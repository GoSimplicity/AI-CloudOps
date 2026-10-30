package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestManagedReviewMissingOperationPreventsDelayedDelivery(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	review := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, ResourcesVerified: true, Reason: "已经核对实际资源和本地记录，不再执行旧请求", OperationID: id, OperatorID: 7}
	view, err := svc.ReviewManagedOperation(context.Background(), review)
	if err != nil || view.State != "reviewed" || view.PreviousState != "not_recorded" || view.Result != nil || view.ReviewedAt == nil {
		t.Fatalf("review missing: %v", err)
	}
	if _, err := svc.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperationID: id, OperatorID: 7}); err == nil {
		t.Fatal("delayed reviewed operation executed")
	}
	if _, err := svc.ReviewManagedOperation(context.Background(), review); err == nil {
		t.Fatal("second review overwrote audit")
	}
	other := *review
	other.ResourceIDs = []string{"rm-other"}
	if _, err := svc.ReviewManagedOperation(context.Background(), &other); err == nil {
		t.Fatal("different request altered reviewed operation")
	}
	var count int64
	db.Model(&model.CloudManagedResource{}).Count(&count)
	if count != 0 {
		t.Fatal("review called cloud sync")
	}
}

func TestManagedReviewAllowsDisabledScopeWithoutEnablingSync(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(account).Update("status", model.CloudAccountDisabled).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(region).Update("status", model.CloudAccountRegionDisabled).Error; err != nil {
		t.Fatal(err)
	}
	review := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, ResourcesVerified: true, Reason: "已核对停用账户内实际资源与本地状态", OperationID: uuid.NewString(), OperatorID: 7}
	if _, err := svc.ReviewManagedOperation(context.Background(), review); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperatorID: 7}); err == nil {
		t.Fatal("review enabled sync of disabled scope")
	}
	history := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"rm-one"}, Status: "unknown", OperatorID: 7, StartedAt: time.Now().UTC()}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveManagedHistory(context.Background(), &model.ResolveCloudManagedHistoryReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", HistoryID: history.ID, OperatorID: 7, ResourcesVerified: true, Reason: "已核对停用范围的未知历史与实际资源状态"}); err != nil {
		t.Fatal(err)
	}
}

func TestManagedOperationReviewPreservesOriginalResultAndHistory(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	id := uuid.NewString()
	sync := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperatorID: 7, OperationID: id}
	result, err := svc.SyncManaged(context.Background(), sync)
	if err != nil || result.Status != "success" {
		t.Fatal(err)
	}
	if err := db.Model(&model.CloudManagedOperation{}).Where("operation_id = ? AND operator_id = ?", id, 7).Update("resource_ids", model.StringList{}).Error; err != nil {
		t.Fatal(err)
	}
	review := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, ResourcesVerified: true, Reason: "已核对该操作的原始成功结果和实际资源", OperationID: id, OperatorID: 7}
	lease, err := svc.regions.AcquireRegionSyncLease(context.Background(), region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReviewManagedOperation(context.Background(), review); err == nil {
		t.Fatal("review bypassed active regional lease")
	}
	if err := svc.regions.ReleaseRegionSyncLease(context.Background(), lease); err != nil {
		t.Fatal(err)
	}
	view, err := svc.ReviewManagedOperation(context.Background(), review)
	if err != nil || view.PreviousState != "completed" || view.Result == nil || view.Result.Status != "success" || len(view.ResourceIDs) != 1 {
		t.Fatalf("result not preserved: %v", err)
	}
	var history model.CloudManagedSyncHistory
	if err := db.First(&history, result.HistoryID).Error; err != nil || history.ResolvedAt == nil || history.ClientOperationID != id || history.Status != "success" {
		t.Fatal("linked history not audited")
	}
	key, _ := operationKey(id, 7)
	if err := svc.store.CompleteOperation(context.Background(), key, []byte(`{}`)); err == nil {
		t.Fatal("late worker overwrote reviewed state")
	}
}

type pausedManagedReservation struct {
	dao.ManagedCloudDAO
	started chan struct{}
	release chan struct{}
}

func (p pausedManagedReservation) BeginOperation(ctx context.Context, row *model.CloudManagedOperation) (bool, *model.CloudManagedOperation, error) {
	created, previous, err := p.ManagedCloudDAO.BeginOperation(ctx, row)
	if created && err == nil {
		close(p.started)
		<-p.release
	}
	return created, previous, err
}
func TestManagedReviewFencesReservedRequestBeforeLeaseAcquisition(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	svc.store = pausedManagedReservation{svc.store, started, release}
	id := uuid.NewString()
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperationID: id, OperatorID: 7}
	done := make(chan *model.CloudManagedSyncResult, 1)
	go func() { result, _ := svc.SyncManaged(context.Background(), req); done <- result }()
	<-started
	review := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, ResourcesVerified: true, Reason: "已经核对实际资源与本地记录，旧请求不可再提交", OperationID: id, OperatorID: 7}
	if _, err := svc.ReviewManagedOperation(context.Background(), review); err != nil {
		t.Fatal(err)
	}
	close(release)
	result := <-done
	if result == nil || result.Status != "unknown" {
		t.Fatal("paused request unexpectedly claimed completion")
	}
	var count int64
	db.Model(&model.CloudManagedResource{}).Count(&count)
	if count != 0 {
		t.Fatal("reserved request committed after review")
	}
	view, err := svc.GetManagedOperation(context.Background(), id, 7)
	if err != nil || view.State != "reviewed" {
		t.Fatal("late request erased review")
	}
}
