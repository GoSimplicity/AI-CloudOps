package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type credentialCloudStub struct {
	fail   bool
	region string
}

type expiryCloudStub struct{ credentialCloudStub }

func (expiryCloudStub) VerifyCredentials(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }

func (s credentialCloudStub) VerifyCredentials(context.Context) error {
	if s.fail {
		return errors.New("secret-canary")
	}
	return nil
}
func (s credentialCloudStub) GetAvailableRegions(context.Context) ([]model.AvailableRegion, error) {
	if s.fail {
		return nil, errors.New("secret-canary")
	}
	region := s.region
	if region == "" {
		region = "test"
	}
	return []model.AvailableRegion{{Region: region, Available: true}}, nil
}
func (s credentialCloudStub) ListInstances(context.Context, []string) ([]*model.TreeCloudResource, error) {
	return []*model.TreeCloudResource{}, nil
}

func TestTemporaryCredentialsSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "temporary.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	assertTemporaryCredentials(t, db)
}

func assertTemporaryCredentials(t *testing.T, db *gorm.DB) {
	t.Helper()
	encryptionKey := strings.Repeat("k", 32)
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}, &model.CloudRegionSyncLease{}, &model.CloudResourceSyncHistory{}); err != nil {
		t.Fatal(err)
	}
	store := dao.NewCloudAccountDAO(db, zap.NewNop())
	svc := mustCloudAccountService(t, encryptionKey, zap.NewNop(), store).(*cloudAccountService)
	expiry := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	token := strings.Repeat("temporary-token-canary", 100)
	req := &model.CreateCloudAccountReq{Name: "temporary-fixture", Provider: model.ProviderAWS, AccessKey: "ASIAFAKEKEY123456", SecretKey: "private-secret-canary", CredentialType: "temporary", SessionToken: token, CredentialExpiresAt: &expiry, Regions: []model.CreateCloudAccountRegionItem{{Region: "us-test-1", IsDefault: true}}}
	if err := svc.CreateCloudAccount(context.Background(), req, 7, "tester"); err != nil {
		t.Fatal(err)
	}
	var saved model.CloudAccount
	if err := db.Where("name = ?", req.Name).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.SessionToken == "" || strings.Contains(saved.SessionToken, "temporary-token-canary") || strings.Contains(saved.AccessKey, "ASIAFAKE") {
		t.Fatal("credential stored in plaintext")
	}
	for _, value := range []any{saved, treeUtils.CloudCredentials{AccessKey: req.AccessKey, SecretKey: req.SecretKey, SessionToken: token}} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, secret := range []string{req.AccessKey, req.SecretKey, token, saved.AccessKey, saved.SecretKey, saved.SessionToken} {
			if secret != "" && strings.Contains(string(encoded), secret) {
				t.Fatal("credential leaked in JSON")
			}
		}
	}
	seen := 0
	svc.credentialFactory = func(provider model.CloudProvider, credentials treeUtils.CloudCredentials, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
		seen++
		if provider != model.ProviderAWS || credentials.SessionToken != token || credentials.AccessKey != req.AccessKey || region != "us-test-1" {
			t.Fatal("incomplete credentials reached cloud factory")
		}
		return credentialCloudStub{}, nil
	}
	if err := svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: saved.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetAvailableRegionsForAccount(context.Background(), saved.ID); err != nil || seen != 2 {
		t.Fatalf("region query failed: %v", err)
	}
	syncService := &treeCloudService{passwordEncryptionKey: encryptionKey, logger: zap.NewNop(), cloudAccountDAO: store, dao: dao.NewTreeCloudDAO(db, zap.NewNop()), credentialFactory: svc.credentialFactory}
	if _, err := syncService.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: saved.ID, SyncMode: model.SyncModeIncremental}); err != nil || seen != 3 {
		t.Fatalf("temporary sync failed: %v", err)
	}
	svc.credentialFactory = func(model.CloudProvider, treeUtils.CloudCredentials, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return nil, errors.New(token)
	}
	if err := svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: saved.ID}); err == nil || strings.Contains(err.Error(), token) {
		t.Fatal("provider factory error leaked credential")
	}
	svc.credentialFactory = syncService.credentialFactory
	if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, AccessKey: "ASIANEWPARTIAL"}); err == nil {
		t.Fatal("partial temporary rotation accepted")
	}
	if err := db.Model(&saved).Update("credential_expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: saved.ID}); err == nil || seen != 3 {
		t.Fatal("expired credentials reached cloud factory")
	}
	newExpiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	rotation := &model.UpdateCloudAccountReq{ID: saved.ID, AccessKey: "ASIAnew", SecretKey: "new-secret-canary", CredentialType: "temporary", SessionToken: "new-token-canary", CredentialExpiresAt: &newExpiry}
	if err := svc.UpdateCloudAccount(context.Background(), rotation); err != nil {
		t.Fatal(err)
	}
	var rotated model.CloudAccount
	if err := db.First(&rotated, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	decoded, err := treeUtils.LoadCloudAccountCredentials(&rotated, encryptionKey)
	if err != nil || decoded.AccessKey != rotation.AccessKey || decoded.SecretKey != rotation.SecretKey || decoded.SessionToken != rotation.SessionToken || rotated.CredentialVersion != 1 {
		t.Fatal("rotation lost credential tuple")
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", decoded, decoded), rotation.SessionToken) {
		t.Fatal("formatted credentials leaked")
	}

	// 凭据与地域同一事务；后续地域插入失败必须回滚整个凭据组。
	trigger := "CREATE TRIGGER reject_credential_region BEFORE INSERT ON cl_tree_cloud_account_region BEGIN SELECT RAISE(ABORT, 'fixture'); END"
	cleanup := "DROP TRIGGER reject_credential_region"
	if db.Dialector.Name() == "mysql" {
		trigger = "ALTER TABLE cl_tree_cloud_account_region ADD CONSTRAINT reject_credential_region CHECK (region <> 'region-rejected')"
		cleanup = "ALTER TABLE cl_tree_cloud_account_region DROP CHECK reject_credential_region"
	}
	if err := db.Exec(trigger).Error; err != nil {
		t.Fatal(err)
	}
	failedRotation := *rotation
	failedRotation.SessionToken = "failed-token-canary"
	failedRotation.Regions = []model.CreateCloudAccountRegionItem{{Region: "us-test-1", IsDefault: true}, {Region: "region-rejected"}}
	if err := svc.UpdateCloudAccount(context.Background(), &failedRotation); err == nil {
		t.Fatal("injected region failure accepted")
	}
	if err := db.Exec(cleanup).Error; err != nil {
		t.Fatal(err)
	}
	var afterFailure model.CloudAccount
	if err := db.First(&afterFailure, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	if afterFailure.SessionToken != rotated.SessionToken || afterFailure.CredentialVersion != rotated.CredentialVersion {
		t.Fatal("credential update escaped failed transaction")
	}

	// 使用受控屏障构造旧读并发轮换，CAS 必须阻止新旧密钥混合。
	paused := &pausedAccountStore{CloudAccountDAO: store, read: make(chan struct{}), resume: make(chan struct{})}
	staleService := mustCloudAccountService(t, encryptionKey, zap.NewNop(), paused)
	stale := *rotation
	stale.SessionToken = "stale-token-canary"
	newest := *rotation
	newest.SessionToken = "newest-token-canary"
	done := make(chan error, 1)
	go func() { done <- staleService.UpdateCloudAccount(context.Background(), &stale) }()
	select {
	case <-paused.read:
	case <-time.After(5 * time.Second):
		t.Fatal("stale read did not complete")
	}
	if err := svc.UpdateCloudAccount(context.Background(), &newest); err != nil {
		close(paused.resume)
		t.Fatal(err)
	}
	close(paused.resume)
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("stale rotation overwrote current credentials")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stale rotation did not return")
	}
	var latest model.CloudAccount
	if err := db.First(&latest, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	credentials, err := treeUtils.LoadCloudAccountCredentials(&latest, encryptionKey)
	if err != nil || credentials.SessionToken != newest.SessionToken || latest.CredentialVersion != 2 {
		t.Fatal("concurrent rotation lost latest tuple")
	}

	// 模式切换必须提交完整新密钥，并在同一事务清除旧令牌和到期时间。
	if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, CredentialType: "long_term"}); err == nil {
		t.Fatal("incomplete type switch accepted")
	}
	if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, CredentialType: "long_term", AccessKey: "LONGkey", SecretKey: "long-secret-canary"}); err != nil {
		t.Fatal(err)
	}
	var long model.CloudAccount
	if err := db.First(&long, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	longCred, err := treeUtils.LoadCloudAccountCredentials(&long, encryptionKey)
	if err != nil || longCred.SessionToken != "" || longCred.ExpiresAt != nil || longCred.CredentialType != "long_term" {
		t.Fatal("switch retained temporary credential")
	}
	if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, SecretKey: "partial-long-secret"}); err != nil {
		t.Fatal("legacy partial long-term rotation failed")
	}

	// 老数据没有 credential_type、到期时间和令牌，仍使用原加密算法。
	if err := db.Model(&long).Update("credential_type", "").Error; err != nil {
		t.Fatal(err)
	}
	var legacy model.CloudAccount
	if err := db.First(&legacy, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	legacyCred, err := treeUtils.LoadCloudAccountCredentials(&legacy, encryptionKey)
	if err != nil || legacyCred.SecretKey != "partial-long-secret" || legacyCred.SessionToken != "" {
		t.Fatal("legacy credential migration failed")
	}
}

func TestHuaweiUnboundedSyncRejectsBeforeProviderOrResourceDAO(t *testing.T) {
	for _, mode := range []model.SyncMode{model.SyncModeFull, model.SyncModeIncremental} {
		account := &model.CloudAccount{Model: model.Model{ID: 7}, Provider: model.ProviderHuawei, Status: model.CloudAccountEnabled}
		svc := &treeCloudService{logger: zap.NewNop(), cloudAccountDAO: storedCloudAccount{account: account}}
		_, err := svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: 7, SyncMode: mode})
		if err == nil || !strings.Contains(err.Error(), "指定实例ID") {
			t.Fatalf("unbounded Huawei %s accepted: %v", mode, err)
		}
	}
}

func TestTemporaryFullSyncAtomicAndExpiredCredentials(t *testing.T) {
	encryptionKey := strings.Repeat("k", 32)
	for _, provider := range []model.CloudProvider{model.ProviderAWS, model.ProviderTencent} {
		t.Run(fmt.Sprint(provider), func(t *testing.T) {
			db, svc, account, region, _ := regionSyncFixture(t)
			svc.passwordEncryptionKey = encryptionKey
			key := "test-ak"
			if provider == model.ProviderAWS {
				key = "ASIAx"
			} else {
				region.Region = "ap-shanghai"
			}
			access, err := treeUtils.EncryptPassword(key, encryptionKey)
			if err != nil {
				t.Fatal(err)
			}
			secret, err := treeUtils.EncryptPassword("test-sk", encryptionKey)
			if err != nil {
				t.Fatal(err)
			}
			token, err := treeUtils.EncryptPassword("test-token", encryptionKey)
			if err != nil {
				t.Fatal(err)
			}
			expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
			if err := db.Model(account).Updates(map[string]any{"provider": provider, "access_key": access, "secret_key": secret, "credential_type": "temporary", "session_token": token, "credential_expires_at": expires}).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Model(region).Update("region", region.Region).Error; err != nil {
				t.Fatal(err)
			}
			old := resource(account, region, "old")
			if err := db.Create(old).Error; err != nil {
				t.Fatal(err)
			}
			calls := 0
			svc.cloudAccountDAO = dao.NewCloudAccountDAO(db, zap.NewNop())
			svc.credentialFactory = func(got model.CloudProvider, c treeUtils.CloudCredentials, _ string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
				calls++
				if got != provider || c.SessionToken != "test-token" || c.CredentialType != "temporary" {
					t.Fatal("temporary credential not propagated")
				}
				return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
					return []*model.TreeCloudResource{resource(account, region, "new")}, nil
				}}, nil
			}
			callback := "test:temporary-log-failure"
			if err := db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
				if _, ok := tx.Statement.Dest.(*model.CloudResourceChangeLog); ok {
					tx.AddError(errors.New("injected failure"))
				}
			}); err != nil {
				t.Fatal(err)
			}
			result, err := svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: account.ID, SyncMode: model.SyncModeFull})
			if err != nil || result.Status != "failed" || result.DeleteCount != 0 || result.NewCount != 0 {
				t.Fatalf("failed sync result=%+v err=%v", result, err)
			}
			var n int64
			db.Model(&model.TreeCloudResource{}).Where("id = ?", old.ID).Count(&n)
			if n != 1 {
				t.Fatal("failed full sync deleted old resource")
			}
			if err := db.Callback().Create().Remove(callback); err != nil {
				t.Fatal(err)
			}
			result, err = svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: account.ID, SyncMode: model.SyncModeFull})
			if err != nil || result.Status != "success" || result.NewCount != 1 || result.DeleteCount != 1 || calls != 2 {
				t.Fatalf("full sync result=%+v calls=%d err=%v", result, calls, err)
			}
			expired := time.Now().UTC().Add(-time.Minute)
			if err := db.Model(account).Update("credential_expires_at", expired).Error; err != nil {
				t.Fatal(err)
			}
			if _, err := svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: account.ID, SyncMode: model.SyncModeFull}); err == nil || calls != 2 {
				t.Fatal("expired credential reached provider")
			}
		})
	}
}

func TestCredentialExpiryCancelsInFlightCloudRequest(t *testing.T) {
	encryptionKey := strings.Repeat("k", 32)
	access, err := treeUtils.EncryptPassword("ASIAFAKEKEY123456", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("secret-fixture", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	token, err := treeUtils.EncryptPassword("token-fixture", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Now().UTC().Add(500 * time.Millisecond)
	account := &model.CloudAccount{Model: model.Model{ID: 7}, Provider: model.ProviderAWS, Status: model.CloudAccountEnabled, CredentialType: "temporary", CredentialExpiresAt: &expires, AccessKey: access, SecretKey: secret, SessionToken: token, Regions: []*model.CloudAccountRegion{{Region: "us-test", IsDefault: true}}}
	svc := &cloudAccountService{passwordEncryptionKey: encryptionKey, logger: zap.NewNop(), dao: storedCloudAccount{account: account}, credentialFactory: func(model.CloudProvider, treeUtils.CloudCredentials, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return expiryCloudStub{}, nil
	}}
	if err := svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: 7}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expiry did not cancel call: %v", err)
	}
}

func TestHuaweiRegionProjectsAreExplicitAndNoPartialResults(t *testing.T) {
	encryptionKey := strings.Repeat("k", 32)
	access, err := treeUtils.EncryptPassword("access-fixture", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("secret-fixture", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	p1, p2 := "project_one", "project_two"
	account := &model.CloudAccount{Model: model.Model{ID: 7}, Provider: model.ProviderHuawei, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret, Regions: []*model.CloudAccountRegion{
		{Model: model.Model{ID: 11}, CloudAccountID: 7, Region: "cn-one", ProjectID: &p1, Status: model.CloudAccountRegionEnabled, IsDefault: true},
		{Model: model.Model{ID: 12}, CloudAccountID: 7, Region: "cn-two", ProjectID: &p2, Status: model.CloudAccountRegionEnabled},
	}}
	seen := make(map[string]string)
	fail := false
	factory := func(_ model.CloudProvider, credentials treeUtils.CloudCredentials, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
		seen[region] = credentials.ProjectID
		return credentialCloudStub{fail: fail && region == "cn-two", region: region}, nil
	}
	svc := &cloudAccountService{passwordEncryptionKey: encryptionKey, logger: zap.NewNop(), dao: storedCloudAccount{account: account}, credentialFactory: factory}
	regions, err := svc.GetAvailableRegionsForAccount(context.Background(), 7)
	if err != nil || len(regions) != 2 || seen["cn-one"] != p1 || seen["cn-two"] != p2 {
		t.Fatalf("project mapping failed: %v", err)
	}
	fail = true
	regions, err = svc.GetAvailableRegionsForAccount(context.Background(), 7)
	if err == nil || regions != nil {
		t.Fatal("partial Huawei region results reported as complete")
	}
	account.Regions[1].ProjectID = nil
	seen = make(map[string]string)
	if _, err := svc.GetAvailableRegionsForAccount(context.Background(), 7); err == nil || len(seen) != 0 {
		t.Fatal("missing project reached provider")
	}
}
