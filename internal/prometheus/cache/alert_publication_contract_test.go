package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	configDAO "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	altconfig "github.com/prometheus/alertmanager/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type alertContractPools struct {
	alertDAO.AlertManagerPoolDAO
	pool *model.MonitorAlertManagerPool
}

func (s alertContractPools) GetMonitorAlertManagerPoolList(context.Context, *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	return []*model.MonitorAlertManagerPool{s.pool}, 1, nil
}

type alertContractGroups struct {
	alertDAO.AlertManagerSendDAO
	rows  []*model.MonitorSendGroup
	fail  bool
	calls int
}

func (s *alertContractGroups) GetMonitorSendGroupByPoolID(context.Context, int) ([]*model.MonitorSendGroup, int64, error) {
	s.calls++
	if s.fail {
		return nil, 0, errors.New("private-backend-error-canary")
	}
	return s.rows, int64(len(s.rows)), nil
}

type alertContractRedis struct {
	redis.Cmdable
	values     map[string]string
	members    map[string]map[string]bool
	failure    string
	onHashRead func()
	writes     int
}

func (s *alertContractRedis) Get(_ context.Context, key string) *redis.StringCmd {
	if strings.Contains(key, ":publication:alertmanager:pool:") {
		if s.onHashRead != nil {
			s.onHashRead()
		}
		if s.failure == "hash read" {
			return redis.NewStringResult("", errors.New("private-backend-error-canary"))
		}
	}
	v, ok := s.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(v, nil)
}
func (s *alertContractRedis) Set(_ context.Context, key string, value any, _ time.Duration) *redis.StatusCmd {
	s.writes++
	isHash := strings.Contains(key, ":publication:alertmanager:pool:")
	pending := strings.HasPrefix(value.(string), "pending:")
	fail := s.failure == "main cache" && strings.Contains(key, ":alertmanager:main:") || s.failure == "webhook cache" && strings.HasPrefix(key, "monitor:webhook_file:") || (s.failure == "hash write" && isHash && !pending) || (s.failure == "pending write" && isHash && pending)
	if fail {
		return redis.NewStatusResult("", errors.New("private-backend-error-canary"))
	}
	s.values[key] = value.(string)
	return redis.NewStatusResult("OK", nil)
}
func (s *alertContractRedis) SMembers(_ context.Context, key string) *redis.StringSliceCmd {
	if s.failure == "members read" {
		return redis.NewStringSliceResult(nil, errors.New("private-backend-error-canary"))
	}
	var out []string
	for value := range s.members[key] {
		out = append(out, value)
	}
	return redis.NewStringSliceResult(out, nil)
}
func (s *alertContractRedis) SAdd(_ context.Context, key string, values ...any) *redis.IntCmd {
	s.writes++
	if s.failure == "members write" {
		return redis.NewIntResult(0, errors.New("private-backend-error-canary"))
	}
	if s.members[key] == nil {
		s.members[key] = map[string]bool{}
	}
	for _, value := range values {
		s.members[key][value.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *alertContractRedis) SRem(_ context.Context, key string, values ...any) *redis.IntCmd {
	s.writes++
	if s.failure == "members delete" {
		return redis.NewIntResult(0, errors.New("private-backend-error-canary"))
	}
	for _, value := range values {
		delete(s.members[key], value.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}
func (s *alertContractRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.writes++
	for _, key := range keys {
		if s.failure == "main delete" && strings.Contains(key, ":alertmanager:main:") || s.failure == "webhook delete" && strings.HasPrefix(key, "monitor:webhook_file:") {
			return redis.NewIntResult(0, errors.New("private-backend-error-canary"))
		}
	}
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

type alertContractConfigs struct {
	configDAO.MonitorConfigDAO
	rows    map[int]*model.MonitorConfig
	next    int
	failure string
	writes  int
}

func (s *alertContractConfigs) GetMonitorConfigsByInstances(_ context.Context, ips []string, kind int8) ([]*model.MonitorConfig, error) {
	if s.failure == "DB read" {
		return nil, errors.New("private-backend-error-canary")
	}
	wanted := map[string]bool{}
	for _, ip := range ips {
		wanted[ip] = true
	}
	var out []*model.MonitorConfig
	for _, row := range s.rows {
		if row.ConfigType == kind && wanted[row.InstanceIP] {
			copy := *row
			out = append(out, &copy)
		}
	}
	return out, nil
}
func (s *alertContractConfigs) save(rows []*model.MonitorConfig) error {
	s.writes++
	for _, row := range rows {
		if s.failure == "main DB" && row.ConfigType == model.ConfigTypeAlertManager || s.failure == "webhook DB" && row.ConfigType == model.ConfigTypeWebhookFile {
			return errors.New("private-backend-error-canary")
		}
	}
	for _, row := range rows {
		copy := *row
		if copy.ID == 0 {
			s.next++
			copy.ID = s.next
		}
		s.rows[copy.ID] = &copy
	}
	return nil
}
func (s *alertContractConfigs) BatchCreateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows)
}
func (s *alertContractConfigs) BatchUpdateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows)
}
func (s *alertContractConfigs) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	if s.failure == "webhook list" {
		return nil, 0, errors.New("private-backend-error-canary")
	}
	var rows []*model.MonitorConfig
	for id := 1; id <= s.next; id++ {
		row := s.rows[id]
		if row != nil && (req.ConfigType == nil || row.ConfigType == *req.ConfigType) {
			copy := *row
			rows = append(rows, &copy)
		}
	}
	total := len(rows)
	start := (req.Page - 1) * req.Size
	if start >= total {
		return nil, int64(total), nil
	}
	end := start + req.Size
	if end > total {
		end = total
	}
	return rows[start:end], int64(total), nil
}
func (s *alertContractConfigs) DeleteMonitorConfig(_ context.Context, id int) error {
	if s.failure == "webhook DB delete" {
		return errors.New("private-backend-error-canary")
	}
	delete(s.rows, id)
	return nil
}
func alertContractCache(t *testing.T) (*alertManagerConfigCache, *alertContractGroups, *alertContractRedis, *alertContractConfigs, *model.MonitorAlertManagerPool, *observer.ObservedLogs) {
	t.Helper()
	pool := &model.MonitorAlertManagerPool{Model: model.Model{ID: 7}, Name: "private-pool-name-canary", AlertManagerInstances: model.StringList{"127.0.0.1", "127.0.0.2"}, ResolveTimeout: "5m", GroupWait: "30s", GroupInterval: "5m", RepeatInterval: "1h", GroupBy: model.StringList{"alertname"}, Receiver: "private-group-name-canary"}
	groups := &alertContractGroups{rows: []*model.MonitorSendGroup{{Model: model.Model{ID: 3}, Name: "private-group-name-canary", Enable: 1, RepeatInterval: "1h", SendResolved: 1}}}
	storage := &alertContractRedis{values: map[string]string{}, members: map[string]map[string]bool{}}
	db := &alertContractConfigs{rows: map[int]*model.MonitorConfig{}}
	core, logs := observer.New(zap.DebugLevel)
	logger := zap.New(core)
	c, e := NewAlertManagerConfigCache(&config.Config{Server: config.ServerConfig{Port: "8891"}, Prometheus: config.PrometheusConfig{AlertWebhookAddr: "http://private-webhook-url-canary.invalid", AlertWebhookFileDir: "/typed/hooks"}}, logger, alertContractPools{pool: pool}, groups, db, NewBatchConfigManager(db, logger), storage)
	if e != nil {
		t.Fatal(e)
	}
	return c.(*alertManagerConfigCache), groups, storage, db, pool, logs
}
func parseAlertContract(t *testing.T, raw string) *altconfig.Config {
	t.Helper()
	cfg, err := altconfig.Load(raw)
	if err != nil {
		t.Fatal("actual Alertmanager v0.31.1 parser rejected generated configuration")
	}
	return cfg
}
func assertAlertLogsSafe(t *testing.T, logs *observer.ObservedLogs) {
	t.Helper()
	for _, entry := range logs.All() {
		text := entry.Message + fmt.Sprint(entry.ContextMap())
		for _, secret := range []string{"private-group-name-canary", "private-pool-name-canary", "private-webhook-url-canary", "private-backend-error-canary"} {
			if strings.Contains(text, secret) {
				t.Fatal("operational log disclosed configuration/backend contents")
			}
		}
	}
}
func TestAlertPublicationFailureRetries(t *testing.T) {
	for _, stage := range []string{"hash read", "pending write", "members read", "main cache", "webhook cache", "members write", "main delete", "webhook delete", "members delete", "hash write", "DB read", "main DB", "webhook DB", "webhook list", "webhook DB delete"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			c, _, r, db, pool, logs := alertContractCache(t)
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			key := buildRedisKeyAlertManagerMain("127.0.0.1")
			hashKey := alertPublicationStateKey(pool.ID)
			before, hash := r.values[key], r.values[hashKey]
			if before == "" || hash == "" {
				t.Fatal("baseline publication incomplete")
			}
			pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
			pool.GroupWait = "45s"
			r.failure, db.failure = stage, stage
			if stage == "webhook DB" {
				c.alertWebhookAddr = "http://private-webhook-url-canary.invalid/rotated"
			}
			err := c.GenerateMainConfig(ctx)
			if err == nil {
				t.Fatal("failed publication reported success")
			}
			if r.values[hashKey] != hash && !strings.HasPrefix(r.values[hashKey], "pending:") {
				t.Fatal("failed publication advanced hash")
			}
			if r.values[key] == "" {
				t.Fatal("failed publication deleted active last-good")
			}
			parseAlertContract(t, r.values[key])
			if stage == "main cache" && r.values[key] != before {
				t.Fatal("failed SET changed last-good")
			}
			r.failure, db.failure = "", ""
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			cfg := parseAlertContract(t, r.values[key])
			if cfg.Route.GroupWait.String() != "45s" || r.values[hashKey] == hash {
				t.Fatal("recovery did not finish the same snapshot")
			}
			if r.values[buildRedisKeyAlertManagerMain("127.0.0.2")] != "" || r.values[buildRedisKeyWebhookFile("127.0.0.2", 3)] != "" {
				t.Fatal("recovery did not retire stale instance artifacts")
			}
			var persisted bool
			for _, row := range db.rows {
				if row.ConfigType == model.ConfigTypeAlertManager && row.InstanceIP == "127.0.0.1" {
					persisted = row.ConfigContent == r.values[key]
				}
			}
			if !persisted {
				t.Fatal("Redis and DB disagree after recovery")
			}
			assertAlertLogsSafe(t, logs)
		})
	}
}
func TestAlertEnabledGroupSnapshot(t *testing.T) {
	t.Run("same snapshot drives hash and render", func(t *testing.T) {
		c, g, r, _, pool, logs := alertContractCache(t)
		ctx := context.Background()
		if err := c.GenerateMainConfig(ctx); err != nil {
			t.Fatal(err)
		}
		key := buildRedisKeyAlertManagerMain("127.0.0.1")
		hashKey := alertPublicationStateKey(pool.ID)
		hash := r.values[hashKey]
		g.rows[0].RepeatInterval = "2h"
		g.rows[0].SendResolved = 2
		if err := c.GenerateMainConfig(ctx); err != nil {
			t.Fatal(err)
		}
		cfg := parseAlertContract(t, r.values[key])
		if cfg.Route.Routes[0].RepeatInterval.String() != "2h" || cfg.Receivers[0].WebhookConfigs[0].SendResolved() {
			t.Fatal("send-group-only changes were skipped")
		}
		if r.values[hashKey] == hash || g.calls != 2 {
			t.Fatal("hash/render did not use one source read per generation")
		}
		assertAlertLogsSafe(t, logs)
	})
	for _, empty := range []string{"disabled", "deleted"} {
		t.Run(empty, func(t *testing.T) {
			c, g, r, db, _, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if empty == "disabled" {
				g.rows[0].Enable = 2
			} else {
				g.rows = nil
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			cfg := parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
			if len(cfg.Route.Routes) != 0 || len(cfg.Receivers) != 1 || len(cfg.Receivers[0].WebhookConfigs) != 0 {
				t.Fatal("zero enabled groups retained active routes")
			}
			if r.values[buildRedisKeyWebhookFile("127.0.0.1", 3)] != "" {
				t.Fatal("old webhook cache retained")
			}
			for _, row := range db.rows {
				if row.ConfigType == model.ConfigTypeWebhookFile {
					t.Fatal("old webhook DB row retained")
				}
			}
		})
	}
	t.Run("DAO failure is not an empty result", func(t *testing.T) {
		c, g, r, _, pool, _ := alertContractCache(t)
		ctx := context.Background()
		if err := c.GenerateMainConfig(ctx); err != nil {
			t.Fatal(err)
		}
		key := buildRedisKeyAlertManagerMain("127.0.0.1")
		hashKey := alertPublicationStateKey(pool.ID)
		before, hash := r.values[key], r.values[hashKey]
		g.fail = true
		if c.GenerateMainConfig(ctx) == nil || r.values[key] != before || r.values[hashKey] != hash {
			t.Fatal("source failure changed or acknowledged last-good")
		}
	})
	t.Run("SDK global HTTP defaults and existing webhook contract", func(t *testing.T) {
		c, _, r, _, pool, _ := alertContractCache(t)
		pool.ResolveTimeout = ""
		pool.GroupWait = ""
		pool.GroupInterval = ""
		pool.RepeatInterval = ""
		if err := c.GenerateMainConfig(context.Background()); err != nil {
			t.Fatal(err)
		}
		cfg := parseAlertContract(t, r.values[buildRedisKeyAlertManagerMain("127.0.0.1")])
		if !cfg.Global.SMTPRequireTLS || cfg.Global.HTTPConfig == nil || !cfg.Global.HTTPConfig.FollowRedirects || !cfg.Global.HTTPConfig.EnableHTTP2 {
			t.Fatal("SDK global defaults were lost")
		}
		if cfg.Global.ResolveTimeout.String() != "5m" || cfg.Route.GroupWait.String() != "30s" || cfg.Route.GroupInterval.String() != "5m" || cfg.Route.RepeatInterval.String() != "1h" {
			t.Fatal("duration fallback changed")
		}
		hook := cfg.Receivers[0].WebhookConfigs[0]
		if hook.URLFile != "/typed/hooks/webhook_private-pool-name-canary_3.yml" || !hook.SendResolved() || hook.HTTPConfig == nil {
			t.Fatal("webhook HTTP/file/resolved contract changed")
		}
	})
}
