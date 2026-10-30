package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
)

type validationHistoryDAO struct{ dao.TreeCloudDAO }

func (validationHistoryDAO) CreateSyncHistory(context.Context, *model.CloudResourceSyncHistory) error {
	return nil
}
func (validationHistoryDAO) AcquireRegionSyncLease(_ context.Context, id int, _ time.Duration) (*dao.RegionSyncLease, error) {
	return &dao.RegionSyncLease{RegionID: id, Token: "fixture", Generation: 1, ExpiresEpochMS: time.Now().Add(time.Minute).UnixMilli()}, nil
}
func (validationHistoryDAO) ReleaseRegionSyncLease(context.Context, *dao.RegionSyncLease) error {
	return nil
}
func (validationHistoryDAO) FinishRegionSyncHistory(context.Context, int, string, string, string) error {
	return nil
}

func TestSyncValidatesCompleteRegionSelectionBeforeSideEffects(t *testing.T) {
	encryptionKey := strings.Repeat("v", 32)
	access, err := treeUtils.EncryptPassword("offline-access", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("offline-secret", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	account := &model.CloudAccount{Model: model.Model{ID: 7}, Provider: model.ProviderAWS, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret, Regions: []*model.CloudAccountRegion{
		{Model: model.Model{ID: 11}, CloudAccountID: 7, Region: "enabled-one", Status: model.CloudAccountRegionEnabled},
		{Model: model.Model{ID: 12}, CloudAccountID: 7, Region: "disabled", Status: model.CloudAccountRegionDisabled},
		{Model: model.Model{ID: 13}, CloudAccountID: 8, Region: "foreign", Status: model.CloudAccountRegionEnabled},
		{Model: model.Model{ID: 14}, CloudAccountID: 7, Region: "enabled-two", Status: model.CloudAccountRegionEnabled},
	}}
	for _, tc := range []struct {
		name        string
		ids         []int
		wantRegions []string
		invalid     bool
	}{
		{"enabled explicit", []int{14, 11}, []string{"enabled-two", "enabled-one"}, false},
		{"empty selects enabled only", nil, []string{"enabled-one", "enabled-two"}, false},
		{"disabled", []int{12}, nil, true},
		{"mixed enabled and disabled", []int{11, 12}, nil, true},
		{"mixed enabled and missing", []int{11, 99}, nil, true},
		{"foreign", []int{13}, nil, true},
		{"zero", []int{0}, nil, true},
		{"duplicate", []int{11, 11}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var called []string
			s := &treeCloudService{passwordEncryptionKey: encryptionKey, logger: zap.NewNop(), cloudAccountDAO: storedCloudAccount{account: account}, dao: validationHistoryDAO{}, clientFactory: func(_ model.CloudProvider, _, _, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
				called = append(called, region)
				return cloudReadStub{err: errors.New("offline fixture: no provider response")}, nil
			}}
			resp, err := s.SyncTreeCloudResource(context.Background(), &model.SyncTreeCloudResourceReq{CloudAccountID: 7, CloudAccountRegionIDs: tc.ids, SyncMode: model.SyncModeFull})
			if tc.invalid && err == nil {
				t.Fatal("expected region validation failure")
			}
			if !tc.invalid && (err != nil || resp == nil || resp.Status != "failed") {
				t.Fatalf("expected structured provider failure: %+v %v", resp, err)
			}
			if tc.invalid && !strings.Contains(err.Error(), "指定的地域") {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(called, ",") != strings.Join(tc.wantRegions, ",") {
				t.Fatalf("provider calls = %v, want %v", called, tc.wantRegions)
			}
		})
	}
}
