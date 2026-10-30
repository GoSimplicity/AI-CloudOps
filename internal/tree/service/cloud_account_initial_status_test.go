package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strings"
	"testing"
)

type initialAccountDAO struct {
	dao.CloudAccountDAO
	account *model.CloudAccount
	writes  int
}

func (s *initialAccountDAO) CheckNameExists(context.Context, string, model.CloudProvider, int) (bool, error) {
	return false, nil
}
func (s *initialAccountDAO) CreateWithTransaction(_ context.Context, fn func(interface{}) error) error {
	return fn(nil)
}
func (s *initialAccountDAO) CreateInTransaction(_ context.Context, account *model.CloudAccount, _ interface{}) error {
	s.writes++
	account.ID = 7
	s.account = account
	return nil
}
func (s *initialAccountDAO) CreateRegionInTransaction(context.Context, *model.CloudAccountRegion, interface{}) error {
	s.writes++
	return nil
}

func TestCreateCloudAccountInitialStatusWithoutSDK(t *testing.T) {
	old := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("k", 32))
	t.Cleanup(func() { viper.Set("tree.password_encryption_key", old) })
	for _, tc := range []struct {
		name     string
		supplied bool
		value    int
		valid    bool
		expected model.CloudAccountStatus
	}{
		{"omitted remains enabled", false, 0, true, model.CloudAccountEnabled}, {"enabled", true, 1, true, model.CloudAccountEnabled}, {"disabled", true, 2, true, model.CloudAccountDisabled},
		{"zero", true, 0, false, 0}, {"unknown", true, 3, false, 0}, {"negative", true, -1, false, 0}, {"overflow", true, 99999, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &initialAccountDAO{}
			service := NewCloudAccountService(zap.NewNop(), store).(*cloudAccountService)
			sdkCalls := 0
			service.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
				sdkCalls++
				return nil, errors.New("unexpected SDK initialization")
			}
			service.credentialFactory = func(model.CloudProvider, treeUtils.CloudCredentials, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
				sdkCalls++
				return nil, errors.New("unexpected SDK initialization")
			}
			req := &model.CreateCloudAccountReq{Name: "disabled-fixture", Provider: model.ProviderAliyun, AccessKey: "fixture-access-key", SecretKey: "fixture-secret-key", Regions: []model.CreateCloudAccountRegionItem{{Region: "cn-hangzhou", IsDefault: true}}}
			if tc.supplied {
				req.Status = &tc.value
			}
			err := service.CreateCloudAccount(context.Background(), req, 7, "fixture")
			if (err == nil) != tc.valid || (store.account != nil) != tc.valid {
				t.Fatalf("valid=%v err=%v wrote=%v", tc.valid, err, store.account != nil)
			}
			if sdkCalls != 0 {
				t.Fatal("creation invoked cloud SDK")
			}
			if !tc.valid && store.writes != 0 {
				t.Fatal("invalid status reached persistence")
			}
			if tc.valid && (store.account.Status != tc.expected || store.writes != 2) {
				t.Fatal("initial status or region persistence changed")
			}
		})
	}
}
