package cache_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	api "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	configservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	scrapeservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/scrape"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type retirementHTTPRedis struct {
	redis.Cmdable
	mu         sync.Mutex
	values     map[string]string
	members    map[string]map[string]bool
	failDelete bool
}

func (r *retirementHTTPRedis) Get(_ context.Context, key string) *redis.StringCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (r *retirementHTTPRedis) Set(_ context.Context, key string, v interface{}, _ time.Duration) *redis.StatusCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values[key] = v.(string)
	return redis.NewStatusResult("OK", nil)
}
func (r *retirementHTTPRedis) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []string
	for ip := range r.members[key] {
		result = append(result, ip)
	}
	return redis.NewStringSliceResult(result, nil)
}
func (r *retirementHTTPRedis) SAdd(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.members[key] == nil {
		r.members[key] = map[string]bool{}
	}
	for _, v := range values {
		r.members[key][v.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (r *retirementHTTPRedis) SRem(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range values {
		delete(r.members[key], v.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (r *retirementHTTPRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failDelete {
		return redis.NewIntResult(0, errors.New("fixture delete failure"))
	}
	for _, key := range keys {
		delete(r.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

type retirementHTTPAM struct{ cache.AlertManagerConfigCache }

func (retirementHTTPAM) GenerateMainConfig(context.Context) error { return nil }

func TestDeletedPoolHTTPAndDatabaseRetirementRecoverFromPersistentTombstone(t *testing.T) {
	for _, mode := range []string{"repeated DELETE", "periodic rebuild", "historical orphan definitions"} {
		t.Run(mode, func(t *testing.T) {
			// errgroup cancellation can discard a database/sql connection. A file
			// fixture models durable tombstones across replacement connections;
			// a single-connection :memory: database would disappear entirely.
			databasePath := filepath.Join(t.TempDir(), "retirement.sqlite")
			db, err := gorm.Open(sqlite.Open(databasePath), &gorm.Config{Logger: logger.Discard})
			if err != nil {
				t.Fatal(err)
			}
			conn, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			conn.SetMaxOpenConns(1)
			defer conn.Close()
			if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}, &model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
				t.Fatal(err)
			}
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "deleted-pool", SupportAlert: 1, SupportRecord: 1, PrometheusInstances: model.StringList{"invalid old target"}}
			if err := db.Create(pool).Error; err != nil {
				t.Fatal(err)
			}
			rows := []*model.MonitorConfig{
				{Model: model.Model{ID: 31}, PoolID: 7, ConfigType: 1, InstanceIP: "127.0.0.1", Name: "prometheus_scrape_pool_7_127.0.0.1.yaml", ConfigContent: "old", Status: 1},
				{Model: model.Model{ID: 32}, PoolID: 7, ConfigType: 3, InstanceIP: "127.0.0.1", Name: "prometheus_alert_rule_7_127.0.0.1.yaml", ConfigContent: "old", Status: 1},
				{Model: model.Model{ID: 33}, PoolID: 7, ConfigType: 4, InstanceIP: "127.0.0.1", Name: "prometheus_record_rule_7_127.0.0.1.yaml", ConfigContent: "old", Status: 1},
				{Model: model.Model{ID: 34}, PoolID: 7, ConfigType: 2, InstanceIP: "127.0.0.1", Name: "AM main", ConfigContent: "keep", Status: 1},
				{Model: model.Model{ID: 35}, PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1", Name: "AM rule", ConfigContent: "keep", Status: 1},
			}
			if err := db.Create(rows).Error; err != nil {
				t.Fatal(err)
			}
			log := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, log, nil)
			configs := configdao.NewMonitorConfigDAO(log, db)
			store := &retirementHTTPRedis{values: map[string]string{}, members: map[string]map[string]bool{}}
			for _, suffix := range []string{"main", "alert_rule", "record_rule"} {
				store.values["aiops:monitor:config:prometheus:"+suffix+":127.0.0.1"] = "old"
			}
			// Unknown legacy-only data has no trustworthy ownership and is untouched.
			store.values["unknown-legacy-key"] = "keep"
			makeManager := func() cache.MonitorCache {
				prom, err := cache.NewPrometheusConfigCache(&config.Config{}, log, pools, nil, configs, nil, store)
				if err != nil {
					t.Fatal(err)
				}
				alertJournal, alertHistory, alertProjection := cache.AlertRuleProjectionDependenciesForTest(db, store)
				rule := cache.NewAlertRuleConfigCache(log, pools, nil, configs, nil, store, alertJournal, alertHistory, alertProjection)
				journal, history, projection := cache.RecordProjectionDependenciesForTest(db, store)
				record := cache.NewRecordRuleConfigCache(log, pools, nil, configs, nil, store, journal, history, projection)
				manager, err := cache.NewMonitorCache(&config.Config{}, prom, retirementHTTPAM{}, rule, record, log)
				if err != nil {
					t.Fatal(err)
				}
				return manager
			}
			manager := makeManager()
			svc := scrapeservice.NewPrometheusPoolService(pools, manager, log, nil, scrapedao.NewScrapeJobDAO(db, log))
			router := gin.New()
			api.NewScrapePoolHandler(log, svc).RegisterRouters(router)
			api.NewMonitorConfigHandler(configservice.NewMonitorConfigService(log, configs)).RegisterRouters(router)
			request := func(method, path string) int {
				t.Helper()
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(method, path, nil))
				var body struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				return body.Code
			}
			if mode == "historical orphan definitions" {
				if err := db.Delete(&model.MonitorScrapePool{}, 7).Error; err != nil {
					t.Fatal(err)
				}
				// Historical corruption is preserved as user data, not cascade-deleted.
				if err := db.Create(&model.MonitorScrapeJob{Name: "orphan", PoolID: 7, Enable: 2}).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.MonitorAlertRule{Name: "orphan", PoolID: 7, Expr: "invalid", Enable: 2}).Error; err != nil {
					t.Fatal(err)
				}
				if err := db.Create(&model.MonitorRecordRule{Name: "orphan", PoolID: 7, Expr: "invalid", Enable: 2}).Error; err != nil {
					t.Fatal(err)
				}
				if err := manager.MonitorCacheManager(context.Background()); err != nil {
					t.Fatal(err)
				}
			} else {
				store.failDelete = true
				if code := request("DELETE", "/api/monitor/scrape_pools/delete/7"); code == 0 {
					t.Fatal("partial retirement returned success")
				}
				if _, err := pools.GetMonitorScrapePoolTombstone(context.Background(), 7); err != nil {
					t.Fatalf("failure lost durable deletion authorization: %v", err)
				}
				store.mu.Lock()
				store.failDelete = false
				store.mu.Unlock()
				if mode == "periodic rebuild" {
					manager = makeManager()
					if err := manager.MonitorCacheManager(context.Background()); err != nil {
						t.Fatal(err)
					}
				} else if code := request("DELETE", "/api/monitor/scrape_pools/delete/7"); code != 0 {
					t.Fatal("repeated DELETE did not recover")
				}
			}
			for _, id := range []int{31, 32, 33} {
				if code := request("GET", fmt.Sprintf("/api/monitor/configs/detail/%d", id)); code == 0 {
					t.Fatal("retired configuration still served by real detail HTTP")
				}
			}
			listResponse := httptest.NewRecorder()
			router.ServeHTTP(listResponse, httptest.NewRequest("GET", "/api/monitor/configs/list?page=1&size=10", nil))
			var listBody struct {
				Code int                                  `json:"code"`
				Data model.ListResp[*model.MonitorConfig] `json:"data"`
			}
			if err := json.Unmarshal(listResponse.Body.Bytes(), &listBody); err != nil || listBody.Code != 0 || len(listBody.Data.Items) != 2 {
				t.Fatal("real list HTTP retained retired rows or lost AM rows")
			}
			active, _, err := configs.GetMonitorConfigList(context.Background(), &model.GetMonitorConfigListReq{ListReq: model.ListReq{Page: 1, Size: 100}})
			if err != nil || len(active) != 2 {
				t.Fatal("ordinary list retained deleted artifacts or lost AM rows")
			}
			for _, table := range []interface{}{&model.MonitorScrapeJob{}, &model.MonitorAlertRule{}, &model.MonitorRecordRule{}} {
				var count int64
				if err := db.Model(table).Where("pool_id = ?", 7).Count(&count).Error; err != nil {
					t.Fatal(err)
				}
				if mode == "historical orphan definitions" && count != 1 {
					t.Fatal("historical user definitions were deleted")
				}
			}
			store.mu.Lock()
			defer store.mu.Unlock()
			if store.values["unknown-legacy-key"] != "keep" {
				t.Fatal("unknown Redis key deleted")
			}
			for _, suffix := range []string{"main", "alert_rule", "record_rule"} {
				if store.values["aiops:monitor:config:prometheus:"+suffix+":127.0.0.1"] != "" {
					t.Fatal("known artifact was not retired")
				}
			}
		})
	}
}

func (retirementHTTPAM) ReconcileDeletedAlertManagerPools(context.Context) error { return nil }
