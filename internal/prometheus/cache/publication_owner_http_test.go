package cache_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	monitorapi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	cachepkg "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	configservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ownerHTTPPools struct {
	scrapedao.ScrapePoolDAO
	pool *model.MonitorScrapePool
}

func (s ownerHTTPPools) GetMonitorScrapePoolList(_ context.Context, req *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	if req.Page > 1 {
		return nil, 1, nil
	}
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

type ownerHTTPRecordSource struct {
	alertdao.AlertManagerRecordDAO
	rule *model.MonitorRecordRule
}

func (s ownerHTTPRecordSource) GetMonitorRecordRuleByPoolId(context.Context, int) ([]*model.MonitorRecordRule, int64, error) {
	return []*model.MonitorRecordRule{s.rule}, 1, nil
}

type ownerHTTPRedis struct {
	redis.Cmdable
	values      map[string]string
	sets        map[string]map[string]bool
	pendingHook func()
}

func (s *ownerHTTPRedis) Get(_ context.Context, key string) *redis.StringCmd {
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *ownerHTTPRedis) Set(_ context.Context, key string, v any, _ time.Duration) *redis.StatusCmd {
	if strings.HasPrefix(v.(string), "pending:") && s.pendingHook != nil {
		hook := s.pendingHook
		s.pendingHook = nil
		hook()
	}
	s.values[key] = v.(string)
	return redis.NewStatusResult("OK", nil)
}
func (s *ownerHTTPRedis) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	var out []string
	for ip := range s.sets[key] {
		out = append(out, ip)
	}
	return redis.NewStringSliceResult(out, nil)
}
func (s *ownerHTTPRedis) SAdd(_ context.Context, key string, items ...any) *redis.IntCmd {
	if s.sets[key] == nil {
		s.sets[key] = map[string]bool{}
	}
	for _, v := range items {
		s.sets[key][v.(string)] = true
	}
	return redis.NewIntResult(int64(len(items)), nil)
}
func (s *ownerHTTPRedis) SRem(_ context.Context, key string, items ...any) *redis.IntCmd {
	for _, v := range items {
		delete(s.sets[key], v.(string))
	}
	return redis.NewIntResult(int64(len(items)), nil)
}
func (s *ownerHTTPRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

func TestGeneratedPublicationOwnerGuardThroughRealConfigCRUD(t *testing.T) {
	for _, mode := range []string{"delete", "update"} {
		t.Run(mode, func(t *testing.T) {
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			if err != nil {
				t.Fatal(err)
			}
			sqlDB, err := db.DB()
			if err != nil {
				t.Fatal(err)
			}
			sqlDB.SetMaxOpenConns(1)
			t.Cleanup(func() { _ = sqlDB.Close() })
			if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
				t.Fatal(err)
			}
			dao := configdao.NewMonitorConfigDAO(zap.NewNop(), db)
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 7}, Name: "ownerHTTP-fixture", SupportAlert: 2, SupportRecord: 1, PrometheusInstances: model.StringList{"127.0.0.1", "127.0.0.2"}}
			rule := &model.MonitorRecordRule{Model: model.Model{ID: 31}, PoolID: 7, Name: "fixture:up", Expr: "up", Enable: 1}
			storage := &ownerHTTPRedis{values: map[string]string{}, sets: map[string]map[string]bool{}}
			journal, history, projection := cachepkg.RecordProjectionDependenciesForTest(db, storage)
			journal = &ownerHTTPJournal{RecordProjectionJournalDAO: journal, before: func() {
				if storage.pendingHook != nil {
					hook := storage.pendingHook
					storage.pendingHook = nil
					hook()
				}
			}}
			cache := cachepkg.NewRecordRuleConfigCache(zap.NewNop(), ownerHTTPPools{pool: pool}, ownerHTTPRecordSource{rule: rule}, dao, cachepkg.NewBatchConfigManager(dao, zap.NewNop()), storage, journal, history, projection)
			if err := cache.GenerateMainConfig(context.Background()); err != nil {
				t.Fatal(err)
			}
			victimIP := "127.0.0.1"
			if mode == "delete" {
				victimIP = "127.0.0.2"
			}
			rows, err := dao.GetMonitorConfigsByInstances(context.Background(), []string{victimIP}, model.ConfigTypeRecordRule)
			if err != nil || len(rows) != 1 {
				t.Fatal("baseline canonical metadata missing")
			}
			id := rows[0].ID
			gin.SetMode(gin.TestMode)
			router := gin.New()
			monitorapi.NewMonitorConfigHandler(configservice.NewMonitorConfigService(zap.NewNop(), dao)).RegisterRouters(router)
			edited := false
			storage.pendingHook = func() {
				body, _ := json.Marshal(model.UpdateMonitorConfigReq{ID: id, Name: "operator-custom", PoolID: 99, InstanceIP: "127.0.0.9", ConfigType: model.ConfigTypeAlertRule, ConfigContent: "operator preserved body", Status: model.ConfigStatusInactive})
				request := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/monitor/configs/update/%d", id), bytes.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				var envelope struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil || response.Code != 200 || envelope.Code != 0 {
					t.Fatal("real registered config update rejected valid ownership edit")
				}
				edited = true
			}
			if mode == "delete" {
				pool.PrometheusInstances = model.StringList{"127.0.0.1"}
			} else {
				rule.Expr = "sum(up)"
			}
			publishErr := cache.GenerateMainConfig(context.Background())
			after, readErr := dao.GetMonitorConfigByID(context.Background(), id)
			preserved := readErr == nil && after.Name == "operator-custom" && after.PoolID == 99 && after.InstanceIP == "127.0.0.9" && after.ConfigType == model.ConfigTypeAlertRule && after.ConfigContent == "operator preserved body"
			t.Logf("mode=%s real CRUD accepted=%v publisher error=%v row still visible=%v ownership/body preserved=%v", mode, edited, publishErr != nil, readErr == nil, preserved)
			if !edited || !preserved {
				t.Fatal("publisher removed or overwrote metadata whose ownership changed through the real HTTP API after preflight")
			}
		})
	}
}

// Interleave the real registered CRUD request after publisher metadata preflight
// and before the actual journal's atomic commit; no receipt/lease is mocked.
type ownerHTTPJournal struct {
	configdao.RecordProjectionJournalDAO
	before func()
}

func (j *ownerHTTPJournal) CommitRecordProjectionPlan(ctx context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	if j.before != nil {
		j.before()
	}
	return j.RecordProjectionJournalDAO.CommitRecordProjectionPlan(ctx, req)
}
