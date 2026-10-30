//go:build integration

package service

import (
	"context"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMySQLManagedResourceAtomicAndFencing(t *testing.T) {
	raw := os.Getenv("CLOUDOPS_MYSQL_TEST_DSN")
	if raw == "" {
		t.Skip("isolated MySQL required")
	}
	cfg, err := mysqlDriver.ParseDSN(raw)
	if err != nil || !regexp.MustCompile(`^schema_[a-f0-9]{24}$`).MatchString(cfg.DBName) || cfg.User != cfg.DBName {
		t.Fatal("refusing non-isolated database")
	}
	cfg.ClientFoundRows = true // duplicate insert row counts must not grant execution
	cfg.Timeout, cfg.ReadTimeout, cfg.WriteTimeout = 5*time.Second, 10*time.Second, 10*time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("isolated connection failed")
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}, &model.CloudRegionSyncLease{}, &model.CloudManagedResource{}, &model.CloudManagedSyncHistory{}, &model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	prior := viper.Get("tree.password_encryption_key")
	viper.Set("tree.password_encryption_key", strings.Repeat("m", 32))
	defer viper.Set("tree.password_encryption_key", prior)
	access, _ := treeUtils.EncryptPassword("test-ak")
	secret, _ := treeUtils.EncryptPassword("test-sk")
	account := &model.CloudAccount{Name: "managed-fixture", Provider: model.ProviderAliyun, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "cn-hangzhou", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(region).Error; err != nil {
		t.Fatal(err)
	}
	svc := NewManagedCloudService(zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop()), dao.NewTreeCloudDAO(db, zap.NewNop()), dao.NewManagedCloudDAO(db)).(*managedCloudService)
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) { return ids, nil }}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one", "id-two"}, OperatorID: 7}
	result, err := svc.SyncManaged(context.Background(), req)
	if err != nil || result.Status != "success" || result.NewCount != 2 {
		t.Fatalf("create=%+v err=%v", result, err)
	}
	var initialHistory model.CloudManagedSyncHistory
	if err := db.First(&initialHistory, result.HistoryID).Error; err != nil || len(initialHistory.ResourceIDs) != 2 || initialHistory.ResourceIDs[0] != "id-one" || initialHistory.ResourceIDs[1] != "id-two" {
		t.Fatal("MySQL history lost requested IDs")
	}
	var rows []model.CloudManagedResource
	if err := db.Order("id").Find(&rows).Error; err != nil || len(rows) != 2 || rows[0].RDS == nil {
		t.Fatal("MySQL JSON metadata missing")
	}
	if err := db.Model(&rows[0]).Update("local_description", "local note").Error; err != nil {
		t.Fatal(err)
	}
	req.ResourceIDs = []string{"id-one"}
	result, err = svc.SyncManaged(context.Background(), req)
	if err != nil || result.Status != "success" || result.UpdateCount != 1 {
		t.Fatalf("update=%+v err=%v", result, err)
	}
	var saved model.CloudManagedResource
	if err := db.First(&saved, rows[0].ID).Error; err != nil || saved.LocalDescription != "local note" || saved.RDS == nil {
		t.Fatal("local metadata or typed JSON lost")
	}
	operationReq := *req
	operationReq.OperationID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	firstOperation, err := svc.SyncManaged(context.Background(), &operationReq)
	if err != nil || firstOperation.Status != "success" {
		t.Fatal("MySQL operation reservation failed")
	}
	replay, err := svc.SyncManaged(context.Background(), &operationReq)
	if err != nil || replay.HistoryID != firstOperation.HistoryID {
		t.Fatal("MySQL clientFoundRows duplicate executed again")
	}
	var operations int64
	if err := db.Model(&model.CloudManagedOperation{}).Count(&operations).Error; err != nil || operations != 1 {
		t.Fatal("MySQL operation deduplication failed")
	}
	reviewReq := &model.ReviewCloudManagedOperationReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: operationReq.Kind, ResourceIDs: append([]string{}, operationReq.ResourceIDs...), ResourcesVerified: true, Reason: "已核对数据库中的操作原始结果与实际资源", OperationID: operationReq.OperationID, OperatorID: 7}
	reviewedOperation, err := svc.ReviewManagedOperation(context.Background(), reviewReq)
	if err != nil || reviewedOperation.State != "reviewed" || reviewedOperation.PreviousState != "completed" || reviewedOperation.Result == nil {
		t.Fatal("MySQL original operation review failed")
	}
	if _, err := svc.ReviewManagedOperation(context.Background(), reviewReq); err == nil {
		t.Fatal("MySQL review overwrote existing audit")
	}
	missingReview := *reviewReq
	missingReview.OperationID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	if view, err := svc.ReviewManagedOperation(context.Background(), &missingReview); err != nil || view.PreviousState != "not_recorded" {
		t.Fatal("MySQL missing operation tombstone failed")
	}
	delayed := operationReq
	delayed.OperationID = missingReview.OperationID
	if _, err := svc.SyncManaged(context.Background(), &delayed); err == nil {
		t.Fatal("MySQL reviewed tombstone allowed delayed delivery")
	}
	var total int64
	db.Model(&model.CloudManagedResource{}).Count(&total)
	if total != 2 {
		t.Fatal("incremental sync removed unrequested resource")
	}
	for _, kind := range []string{"clb", "cache"} {
		req.Kind = kind
		result, err = svc.SyncManaged(context.Background(), req)
		if err != nil || result.Status != "success" || result.NewCount != 1 {
			t.Fatalf("%s typed sync failed", kind)
		}
		var typed model.CloudManagedResource
		if err := db.Where("kind = ? AND provider_id = ?", kind, "id-one").First(&typed).Error; err != nil || (kind == "clb" && typed.CLB == nil) || (kind == "cache" && typed.Cache == nil) {
			t.Fatalf("%s JSON metadata missing", kind)
		}
	}
	req.Kind = "rds"
	callback := "test:managed-history-failure"
	if err := db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_tree_cloud_managed_sync_history" {
			if values, ok := tx.Statement.Dest.(map[string]any); ok && values["status"] == "success" {
				tx.AddError(errors.New("fixture failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	req.ResourceIDs = []string{"id-three"}
	result, err = svc.SyncManaged(context.Background(), req)
	if err != nil || result.Status != "failed" {
		t.Fatalf("rollback=%+v err=%v", result, err)
	}
	var failedHistory model.CloudManagedSyncHistory
	if err := db.First(&failedHistory, result.HistoryID).Error; err != nil || len(failedHistory.ResourceIDs) != 1 || failedHistory.ResourceIDs[0] != "id-three" {
		t.Fatal("failed MySQL history lost requested IDs")
	}
	if err := db.Callback().Update().Remove(callback); err != nil {
		t.Fatal(err)
	}
	db.Model(&model.CloudManagedResource{}).Where("provider_id = ?", "id-three").Count(&total)
	if total != 0 {
		t.Fatal("audit failure committed resource")
	}
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		index := calls.Add(1)
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			if index == 1 {
				close(started)
				<-release
			}
			return ids, nil
		}}, nil
	}
	req.ResourceIDs = []string{"id-four"}
	done := make(chan *model.CloudManagedSyncResult, 1)
	go func() { r, _ := svc.SyncManaged(context.Background(), req); done <- r }()
	<-started
	busy, err := svc.SyncManaged(context.Background(), req)
	if err != nil || busy.Status != "failed" || calls.Load() != 1 {
		t.Fatal("lease did not prevent concurrent cloud read")
	}
	if err := db.Model(&model.CloudRegionSyncLease{}).Where("region_id = ?", region.ID).Update("expires_epoch_ms", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncManaged(context.Background(), req); err == nil {
		t.Fatal("unreviewed expired sync accepted")
	}
	var previous model.CloudManagedSyncHistory
	if err := db.Where("status = ?", "running").First(&previous).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveManagedHistory(context.Background(), &model.ResolveCloudManagedHistoryReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", HistoryID: previous.ID, OperatorID: 9, Reason: "已经核对实际资源与本地记录的状态", ResourcesVerified: true}); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.SyncManaged(context.Background(), req)
	if err != nil || fresh.Status != "success" {
		t.Fatal("new owner failed")
	}
	close(release)
	stale := <-done
	if stale.Status != "failed" {
		t.Fatal("stale writer committed")
	}
	db.Model(&model.CloudManagedResource{}).Where("provider_id = ?", "id-four").Count(&total)
	if total != 1 {
		t.Fatal("stale writer duplicated resource")
	}
	if err := dao.NewCloudAccountDAO(db, zap.NewNop()).Delete(context.Background(), account.ID); !errors.Is(err, dao.ErrCloudAccountInUse) {
		t.Fatal("managed resource did not block account deletion")
	}
	if err := dao.NewCloudAccountRegionDAO(db, zap.NewNop()).Delete(context.Background(), region.ID); !errors.Is(err, dao.ErrCloudAccountRegionInUse) {
		t.Fatal("managed resource did not block region deletion")
	}
	other := &model.CloudAccount{Name: "managed-delete-race", Provider: model.ProviderAliyun, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	otherRegion := &model.CloudAccountRegion{CloudAccountID: other.ID, Region: "cn-hangzhou", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(otherRegion).Error; err != nil {
		t.Fatal(err)
	}
	readStarted, allowReturn := make(chan struct{}), make(chan struct{})
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			close(readStarted)
			<-allowReturn
			return ids, nil
		}}, nil
	}
	otherReq := &model.SyncCloudManagedResourcesReq{CloudAccountID: other.ID, CloudAccountRegionID: otherRegion.ID, Kind: "cache", ResourceIDs: []string{"id-five"}, OperatorID: 7}
	otherDone := make(chan *model.CloudManagedSyncResult, 1)
	go func() { r, _ := svc.SyncManaged(context.Background(), otherReq); otherDone <- r }()
	<-readStarted
	if err := dao.NewCloudAccountRegionDAO(db, zap.NewNop()).Delete(context.Background(), otherRegion.ID); err != nil {
		t.Fatal(err)
	}
	close(allowReturn)
	otherResult := <-otherDone
	if otherResult == nil || otherResult.Status != "failed" {
		t.Fatal("sync committed after region deletion")
	}
	db.Model(&model.CloudManagedResource{}).Where("cloud_account_id = ?", other.ID).Count(&total)
	if total != 0 {
		t.Fatal("deleted region acquired managed resource")
	}
	third := &model.CloudAccount{Name: "managed-account-delete-race", Provider: model.ProviderAliyun, Status: model.CloudAccountEnabled, AccessKey: access, SecretKey: secret}
	if err := db.Create(third).Error; err != nil {
		t.Fatal(err)
	}
	thirdRegion := &model.CloudAccountRegion{CloudAccountID: third.ID, Region: "cn-hangzhou", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(thirdRegion).Error; err != nil {
		t.Fatal(err)
	}
	readStarted, allowReturn = make(chan struct{}), make(chan struct{})
	thirdReq := &model.SyncCloudManagedResourcesReq{CloudAccountID: third.ID, CloudAccountRegionID: thirdRegion.ID, Kind: "clb", ResourceIDs: []string{"id-six"}, OperatorID: 7}
	thirdDone := make(chan *model.CloudManagedSyncResult, 1)
	go func() { r, _ := svc.SyncManaged(context.Background(), thirdReq); thirdDone <- r }()
	<-readStarted
	if err := dao.NewCloudAccountDAO(db, zap.NewNop()).Delete(context.Background(), third.ID); err != nil {
		t.Fatal(err)
	}
	close(allowReturn)
	thirdResult := <-thirdDone
	if thirdResult == nil || thirdResult.Status != "failed" {
		t.Fatal("sync committed after account deletion")
	}
	db.Model(&model.CloudManagedResource{}).Where("cloud_account_id = ?", third.ID).Count(&total)
	if total != 0 {
		t.Fatal("deleted account acquired managed resource")
	}
}
