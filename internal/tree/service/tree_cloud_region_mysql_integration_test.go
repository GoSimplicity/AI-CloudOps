//go:build integration

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMySQLCloudRegionSyncFencingAndAtomicRollback(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("explicit isolated MySQL required")
	}
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated MySQL")
	}
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeNode{}, &model.TreeCloudResource{}, &model.CloudResourceChangeLog{}, &model.CloudResourceSyncHistory{}, &model.CloudRegionSyncLease{}); err != nil {
		t.Fatal(err)
	}
	account := &model.CloudAccount{Name: "fixture", Provider: model.ProviderAWS, Status: model.CloudAccountEnabled, AccessKey: "cipher", SecretKey: "cipher"}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "us-west-2", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(region).Error; err != nil {
		t.Fatal(err)
	}
	existing := resource(account, region, "same")
	existing.Name = "before"
	existing.Password = "private-cipher"
	existing.Username = "ssh-user"
	if err := db.Create(existing).Error; err != nil {
		t.Fatal(err)
	}
	svc := &treeCloudService{dao: dao.NewTreeCloudDAO(db, zap.NewNop()), logger: zap.NewNop()}
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		index := calls.Add(1)
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			r := resource(account, region, "same")
			if index == 1 {
				close(started)
				<-release
				r.Name = "stale"
			} else {
				r.Name = "fresh"
			}
			return []*model.TreeCloudResource{r}, nil
		}}, nil
	}
	credentials := treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}
	req := &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}
	done := make(chan error, 1)
	go func() {
		_, err := svc.syncOneRegion(context.Background(), account, credentials, region, req)
		done <- err
	}()
	<-started
	if _, err := svc.syncOneRegion(context.Background(), account, credentials, region, req); !errors.Is(err, dao.ErrRegionSyncBusy) {
		t.Fatalf("expected busy: %v", err)
	}
	if err := db.Model(&model.CloudRegionSyncLease{}).Where("region_id = ?", region.ID).Update("expires_epoch_ms", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.syncOneRegion(context.Background(), account, credentials, region, req); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; !errors.Is(err, dao.ErrRegionSyncLeaseLost) {
		t.Fatalf("stale writer: %v", err)
	}
	var current model.TreeCloudResource
	if err := db.First(&current, existing.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Name != "fresh" || current.Password != "private-cipher" || current.Username != "ssh-user" {
		t.Fatal("fencing or SSH preservation failed")
	}
	var histories []model.CloudResourceSyncHistory
	if err := db.Where("region_id = ?", region.ID).Find(&histories).Error; err != nil {
		t.Fatal(err)
	}
	counts := map[model.SyncStatus]int{}
	for _, h := range histories {
		counts[h.SyncStatus]++
	}
	if counts["success"] != 1 || counts["failed"] != 2 {
		t.Fatalf("history status counts=%v", counts)
	}
	crashed := model.CloudResourceSyncHistory{CloudAccountID: account.ID, RegionID: region.ID, Region: region.Region, OperationID: "crash-fixture", LeaseExpiresMS: 1, SyncMode: model.SyncModeFull, SyncStatus: "running", StartTime: time.Now().UTC()}
	if err := db.Create(&crashed).Error; err != nil {
		t.Fatal(err)
	}
	projected, _, err := svc.dao.GetSyncHistoryList(context.Background(), &model.GetCloudResourceSyncHistoryReq{ListReq: model.ListReq{Page: 1, Size: 20}, SyncStatus: "unknown"})
	if err != nil || len(projected) != 1 || projected[0].SyncStatus != "unknown" || projected[0].RetrySafe == nil || *projected[0].RetrySafe {
		t.Fatal("crashed history was not safely projected")
	}
	var persisted model.CloudResourceSyncHistory
	if err := db.First(&persisted, crashed.ID).Error; err != nil || persisted.SyncStatus != "running" {
		t.Fatal("history read mutated state")
	}
	callback := "test:fail-region-log"
	if err := db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if _, ok := tx.Statement.Dest.(*model.CloudResourceChangeLog); ok {
			tx.AddError(errors.New("injected log failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Create().Remove(callback)
	if _, err := svc.syncOneRegion(context.Background(), account, credentials, region, req); err == nil {
		t.Fatal("missing injected failure")
	}
	if err := db.First(&current, existing.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.Name != "fresh" {
		t.Fatal("log failure failed to roll back changes")
	}
}

func TestMySQLTemporaryFullSyncAtomic(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("explicit isolated MySQL required")
	}
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated MySQL")
	}
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeNode{}, &model.TreeCloudResource{}, &model.CloudResourceChangeLog{}, &model.CloudResourceSyncHistory{}, &model.CloudRegionSyncLease{}); err != nil {
		t.Fatal(err)
	}
	encryptionKey := strings.Repeat("v", 32)
	for _, provider := range []model.CloudProvider{model.ProviderAWS, model.ProviderTencent} {
		key := "test-ak"
		regionCode := "ap-shanghai"
		if provider == model.ProviderAWS {
			key = "ASIAx"
			regionCode = "us-west-2"
		}
		access, _ := treeUtils.EncryptPassword(key, encryptionKey)
		secret, _ := treeUtils.EncryptPassword("test-sk", encryptionKey)
		token, _ := treeUtils.EncryptPassword("test-token", encryptionKey)
		expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
		account := &model.CloudAccount{Name: fmt.Sprintf("fixture-%d", provider), Provider: provider, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret, CredentialType: "temporary", SessionToken: token, CredentialExpiresAt: &expires}
		if err := db.Create(account).Error; err != nil {
			t.Fatal(err)
		}
		region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: regionCode, Status: model.CloudAccountRegionEnabled}
		if err := db.Create(region).Error; err != nil {
			t.Fatal(err)
		}
		old := resource(account, region, fmt.Sprintf("old-%d", provider))
		old.Password = "cipher-kept"
		if err := db.Create(old).Error; err != nil {
			t.Fatal(err)
		}
		svc := &treeCloudService{passwordEncryptionKey: encryptionKey, dao: dao.NewTreeCloudDAO(db, zap.NewNop()), cloudAccountDAO: dao.NewCloudAccountDAO(db, zap.NewNop()), logger: zap.NewNop()}
		svc.credentialFactory = func(got model.CloudProvider, c treeUtils.CloudCredentials, _ string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
			if got != provider || c.SessionToken != "test-token" {
				t.Fatal("temporary token not propagated")
			}
			return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
				r := resource(account, region, old.InstanceID)
				r.Name = "fresh"
				return []*model.TreeCloudResource{r}, nil
			}}, nil
		}
		callback := fmt.Sprintf("test:temporary-audit-%d", provider)
		if err := db.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
			if _, ok := tx.Statement.Dest.(*model.CloudResourceChangeLog); ok {
				tx.AddError(errors.New("injected audit failure"))
			}
		}); err != nil {
			t.Fatal(err)
		}
		resp, err := svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: account.ID, SyncMode: model.SyncModeFull})
		if err != nil || resp.Status != "failed" {
			t.Fatalf("failure status=%+v err=%v", resp, err)
		}
		var row model.TreeCloudResource
		if err := db.First(&row, old.ID).Error; err != nil || row.Name == "fresh" {
			t.Fatal("failed transaction changed resource")
		}
		if err := db.Callback().Create().Remove(callback); err != nil {
			t.Fatal(err)
		}
		resp, err = svc.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: account.ID, SyncMode: model.SyncModeFull})
		if err != nil || resp.Status != "success" || resp.UpdateCount != 1 {
			t.Fatalf("success status=%+v err=%v", resp, err)
		}
		row = model.TreeCloudResource{}
		if err := db.First(&row, old.ID).Error; err != nil || row.Name != "fresh" || row.Password != "cipher-kept" {
			t.Fatal("local credentials or successful update incorrect")
		}
	}
}
