package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	base "github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type passwordStorageLocalDAO struct {
	dao.TreeLocalDAO
	row              model.TreeLocalResource
	creates, updates int
}

func (d *passwordStorageLocalDAO) GetByIP(context.Context, string) (*model.TreeLocalResource, error) {
	return nil, nil
}
func (d *passwordStorageLocalDAO) GetByID(context.Context, int) (*model.TreeLocalResource, error) {
	copy := d.row
	return &copy, nil
}
func (d *passwordStorageLocalDAO) Create(_ context.Context, row *model.TreeLocalResource) error {
	d.creates++
	d.row = *row
	return nil
}
func (d *passwordStorageLocalDAO) Update(_ context.Context, _ int, changes map[string]any) error {
	d.updates++
	if value, ok := changes["password"].(string); ok {
		d.row.Password = value
	}
	if value, ok := changes["key"].(string); ok {
		d.row.Key = value
	}
	return nil
}

type passwordStorageCloudDAO struct {
	dao.TreeCloudDAO
	row     model.TreeCloudResource
	updates int
	audit   chan struct{}
}

func (d *passwordStorageCloudDAO) GetByID(context.Context, int) (*model.TreeCloudResource, error) {
	copy := d.row
	return &copy, nil
}
func (d *passwordStorageCloudDAO) UpdateMetadata(_ context.Context, _ int, changes map[string]any) error {
	d.updates++
	if value, ok := changes["password"].(string); ok {
		d.row.Password = value
	}
	if value, ok := changes["key"].(string); ok {
		d.row.Key = value
	}
	return nil
}
func (d *passwordStorageCloudDAO) CreateChangeLog(context.Context, *model.CloudResourceChangeLog) error {
	d.audit <- struct{}{}
	return nil
}
func waitPasswordAudit(t *testing.T, audit chan struct{}, count int) {
	t.Helper()
	for range count {
		select {
		case <-audit:
		case <-time.After(3 * time.Second):
			t.Fatal("owned audit fixture did not finish")
		}
	}
}

func TestSSHPasswordStorageBoundaryBeforeEveryDAOChange(t *testing.T) {
	key := strings.Repeat("0f", 16)
	for _, path := range []string{"local-create", "local-update", "cloud-update"} {
		for _, tc := range []struct {
			name, password string
			valid          bool
			length         int
		}{
			{"ASCII-at-limit", strings.Repeat("p", 347), true, 500},
			{"ASCII-over-limit", strings.Repeat("p", 348), false, 504},
			{"UTF8-at-limit", strings.Repeat("界", 115) + "ab", true, 500},
			{"UTF8-over-limit", strings.Repeat("界", 116), false, 504},
			{"existing-masked-literal", "******", true, 48},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				cfg := &config.Config{Tree: config.TreeConfig{PasswordEncryptionKey: key}}
				core, logs := observer.New(zap.DebugLevel)
				logger := zap.New(core)
				local := &passwordStorageLocalDAO{row: model.TreeLocalResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModePassword, Password: "original-ciphertext"}}
				cloud := &passwordStorageCloudDAO{row: model.TreeCloudResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModePassword, Password: "original-ciphertext"}, audit: make(chan struct{}, 4)}
				var err error
				switch path {
				case "local-create":
					service, construction := NewTreeLocalService(cfg, logger, local)
					if construction != nil {
						t.Fatal(construction)
					}
					err = service.CreateTreeLocal(context.Background(), &model.CreateTreeLocalResourceReq{Name: "fixture", IpAddr: "192.0.2.5", AuthMode: model.AuthModePassword, Password: tc.password})
				case "local-update":
					service, construction := NewTreeLocalService(cfg, logger, local)
					if construction != nil {
						t.Fatal(construction)
					}
					err = service.UpdateTreeLocal(context.Background(), &model.UpdateTreeLocalResourceReq{ID: 7, Password: tc.password})
				case "cloud-update":
					service, construction := NewTreeCloudService(cfg, logger, cloud, nil)
					if construction != nil {
						t.Fatal(construction)
					}
					err = service.UpdateTreeCloudResource(context.Background(), &model.UpdateTreeCloudResourceReq{ID: 7, Password: tc.password})
				}
				writes := local.creates + local.updates + cloud.updates
				if !tc.valid {
					if err == nil || err.Error() != "SSH密码超出加密存储限制" || writes != 0 {
						t.Fatal("oversized ciphertext was not rejected before the DAO")
					}
					if local.row.Password != "original-ciphertext" || cloud.row.Password != "original-ciphertext" || len(cloud.audit) != 0 {
						t.Fatal("rejected password changed stored material or generated an audit")
					}
					for _, entry := range logs.All() {
						if strings.Contains(entry.Message, tc.password) || strings.Contains(entry.Message, key) {
							t.Fatal("length failure logged private material")
						}
					}
					return
				}
				if err != nil || writes != 1 {
					t.Fatal("in-budget password was not saved")
				}
				ciphertext := local.row.Password
				if path == "cloud-update" {
					ciphertext = cloud.row.Password
					waitPasswordAudit(t, cloud.audit, 1)
				}
				if len(ciphertext) != tc.length || len(ciphertext) > 500 {
					t.Fatal("ciphertext did not match the storage budget")
				}
				actual, err := base.DecryptSecretKey(ciphertext, []byte(key))
				if err != nil || actual != tc.password {
					t.Fatal("saved credential did not preserve the original plaintext and key format")
				}
			})
		}
	}
}

func TestSSHPasswordStorageGuardsKeepEmptyAndKeyAuthSemantics(t *testing.T) {
	for _, path := range []string{"local-create", "local-update", "cloud-update"} {
		t.Run(path, func(t *testing.T) {
			cfg := &config.Config{} // Key authentication must not need an AES key.
			local := &passwordStorageLocalDAO{row: model.TreeLocalResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModeKey, Password: "original-ciphertext", Key: "old-key"}}
			cloud := &passwordStorageCloudDAO{row: model.TreeCloudResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModeKey, Password: "original-ciphertext", Key: "old-key"}, audit: make(chan struct{}, 4)}
			var err error
			switch path {
			case "local-create":
				service, construction := NewTreeLocalService(cfg, zap.NewNop(), local)
				if construction != nil {
					t.Fatal(construction)
				}
				err = service.CreateTreeLocal(context.Background(), &model.CreateTreeLocalResourceReq{Name: "fixture", IpAddr: "192.0.2.5", AuthMode: model.AuthModeKey, Key: "new-key", Password: strings.Repeat("unused", 100)})
			case "local-update":
				service, construction := NewTreeLocalService(cfg, zap.NewNop(), local)
				if construction != nil {
					t.Fatal(construction)
				}
				err = service.UpdateTreeLocal(context.Background(), &model.UpdateTreeLocalResourceReq{ID: 7, Key: "new-key", Password: strings.Repeat("unused", 100)})
			case "cloud-update":
				service, construction := NewTreeCloudService(cfg, zap.NewNop(), cloud, nil)
				if construction != nil {
					t.Fatal(construction)
				}
				err = service.UpdateTreeCloudResource(context.Background(), &model.UpdateTreeCloudResourceReq{ID: 7, Key: "new-key"})
				waitPasswordAudit(t, cloud.audit, 1)
			}
			if err != nil {
				t.Fatal("key-only write unexpectedly required a password cipher key")
			}
			if path == "local-create" {
				if local.row.Password != "" || local.row.Key != "new-key" {
					t.Fatal("key-only create started storing the unused password")
				}
			} else if path == "local-update" {
				if local.row.Password != "original-ciphertext" || local.row.Key != "new-key" {
					t.Fatal("key-only local update changed the stored password")
				}
			} else if cloud.row.Password != "original-ciphertext" || cloud.row.Key != "new-key" {
				t.Fatal("empty cloud password did not preserve stored ciphertext")
			}
		})
	}
	for _, path := range []string{"local-create", "local-update", "cloud-update"} {
		t.Run(path+"/empty-password-auth", func(t *testing.T) {
			local := &passwordStorageLocalDAO{row: model.TreeLocalResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModePassword, Password: "original-ciphertext"}}
			cloud := &passwordStorageCloudDAO{row: model.TreeCloudResource{Model: model.Model{ID: 7}, AuthMode: model.AuthModePassword, Password: "original-ciphertext"}, audit: make(chan struct{}, 1)}
			var err error
			if path == "cloud-update" {
				service, construction := NewTreeCloudService(&config.Config{}, zap.NewNop(), cloud, nil)
				if construction != nil {
					t.Fatal(construction)
				}
				err = service.UpdateTreeCloudResource(context.Background(), &model.UpdateTreeCloudResourceReq{ID: 7})
			} else {
				service, construction := NewTreeLocalService(&config.Config{}, zap.NewNop(), local)
				if construction != nil {
					t.Fatal(construction)
				}
				if path == "local-create" {
					err = service.CreateTreeLocal(context.Background(), &model.CreateTreeLocalResourceReq{Name: "fixture", IpAddr: "192.0.2.5", AuthMode: model.AuthModePassword})
				} else {
					err = service.UpdateTreeLocal(context.Background(), &model.UpdateTreeLocalResourceReq{ID: 7})
				}
			}
			if err != nil {
				t.Fatal("empty input started requiring a cipher key")
			}
			if path == "local-create" {
				if local.row.Password != "" {
					t.Fatal("empty create stored a new password")
				}
			} else if local.row.Password != "original-ciphertext" || cloud.row.Password != "original-ciphertext" {
				t.Fatal("empty update overwrote stored ciphertext")
			}
		})
	}
}
