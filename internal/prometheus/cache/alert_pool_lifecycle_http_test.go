package cache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	api "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	alertservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	configservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type deletedAMHTTPProm struct{ cache.PrometheusConfigCache }

func (deletedAMHTTPProm) GenerateMainConfig(context.Context) error          { return nil }
func (deletedAMHTTPProm) ReconcileDeletedScrapePools(context.Context) error { return nil }

type deletedAMHTTPRule struct{ cache.AlertRuleConfigCache }

func (deletedAMHTTPRule) GenerateMainConfig(context.Context) error          { return nil }
func (deletedAMHTTPRule) ReconcileDeletedScrapePools(context.Context) error { return nil }

type deletedAMHTTPRecord struct{ cache.RecordRuleConfigCache }

func (deletedAMHTTPRecord) GenerateMainConfig(context.Context) error          { return nil }
func (deletedAMHTTPRecord) ReconcileDeletedScrapePools(context.Context) error { return nil }

func TestDeletedAMPoolRealHTTPAndDatabaseRecovery(t *testing.T) {
	for _, mode := range []string{"repeated DELETE", "periodic rebuild", "historical orphan group"} {
		t.Run(mode, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "am-delete.sqlite")), &gorm.Config{Logger: logger.Discard, DisableForeignKeyConstraintWhenMigrating: true})
			if err != nil {
				t.Fatal(err)
			}
			conn, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			conn.SetMaxOpenConns(1)
			defer conn.Close()
			if err := db.AutoMigrate(&model.MonitorAlertManagerPool{}, &model.MonitorSendGroup{}, &model.MonitorConfig{}, &model.MonitorScrapePool{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
				t.Fatal(err)
			}
			pool := &model.MonitorAlertManagerPool{Model: model.Model{ID: 7}, Name: "old-am", AlertManagerInstances: model.StringList{"invalid old target"}, ResolveTimeout: "invalid"}
			if err := db.Create(pool).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "same-ID Scrape pool"}).Error; err != nil {
				t.Fatal(err)
			}
			rows := []*model.MonitorConfig{{Model: model.Model{ID: 31}, PoolID: 7, ConfigType: 2, InstanceIP: "127.0.0.1", Name: "alertmanager_pool_7_127.0.0.1.yaml", ConfigContent: "old", Status: 1}, {Model: model.Model{ID: 32}, PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", Name: "webhook_old-am_3_127.0.0.1", ConfigContent: "old", Status: 1}}
			for _, kind := range []int8{1, 3, 4} {
				rows = append(rows, &model.MonitorConfig{PoolID: 7, ConfigType: kind, InstanceIP: "127.0.0.1", Name: fmt.Sprintf("Scrape type %d", kind), Status: 1})
			}
			if err := db.Create(rows).Error; err != nil {
				t.Fatal(err)
			}
			log := zap.NewNop()
			pools := alertdao.NewAlertManagerPoolDAO(db, log)
			groups := alertdao.NewAlertManagerSendDAO(db, log)
			configs := configdao.NewMonitorConfigDAO(log, db)
			store := &retirementHTTPRedis{values: map[string]string{"aiops:monitor:config:alertmanager:main:127.0.0.1": "old", "monitor:webhook_file:127.0.0.1:3": "old", "unindexed-legacy-unknown": "keep"}, members: map[string]map[string]bool{}}
			makeManager := func() cache.MonitorCache {
				am, err := cache.NewAlertManagerConfigCache(&config.Config{}, log, pools, groups, configs, nil, store)
				if err != nil {
					t.Fatal(err)
				}
				result, err := cache.NewMonitorCache(&config.Config{}, deletedAMHTTPProm{}, am, deletedAMHTTPRule{}, deletedAMHTTPRecord{}, log)
				if err != nil {
					t.Fatal(err)
				}
				return result
			}
			manager := makeManager()
			svc := alertservice.NewAlertManagerPoolService(pools, groups, log, manager)
			router := gin.New()
			api.NewAlertPoolHandler(svc).RegisterRouters(router)
			api.NewMonitorConfigHandler(configservice.NewMonitorConfigService(log, configs)).RegisterRouters(router)
			request := func(method, path string) int {
				t.Helper()
				w := httptest.NewRecorder()
				router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
				var result struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				return result.Code
			}
			if mode == "historical orphan group" {
				if err := db.Delete(&model.MonitorAlertManagerPool{}, 7).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.MonitorSendGroup{Name: "historical user definition", PoolID: 7, Enable: 2, RepeatInterval: "invalid"}).Error; err != nil {
					t.Fatal(err)
				}
				if code := request("DELETE", "/api/monitor/alert_manager_pools/delete/7"); code != 0 {
					t.Fatal("true tombstone retry was blocked by historical group")
				}
			} else {
				store.failDelete = true
				if code := request("DELETE", "/api/monitor/alert_manager_pools/delete/7"); code == 0 {
					t.Fatal("partial cleanup reported completed deletion")
				}
				if _, err := pools.GetMonitorAlertManagerPoolTombstone(context.Background(), 7); err != nil {
					t.Fatalf("durable AM tombstone missing: %v", err)
				}
				store.mu.Lock()
				store.failDelete = false
				store.mu.Unlock()
				if mode == "periodic rebuild" {
					manager = makeManager()
					if err := manager.MonitorCacheManager(context.Background()); err != nil {
						t.Fatal(err)
					}
				} else if code := request("DELETE", "/api/monitor/alert_manager_pools/delete/7"); code != 0 {
					t.Fatal("repeated DELETE did not converge")
				}
			}
			for _, id := range []int{31, 32} {
				if code := request("GET", fmt.Sprintf("/api/monitor/configs/detail/%d", id)); code == 0 {
					t.Fatal("retired AM config remains visible by detail HTTP")
				}
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/api/monitor/configs/list?page=1&size=10", nil))
			var listing struct {
				Code int                                  `json:"code"`
				Data model.ListResp[*model.MonitorConfig] `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &listing); err != nil || listing.Code != 0 || len(listing.Data.Items) != 3 {
				t.Fatal("list HTTP retained AM output or lost same-ID Scrape output")
			}
			for _, row := range listing.Data.Items {
				if row.ConfigType == 2 || row.ConfigType == 5 {
					t.Fatal("deleted AM config still listed")
				}
			}
			if err := db.First(&model.MonitorScrapePool{}, 7).Error; err != nil {
				t.Fatal("AM deletion changed same-ID Scrape parent")
			}
			if mode == "historical orphan group" {
				var n int64
				if err := db.Model(&model.MonitorSendGroup{}).Where("pool_id = ?", 7).Count(&n).Error; err != nil || n != 1 {
					t.Fatal("historical user definition was deleted")
				}
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.values["aiops:monitor:config:alertmanager:main:127.0.0.1"] != "" || store.values["monitor:webhook_file:127.0.0.1:3"] != "" {
				t.Fatal("retired AM Redis output remains")
			}
			if store.values["unindexed-legacy-unknown"] != "keep" {
				t.Fatal("unknown legacy key was guessed/deleted")
			}
		})
	}
}
