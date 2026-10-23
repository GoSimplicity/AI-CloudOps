package cache

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"strings"
	"testing"
	"time"
)

type refreshPools struct {
	scrapeDAO.ScrapePoolDAO
	pool *model.MonitorScrapePool
}

func (s refreshPools) GetMonitorScrapePoolList(context.Context, *model.GetMonitorScrapePoolListReq) ([]*model.MonitorScrapePool, int64, error) {
	return []*model.MonitorScrapePool{s.pool}, 1, nil
}

type refreshRedis struct {
	redis.Cmdable
	values map[string]string
	writes int
}

func (s *refreshRedis) Get(_ context.Context, key string) *redis.StringCmd {
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *refreshRedis) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	s.values[key] = value.(string)
	s.writes++
	return redis.NewStatusResult("OK", nil)
}
func (*refreshRedis) SMembers(context.Context, string) *redis.StringSliceCmd {
	return redis.NewStringSliceResult(nil, nil)
}
func (*refreshRedis) SAdd(context.Context, string, ...interface{}) *redis.IntCmd {
	return redis.NewIntResult(1, nil)
}

type refreshConfigs struct{ configDAO.MonitorConfigDAO }

func (refreshConfigs) GetMonitorConfigsByInstances(context.Context, []string, int8) ([]*model.MonitorConfig, error) {
	return nil, nil
}
func (refreshConfigs) BatchCreateMonitorConfigs(context.Context, []*model.MonitorConfig) error {
	return nil
}

func TestJobTokenRotationRefreshesCachedOperationalConfig(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	job := &model.MonitorScrapeJob{Name: "fixture", ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, Scheme: "https", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10, IpAddress: "127.0.0.1:9100", BearerToken: "fixture-token-before"}
	storage := &refreshRedis{values: map[string]string{}}
	cache := &prometheusConfigCache{logger: logger, redis: storage, scrapeJobDAO: scrapeJobsStub{jobs: []*model.MonitorScrapeJob{job}}, scrapePoolDAO: refreshPools{pool: &model.MonitorScrapePool{Model: model.Model{ID: 1}, Name: "fixture", PrometheusInstances: model.StringList{"127.0.0.1"}, ScrapeInterval: 30, ScrapeTimeout: 10}}, batchManager: NewBatchConfigManager(refreshConfigs{}, logger)}
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	writes := storage.writes
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	if storage.writes != writes {
		t.Fatal("unchanged configuration rewrote cache")
	}
	job.BearerToken = "fixture-token-after"
	if err := cache.GenerateMainConfig(context.Background()); err != nil {
		t.Fatal(err)
	}
	config := storage.values[buildRedisKeyPrometheusMain("127.0.0.1")]
	if storage.writes <= writes || !strings.Contains(config, job.BearerToken) || strings.Contains(config, "fixture-token-before") {
		t.Fatal("job-only credential rotation skipped by pool hash")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, "fixture-token-") || strings.Contains(fmt.Sprint(entry.ContextMap()), "fixture-token-") {
			t.Fatal("credential leaked into log")
		}
	}
}
