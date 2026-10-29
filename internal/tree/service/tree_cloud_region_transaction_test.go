package service

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type snapshotClient struct {
	treeUtils.CloudComputeClient
	read func(context.Context) ([]*model.TreeCloudResource, error)
}

func (c snapshotClient) ListInstances(ctx context.Context, _ []string) ([]*model.TreeCloudResource, error) {
	return c.read(ctx)
}

func regionSyncFixture(t *testing.T) (*gorm.DB, *treeCloudService, *model.CloudAccount, *model.CloudAccountRegion, *model.TreeNode) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "cloud.db")+"?_busy_timeout=5000"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeNode{}, &model.TreeCloudResource{}, &model.CloudResourceChangeLog{}, &model.CloudResourceSyncHistory{}, &model.CloudRegionSyncLease{}); err != nil {
		t.Fatal(err)
	}
	account := &model.CloudAccount{Name: "offline", Provider: model.ProviderAWS, Status: model.CloudAccountEnabled, AccessKey: "cipher", SecretKey: "cipher"}
	if err := db.Create(account).Error; err != nil {
		t.Fatal(err)
	}
	region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "us-west-2", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(region).Error; err != nil {
		t.Fatal(err)
	}
	node := &model.TreeNode{Name: "offline-node"}
	if err := db.Create(node).Error; err != nil {
		t.Fatal(err)
	}
	return db, &treeCloudService{dao: dao.NewTreeCloudDAO(db, zap.NewNop()), logger: zap.NewNop()}, account, region, node
}

func resource(account *model.CloudAccount, region *model.CloudAccountRegion, id string) *model.TreeCloudResource {
	return &model.TreeCloudResource{Name: id, InstanceID: id, ResourceType: model.ResourceTypeECS, CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Region: region.Region, Status: model.CloudResourceRunning}
}

func TestFullRegionReconcilesAtomicallyAndPreservesLocalSSH(t *testing.T) {
	db, svc, account, region, node := regionSyncFixture(t)
	keep, gone := resource(account, region, "keep"), resource(account, region, "gone")
	keep.Username = "local-user"
	keep.PublicIP = "192.0.2.1"
	keep.Password = "private-cipher"
	keep.Key = "local-private-key"
	keep.Port = 2222
	if err := db.Create(keep).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(gone).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.dao.BindTreeNodes(context.Background(), keep.ID, []int{node.ID}); err != nil {
		t.Fatal(err)
	}
	if err := svc.dao.BindTreeNodes(context.Background(), gone.ID, []int{node.ID}); err != nil {
		t.Fatal(err)
	}
	snapshot := []*model.TreeCloudResource{resource(account, region, "keep"), resource(account, region, "new")}
	snapshot[0].Name = "updated"
	snapshot[0].Password = "provider-must-not-overwrite"
	snapshot[1].Password, snapshot[1].Key, snapshot[1].Username, snapshot[1].Port = "provider-password", "provider-key", "provider-user", 9999
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) { return snapshot, nil }}, nil
	}
	result, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull, AutoBind: true, BindNodeID: node.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.NewCount != 1 || result.UpdateCount != 1 || result.DeleteCount != 1 {
		t.Fatalf("stats=%+v", result)
	}
	var history model.CloudResourceSyncHistory
	if err := db.First(&history, result.HistoryID).Error; err != nil || history.SyncStatus != "success" || history.NewCount != 1 || history.UpdateCount != 1 || history.DeleteCount != 1 {
		t.Fatalf("transactional history=%+v err=%v", history, err)
	}
	var stored model.TreeCloudResource
	if err := db.First(&stored, keep.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Name != "updated" || stored.PublicIP != "" || stored.Username != "local-user" || stored.Password != "private-cipher" || stored.Key != "local-private-key" || stored.Port != 2222 {
		t.Fatal("provider update overwrote local SSH data")
	}
	var keptBinding int64
	db.Table("cl_tree_node_cloud").Where("tree_cloud_resource_id = ? AND tree_node_id = ?", keep.ID, node.ID).Count(&keptBinding)
	if keptBinding != 1 {
		t.Fatal("existing local binding was lost")
	}
	var goneCount int64
	db.Model(&model.TreeCloudResource{}).Where("id = ?", gone.ID).Count(&goneCount)
	if goneCount != 0 {
		t.Fatal("absent resource remains active")
	}
	var bindings int64
	db.Table("cl_tree_node_cloud").Where("tree_cloud_resource_id = ?", gone.ID).Count(&bindings)
	if bindings != 0 {
		t.Fatal("deleted resource binding remains")
	}
	var created model.TreeCloudResource
	if err := db.Where("instance_id = ?", "new").First(&created).Error; err != nil {
		t.Fatal(err)
	}
	if created.Password != "" || created.Key != "" || created.Username != "" || created.Port == 9999 {
		t.Fatal("provider injected local SSH credentials")
	}
	db.Table("cl_tree_node_cloud").Where("tree_cloud_resource_id = ? AND tree_node_id = ?", created.ID, node.ID).Count(&bindings)
	if bindings != 1 {
		t.Fatal("new resource missing binding")
	}
	var logs int64
	db.Model(&model.CloudResourceChangeLog{}).Count(&logs)
	if logs != 3 {
		t.Fatalf("change logs=%d", logs)
	}
}

func TestRegionWriteFailureAndCancellationRollBackAllChanges(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "write_failure", true: "cancelled"}[cancelled], func(t *testing.T) {
			db, svc, account, region, node := regionSyncFixture(t)
			old := resource(account, region, "old")
			if err := db.Create(old).Error; err != nil {
				t.Fatal(err)
			}
			if err := svc.dao.BindTreeNodes(context.Background(), old.ID, []int{node.ID}); err != nil {
				t.Fatal(err)
			}
			svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
				return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
					return []*model.TreeCloudResource{resource(account, region, "new")}, nil
				}}, nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			key := "test:fail-change-log"
			if err := db.Callback().Create().Before("gorm:create").Register(key, func(tx *gorm.DB) {
				if _, ok := tx.Statement.Dest.(*model.CloudResourceChangeLog); ok {
					if cancelled {
						cancel()
					} else {
						tx.AddError(errors.New("fixture failure"))
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			defer db.Callback().Create().Remove(key)
			result, err := svc.syncOneRegion(ctx, account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull, AutoBind: true, BindNodeID: node.ID})
			if err == nil || result.NewCount != 0 || result.DeleteCount != 0 {
				t.Fatalf("partial result=%+v err=%v", result, err)
			}
			var history model.CloudResourceSyncHistory
			if err := db.First(&history, result.HistoryID).Error; err != nil {
				t.Fatal(err)
			}
			want := model.SyncStatus("failed")
			if cancelled {
				want = "cancelled"
			}
			if history.SyncStatus != want {
				t.Fatalf("failure history status=%s", history.SyncStatus)
			}
			var count int64
			db.Model(&model.TreeCloudResource{}).Where("id = ?", old.ID).Count(&count)
			if count != 1 {
				t.Fatal("old resource deleted despite rollback")
			}
			db.Table("cl_tree_node_cloud").Where("tree_cloud_resource_id = ?", old.ID).Count(&count)
			if count != 1 {
				t.Fatal("binding deleted despite rollback")
			}
			db.Model(&model.CloudResourceChangeLog{}).Count(&count)
			if count != 0 {
				t.Fatal("log committed despite rollback")
			}
		})
	}
}

func TestRegionLeaseSerializesReadersAndRejectsStaleOwner(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	var reads atomic.Int32
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(ctx context.Context) ([]*model.TreeCloudResource, error) {
			reads.Add(1)
			close(started)
			select {
			case <-release:
				return []*model.TreeCloudResource{}, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}}, nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull})
		done <- err
	}()
	<-started
	if _, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeIncremental}); !errors.Is(err, dao.ErrRegionSyncBusy) {
		t.Fatalf("expected lease busy: %v", err)
	}
	if reads.Load() != 1 {
		t.Fatal("busy sync reached provider")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	first, err := svc.dao.AcquireRegionSyncLease(context.Background(), region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.CloudRegionSyncLease{}).Where("region_id = ?", region.ID).Update("expires_epoch_ms", 0).Error; err != nil {
		t.Fatal(err)
	}
	second, err := svc.dao.AcquireRegionSyncLease(context.Background(), region.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	err = svc.dao.ApplyRegionSync(context.Background(), account, region, first, func(dao.TreeCloudDAO) error { return nil })
	if !errors.Is(err, dao.ErrRegionSyncLeaseLost) {
		t.Fatalf("stale lease accepted: %v", err)
	}
	if err := svc.dao.ReleaseRegionSyncLease(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := svc.dao.ApplyRegionSync(context.Background(), account, region, second, func(dao.TreeCloudDAO) error { return nil }); err != nil {
		t.Fatalf("new lease lost: %v", err)
	}
}

func TestCredentialRotationDuringCloudReadPreventsCommit(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			close(started)
			<-release
			return []*model.TreeCloudResource{resource(account, region, "new")}, nil
		}}, nil
	}
	done := make(chan error, 1)
	go func() {
		_, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull})
		done <- err
	}()
	<-started
	if err := db.Model(&model.CloudAccount{}).Where("id = ?", account.ID).Update("credential_version", 1).Error; err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err == nil {
		t.Fatal("stale credentials committed")
	}
	var count int64
	db.Model(&model.TreeCloudResource{}).Count(&count)
	if count != 0 {
		t.Fatal("stale snapshot modified data")
	}
}

func TestExpiredReaderCannotOverwriteNewerSnapshot(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		index := calls.Add(1)
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			if index == 1 {
				close(started)
				<-release
				r := resource(account, region, "same")
				r.Name = "stale"
				return []*model.TreeCloudResource{r}, nil
			}
			r := resource(account, region, "same")
			r.Name = "fresh"
			return []*model.TreeCloudResource{r}, nil
		}}, nil
	}
	credentials := treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}
	done := make(chan error, 1)
	go func() {
		_, err := svc.syncOneRegion(context.Background(), account, credentials, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull})
		done <- err
	}()
	<-started
	if err := db.Model(&model.CloudRegionSyncLease{}).Where("region_id = ?", region.ID).Update("expires_epoch_ms", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.syncOneRegion(context.Background(), account, credentials, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; !errors.Is(err, dao.ErrRegionSyncLeaseLost) {
		t.Fatalf("stale reader result: %v", err)
	}
	var row model.TreeCloudResource
	if err := db.Where("instance_id = ?", "same").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Name != "fresh" {
		t.Fatal("stale cloud snapshot overwrote fresh result")
	}
}

func TestMultiRegionPartialReportsOnlyCommittedChanges(t *testing.T) {
	db, svc, account, first, _ := regionSyncFixture(t)
	second := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "us-east-1", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(second).Error; err != nil {
		t.Fatal(err)
	}
	svc.clientFactory = func(_ model.CloudProvider, _, _, region string, _ *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			if region == second.Region {
				return nil, errors.New("private-credential-canary")
			}
			return []*model.TreeCloudResource{resource(account, first, "created")}, nil
		}}, nil
	}
	resp := &model.SyncCloudResourceResp{}
	err := svc.syncCloudResourcesWithCredentials(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, []*model.CloudAccountRegion{first, second}, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}, resp)
	if err == nil || resp.NewCount != 1 || resp.FailedCount != 1 || len(resp.Regions) != 2 || resp.Regions[0].Status != "success" || resp.Regions[1].Status != "failed" || resp.Regions[1].ErrorCategory != "sync_failed" {
		t.Fatalf("incorrect partial result: %+v", resp)
	}
	var count int64
	db.Model(&model.TreeCloudResource{}).Count(&count)
	if count != 1 {
		t.Fatal("committed region missing")
	}
}

type commitUnknownDAO struct{ dao.TreeCloudDAO }

func (d commitUnknownDAO) ApplyRegionSync(context.Context, *model.CloudAccount, *model.CloudAccountRegion, *dao.RegionSyncLease, func(dao.TreeCloudDAO) error) error {
	return dao.ErrRegionSyncCommitUnknown
}
func TestUnknownCommitDoesNotReportRollbackOrReleaseLease(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	svc.dao = commitUnknownDAO{svc.dao}
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) { return []*model.TreeCloudResource{}, nil }}, nil
	}
	resp := &model.SyncCloudResourceResp{}
	_ = svc.syncCloudResourcesWithCredentials(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, []*model.CloudAccountRegion{region}, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}, resp)
	if len(resp.Regions) != 1 || resp.Regions[0].Status != "unknown" || resp.Regions[0].ErrorCategory != "commit_unknown" || resp.NewCount != 0 {
		t.Fatalf("unknown result=%+v", resp)
	}
	if resp.Regions[0].RetrySafe || resp.Regions[0].HistoryID <= 0 {
		t.Fatal("unknown outcome omitted unsafe retry/audit status")
	}
	var history model.CloudResourceSyncHistory
	if err := db.First(&history, resp.Regions[0].HistoryID).Error; err != nil || history.SyncStatus != "unknown" {
		t.Fatalf("unknown history=%+v err=%v", history, err)
	}
	var lease model.CloudRegionSyncLease
	if err := db.First(&lease, region.ID).Error; err != nil || lease.Token == "" {
		t.Fatal("unknown commit lease was released")
	}
}

func TestExpiredRunningHistoryProjectsUnknownWithoutWriting(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	row := model.CloudResourceSyncHistory{CloudAccountID: account.ID, RegionID: region.ID, Region: region.Region, OperationID: "fixture", LeaseExpiresMS: 1, SyncMode: model.SyncModeFull, SyncStatus: "running", StartTime: time.Now().UTC()}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	histories, _, err := svc.dao.GetSyncHistoryList(context.Background(), &model.GetCloudResourceSyncHistoryReq{ListReq: model.ListReq{Page: 1, Size: 20}, SyncStatus: "unknown"})
	if err != nil || len(histories) != 1 || histories[0].SyncStatus != "unknown" || histories[0].RetrySafe == nil || *histories[0].RetrySafe {
		t.Fatalf("history projection=%+v err=%v", histories, err)
	}
	encoded, err := json.Marshal(histories)
	if err != nil || strings.Contains(string(encoded), "fixture") {
		t.Fatal("history response leaked operation token")
	}
	var stored model.CloudResourceSyncHistory
	if err := db.First(&stored, row.ID).Error; err != nil || stored.SyncStatus != "running" {
		t.Fatal("history GET mutated persistent state")
	}
}

func TestRegionSuccessHistoryFailureRollsBackChanges(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	old := resource(account, region, "old")
	if err := db.Create(old).Error; err != nil {
		t.Fatal(err)
	}
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			return []*model.TreeCloudResource{resource(account, region, "new")}, nil
		}}, nil
	}
	callback := "test:fail-success-history"
	if err := db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "cl_tree_cloud_resource_sync_history" {
			if values, ok := tx.Statement.Dest.(map[string]any); ok && values["sync_status"] == "success" {
				tx.AddError(errors.New("injected audit failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	defer db.Callback().Update().Remove(callback)
	result, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull})
	if err == nil || result.HistoryID <= 0 {
		t.Fatal("audit failure was not reported")
	}
	var count int64
	db.Model(&model.TreeCloudResource{}).Where("id = ?", old.ID).Count(&count)
	if count != 1 {
		t.Fatal("resource committed without success audit")
	}
	var history model.CloudResourceSyncHistory
	if err := db.First(&history, result.HistoryID).Error; err != nil || history.SyncStatus != "failed" {
		t.Fatalf("history=%+v err=%v", history, err)
	}
}

type brokenAuditDAO struct{ dao.TreeCloudDAO }

func (brokenAuditDAO) CreateSyncHistory(context.Context, *model.CloudResourceSyncHistory) error {
	return errors.New("private-database-canary")
}

func TestRegionAuditUnavailableFailsBeforeCloudRead(t *testing.T) {
	_, svc, account, region, _ := regionSyncFixture(t)
	svc.dao = brokenAuditDAO{svc.dao}
	var calls atomic.Int32
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		calls.Add(1)
		return nil, nil
	}
	resp := &model.SyncCloudResourceResp{}
	_ = svc.syncCloudResourcesWithCredentials(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, []*model.CloudAccountRegion{region}, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}, resp)
	if calls.Load() != 0 || len(resp.Regions) != 1 || resp.Regions[0].ErrorCategory != "audit_unavailable" || resp.Regions[0].HistoryID != 0 || resp.Regions[0].RetrySafe {
		t.Fatalf("audit result=%+v", resp)
	}
}

func TestCancelledBeforeLeaseWritesExplicitCancelledHistory(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp := &model.SyncCloudResourceResp{}
	_ = svc.syncCloudResourcesWithCredentials(ctx, account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, []*model.CloudAccountRegion{region}, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull}, resp)
	if len(resp.Regions) != 1 || resp.Regions[0].Status != "cancelled" || resp.Regions[0].HistoryID <= 0 {
		t.Fatalf("cancel result=%+v", resp)
	}
	var history model.CloudResourceSyncHistory
	if err := db.First(&history, resp.Regions[0].HistoryID).Error; err != nil || history.SyncStatus != "cancelled" {
		t.Fatalf("cancel history=%+v err=%v", history, err)
	}
}

func TestFullSyncRestoresPreviouslyDeletedResourceWithoutLosingSSH(t *testing.T) {
	db, svc, account, region, _ := regionSyncFixture(t)
	old := resource(account, region, "restored")
	old.Username = "local-user"
	old.Password = "cipher-kept"
	if err := db.Create(old).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(old).Error; err != nil {
		t.Fatal(err)
	}
	svc.clientFactory = func(model.CloudProvider, string, string, string, *zap.Logger) (treeUtils.CloudComputeClient, error) {
		return snapshotClient{read: func(context.Context) ([]*model.TreeCloudResource, error) {
			return []*model.TreeCloudResource{resource(account, region, "restored")}, nil
		}}, nil
	}
	result, err := svc.syncOneRegion(context.Background(), account, treeUtils.CloudCredentials{AccessKey: "offline-access", SecretKey: "offline-secret"}, region, &model.SyncTreeCloudResourceReq{SyncMode: model.SyncModeFull})
	if err != nil || result.NewCount != 1 {
		t.Fatalf("restore=%+v err=%v", result, err)
	}
	var got model.TreeCloudResource
	if err := db.First(&got, old.ID).Error; err != nil || got.Username != "local-user" || got.Password != "cipher-kept" {
		t.Fatal("restored resource lost local SSH fields")
	}
}
