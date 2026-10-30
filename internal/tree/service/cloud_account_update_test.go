package service

import (
	"context"
	"errors"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCloudAccountRegionDeltaSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "regions.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	defer pool.Close()
	assertCloudAccountRegionDelta(t, db)
}
func assertCloudAccountRegionDelta(t *testing.T, db *gorm.DB) {
	t.Helper()
	encryptionKey := strings.Repeat("k", 32)
	if err := db.AutoMigrate(&model.CloudAccount{}, &model.CloudAccountRegion{}, &model.TreeCloudResource{}, &model.CloudManagedResource{}); err != nil {
		t.Fatal("region fixture schema failed")
	}
	store := dao.NewCloudAccountDAO(db, zap.NewNop())
	service := mustCloudAccountService(t, encryptionKey, zap.NewNop(), store)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	seed := func(t *testing.T, name string) (*model.CloudAccount, *model.CloudAccountRegion) {
		t.Helper()
		account := &model.CloudAccount{Name: name, Provider: 1, AccessKey: "encrypted-before", SecretKey: "encrypted-before", CreateUserName: "fixture"}
		if err := db.Omit("Regions", "CloudResources").Create(account).Error; err != nil {
			t.Fatal("account fixture failed")
		}
		region := &model.CloudAccountRegion{CloudAccountID: account.ID, Region: "cn-first", RegionName: "原地域", Status: 2, IsDefault: true, Description: "preserve"}
		if err := db.Omit("CloudAccount").Create(region).Error; err != nil {
			t.Fatal("region fixture failed")
		}
		return account, region
	}
	selected := func(codes ...string) []model.CreateCloudAccountRegionItem {
		items := make([]model.CreateCloudAccountRegionItem, len(codes))
		for i, code := range codes {
			items[i] = model.CreateCloudAccountRegionItem{Region: code, RegionName: code, IsDefault: i == 0, Description: "preserve"}
		}
		return items
	}
	t.Run("preserve IDs status and resource association", func(t *testing.T) {
		account, region := seed(t, "preserve-account")
		resource := &model.TreeCloudResource{Name: "fixture", CloudAccountID: account.ID, CloudAccountRegionID: region.ID, Region: region.Region, ResourceType: 1, Status: 1, InstanceID: "fixture-preserve"}
		if err := db.Omit("CloudAccount", "CloudAccountRegion", "TreeNodes").Create(resource).Error; err != nil {
			t.Fatal("resource fixture failed")
		}
		if err := service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Name: "preserved-name", Regions: selected("cn-first", "cn-second")}); err != nil {
			var mysqlErr *mysqlDriver.MySQLError
			if errors.As(err, &mysqlErr) {
				t.Fatalf("retained region failed with MySQL code %d", mysqlErr.Number)
			}
			t.Fatal("retaining a region collided with its unique index")
		}
		var saved model.CloudAccountRegion
		if err := db.First(&saved, region.ID).Error; err != nil || saved.Status != 2 {
			t.Fatal("unchanged region identity or state lost")
		}
		var linked model.TreeCloudResource
		if err := db.First(&linked, resource.ID).Error; err != nil || linked.CloudAccountRegionID != region.ID {
			t.Fatal("resource association changed")
		}
		if err := service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Name: "must-rollback", Regions: selected("cn-second")}); err == nil {
			t.Fatal("region with live resource reference was removed")
		}
		var latest model.CloudAccount
		if err := db.First(&latest, account.ID).Error; err != nil || latest.Name != "preserved-name" {
			t.Fatal("rejected removal partially updated account metadata")
		}
	})
	t.Run("restore historical region", func(t *testing.T) {
		account, region := seed(t, "restore-account")
		if err := service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Regions: selected("cn-second")}); err != nil {
			t.Fatal("region removal failed")
		}
		if err := service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Regions: selected("cn-first")}); err != nil {
			t.Fatal("historical region could not be restored under original unique index")
		}
		var saved model.CloudAccountRegion
		if err := db.First(&saved, region.ID).Error; err != nil || saved.Status != 2 {
			t.Fatal("restoring region changed identity/status")
		}
	})
	t.Run("all fields roll back on region write failure", func(t *testing.T) {
		account, region := seed(t, "rollback-account")
		trigger := "CREATE TRIGGER fixture_reject_region BEFORE INSERT ON cl_tree_cloud_account_region BEGIN SELECT RAISE(ABORT, 'fixture rejected'); END"
		cleanup := "DROP TRIGGER fixture_reject_region"
		if db.Dialector.Name() == "mysql" {
			trigger = "ALTER TABLE cl_tree_cloud_account_region ADD CONSTRAINT fixture_reject_region CHECK (region <> 'cn-rejected')"
			cleanup = "ALTER TABLE cl_tree_cloud_account_region DROP CHECK fixture_reject_region"
		}
		if err := db.Exec(trigger).Error; err != nil {
			t.Fatal("failure trigger setup failed")
		}
		defer db.Exec(cleanup)
		err := service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Name: "must-rollback", AccessKey: "fake-updated-key", SecretKey: "fake-updated-secret", Regions: selected("cn-first", "cn-rejected")})
		if err == nil {
			t.Fatal("injected region failure did not fail update")
		}
		var saved model.CloudAccount
		if err := db.First(&saved, account.ID).Error; err != nil || saved.Name != account.Name || saved.AccessKey != account.AccessKey || saved.SecretKey != account.SecretKey {
			t.Fatal("region failure committed metadata or credentials outside transaction")
		}
		var original model.CloudAccountRegion
		if err := db.First(&original, region.ID).Error; err != nil || original.RegionName != region.RegionName {
			t.Fatal("region failure partially committed region changes")
		}
	})
	t.Run("omitted regions preserve newer concurrent selection", func(t *testing.T) {
		account, region := seed(t, "concurrent-account")
		paused := &pausedAccountStore{CloudAccountDAO: store, read: make(chan struct{}), resume: make(chan struct{})}
		staleService := mustCloudAccountService(t, encryptionKey, zap.NewNop(), paused)
		deadline, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- staleService.UpdateCloudAccount(deadline, &model.UpdateCloudAccountReq{ID: account.ID, Name: "new-name"})
		}()
		select {
		case <-paused.read:
		case <-deadline.Done():
			t.Fatal("snapshot handshake missing")
		}
		if err := service.UpdateCloudAccount(deadline, &model.UpdateCloudAccountReq{ID: account.ID, Regions: selected("cn-first", "cn-new")}); err != nil {
			close(paused.resume)
			t.Fatal("concurrent region update failed")
		}
		close(paused.resume)
		if err := <-done; err != nil {
			t.Fatal("metadata patch failed")
		}
		var saved []model.CloudAccountRegion
		if err := db.Where("cloud_account_id = ?", account.ID).Find(&saved).Error; err != nil || len(saved) != 2 {
			t.Fatal("omitted regions overwrote a newer selection")
		}
		var common model.CloudAccountRegion
		if err := db.First(&common, region.ID).Error; err != nil {
			t.Fatal("concurrent edit changed common region ID")
		}
	})
	t.Run("concurrent complete selections serialize", func(t *testing.T) {
		if db.Dialector.Name() != "mysql" {
			t.Skip("row-level lock assertion requires production MySQL")
		}
		account, region := seed(t, "serialized-account")
		start := make(chan struct{})
		results := make(chan error, 2)
		var group sync.WaitGroup
		for _, extra := range []string{"cn-a", "cn-b"} {
			group.Add(1)
			go func(code string) {
				defer group.Done()
				<-start
				results <- service.UpdateCloudAccount(ctx, &model.UpdateCloudAccountReq{ID: account.ID, Regions: selected("cn-first", code)})
			}(extra)
		}
		close(start)
		group.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Fatal("concurrent region update violated uniqueness or transaction ownership")
			}
		}
		var saved []model.CloudAccountRegion
		if err := db.Where("cloud_account_id = ?", account.ID).Find(&saved).Error; err != nil || len(saved) != 2 {
			t.Fatal("concurrent selections merged or lost rows")
		}
		var common model.CloudAccountRegion
		if err := db.First(&common, region.ID).Error; err != nil {
			t.Fatal("concurrent selections replaced stable region")
		}
	})
}

type pausedAccountStore struct {
	dao.CloudAccountDAO
	read   chan struct{}
	resume chan struct{}
}

func (s *pausedAccountStore) GetByID(ctx context.Context, id int) (*model.CloudAccount, error) {
	value, err := s.CloudAccountDAO.GetByID(ctx, id)
	close(s.read)
	select {
	case <-s.resume:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return value, err
}
