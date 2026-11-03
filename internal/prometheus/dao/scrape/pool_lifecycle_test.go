package scrape_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	api "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alert "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	scrape "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	service "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func lifecycleDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}, &model.MonitorAlertManagerPool{}); err != nil {
		t.Fatal(err)
	}
	for id := 1; id <= 2; id++ {
		if err := db.Create(&model.MonitorScrapePool{Model: model.Model{ID: id}, Name: fmt.Sprintf("pool-%d", id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return db
}

type definitionCase struct {
	name   string
	create func(*gorm.DB, int, int8) error
	remove func(*gorm.DB) error
}

func definitions() []definitionCase {
	return []definitionCase{
		{"job", func(db *gorm.DB, p int, e int8) error {
			return scrape.NewScrapeJobDAO(db, zap.NewNop()).CreateMonitorScrapeJob(context.Background(), &model.MonitorScrapeJob{Name: "job", PoolID: p, Enable: e})
		}, func(db *gorm.DB) error { return db.Where("pool_id = ?", 1).Delete(&model.MonitorScrapeJob{}).Error }},
		{"alert", func(db *gorm.DB, p int, e int8) error {
			return alert.NewAlertManagerRuleDAO(db, zap.NewNop(), nil).CreateMonitorAlertRule(context.Background(), &model.MonitorAlertRule{Name: "alert", PoolID: p, Enable: e})
		}, func(db *gorm.DB) error { return db.Where("pool_id = ?", 1).Delete(&model.MonitorAlertRule{}).Error }},
		{"record", func(db *gorm.DB, p int, e int8) error {
			return alert.NewAlertManagerRecordDAO(db, zap.NewNop(), nil).CreateMonitorRecordRule(context.Background(), &model.MonitorRecordRule{Name: "record", PoolID: p, Enable: e})
		}, func(db *gorm.DB) error { return db.Where("pool_id = ?", 1).Delete(&model.MonitorRecordRule{}).Error }},
	}
}

func TestPoolDeletionProtectsAllUndeletedDefinitions(t *testing.T) {
	for _, def := range definitions() {
		for _, enabled := range []int8{1, 2} {
			t.Run(fmt.Sprintf("%s/enabled-%d", def.name, enabled), func(t *testing.T) {
				db := lifecycleDB(t)
				dao := scrape.NewScrapePoolDAO(db, zap.NewNop(), nil)
				if err := def.create(db, 1, enabled); err != nil {
					t.Fatal(err)
				}
				if err := dao.DeleteMonitorScrapePool(context.Background(), 1); err == nil {
					t.Fatal("referenced pool deleted")
				}
				if _, err := dao.GetMonitorScrapePoolById(context.Background(), 1); err != nil {
					t.Fatal(err)
				}
				if err := def.remove(db); err != nil {
					t.Fatal(err)
				}
				if err := dao.DeleteMonitorScrapePool(context.Background(), 1); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestDefinitionCreationRejectsMissingAndDeletedParents(t *testing.T) {
	for _, def := range definitions() {
		for _, id := range []int{0, -1, 99, 1} {
			t.Run(fmt.Sprintf("%s/pool-%d", def.name, id), func(t *testing.T) {
				db := lifecycleDB(t)
				if id == 1 {
					if err := db.Delete(&model.MonitorScrapePool{}, 1).Error; err != nil {
						t.Fatal(err)
					}
				}
				if err := def.create(db, id, 2); err == nil {
					t.Fatal("unavailable parent accepted")
				}
			})
		}
	}
}

func TestDefinitionMigrationChecksTargetAndPreservesSource(t *testing.T) {
	for _, def := range definitions() {
		t.Run(def.name, func(t *testing.T) {
			db := lifecycleDB(t)
			if err := def.create(db, 1, 2); err != nil {
				t.Fatal(err)
			}
			update := func(target int) error {
				switch def.name {
				case "job":
					return scrape.NewScrapeJobDAO(db, zap.NewNop()).UpdateMonitorScrapeJob(context.Background(), &model.MonitorScrapeJob{Model: model.Model{ID: 1}, Name: "changed", PoolID: target, Enable: 2}, nil)
				case "alert":
					return alert.NewAlertManagerRuleDAO(db, zap.NewNop(), nil).UpdateMonitorAlertRule(context.Background(), &model.UpdateMonitorAlertRuleReq{ID: 1, Name: "changed", PoolID: target, Enable: 2})
				default:
					return alert.NewAlertManagerRecordDAO(db, zap.NewNop(), nil).UpdateMonitorRecordRule(context.Background(), &model.MonitorRecordRule{Model: model.Model{ID: 1}, Name: "changed", PoolID: target, Enable: 2})
				}
			}
			if err := update(99); err == nil {
				t.Fatal("missing target accepted")
			}
			poolDAO := scrape.NewScrapePoolDAO(db, zap.NewNop(), nil)
			if err := poolDAO.DeleteMonitorScrapePool(context.Background(), 1); err == nil {
				t.Fatal("failed migration removed old association")
			}
			if err := update(2); err != nil {
				t.Fatal(err)
			}
			if err := poolDAO.DeleteMonitorScrapePool(context.Background(), 1); err != nil {
				t.Fatal(err)
			}
			if err := poolDAO.DeleteMonitorScrapePool(context.Background(), 2); err == nil {
				t.Fatal("new association missing")
			}
		})
	}
}

func TestPoolDeletionReferenceReadFailureRollsBack(t *testing.T) {
	db := lifecycleDB(t)
	if err := db.Callback().Query().Before("gorm:query").Register("fixture:read-failure", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.ModelType == reflect.TypeOf(model.MonitorAlertRule{}) {
			tx.AddError(errors.New("fixture failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := scrape.NewScrapePoolDAO(db, zap.NewNop(), nil).DeleteMonitorScrapePool(context.Background(), 1); err == nil {
		t.Fatal("reference read failure accepted")
	}
	var count int64
	if err := db.Model(&model.MonitorScrapePool{}).Where("id = ?", 1).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("pool lost: count=%d err=%v", count, err)
	}
}

func TestRecordZeroPoolUpdatePreservesExistingAssociation(t *testing.T) {
	db := lifecycleDB(t)
	dao := alert.NewAlertManagerRecordDAO(db, zap.NewNop(), nil)
	row := &model.MonitorRecordRule{Name: "original", PoolID: 1, Enable: 2}
	if err := dao.CreateMonitorRecordRule(context.Background(), row); err != nil {
		t.Fatal(err)
	}
	if err := dao.UpdateMonitorRecordRule(context.Background(), &model.MonitorRecordRule{Model: model.Model{ID: row.ID}, Name: "changed"}); err != nil {
		t.Fatal(err)
	}
	got, err := dao.GetMonitorRecordRuleById(context.Background(), row.ID)
	if err != nil || got.PoolID != 1 || got.Name != "changed" || got.Enable != 2 {
		t.Fatalf("omitted field contract changed: err=%v", err)
	}
}

// SQLite's one-connection fixture serializes complete transactions. This proves
// both observable outcomes without claiming to validate MySQL row locking.
func TestConcurrentCreateAndDeleteCannotLeaveDefinitionWithDeletedPool(t *testing.T) {
	for _, def := range definitions() {
		t.Run(def.name, func(t *testing.T) {
			db := lifecycleDB(t)
			start := make(chan struct{})
			var wg sync.WaitGroup
			var createErr, deleteErr error
			wg.Add(2)
			go func() { defer wg.Done(); <-start; createErr = def.create(db, 1, 2) }()
			go func() {
				defer wg.Done()
				<-start
				deleteErr = scrape.NewScrapePoolDAO(db, zap.NewNop(), nil).DeleteMonitorScrapePool(context.Background(), 1)
			}()
			close(start)
			wg.Wait()
			if (createErr == nil) == (deleteErr == nil) {
				t.Fatalf("expected exactly one operation to succeed: create=%v delete=%v", createErr, deleteErr)
			}
		})
	}
}

type lifecycleCache struct{ cache.MonitorCache }

func (lifecycleCache) MonitorCacheManager(context.Context) error { return nil }

func TestPoolDeleteHTTPRejectsDisabledDefinitions(t *testing.T) {
	for _, def := range definitions() {
		t.Run(def.name, func(t *testing.T) {
			db := lifecycleDB(t)
			if err := def.create(db, 1, 2); err != nil {
				t.Fatal(err)
			}
			poolDAO := scrape.NewScrapePoolDAO(db, zap.NewNop(), nil)
			svc := service.NewPrometheusPoolService(poolDAO, lifecycleCache{}, zap.NewNop(), nil, scrape.NewScrapeJobDAO(db, zap.NewNop()))
			handler := api.NewScrapePoolHandler(zap.NewNop(), svc)
			router := gin.New()
			router.DELETE("/pool/:id", handler.DeleteMonitorScrapePool)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest("DELETE", "/pool/1", nil))
			var body struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Code == 0 {
				t.Fatal("HTTP delete accepted referenced pool")
			}
			if _, err := poolDAO.GetMonitorScrapePoolById(context.Background(), 1); err != nil {
				t.Fatal("HTTP rejected but pool was deleted")
			}
		})
	}
}

func (lifecycleCache) RetireDeletedScrapePool(context.Context, int) error { return nil }
