package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	scrapeDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	pc "github.com/prometheus/prometheus/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type promPublicationJobs struct {
	scrapeDAO.ScrapeJobDAO
	jobs  []*model.MonitorScrapeJob
	err   error
	calls int
}

func (s *promPublicationJobs) GetMonitorScrapeJobsByPoolId(context.Context, int) ([]*model.MonitorScrapeJob, error) {
	s.calls++
	return s.jobs, s.err
}

type promPublicationRedis struct{ *recordingPublicationRedis }

func (s *promPublicationRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	if (s.failure == "set" && key == buildRedisKeyPrometheusMain("127.0.0.1")) ||
		(s.failure == "hash" && key == buildRedisHashKeyPrometheus("fixture")) {
		return redis.NewStatusResult("", errors.New("fixture main configuration write failure"))
	}
	return s.refreshRedis.Set(ctx, key, value, ttl)
}

func (s *promPublicationRedis) Get(ctx context.Context, key string) *redis.StringCmd {
	if s.failure == "get" {
		return redis.NewStringResult("", errors.New("fixture hash read failure"))
	}
	return s.refreshRedis.Get(ctx, key)
}

func (s *promPublicationRedis) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	if s.failure == "smembers" {
		return redis.NewStringSliceResult(nil, errors.New("fixture membership read failure"))
	}
	return s.recordingPublicationRedis.SMembers(ctx, key)
}

type promPublicationFixture struct {
	cache   *prometheusConfigCache
	jobs    *promPublicationJobs
	storage *promPublicationRedis
	configs *recordingPublicationConfigs
	pool    *model.MonitorScrapePool
}

func newPromPublicationFixture() promPublicationFixture {
	jobs := &promPublicationJobs{jobs: []*model.MonitorScrapeJob{{
		Name: "fixture", ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic,
		Scheme: "https", MetricsPath: "/metrics", ScrapeInterval: 30, ScrapeTimeout: 10,
		IpAddress: "127.0.0.1:9100", BearerToken: "fixture-token-before",
	}}}
	storage := &promPublicationRedis{&recordingPublicationRedis{
		refreshRedis: &refreshRedis{values: map[string]string{}}, members: map[string]bool{},
	}}
	configs := &recordingPublicationConfigs{rows: map[string]*model.MonitorConfig{}}
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 1}, Name: "fixture",
		PrometheusInstances: model.StringList{"127.0.0.1"}, ScrapeInterval: 30, ScrapeTimeout: 10}
	logger := zap.NewNop()
	cache := &prometheusConfigCache{logger: logger, redis: storage, scrapeJobDAO: jobs,
		scrapePoolDAO: refreshPools{pool: pool}, batchManager: NewBatchConfigManager(configs, logger)}
	return promPublicationFixture{cache: cache, jobs: jobs, storage: storage, configs: configs, pool: pool}
}

func parsePublishedPromConfig(t *testing.T, content string) *pc.Config {
	t.Helper()
	if content == "" {
		t.Fatal("expected a published Prometheus configuration")
	}
	config, err := pc.Load(content, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal("Prometheus rejected the published configuration")
	}
	return config
}

func TestPromPublicationRetriesFailedSnapshot(t *testing.T) {
	for _, test := range []struct {
		name, failure string
		untracked     bool
		stale         bool
		multiple      bool
	}{
		{name: "configuration write with membership", failure: "set"},
		{name: "configuration write without membership", failure: "set", untracked: true},
		{name: "partial current instance writes", failure: "set", multiple: true},
		{name: "hash read", failure: "get"},
		{name: "membership read", failure: "smembers", stale: true},
		{name: "membership write", failure: "sadd", untracked: true},
		{name: "persistence", failure: "database"},
		{name: "stale configuration deletion", failure: "del", stale: true},
		{name: "stale membership deletion", failure: "srem", stale: true},
		{name: "completion marker", failure: "hash"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newPromPublicationFixture()
			if test.multiple {
				fixture.pool.PrometheusInstances = append(fixture.pool.PrometheusInstances, "127.0.0.2")
			}
			ctx := context.Background()
			if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			configKey, hashKey := buildRedisKeyPrometheusMain("127.0.0.1"), buildRedisHashKeyPrometheus(fixture.pool.Name)
			previousConfig, previousHash := fixture.storage.values[configKey], fixture.storage.values[hashKey]
			parsePublishedPromConfig(t, previousConfig)
			if previousHash == "" || fixture.configs.rows["127.0.0.1"] == nil || !fixture.storage.members["127.0.0.1"] {
				t.Fatal("initial publication did not establish last-good states")
			}
			if test.untracked {
				delete(fixture.storage.members, "127.0.0.1")
			}
			if test.stale {
				fixture.storage.members["127.0.0.2"] = true
				fixture.storage.values[buildRedisKeyPrometheusMain("127.0.0.2")] = previousConfig
			}
			fixture.jobs.jobs[0].BearerToken = "fixture-token-after"
			fixture.storage.failure = test.failure
			fixture.configs.failure = test.failure == "database"
			if err := fixture.cache.GenerateMainConfig(ctx); err == nil {
				t.Error("failed publication must report incomplete delivery")
			}
			if fixture.storage.values[hashKey] != previousHash {
				t.Error("failed publication advanced its completion marker")
			}
			if test.failure == "set" && fixture.storage.values[configKey] != previousConfig {
				t.Error("failed current-IP write destroyed last-good configuration")
			}
			fixture.storage.failure, fixture.configs.failure = "", false
			if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			published := fixture.storage.values[configKey]
			parsePublishedPromConfig(t, published)
			if !strings.Contains(published, "fixture-token-after") || strings.Contains(published, "fixture-token-before") {
				t.Error("recovery skipped the pending credential rotation")
			}
			if fixture.storage.values[hashKey] == previousHash || !fixture.storage.members["127.0.0.1"] {
				t.Error("recovery did not publish membership and acknowledge the snapshot")
			}
			if saved := fixture.configs.rows["127.0.0.1"]; saved == nil || saved.ConfigContent != published {
				t.Error("recovery did not persist the published configuration")
			}
			for _, ip := range fixture.pool.PrometheusInstances {
				content := fixture.storage.values[buildRedisKeyPrometheusMain(ip)]
				parsePublishedPromConfig(t, content)
				if saved := fixture.configs.rows[ip]; saved == nil || saved.ConfigContent != content || !fixture.storage.members[ip] || !strings.Contains(content, "fixture-token-after") {
					t.Error("recovery did not complete every current instance")
				}
			}
			if test.stale && (fixture.storage.members["127.0.0.2"] || fixture.storage.values[buildRedisKeyPrometheusMain("127.0.0.2")] != "") {
				t.Error("recovery did not finish stale configuration cleanup")
			}
			writes, saves := fixture.storage.writes, fixture.configs.attempts
			if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if fixture.storage.writes != writes || fixture.configs.attempts != saves {
				t.Error("unchanged acknowledged snapshot was published again")
			}
		})
	}
}

func TestPromPublicationEmptyActiveJobsStopsOldScraping(t *testing.T) {
	fixture := newPromPublicationFixture()
	ctx := context.Background()
	if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	hashKey := buildRedisHashKeyPrometheus(fixture.pool.Name)
	oldHash := fixture.storage.values[hashKey]
	fixture.jobs.jobs = nil
	if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	content := fixture.storage.values[buildRedisKeyPrometheusMain("127.0.0.1")]
	if config := parsePublishedPromConfig(t, content); len(config.ScrapeConfigs) != 0 {
		t.Error("empty active list retained an old scrape target")
	}
	if fixture.storage.values[hashKey] == oldHash || fixture.configs.rows["127.0.0.1"].ConfigContent != content {
		t.Error("empty active list was not published and persisted")
	}
}

func TestPromPublicationInvalidSnapshotRetainsLastGood(t *testing.T) {
	for _, kind := range []string{"DAO error", "invalid auth", "invalid timing", "invalid relabel", "nil job"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newPromPublicationFixture()
			ctx := context.Background()
			if err := fixture.cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			configKey, hashKey := buildRedisKeyPrometheusMain("127.0.0.1"), buildRedisHashKeyPrometheus(fixture.pool.Name)
			config, hash, writes := fixture.storage.values[configKey], fixture.storage.values[hashKey], fixture.storage.writes
			switch kind {
			case "DAO error":
				fixture.jobs.err = errors.New("fixture-private-read-failure")
			case "invalid auth":
				fixture.jobs.jobs[0].BearerTokenFile = "/fixture/token"
			case "invalid timing":
				fixture.jobs.jobs[0].ScrapeInterval = 5
			case "invalid relabel":
				fixture.jobs.jobs[0].RelabelConfigsYamlString = "["
			case "nil job":
				fixture.jobs.jobs = []*model.MonitorScrapeJob{nil}
			}
			if err := fixture.cache.GenerateMainConfig(ctx); err == nil {
				t.Error("invalid snapshot did not report generation failure")
			}
			if fixture.storage.values[configKey] != config || fixture.storage.values[hashKey] != hash || fixture.storage.writes != writes {
				t.Error("invalid snapshot replaced last-good data or its hash")
			}
		})
	}
}

func TestPromPublicationUsesOneJobSnapshotIncludingRotation(t *testing.T) {
	fixture := newPromPublicationFixture()
	for iteration := 0; iteration < 3; iteration++ {
		if iteration == 1 {
			fixture.jobs.jobs[0].BearerToken = "fixture-token-after"
		}
		previous := fixture.jobs.calls
		if err := fixture.cache.GenerateMainConfig(context.Background()); err != nil {
			t.Fatal(err)
		}
		if fixture.jobs.calls-previous != 1 {
			t.Fatal("refresh did not use exactly one job DAO snapshot")
		}
	}
	if !strings.Contains(fixture.storage.values[buildRedisKeyPrometheusMain("127.0.0.1")], "fixture-token-after") {
		t.Fatal("job-only token rotation was lost")
	}
}

func TestPromPublicationErrorsDoNotExposePrivateConfig(t *testing.T) {
	const privateValue = "fixture-private-configuration-canary"
	for _, kind := range []string{"DAO", "relabel", "remote write", "remote read"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newPromPublicationFixture()
			core, logs := observer.New(zap.DebugLevel)
			fixture.cache.logger = zap.New(core)
			switch kind {
			case "DAO":
				fixture.jobs.err = errors.New(privateValue)
			case "relabel":
				fixture.jobs.jobs[0].RelabelConfigsYamlString = privateValue
			case "remote write":
				fixture.pool.RemoteWriteUrl = "http://" + privateValue + "@["
			case "remote read":
				fixture.pool.RemoteReadUrl = "http://" + privateValue + "@["
			}
			err := fixture.cache.GenerateMainConfig(context.Background())
			if err == nil {
				t.Fatal("invalid private configuration was not rejected")
			}
			if strings.Contains(err.Error(), privateValue) {
				t.Fatal("returned error exposed private configuration")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message, privateValue) || strings.Contains(fmt.Sprint(entry.ContextMap()), privateValue) {
					t.Fatal("log exposed private configuration")
				}
			}
		})
	}
}
