package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	base "github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func driftEncryptionInputs(t *testing.T, cfg *config.Config) {
	t.Helper()
	old := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("global-drift", 3))
	t.Cleanup(func() { viper.Set("tree.password_encryption_key", old) })
	cfg.Tree.PasswordEncryptionKey = strings.Repeat("x", 32)
}
func assertPersistedWithKey(t *testing.T, ciphertext, plaintext, encryptionKey string) {
	t.Helper()
	opened, err := base.DecryptSecretKey(ciphertext, []byte(encryptionKey))
	if err != nil || opened != plaintext || ciphertext == plaintext {
		t.Fatal("service did not persist the intended plaintext with its typed key")
	}
}

type cryptoLocalDAO struct {
	dao.TreeLocalDAO
	stored *model.TreeLocalResource
}

func (d *cryptoLocalDAO) GetByIP(context.Context, string) (*model.TreeLocalResource, error) {
	return nil, nil
}
func (d *cryptoLocalDAO) Create(_ context.Context, row *model.TreeLocalResource) error {
	copy := *row
	copy.ID = 7
	d.stored = &copy
	return nil
}
func (d *cryptoLocalDAO) GetByID(context.Context, int) (*model.TreeLocalResource, error) {
	copy := *d.stored
	return &copy, nil
}

func TestLocalPasswordUsesTypedSnapshotForWriteAndConnectionRead(t *testing.T) {
	encryptionKey := strings.Repeat("0f", 16)
	cfg := &config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}
	store := &cryptoLocalDAO{}
	service, err := NewTreeLocalService(cfg, zap.NewNop(), store)
	if err != nil {
		t.Fatal(err)
	}
	driftEncryptionInputs(t, cfg)
	const plaintext = "local-synthetic-password"
	if err := service.CreateTreeLocal(context.Background(), &model.CreateTreeLocalResourceReq{Name: "fixture", IpAddr: "192.0.2.5", AuthMode: model.AuthModePassword, Password: plaintext}); err != nil {
		t.Fatal(err)
	}
	assertPersistedWithKey(t, store.stored.Password, plaintext, encryptionKey)
	for range 2 {
		connected, err := service.GetTreeLocalForConnection(context.Background(), &model.GetTreeLocalResourceDetailReq{ID: 7})
		if err != nil || connected.Password != plaintext {
			t.Fatal("typed local connection read changed after cfg/global drift")
		}
	}
	wrong, err := NewTreeLocalService(cfg, zap.NewNop(), store)
	if err != nil {
		t.Fatal(err)
	}
	_, err = wrong.GetTreeLocalForConnection(context.Background(), &model.GetTreeLocalResourceDetailReq{ID: 7})
	if err == nil || strings.Contains(err.Error(), plaintext) || strings.Contains(err.Error(), encryptionKey) || strings.Contains(err.Error(), store.stored.Password) {
		t.Fatal("wrong-key read did not fail safely")
	}
}

type cryptoCloudDAO struct {
	dao.TreeCloudDAO
	stored *model.TreeCloudResource
	logged chan struct{}
}

func (d *cryptoCloudDAO) GetByID(context.Context, int) (*model.TreeCloudResource, error) {
	copy := *d.stored
	return &copy, nil
}
func (d *cryptoCloudDAO) UpdateMetadata(_ context.Context, _ int, metadata map[string]any) error {
	if value, ok := metadata["password"].(string); ok {
		d.stored.Password = value
	}
	return nil
}
func (d *cryptoCloudDAO) CreateChangeLog(context.Context, *model.CloudResourceChangeLog) error {
	d.logged <- struct{}{}
	return nil
}

func TestCloudSSHPasswordUsesTypedSnapshotForWriteAndConnectionRead(t *testing.T) {
	encryptionKey := strings.Repeat("0f", 16)
	cfg := &config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}
	store := &cryptoCloudDAO{stored: &model.TreeCloudResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModePassword}, logged: make(chan struct{}, 1)}
	service, err := NewTreeCloudService(cfg, zap.NewNop(), store, nil)
	if err != nil {
		t.Fatal(err)
	}
	driftEncryptionInputs(t, cfg)
	const plaintext = "cloud-synthetic-password"
	if err := service.UpdateTreeCloudResource(context.Background(), &model.UpdateTreeCloudResourceReq{ID: 7, Password: plaintext}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-store.logged: // All fixture audit goroutines finish before the test leaves.
	case <-time.After(5 * time.Second):
		t.Fatal("fixture audit did not finish")
	}
	assertPersistedWithKey(t, store.stored.Password, plaintext, encryptionKey)
	connected, err := service.GetTreeCloudResourceForConnection(context.Background(), &model.GetTreeCloudResourceDetailReq{ID: 7})
	if err != nil || connected.Password != plaintext {
		t.Fatal("typed cloud connection read changed after cfg/global drift")
	}
}

type cryptoAccountDAO struct{ initialAccountDAO }

func (d *cryptoAccountDAO) GetByID(context.Context, int) (*model.CloudAccount, error) {
	copy := *d.account
	copy.Regions = []*model.CloudAccountRegion{{Region: "cn-hangzhou", IsDefault: true, Status: model.CloudAccountRegionEnabled}}
	return &copy, nil
}

func TestAccountCredentialWriteAndCloudFactoryUseTypedSnapshot(t *testing.T) {
	encryptionKey := strings.Repeat("0f", 16)
	cfg := &config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: encryptionKey}}
	store := &cryptoAccountDAO{}
	service, err := NewCloudAccountService(cfg, zap.NewNop(), store)
	if err != nil {
		t.Fatal(err)
	}
	typed := service.(*cloudAccountService)
	driftEncryptionInputs(t, cfg)
	const access = "synthetic-access-key"
	const secret = "synthetic-secret-key"
	req := &model.CreateCloudAccountReq{Name: "typed-fixture", Provider: model.ProviderAliyun, AccessKey: access, SecretKey: secret, Regions: []model.CreateCloudAccountRegionItem{{Region: "cn-hangzhou", IsDefault: true}}}
	if err := service.CreateCloudAccount(context.Background(), req, 7, "fixture"); err != nil {
		t.Fatal(err)
	}
	assertPersistedWithKey(t, store.account.AccessKey, access, encryptionKey)
	assertPersistedWithKey(t, store.account.SecretKey, secret, encryptionKey)
	called := false
	typed.credentialFactory = func(_ model.CloudProvider, credentials treeUtils.CloudCredentials, _ string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
		called = true
		if credentials.AccessKey != access || credentials.SecretKey != secret {
			t.Fatal("cloud factory did not receive the stored plaintext tuple")
		}
		return credentialCloudStub{}, nil
	}
	if err := typed.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: 7}); err != nil || !called {
		t.Fatal("typed account read/factory boundary failed")
	}
	wrong, err := NewCloudAccountService(cfg, zap.NewNop(), store)
	if err != nil {
		t.Fatal(err)
	}
	called = false
	wrong.(*cloudAccountService).credentialFactory = typed.credentialFactory
	err = wrong.VerifyCloudAccount(context.Background(), &model.VerifyCloudAccountReq{ID: 7})
	if err == nil || called || strings.Contains(err.Error(), access) || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), encryptionKey) || strings.Contains(err.Error(), store.account.SecretKey) {
		t.Fatal("wrong-key account read did not fail safely before the cloud factory")
	}
}

func TestManagedCredentialReadUsesTypedSnapshotBeforeFakeFactory(t *testing.T) {
	_, template, account, region := managedServiceFixture(t)
	cfg := &config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: template.passwordEncryptionKey}}
	service, err := NewManagedCloudService(cfg, zap.NewNop(), template.accounts, template.regions, template.store)
	if err != nil {
		t.Fatal(err)
	}
	typed := service.(*managedCloudService)
	called := false
	typed.factory = func(credentials treeUtils.CloudCredentials, region string) (managedReader, error) {
		called = true
		if credentials.AccessKey != "test-ak" || credentials.SecretKey != "test-sk" {
			t.Fatal("managed factory did not receive typed-key plaintext credentials")
		}
		return template.factory(credentials, region)
	}
	driftEncryptionInputs(t, cfg)
	result, err := service.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"typed-one"}, OperatorID: 7})
	if err != nil || result.Status != "success" || !called {
		t.Fatal("managed typed snapshot read failed")
	}
}

func TestCryptoServiceConstructorsRejectNilConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		create func() (any, error)
	}{
		{"local", func() (any, error) { return NewTreeLocalService(nil, nil, nil) }},
		{"cloud", func() (any, error) { return NewTreeCloudService(nil, nil, nil, nil) }},
		{"account", func() (any, error) { return NewCloudAccountService(nil, nil, nil) }},
		{"managed", func() (any, error) { return NewManagedCloudService(nil, nil, nil, nil, nil) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value, err := tc.create()
			if err == nil || value != nil {
				t.Fatal("nil config was silently accepted")
			}
		})
	}
}
