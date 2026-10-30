package service

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"github.com/google/uuid"
	"sync/atomic"
	"testing"
	"time"
)

func TestManagedOperationDeduplicatesAndIsolatesOwner(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		calls.Add(1)
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) { return ids, nil }}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one", "rm-two"}, OperatorID: 7, OperationID: uuid.NewString()}
	first, err := svc.SyncManaged(context.Background(), req)
	if err != nil || first.Status != "success" {
		t.Fatalf("first: %v", err)
	}
	copyReq := *req
	copyReq.ResourceIDs = []string{"rm-two", "rm-one"}
	again, err := svc.SyncManaged(context.Background(), &copyReq)
	if err != nil || again.HistoryID != first.HistoryID || calls.Load() != 1 {
		t.Fatal("duplicate operation executed")
	}
	copyReq.ResourceIDs = []string{"rm-three"}
	if _, err := svc.SyncManaged(context.Background(), &copyReq); err == nil || calls.Load() != 1 {
		t.Fatal("different request reused operation")
	}
	if _, err := svc.GetManagedOperation(context.Background(), req.OperationID, 8); err == nil {
		t.Fatal("cross-owner operation exposed")
	}
	view, err := svc.GetManagedOperation(context.Background(), req.OperationID, 7)
	if err != nil || view.State != "completed" || view.Result.HistoryID != first.HistoryID {
		t.Fatal("durable result missing")
	}
	restarted := *svc
	restarted.store = dao.NewManagedCloudDAO(db)
	if _, err := restarted.SyncManaged(context.Background(), req); err != nil || calls.Load() != 1 {
		t.Fatal("restart lost operation deduplication")
	}
}

func TestManagedOperationInFlightAndUnknownNeverRunAgain(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		calls.Add(1)
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			close(started)
			<-release
			return ids, nil
		}}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperatorID: 7, OperationID: uuid.NewString()}
	done := make(chan error, 1)
	go func() { _, err := svc.SyncManaged(context.Background(), req); done <- err }()
	<-started
	if _, err := svc.SyncManaged(context.Background(), req); err == nil || calls.Load() != 1 {
		t.Fatal("concurrent operation executed")
	}
	key, _ := operationKey(req.OperationID, 7)
	if err := db.Model(&model.CloudManagedOperation{}).Where("`key` = ?", key).Update("created_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	view, err := svc.GetManagedOperation(context.Background(), req.OperationID, 7)
	if err != nil || view.State != "unknown" {
		t.Fatal("expired operation not unknown")
	}
	if _, err := svc.SyncManaged(context.Background(), req); err == nil || calls.Load() != 1 {
		t.Fatal("unknown operation replayed")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

type failOperationCompletion struct{ dao.ManagedCloudDAO }

func (f failOperationCompletion) CompleteOperation(context.Context, string, []byte) error {
	return errors.New("fixture completion failure")
}
func TestManagedOperationMissingAcknowledgementKeepsUncertainty(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	svc.store = failOperationCompletion{svc.store}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"rm-one"}, OperatorID: 7, OperationID: uuid.NewString()}
	result, err := svc.SyncManaged(context.Background(), req)
	if err != nil || result.Status != "unknown" || result.RetrySafe || result.HistoryID == 0 {
		t.Fatal("missing persistence acknowledgement claimed success")
	}
	if _, err := svc.SyncManaged(context.Background(), req); err == nil {
		t.Fatal("uncertain operation replayed")
	}
}

func TestManagedOperationCorruptPersistedResultIsNotSuccess(t *testing.T) {
	for _, payload := range []string{`{}`, `{"operation_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","status":"success","count":1,"new_count":1,"update_count":1,"retry_safe":true}`, `{"operation_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","status":"success","count":1,"new_count":1,"update_count":0,"retry_safe":null}`, `{"operation_id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","status":"unknown","count":0,"new_count":0,"update_count":0,"retry_safe":true}`} {
		if _, err := operationView(&model.CloudManagedOperation{OperationID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", State: "completed", Result: []byte(payload)}); err == nil {
			t.Fatal("corrupted result accepted")
		}
	}
}
