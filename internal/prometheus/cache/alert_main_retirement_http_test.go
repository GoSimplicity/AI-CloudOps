package cache_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	api "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	configservice "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type amMainHTTPPools struct {
	alertdao.AlertManagerPoolDAO
	pool *model.MonitorAlertManagerPool
}

func (s amMainHTTPPools) GetMonitorAlertManagerPoolList(context.Context, *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	return []*model.MonitorAlertManagerPool{s.pool}, 1, nil
}

type amMainHTTPGroups struct{ alertdao.AlertManagerSendDAO }

func (amMainHTTPGroups) GetMonitorSendGroupByPoolID(context.Context, int) ([]*model.MonitorSendGroup, int64, error) {
	return []*model.MonitorSendGroup{{Model: model.Model{ID: 3}, Name: "test-group", Enable: 1, RepeatInterval: "1h"}}, 1, nil
}
func TestAMMainUnindexedRetirementRemovesHTTPArtifacts(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "main.sqlite")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	defer conn.Close()
	if err := db.AutoMigrate(&model.MonitorConfig{}); err != nil {
		t.Fatal(err)
	}
	pool := &model.MonitorAlertManagerPool{Model: model.Model{ID: 7}, Name: "before", AlertManagerInstances: model.StringList{"127.0.0.1"}, ResolveTimeout: "5m", GroupWait: "30s", GroupInterval: "5m", RepeatInterval: "1h", GroupBy: model.StringList{"alertname"}}
	log := zap.NewNop()
	dao := configdao.NewMonitorConfigDAO(log, db)
	r := &retirementHTTPRedis{values: map[string]string{}, members: map[string]map[string]bool{}}
	c, err := cache.NewAlertManagerConfigCache(&config.Config{Server: config.ServerConfig{Port: "8891"}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: "http://127.0.0.1:8891", AlertWebhookFileDir: "/tmp/fixture"}}, log, amMainHTTPPools{pool: pool}, amMainHTTPGroups{}, dao, nil, r)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	old, err := dao.GetMonitorConfigByInstance(ctx, "127.0.0.1", 2)
	if err != nil {
		t.Fatal(err)
	}
	clear(r.members["aiops:monitor:config:alertmanager:main:pool:7:ips"])
	pool.Name = "after"
	pool.AlertManagerInstances = model.StringList{"127.0.0.2"}
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	api.NewMonitorConfigHandler(configservice.NewMonitorConfigService(log, dao)).RegisterRouters(router)
	detail := httptest.NewRecorder()
	router.ServeHTTP(detail, httptest.NewRequest("GET", fmt.Sprintf("/api/monitor/configs/detail/%d", old.ID), nil))
	var body struct {
		Code int                 `json:"code"`
		Data model.MonitorConfig `json:"data"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest("GET", "/api/monitor/configs/list?page=1&size=10", nil))
	var listBody struct {
		Code int                                  `json:"code"`
		Data model.ListResp[*model.MonitorConfig] `json:"data"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &listBody); err != nil {
		t.Fatal(err)
	}
	oldListed := false
	for _, row := range listBody.Data.Items {
		if row.ID == old.ID {
			oldListed = true
		}
	}
	if body.Code == 0 || oldListed || listBody.Code != 0 {
		t.Fatal("real HTTP retained retired active main")
	}
	if r.values["aiops:monitor:config:alertmanager:main:127.0.0.1"] != "" {
		t.Fatal("retired main Redis artifact remains")
	}
	if r.values["aiops:monitor:config:alertmanager:main:127.0.0.2"] == "" {
		t.Fatal("current main Redis artifact missing")
	}
}
