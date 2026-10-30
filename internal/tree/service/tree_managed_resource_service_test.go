package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type managedFixture struct {
	read func(context.Context, []string, string) ([]string, error)
}

func (f managedFixture) ListRDS(c context.Context, ids []string) ([]model.AliyunRDSInstance, error) {
	got, err := f.read(c, ids, "rds")
	if err != nil {
		return nil, err
	}
	out := make([]model.AliyunRDSInstance, 0, len(got))
	for _, id := range got {
		out = append(out, model.AliyunRDSInstance{InstanceID: id, Region: "cn-hangzhou", Name: "fixture-db", Engine: "MySQL"})
	}
	return out, nil
}
func (f managedFixture) ListSLB(c context.Context, ids []string) ([]model.AliyunLoadBalancer, error) {
	got, err := f.read(c, ids, "clb")
	if err != nil {
		return nil, err
	}
	out := make([]model.AliyunLoadBalancer, 0, len(got))
	for _, id := range got {
		out = append(out, model.AliyunLoadBalancer{InstanceID: id, Region: "cn-hangzhou", Name: "fixture-clb"})
	}
	return out, nil
}
func (f managedFixture) ListCache(c context.Context, ids []string) ([]model.AliyunCacheInstance, error) {
	got, err := f.read(c, ids, "cache")
	if err != nil {
		return nil, err
	}
	out := make([]model.AliyunCacheInstance, 0, len(got))
	for _, id := range got {
		out = append(out, model.AliyunCacheInstance{InstanceID: id, Region: "cn-hangzhou", Name: "fixture-cache", InstanceType: "Redis"})
	}
	return out, nil
}

func managedServiceFixture(t *testing.T) (*gorm.DB, *managedCloudService, *model.CloudAccount, *model.CloudAccountRegion) {
	t.Helper()
	db, _, account, region, _ := regionSyncFixture(t)
	if err := db.AutoMigrate(&model.CloudManagedResource{}, &model.CloudManagedSyncHistory{}, &model.CloudManagedOperation{}); err != nil {
		t.Fatal(err)
	}
	encryptionKey := strings.Repeat("k", 32)
	access, err := treeUtils.EncryptPassword("test-ak", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	secret, err := treeUtils.EncryptPassword("test-sk", encryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(account).Updates(map[string]any{"provider": model.ProviderAliyun, "access_key": access, "secret_key": secret}).Error; err != nil {
		t.Fatal(err)
	}
	region.Region = "cn-hangzhou"
	if err := db.Model(region).Update("region", region.Region).Error; err != nil {
		t.Fatal(err)
	}
	svc := mustManagedCloudService(t, encryptionKey, zap.NewNop(), dao.NewCloudAccountDAO(db, zap.NewNop()), dao.NewTreeCloudDAO(db, zap.NewNop()), dao.NewManagedCloudDAO(db)).(*managedCloudService)
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			return append([]string{}, ids...), nil
		}}, nil
	}
	return db, svc, account, region
}

func TestManagedIncrementalTypesPreserveLocalAndNeverDelete(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	for _, kind := range []string{"rds", "clb", "cache"} {
		t.Run(kind, func(t *testing.T) {
			req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: kind, ResourceIDs: []string{"id-one", "id-two"}, OperatorID: 7}
			result, err := svc.SyncManaged(context.Background(), req)
			if err != nil || result.Status != "success" || result.NewCount != 2 || result.HistoryID <= 0 {
				t.Fatalf("sync=%+v err=%v", result, err)
			}
			var firstHistory model.CloudManagedSyncHistory
			if err := db.First(&firstHistory, result.HistoryID).Error; err != nil || len(firstHistory.ResourceIDs) != 2 || firstHistory.ResourceIDs[0] != "id-one" || firstHistory.ResourceIDs[1] != "id-two" {
				t.Fatal("successful history lost request IDs")
			}
			listReq := &model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: kind, Page: 1, Size: 20}
			listed, err := svc.ListManaged(context.Background(), listReq)
			if err != nil || listed.Total != 2 || len(listed.Items) != 2 {
				t.Fatal("typed rows missing")
			}
			first := listed.Items[0]
			if got, err := svc.GetManaged(context.Background(), listReq, first.ID); err != nil || got.ProviderID != first.ProviderID {
				t.Fatal("scoped detail missing")
			}
			wrong := *listReq
			wrong.Kind = "rds"
			if kind == "rds" {
				wrong.Kind = "clb"
			}
			if _, err := svc.GetManaged(context.Background(), &wrong, first.ID); err == nil {
				t.Fatal("cross-type detail leaked")
			}
			if err := db.Model(&model.CloudManagedResource{}).Where("id = ?", first.ID).Update("local_description", "locally-maintained").Error; err != nil {
				t.Fatal(err)
			}
			req.ResourceIDs = []string{"id-one"}
			result, err = svc.SyncManaged(context.Background(), req)
			if err != nil || result.Status != "success" || result.UpdateCount != 1 {
				t.Fatalf("update=%+v err=%v", result, err)
			}
			listed, err = svc.ListManaged(context.Background(), listReq)
			if err != nil || listed.Total != 2 || listed.Items[0].LocalDescription != "locally-maintained" {
				t.Fatal("incremental sync deleted or overwrote local data")
			}
			var history model.CloudManagedSyncHistory
			if err := db.First(&history, result.HistoryID).Error; err != nil || history.Status != "success" || history.OperatorID != 7 {
				t.Fatal("missing transactional history")
			}
			if len(history.ResourceIDs) != 1 || history.ResourceIDs[0] != "id-one" {
				t.Fatal("updated history lost requested IDs")
			}
			data, err := json.Marshal(listed.Items)
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"test-ak", "test-sk", "password", "ssh"} {
				if strings.Contains(strings.ToLower(string(data)), secret) {
					t.Fatal("response leaked sensitive field")
				}
			}
			var ecs int64
			db.Model(&model.TreeCloudResource{}).Count(&ecs)
			if ecs != 0 {
				t.Fatal("managed metadata was written to ECS table")
			}
		})
	}
}

func TestManagedFailureAndAuditRollbackAreSafe(t *testing.T) {
	for _, mode := range []string{"partial", "audit", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			db, svc, account, region := managedServiceFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
				return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
					if mode == "partial" {
						return []string{ids[0]}, nil
					}
					if mode == "cancel" {
						cancel()
						return ids, nil
					}
					return ids, nil
				}}, nil
			}
			if mode == "audit" {
				callback := "test:managed-audit-failure"
				if err := db.Callback().Update().Before("gorm:update").Register(callback, func(tx *gorm.DB) {
					if tx.Statement.Table == "cl_tree_cloud_managed_sync_history" {
						if values, ok := tx.Statement.Dest.(map[string]any); ok && values["status"] == "success" {
							tx.AddError(errors.New("fixture private failure"))
						}
					}
				}); err != nil {
					t.Fatal(err)
				}
				defer db.Callback().Update().Remove(callback)
			}
			result, err := svc.SyncManaged(ctx, &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one", "id-two"}, OperatorID: 7})
			if err != nil || result == nil || result.Status == "success" || result.HistoryID <= 0 {
				t.Fatalf("failure=%+v err=%v", result, err)
			}
			var n int64
			db.Model(&model.CloudManagedResource{}).Count(&n)
			if n != 0 {
				t.Fatal("failure committed partial metadata")
			}
			var h model.CloudManagedSyncHistory
			if err := db.First(&h, result.HistoryID).Error; err != nil {
				t.Fatal(err)
			}
			if len(h.ResourceIDs) != 2 || h.ResourceIDs[0] != "id-one" || h.ResourceIDs[1] != "id-two" {
				t.Fatal("failed history lost requested IDs")
			}
			if h.Status != "failed" && h.Status != "cancelled" {
				t.Fatalf("history status=%s", h.Status)
			}
		})
	}
}

func TestManagedSyncSerializesAndFencesExpiredOwner(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	start, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		index := calls.Add(1)
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			if index == 1 {
				close(start)
				<-release
			}
			return ids, nil
		}}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7}
	done := make(chan *model.CloudManagedSyncResult, 1)
	go func() { r, _ := svc.SyncManaged(context.Background(), req); done <- r }()
	<-start
	busy, err := svc.SyncManaged(context.Background(), req)
	if err != nil || busy.Status != "failed" || busy.ErrorCategory != "busy" || calls.Load() != 1 {
		t.Fatalf("busy=%+v err=%v", busy, err)
	}
	if err := db.Model(&model.CloudRegionSyncLease{}).Where("region_id = ?", region.ID).Update("expires_epoch_ms", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.SyncManaged(context.Background(), req); err == nil {
		t.Fatal("expired in-flight result bypassed review")
	}
	var previous model.CloudManagedSyncHistory
	if err := db.Where("status = ?", "running").First(&previous).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.ResolveManagedHistory(context.Background(), &model.ResolveCloudManagedHistoryReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", HistoryID: previous.ID, OperatorID: 7, Reason: "已经核对实际资源与本地记录的状态", ResourcesVerified: true}); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.SyncManaged(context.Background(), req)
	if err != nil || fresh.Status != "success" {
		t.Fatalf("fresh=%+v err=%v", fresh, err)
	}
	close(release)
	stale := <-done
	var reviewed model.CloudManagedSyncHistory
	if err := db.First(&reviewed, previous.ID).Error; err != nil {
		t.Fatal(err)
	}
	if reviewed.ResolvedAt == nil || reviewed.Status != "running" {
		t.Fatal("late worker rewrote reviewed history")
	}
	if stale.Status != "failed" {
		t.Fatalf("stale=%+v", stale)
	}
	var n int64
	db.Model(&model.CloudManagedResource{}).Count(&n)
	if n != 1 {
		t.Fatal("stale writer duplicated data")
	}
}

func TestManagedValidationRejectsUnsupportedAndUnscopedReads(t *testing.T) {
	_, svc, account, region := managedServiceFixture(t)
	for _, req := range []*model.SyncCloudManagedResourcesReq{
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", OperatorID: 7},
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "ecs", ResourceIDs: []string{"id-one"}, OperatorID: 7},
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one", "id-one"}, OperatorID: 7},
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID + 1, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7},
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}},
	} {
		if _, err := svc.SyncManaged(context.Background(), req); err == nil {
			t.Fatal("invalid or unscoped request accepted")
		}
	}
}

func TestManagedHistoryExpiredProjectionIsReadOnly(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	history := model.CloudManagedSyncHistory{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"id-one", "id-two"}, Status: "running", OperationID: "private-fixture", LeaseExpiresMS: 1, StartedAt: time.Now().UTC(), OperatorID: 7}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	result, err := svc.ListManagedHistory(context.Background(), &model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", Page: 1, Size: 20})
	if err != nil || len(result.Items) != 1 || result.Items[0].Status != "unknown" || result.Items[0].RetrySafe == nil || *result.Items[0].RetrySafe {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(result.Items[0].ResourceIDs) != 2 || result.Items[0].ResourceIDs[1] != "id-two" {
		t.Fatal("restart history lost requested IDs")
	}
	encoded, _ := json.Marshal(result)
	if strings.Contains(string(encoded), "private-fixture") {
		t.Fatal("operation token leaked")
	}
	var actual model.CloudManagedSyncHistory
	if err := db.First(&actual, history.ID).Error; err != nil || actual.Status != "running" {
		t.Fatal("GET history changed data")
	}
}

type unavailableManagedAudit struct{ dao.ManagedCloudDAO }

func (unavailableManagedAudit) CreateHistory(context.Context, *model.CloudManagedSyncHistory) error {
	return errors.New("private-audit-canary")
}

type unknownManagedCommit struct{ dao.TreeCloudDAO }

func (unknownManagedCommit) ApplyRegionSync(context.Context, *model.CloudAccount, *model.CloudAccountRegion, *dao.RegionSyncLease, func(dao.TreeCloudDAO) error) error {
	return dao.ErrRegionSyncCommitUnknown
}

func TestManagedUnknownAndAuditUnavailableFailClosed(t *testing.T) {
	for _, mode := range []string{"unknown", "audit", "private-provider"} {
		t.Run(mode, func(t *testing.T) {
			db, svc, account, region := managedServiceFixture(t)
			if mode == "unknown" {
				svc.regions = unknownManagedCommit{svc.regions}
			}
			if mode == "audit" {
				svc.store = unavailableManagedAudit{svc.store}
			}
			if mode == "private-provider" {
				svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
					return nil, errors.New("private-provider-canary")
				}
			}
			result, err := svc.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7})
			if err != nil || result == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			switch mode {
			case "unknown":
				if result.Status != "unknown" || result.RetrySafe || result.HistoryID <= 0 {
					t.Fatal("unknown commit was misreported")
				}
				var h model.CloudManagedSyncHistory
				if err := db.First(&h, result.HistoryID).Error; err != nil || h.Status != "unknown" {
					t.Fatal("unknown history missing")
				}
				if len(h.ResourceIDs) != 1 || h.ResourceIDs[0] != "id-one" {
					t.Fatal("unknown history lost requested ID")
				}
			case "audit":
				if result.ErrorCategory != "audit_unavailable" || result.RetrySafe || result.HistoryID != 0 {
					t.Fatal("audit failure was hidden")
				}
			case "private-provider":
				if result.Status != "failed" || result.HistoryID <= 0 {
					t.Fatal("provider failure was hidden")
				}
			}
			data, _ := json.Marshal(result)
			if strings.Contains(string(data), "private-") {
				t.Fatal("private failure escaped")
			}
			var n int64
			db.Model(&model.CloudManagedResource{}).Count(&n)
			if n != 0 {
				t.Fatal("failure created resource")
			}
		})
	}
}

func TestManagedCredentialRotationDuringReadRollsBack(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			close(started)
			<-release
			return ids, nil
		}}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7}
	done := make(chan *model.CloudManagedSyncResult, 1)
	go func() { r, _ := svc.SyncManaged(context.Background(), req); done <- r }()
	<-started
	if err := db.Model(&model.CloudAccount{}).Where("id = ?", account.ID).Update("credential_version", 1).Error; err != nil {
		t.Fatal(err)
	}
	close(release)
	result := <-done
	if result == nil || result.Status != "failed" {
		t.Fatalf("result=%+v", result)
	}
	var count int64
	db.Model(&model.CloudManagedResource{}).Count(&count)
	if count != 0 {
		t.Fatal("stale credential snapshot committed")
	}
}

func TestManagedResourcesBlockAccountAndRegionRemoval(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	result, err := svc.SyncManaged(context.Background(), &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7})
	if err != nil || result.Status != "success" {
		t.Fatal("fixture sync failed")
	}
	accounts := dao.NewCloudAccountDAO(db, zap.NewNop())
	regions := dao.NewCloudAccountRegionDAO(db, zap.NewNop())
	if err := accounts.Delete(context.Background(), account.ID); !errors.Is(err, dao.ErrCloudAccountInUse) {
		t.Fatalf("account removal=%v", err)
	}
	if err := regions.Delete(context.Background(), region.ID); !errors.Is(err, dao.ErrCloudAccountRegionInUse) {
		t.Fatalf("region removal=%v", err)
	}
	count, err := regions.GetResourceCountByRegion(context.Background(), region.ID)
	if err != nil || count != 1 {
		t.Fatalf("region count=%d err=%v", count, err)
	}
	if err := accounts.UpdateAccountAndRegions(context.Background(), &model.CloudAccount{Model: model.Model{ID: account.ID}}, nil, []model.CreateCloudAccountRegionItem{{Region: "cn-beijing"}}); !errors.Is(err, dao.ErrCloudAccountRegionInUse) {
		t.Fatalf("region replacement=%v", err)
	}
	second := &model.CloudAccount{Name: "other-account", Provider: model.ProviderAliyun, Status: model.CloudAccountEnabled, AccessKey: "cipher", SecretKey: "cipher"}
	if err := db.Create(second).Error; err != nil {
		t.Fatal(err)
	}
	if err := accounts.BatchDelete(context.Background(), []int{account.ID, second.ID}); !errors.Is(err, dao.ErrCloudAccountInUse) {
		t.Fatalf("batch removal=%v", err)
	}
	var n int64
	db.Model(&model.CloudAccount{}).Where("id IN ?", []int{account.ID, second.ID}).Count(&n)
	if n != 2 {
		t.Fatal("batch delete partially committed")
	}
}

func TestRegionDeletionDuringManagedReadPreventsCommit(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	start, release := make(chan struct{}), make(chan struct{})
	svc.factory = func(treeUtils.CloudCredentials, string) (managedReader, error) {
		return managedFixture{read: func(_ context.Context, ids []string, _ string) ([]string, error) {
			close(start)
			<-release
			return ids, nil
		}}, nil
	}
	req := &model.SyncCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: []string{"id-one"}, OperatorID: 7}
	done := make(chan *model.CloudManagedSyncResult, 1)
	go func() { result, _ := svc.SyncManaged(context.Background(), req); done <- result }()
	<-start
	if err := dao.NewCloudAccountRegionDAO(db, zap.NewNop()).Delete(context.Background(), region.ID); err != nil {
		t.Fatal(err)
	}
	close(release)
	result := <-done
	if result == nil || result.Status != "failed" {
		t.Fatalf("stale sync=%+v", result)
	}
	var n int64
	db.Model(&model.CloudManagedResource{}).Count(&n)
	if n != 0 {
		t.Fatal("sync wrote to deleted region")
	}
}

func TestManagedHistoryDoesNotCrossAccountOrRegion(t *testing.T) {
	db, svc, account, region := managedServiceFixture(t)
	other := &model.CloudAccount{Name: "other-history", Provider: model.ProviderAliyun, Status: model.CloudAccountEnabled, AccessKey: "cipher", SecretKey: "cipher"}
	if err := db.Create(other).Error; err != nil {
		t.Fatal(err)
	}
	otherRegion := &model.CloudAccountRegion{CloudAccountID: other.ID, Region: "cn-hangzhou", Status: model.CloudAccountRegionEnabled}
	if err := db.Create(otherRegion).Error; err != nil {
		t.Fatal(err)
	}
	entries := []*model.CloudManagedSyncHistory{
		{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", ResourceIDs: model.StringList{"own-id"}, Status: "failed", OperatorID: 7, StartedAt: time.Now().UTC()},
		{CloudAccountID: other.ID, CloudAccountRegionID: otherRegion.ID, Kind: "rds", ResourceIDs: model.StringList{"other-id"}, Status: "failed", OperatorID: 7, StartedAt: time.Now().UTC()},
	}
	if err := db.Create(&entries).Error; err != nil {
		t.Fatal(err)
	}
	req := &model.ListCloudManagedResourcesReq{CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Kind: "rds", Page: 1, Size: 20}
	result, err := svc.ListManagedHistory(context.Background(), req)
	if err != nil || len(result.Items) != 1 || len(result.Items[0].ResourceIDs) != 1 || result.Items[0].ResourceIDs[0] != "own-id" {
		t.Fatal("history crossed account scope")
	}
	req.CloudAccountRegionID = otherRegion.ID
	if _, err := svc.ListManagedHistory(context.Background(), req); err == nil {
		t.Fatal("foreign region history accepted")
	}
}
