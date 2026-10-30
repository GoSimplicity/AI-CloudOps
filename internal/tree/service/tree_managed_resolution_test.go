package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"testing"
	"time"
)

func TestManagedResolutionPreservesOutcomeAndUnblocksOnlyReviewedHistory(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	history := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-test"}, Status: "unknown", OperatorID: 7, StartedAt: time.Now().UTC()}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	sync := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-test"}, OperatorID: 7}
	if _, err := svc.SyncManaged(ctx, sync); err == nil {
		t.Fatal("unreviewed unknown permitted another sync")
	}
	req := &model.ResolveCloudManagedHistoryReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", HistoryID: history.ID, OperatorID: 9, Reason: "已经核对云端资源和本地记录的实际状态", ResourcesVerified: true}
	wrong := *req
	wrong.CloudAccountRegionID++
	if err := svc.ResolveManagedHistory(ctx, &wrong); err == nil {
		t.Fatal("cross-region review accepted")
	}
	unverified := *req
	unverified.ResourcesVerified = false
	if err := svc.ResolveManagedHistory(ctx, &unverified); err == nil {
		t.Fatal("unverified review accepted")
	}
	lease, err := svc.regions.AcquireRegionSyncLease(ctx, region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveManagedHistory(ctx, req); err == nil {
		t.Fatal("active sync lease bypassed")
	}
	if err := svc.regions.ReleaseRegionSyncLease(ctx, lease); err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveManagedHistory(ctx, req); err != nil {
		t.Fatal(err)
	}
	var saved model.CloudManagedSyncHistory
	if err := db.First(&saved, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != "unknown" || saved.ResolvedBy != 9 || saved.ResolvedAt == nil || saved.ResolutionReason != req.Reason {
		t.Fatal("review rewrote outcome or omitted audit")
	}
	if err := svc.ResolveManagedHistory(ctx, req); err == nil {
		t.Fatal("second review overwrote audit")
	}
	result, err := svc.SyncManaged(ctx, sync)
	if err != nil || result.Status != "success" {
		t.Fatalf("reviewed history blocked new sync: %v", err)
	}
}

func TestManagedRunningHistoryRequiresReviewEvenBeforeOriginalExpiry(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	row := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-test"}, Status: "running", LeaseExpiresMS: time.Now().Add(time.Hour).UnixMilli(), OperatorID: 7, StartedAt: time.Now().UTC()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-test"}, OperatorID: 7}); err == nil {
		t.Fatal("orphan running record bypassed review")
	}
}
