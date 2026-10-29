package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fixtureGCPJSON(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]string{"type": "service_account", "project_id": "sample-project", "client_email": "robot@sample-project.iam.gserviceaccount.com", "private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})), "private_key_id": "fixture-key", "token_uri": "https://oauth2.googleapis.com/token"})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func TestProviderCredentialsSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "provider.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	assertProviderCredentials(t, db)
}
func assertProviderCredentials(t *testing.T, db *gorm.DB) {
	t.Helper()
	old := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("k", 32))
	t.Cleanup(func() { viper.Set("tree.password_encryption_key", old) })
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	store := dao.NewCloudAccountDAO(db, zap.NewNop())
	svc := NewCloudAccountService(zap.NewNop(), store).(*cloudAccountService)
	azure := &model.ProviderCredentialInput{TenantID: "11111111-1111-1111-1111-111111111111", ClientID: "22222222-2222-2222-2222-222222222222", SubscriptionID: "33333333-3333-3333-3333-333333333333", ClientSecret: "azure-secret-canary"}
	gcp := &model.ProviderCredentialInput{ProjectID: "sample-project", ServiceAccountJSON: fixtureGCPJSON(t)}
	for _, tc := range []struct {
		name       string
		provider   model.CloudProvider
		credential *model.ProviderCredentialInput
		regions    []model.CreateCloudAccountRegionItem
	}{
		{"azure", model.ProviderAzure, azure, []model.CreateCloudAccountRegionItem{{Region: "eastus", IsDefault: true}, {Region: "westus"}}},
		{"gcp", model.ProviderGCP, gcp, []model.CreateCloudAccountRegionItem{{Region: "us-central1-a", IsDefault: true}, {Region: "us-central1-b"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := &model.CreateCloudAccountReq{Name: "provider-" + tc.name, Provider: tc.provider, ProviderCredentials: tc.credential, Regions: tc.regions}
			if err := svc.CreateCloudAccount(context.Background(), req, 7, "fixture"); err != nil {
				t.Fatal(err)
			}
			var saved model.CloudAccount
			if err := db.Where("name = ?", req.Name).First(&saved).Error; err != nil {
				t.Fatal(err)
			}
			if saved.ProviderCredentials == "" || strings.Contains(saved.ProviderCredentials, "azure-secret-canary") || strings.Contains(saved.ProviderCredentials, "PRIVATE KEY") {
				t.Fatal("missing ciphertext or plaintext stored")
			}
			creds, err := treeUtils.LoadCloudAccountCredentials(&saved)
			if err != nil {
				t.Fatal(err)
			}
			if tc.provider == model.ProviderAzure && creds.AzureClientSecret != azure.ClientSecret || tc.provider == model.ProviderGCP && creds.GCPServiceAccountJSON != gcp.ServiceAccountJSON {
				t.Fatal("credential tuple lost")
			}
			for _, v := range []any{saved, creds} {
				body, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(body), "azure-secret-canary") || strings.Contains(string(body), "PRIVATE KEY") || strings.Contains(string(body), saved.ProviderCredentials) {
					t.Fatal("credential leaked in response")
				}
			}
			detail, err := svc.GetCloudAccountDetail(context.Background(), &model.GetCloudAccountDetailReq{ID: saved.ID})
			if err != nil || detail.ProviderCredentials != "" {
				t.Fatal("credential not sanitized")
			}
			calls := 0
			svc.credentialFactory = func(provider model.CloudProvider, c treeUtils.CloudCredentials, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
				calls++
				if provider != tc.provider {
					return nil, errors.New("wrong provider")
				}
				if provider == model.ProviderAzure && c.AzureClientSecret != azure.ClientSecret || provider == model.ProviderGCP && c.GCPServiceAccountJSON != gcp.ServiceAccountJSON {
					return nil, errors.New("wrong tuple")
				}
				return credentialCloudStub{region: region}, nil
			}
			if err := svc.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: saved.ID}); err != nil {
				t.Fatal(err)
			}
			regions, err := svc.GetAvailableRegionsForAccount(context.Background(), saved.ID)
			if err != nil || len(regions) != 2 || calls != 3 {
				t.Fatalf("region verification failed: %v", err)
			}
			svc.credentialFactory = func(_ model.CloudProvider, _ treeUtils.CloudCredentials, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
				return credentialCloudStub{region: region, fail: region == tc.regions[1].Region}, nil
			}
			partialRegions, regionErr := svc.GetAvailableRegionsForAccount(context.Background(), saved.ID)
			if regionErr == nil || partialRegions != nil || strings.Contains(regionErr.Error(), "secret-canary") {
				t.Fatal("failed region returned partial data or leaked error")
			}
			partial := *tc.credential
			if tc.provider == model.ProviderAzure {
				partial.ClientSecret = ""
			} else {
				partial.ServiceAccountJSON = ""
			}
			if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, ProviderCredentials: &partial}); err == nil {
				t.Fatal("partial credential update accepted")
			}
			replacement := *tc.credential
			if tc.provider == model.ProviderAzure {
				replacement.ClientSecret = "azure-new-secret-canary"
			} else {
				replacement.ServiceAccountJSON = fixtureGCPJSON(t)
			}
			if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, ProviderCredentials: &replacement}); err != nil {
				t.Fatal(err)
			}
			var rotated model.CloudAccount
			if err := db.First(&rotated, saved.ID).Error; err != nil {
				t.Fatal(err)
			}
			if rotated.CredentialVersion != 1 || rotated.ProviderCredentials == saved.ProviderCredentials {
				t.Fatal("credential rotation was not atomic")
			}
			decoded, err := treeUtils.LoadCloudAccountCredentials(&rotated)
			if err != nil {
				t.Fatal(err)
			}
			if tc.provider == model.ProviderAzure && decoded.AzureClientSecret != replacement.ClientSecret || tc.provider == model.ProviderGCP && decoded.GCPServiceAccountJSON != replacement.ServiceAccountJSON {
				t.Fatal("rotation returned wrong tuple")
			}
			// A failed region update must roll back the replacement tuple.
			failedRegion := "eastus2"
			if tc.provider == model.ProviderGCP {
				failedRegion = "us-central1-c"
			}
			trigger := "CREATE TRIGGER reject_provider_region BEFORE INSERT ON cl_tree_cloud_account_region BEGIN SELECT RAISE(ABORT, 'fixture'); END"
			cleanup := "DROP TRIGGER reject_provider_region"
			if db.Dialector.Name() == "mysql" {
				trigger = fmt.Sprintf("ALTER TABLE cl_tree_cloud_account_region ADD CONSTRAINT reject_provider_region CHECK (region <> '%s')", failedRegion)
				cleanup = "ALTER TABLE cl_tree_cloud_account_region DROP CHECK reject_provider_region"
			}
			if err := db.Exec(trigger).Error; err != nil {
				t.Fatal(err)
			}
			bad := *tc.credential
			badRegions := append([]model.CreateCloudAccountRegionItem{}, tc.regions...)
			badRegions = append(badRegions, model.CreateCloudAccountRegionItem{Region: failedRegion})
			err = svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, ProviderCredentials: &bad, Regions: badRegions})
			if err == nil {
				t.Fatal("failed region update committed")
			}
			if err := db.Exec(cleanup).Error; err != nil {
				t.Fatal(err)
			}
			var after model.CloudAccount
			if err := db.First(&after, saved.ID).Error; err != nil {
				t.Fatal(err)
			}
			if after.ProviderCredentials != rotated.ProviderCredentials || after.CredentialVersion != rotated.CredentialVersion {
				t.Fatal("failed transaction changed credentials")
			}
			paused := &pausedAccountStore{CloudAccountDAO: store, read: make(chan struct{}), resume: make(chan struct{})}
			staleService := NewCloudAccountService(zap.NewNop(), paused)
			done := make(chan error, 1)
			go func() {
				done <- staleService.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, ProviderCredentials: tc.credential})
			}()
			select {
			case <-paused.read:
			case <-time.After(5 * time.Second):
				t.Fatal("stale read timeout")
			}
			if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: saved.ID, ProviderCredentials: &replacement}); err != nil {
				close(paused.resume)
				t.Fatal(err)
			}
			close(paused.resume)
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("stale credentials overwrote new")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("rotation timeout")
			}
		})
	}
	// Older Azure rows accepted generic AK/SK but had no working Azure client.
	// Explicit migration must not retain those obsolete encrypted credentials.
	legacyAccess, err := treeUtils.EncryptPassword("legacy-azure-access")
	if err != nil {
		t.Fatal(err)
	}
	legacySecret, err := treeUtils.EncryptPassword("legacy-azure-secret")
	if err != nil {
		t.Fatal(err)
	}
	legacy := &model.CloudAccount{Name: "legacy-azure", Provider: model.ProviderAzure, AccessKey: legacyAccess, SecretKey: legacySecret, Status: model.CloudAccountEnabled}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := treeUtils.LoadCloudAccountCredentials(legacy); err == nil {
		t.Fatal("legacy generic keys accepted as Azure credentials")
	}
	if err := svc.UpdateCloudAccount(context.Background(), &model.UpdateCloudAccountReq{ID: legacy.ID, ProviderCredentials: azure}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(legacy, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if legacy.AccessKey != "" || legacy.SecretKey != "" || legacy.ProviderCredentials == "" {
		t.Fatal("legacy keys were not cleared during conversion")
	}

	if strings.Contains(fmt.Sprintf("%+v %#v", *azure, *azure), azure.ClientSecret) {
		t.Fatal("formatted input leaked secret")
	}
	oversized := &model.CreateCloudAccountReq{Name: "oversized", Provider: model.ProviderAWS, AccessKey: strings.Repeat("a", 500), SecretKey: "0123456789", Regions: []model.CreateCloudAccountRegionItem{{Region: "us-east-1"}}}
	if err := svc.CreateCloudAccount(context.Background(), oversized, 7, "fixture"); err == nil {
		t.Fatal("oversized ciphertext accepted")
	}
}

func TestProviderCredentialFactoryWithoutCloudCalls(t *testing.T) {
	azure := &model.ProviderCredentialInput{TenantID: "11111111-1111-1111-1111-111111111111", ClientID: "22222222-2222-2222-2222-222222222222", SubscriptionID: "33333333-3333-3333-3333-333333333333", ClientSecret: "fixture-secret"}
	gcp := &model.ProviderCredentialInput{ProjectID: "sample-project", ServiceAccountJSON: fixtureGCPJSON(t)}
	for _, tc := range []struct {
		name     string
		provider model.CloudProvider
		input    *model.ProviderCredentialInput
		region   string
	}{{"azure", model.ProviderAzure, azure, "eastus"}, {"gcp", model.ProviderGCP, gcp, "us-central1-a"}} {
		t.Run(tc.name, func(t *testing.T) {
			c := treeUtils.ProviderCredentialsFromInput(tc.input)
			client, err := treeUtils.NewCloudComputeClientWithCredentials(tc.provider, c, tc.region, zap.NewNop())
			if err != nil {
				t.Fatal(err)
			}
			switch tc.provider {
			case model.ProviderAzure:
				if _, ok := client.(*treeUtils.AzureClient); !ok {
					t.Fatal("wrong factory dispatch")
				}
			case model.ProviderGCP:
				if _, ok := client.(*treeUtils.GCPClient); !ok {
					t.Fatal("wrong factory dispatch")
				}
			}
			if _, err := treeUtils.NewCloudComputeClientWithCredentials(tc.provider, c, "../invalid", zap.NewNop()); err == nil {
				t.Fatal("invalid region accepted")
			}
		})
	}
}

func TestProviderRegionAPIValidatesBeforeWrite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "region.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}); err != nil {
		t.Fatal(err)
	}
	account := &model.CloudAccount{Name: "gcp-region", Provider: model.ProviderGCP, Status: model.CloudAccountEnabled}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewCloudAccountRegionService(zap.NewNop(), dao.NewCloudAccountRegionDAO(db, zap.NewNop()), NewCloudAccountService(zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop())))
	if err := svc.CreateCloudAccountRegion(context.Background(), &model.CreateCloudAccountRegionReq{CloudAccountID: account.ID, Region: "eastus"}); err == nil {
		t.Fatal("GCP accepted non-zone region")
	}
	if err := svc.BatchCreateCloudAccountRegion(context.Background(), &model.BatchCreateCloudAccountRegionReq{CloudAccountID: account.ID, Regions: []model.CreateCloudAccountRegionItem{{Region: "us-central1-a"}, {Region: "../bad"}}}); err == nil {
		t.Fatal("GCP accepted invalid batch")
	}
	var count int64
	if err := db.Model(&model.CloudAccountRegion{}).Where("cloud_account_id = ?", account.ID).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("invalid region batch wrote state")
	}
	if err := svc.CreateCloudAccountRegion(context.Background(), &model.CreateCloudAccountRegionReq{CloudAccountID: account.ID, Region: "us-central1-a"}); err != nil {
		t.Fatal(err)
	}
}
