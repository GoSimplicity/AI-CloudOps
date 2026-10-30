package service

import (
	"context"
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestManagedScopeBlocksUncertainOperationWithoutHistoryAcrossOwners(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	id := uuid.NewString()
	key, _ := operationKey(id, 8)
	payload, _ := json.Marshal(&model.CloudManagedSyncResult{OperationID: id, Status: "failed", ErrorCategory: "audit_unavailable", RetrySafe: false})
	row := model.CloudManagedOperation{Key: key, OperationID: id, OperatorID: 8, CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-one"}, RequestHash: managedRequestHash(account.ID, region.ID, "rds", []string{"rm-one"}), ClaimToken: uuid.NewString(), State: "completed", AdmissionVersion: 1, Result: payload}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	request := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperatorID: 7, OperationID: uuid.NewString()}
	if _, err := svc.SyncManaged(context.Background(), request); err == nil {
		t.Fatal("new ID bypassed another owner's uncertain operation")
	}
	request.OperationID = ""
	if _, err := svc.SyncManaged(context.Background(), request); err == nil {
		t.Fatal("legacy request bypassed durable scope guard")
	}
	own, err := svc.ListManagedOperations(context.Background(), &model.ListCloudManagedOperationsReq{ListCloudManagedResourcesReq: model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds"}, ViewerID: 7, UnconfirmedOnly: true})
	if err != nil || own.Total != 0 || !own.ScopeBlocked {
		t.Fatal("other user's uncertainty leaked")
	}
	if _, err := svc.ListManagedOperations(context.Background(), &model.ListCloudManagedOperationsReq{ListCloudManagedResourcesReq: model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds"}, ViewerID: 7, AllOwners: true}); err == nil {
		t.Fatal("untrusted cross-owner list allowed")
	}
	all, err := svc.ListManagedOperations(context.Background(), &model.ListCloudManagedOperationsReq{ListCloudManagedResourcesReq: model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds"}, ViewerID: 7, TrustedSuper: true, AllOwners: true, UnconfirmedOnly: true})
	if err != nil || all.Total != 1 || all.Items[0].OwnerID != 8 {
		t.Fatal("privileged scope recovery list missing")
	}
	review := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, ResourcesVerified: true, Reason: "管理员已核对该操作实际资源和本地状态", OperationID: id, OperatorID: 7, OwnerID: 8}
	if _, err := svc.ReviewManagedOperation(context.Background(), review); err == nil {
		t.Fatal("untrusted cross-owner review allowed")
	}
	linked := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-one"}, ClientOperationID: id, Status: "unknown", OperatorID: 8, StartedAt: time.Now().UTC()}
	if err := db.Create(&linked).Error; err != nil {
		t.Fatal(err)
	}
	review.TrustedSuper = true
	reviewed, err := svc.ReviewManagedOperation(context.Background(), review)
	if err != nil || reviewed.OwnerID != 8 || reviewed.ReviewedBy != 7 {
		t.Fatal("review actor confused with original owner")
	}
	var audited model.CloudManagedSyncHistory
	if err := db.First(&audited, linked.ID).Error; err != nil || audited.OperatorID != 8 || audited.ResolvedBy != 7 || audited.ResolvedAt == nil {
		t.Fatal("history lost original owner or actual reviewer")
	}
	request.OperationID = uuid.NewString()
	result, err := svc.SyncManaged(context.Background(), request)
	if err != nil || result.Status != "success" {
		t.Fatalf("audited scope stayed blocked: %v", err)
	}
	cleared, err := svc.ListManagedOperations(context.Background(), &model.ListCloudManagedOperationsReq{ListCloudManagedResourcesReq: model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds"}, ViewerID: 7, UnconfirmedOnly: true})
	if err != nil || cleared.ScopeBlocked || cleared.Total != 0 {
		t.Fatal("scope summary did not reflect completed review")
	}

}

func TestManagedAdmissionOrdersPendingReservationsAndPersistsStart(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	now := time.Now().UTC().Add(-time.Minute)
	firstID, secondID := uuid.NewString(), uuid.NewString()
	firstKey, _ := operationKey(firstID, 7)
	secondKey, _ := operationKey(secondID, 7)
	first := model.CloudManagedOperation{Key: firstKey, OperationID: firstID, OperatorID: 7, CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", State: "running", AdmissionVersion: 1, ClaimToken: "first", RequestHash: "first", CreatedAt: now}
	second := first
	second.Key = secondKey
	second.OperationID = secondID
	second.ClaimToken = "second"
	second.CreatedAt = now.Add(time.Second)
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	lease, err := svc.regions.AcquireRegionSyncLease(context.Background(), region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, second.Kind, second.Key, second.ClaimToken); err == nil {
		t.Fatal("later reservation bypassed earlier pending operation")
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, first.Kind, first.Key, first.ClaimToken); err != nil {
		t.Fatal(err)
	}
	var saved model.CloudManagedOperation
	if err := db.Where("`key` = ?", first.Key).First(&saved).Error; err != nil || saved.StartedAt == nil {
		t.Fatal("admission was not durable")
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, first.Kind, first.Key, first.ClaimToken); err == nil {
		t.Fatal("one reservation admitted twice")
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, second.Kind, second.Key, second.ClaimToken); err == nil {
		t.Fatal("started operation did not block later request")
	}
	if err := svc.regions.ReleaseRegionSyncLease(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
}

func TestManagedScopeGuardRejectsMalformedSafetyFlags(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	id := uuid.NewString()
	key, _ := operationKey(id, 7)
	row := model.CloudManagedOperation{Key: key, OperationID: id, OperatorID: 7, CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ClaimToken: "fixture", RequestHash: "fixture", State: "completed"}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{`{}`, `{"retry_safe":"true","status":"failed"}`, `{"retry_safe":true,"status":"unknown"}`, `{"retry_safe":true,"status":"success","count":-1}`, `not-json`} {
		if err := db.Model(&model.CloudManagedOperation{}).Where("`key` = ?", key).Update("result", []byte(payload)).Error; err != nil {
			t.Fatal(err)
		}
		blocked, err := svc.store.HasUnresolvedOperations(context.Background(), account.ID, region.ID, "rds", "")
		if err == nil && !blocked {
			t.Fatal("malformed stored result was declared safe")
		}
	}
}

func TestManagedAdmissionEqualTimestampsHaveDeterministicWinner(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	firstID, secondID := uuid.NewString(), uuid.NewString()
	firstKey, _ := operationKey(firstID, 7)
	secondKey, _ := operationKey(secondID, 7)
	if firstKey > secondKey {
		firstID, secondID = secondID, firstID
		firstKey, secondKey = secondKey, firstKey
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	first := model.CloudManagedOperation{Key: firstKey, OperationID: firstID, OperatorID: 7, CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", State: "running", AdmissionVersion: 1, ClaimToken: "first", RequestHash: "fixture", CreatedAt: now}
	second := first
	second.Key = secondKey
	second.OperationID = secondID
	second.ClaimToken = "second"
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	lease, err := svc.regions.AcquireRegionSyncLease(context.Background(), region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, "rds", second.Key, second.ClaimToken); err == nil {
		t.Fatal("later tie winner bypassed order")
	}
	if err := svc.regions.AdmitManagedOperation(context.Background(), account, region, lease, "rds", first.Key, first.ClaimToken); err != nil {
		t.Fatal(err)
	}
	_ = svc.regions.ReleaseRegionSyncLease(context.Background(), lease)
}

func TestManagedScopeSummaryBlocksUnknownHistoryState(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	row := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-one"}, Status: "unexpected", OperatorID: 7, StartedAt: time.Now().UTC()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	response, err := svc.ListManagedOperations(context.Background(), &model.ListCloudManagedOperationsReq{ListCloudManagedResourcesReq: model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds"}, ViewerID: 7})
	if err != nil || !response.ScopeBlocked {
		t.Fatal("unknown history status was declared safe")
	}
}
