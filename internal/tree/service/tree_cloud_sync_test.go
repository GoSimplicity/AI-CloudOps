package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/spf13/viper"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestFilteredFullSyncRejectedBeforeSideEffects(t *testing.T) {
	s := &treeCloudService{}
	_, err := s.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull, InstanceIDs: []string{"i-one"}})
	if err == nil {
		t.Fatal("filtered full snapshot could delete unqueried resources")
	}
}

func TestRegionFailuresAreNotReportedAsSuccessfulSync(t *testing.T) {
	s := &treeCloudService{logger: zap.NewNop()}
	resp := &model.SyncCloudResourceResp{}
	err := s.syncCloudResourcesForMultipleRegions(context.Background(), &model.CloudAccount{Provider: model.ProviderAliyun}, "", "", []*model.CloudAccountRegion{{Region: "one"}, {Region: "two"}}, &model.SyncTreeCloudResourceReq{}, resp)
	if err == nil || len(resp.Regions) != 2 || resp.Regions[0].Region != "one" || resp.Regions[1].Region != "two" || resp.Regions[0].Status != "failed" || resp.Regions[1].Status != "failed" || resp.FailedCount != 2 {
		t.Fatalf("region failures lost: %+v, %v", resp, err)
	}
}

func TestUnsupportedProviderDoesNotMasqueradeAsVerifiedRegionList(t *testing.T) {
	for _, provider := range []model.CloudProvider{model.ProviderHuawei, model.ProviderAzure, model.ProviderGCP} {
		if _, err := treeUtils.NewCloudComputeClient(provider, "offline-placeholder", "offline-placeholder", "region", zap.NewNop()); err == nil {
			t.Fatal("unimplemented provider accepted")
		}
	}
}

type cloudReadStub struct {
	treeUtils.CloudComputeClient
	err     error
	regions []model.AvailableRegion
}

func (s cloudReadStub) ListInstances(context.Context, []string) ([]*model.TreeCloudResource, error) {
	return nil, s.err
}
func (s cloudReadStub) GetAvailableRegions(context.Context) ([]model.AvailableRegion, error) {
	return s.regions, s.err
}

func TestProviderFailureNeverReconcilesLocalAssets(t *testing.T) {
	for _, provider := range []model.CloudProvider{model.ProviderAliyun, model.ProviderAWS, model.ProviderTencent} {
		s := &treeCloudService{logger: zap.NewNop(), clientFactory: func(got model.CloudProvider, _, _, _ string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
			if got != provider {
				t.Fatal("wrong provider selected")
			}
			return cloudReadStub{err: errors.New("denied")}, nil
		}}
		// A nil DAO is intentional: an incomplete provider read must never reach
		// local deletion, update, or creation, even when full sync was requested.
		err := s.syncCloudResourcesForMultipleRegions(context.Background(), &model.CloudAccount{Provider: provider}, "offline", "offline", []*model.CloudAccountRegion{{Region: "region"}}, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}, &model.SyncCloudResourceResp{})
		if err == nil {
			t.Fatal("failed read reported sync success")
		}
	}
}

type storedCloudAccount struct {
	dao.CloudAccountDAO
	account *model.CloudAccount
}

func (s storedCloudAccount) GetByID(context.Context, int) (*model.CloudAccount, error) {
	return s.account, nil
}

func TestRegionQueryUsesStoredAccountWithoutReturningCredentials(t *testing.T) {
	viper.Set("tree.password_encryption_key", strings.Repeat("a", 32))
	defer viper.Set("tree.password_encryption_key", nil)
	access, err := treeUtils.EncryptPassword("offline-access")
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("offline-secret")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	s := &cloudAccountService{dao: storedCloudAccount{account: &model.CloudAccount{Provider: model.ProviderAWS, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret, Regions: []*model.CloudAccountRegion{{Region: "us-west-2", IsDefault: true}}}}, clientFactory: func(provider model.CloudProvider, ak, sk, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
		calls++
		if provider != model.ProviderAWS || ak != "offline-access" || sk != "offline-secret" || region != "us-west-2" {
			t.Fatal("stored account context lost")
		}
		return cloudReadStub{regions: []model.AvailableRegion{{Region: "us-west-2", Available: true}}}, nil
	}}
	regions, err := s.GetAvailableRegionsForAccount(context.Background(), 7)
	if err != nil || calls != 1 || len(regions) != 1 {
		t.Fatalf("query failed: %v", err)
	}
}

func TestURLCredentialsRejectedBeforeProviderCall(t *testing.T) {
	s := &cloudAccountRegionService{}
	_, err := s.GetAvailableRegions(context.Background(), &model.GetAvailableRegionsReq{Provider: model.ProviderAWS, AccessKey: "do-not-send-in-url"})
	if err == nil {
		t.Fatal("URL credentials accepted")
	}
}

func TestProviderSnapshotClearsStaleFieldsButPreservesLocalAccess(t *testing.T) {
	now := time.Now()
	fields := providerResourceFields(&model.TreeCloudResource{InstanceID: "instance", Status: model.CloudResourceRunning, LastSyncTime: &now})
	if fields["public_ip"] != "" || fields["last_sync_time"] == nil {
		t.Fatal("stale provider fields not reconciled")
	}
	for _, field := range []string{"password", "key", "username", "port", "environment", "tree_nodes"} {
		if _, exists := fields[field]; exists {
			t.Fatalf("provider snapshot changed local field %s", field)
		}
	}
}
