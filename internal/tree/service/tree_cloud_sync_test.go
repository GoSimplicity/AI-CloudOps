package service

import (
	"context"
	"strings"
	"testing"

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
	err := s.syncAliyunResourcesForMultipleRegions(context.Background(), &model.CloudAccount{}, "", "", []*model.CloudAccountRegion{{Region: "one"}, {Region: "two"}}, &model.SyncTreeCloudResourceReq{}, &model.SyncCloudResourceResp{})
	if err == nil || !strings.Contains(err.Error(), "one") || !strings.Contains(err.Error(), "two") {
		t.Fatalf("region failures lost: %v", err)
	}
}

func TestUnsupportedProviderDoesNotMasqueradeAsVerifiedRegionList(t *testing.T) {
	s := &cloudAccountRegionService{logger: zap.NewNop()}
	_, err := s.GetAvailableRegions(context.Background(), &model.GetAvailableRegionsReq{Provider: model.ProviderTencent, AccessKey: "offline-placeholder", SecretKey: "offline-placeholder"})
	if err == nil {
		t.Fatal("unimplemented provider was reported as a verified region response")
	}
}
