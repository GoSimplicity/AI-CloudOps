package cache

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"
)

// These in-memory clients keep Redis configuration, membership, and persistence
// independent so a failure cannot appear successful through a shared fake state.
type recordingPublicationRedis struct {
	*refreshRedis
	members       map[string]bool
	failure       string
	artifactCalls int
}

func (s *recordingPublicationRedis) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	hashKey := recordPublicationKey(7)
	pending := strings.HasPrefix(value.(string), "pending:")
	if key != hashKey {
		s.artifactCalls++
	}
	if key == hashKey && (s.failure == "pending" && pending || s.failure == "hash" && !pending) {
		return redis.NewStatusResult("", errors.New("fixture publication marker failure"))
	}
	if s.failure == "set" && key == buildRedisKeyRecordRule("127.0.0.1") {
		return redis.NewStatusResult("", errors.New("fixture configuration write failure"))
	}
	return s.refreshRedis.Set(ctx, key, value, ttl)
}

func (s *recordingPublicationRedis) SMembers(context.Context, string) *redis.StringSliceCmd {
	var members []string
	for ip := range s.members {
		members = append(members, ip)
	}
	return redis.NewStringSliceResult(members, nil)
}

func (s *recordingPublicationRedis) SAdd(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.artifactCalls++
	if s.failure == "sadd" {
		return redis.NewIntResult(0, errors.New("fixture membership write failure"))
	}
	for _, value := range values {
		s.members[value.(string)] = true
	}
	return redis.NewIntResult(int64(len(values)), nil)
}

func (s *recordingPublicationRedis) Del(_ context.Context, keys ...string) *redis.IntCmd {
	s.artifactCalls++
	if s.failure == "del" {
		return redis.NewIntResult(0, errors.New("fixture stale configuration deletion failure"))
	}
	for _, key := range keys {
		delete(s.values, key)
	}
	return redis.NewIntResult(int64(len(keys)), nil)
}

func (s *recordingPublicationRedis) SRem(_ context.Context, _ string, values ...interface{}) *redis.IntCmd {
	s.artifactCalls++
	if s.failure == "srem" {
		return redis.NewIntResult(0, errors.New("fixture stale membership deletion failure"))
	}
	for _, value := range values {
		delete(s.members, value.(string))
	}
	return redis.NewIntResult(int64(len(values)), nil)
}

type recordingPublicationConfigs struct {
	refreshConfigs
	rows     map[string]*model.MonitorConfig
	failure  bool
	attempts int
	next     int
}

func (s *recordingPublicationConfigs) GetMonitorConfigsByInstances(_ context.Context, ips []string, _ int8) ([]*model.MonitorConfig, error) {
	var rows []*model.MonitorConfig
	for _, ip := range ips {
		if saved := s.rows[ip]; saved != nil {
			copy := *saved
			rows = append(rows, &copy)
		}
	}
	return rows, nil
}

func (s *recordingPublicationConfigs) save(rows []*model.MonitorConfig) error {
	s.attempts++
	if s.failure {
		return errors.New("fixture configuration persistence failure")
	}
	for _, row := range rows {
		copy := *row
		if copy.ID == 0 {
			s.next++
			copy.ID = s.next
		}
		s.rows[row.InstanceIP] = &copy
	}
	return nil
}

func (s *recordingPublicationConfigs) BatchCreateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows)
}

func (s *recordingPublicationConfigs) BatchUpdateMonitorConfigs(_ context.Context, rows []*model.MonitorConfig) error {
	return s.save(rows)
}

func (s *recordingPublicationConfigs) GetMonitorConfigList(_ context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	var rows []*model.MonitorConfig
	for _, row := range s.rows {
		if req.ConfigType == nil || row.ConfigType == *req.ConfigType {
			copy := *row
			rows = append(rows, &copy)
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
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
func (s *recordingPublicationConfigs) DeleteMonitorConfig(_ context.Context, id int) error {
	s.attempts++
	if s.failure {
		return errors.New("fixture configuration persistence failure")
	}
	for ip, row := range s.rows {
		if row.ID == id {
			delete(s.rows, ip)
		}
	}
	return nil
}

func TestRecordingPublicationRetriesFailedSnapshot(t *testing.T) {
	for _, test := range []struct {
		name      string
		failure   string
		untracked bool
		stale     bool
	}{
		{name: "configuration write with existing membership", failure: "set"},
		{name: "configuration write without membership", failure: "set", untracked: true},
		{name: "membership write", failure: "sadd", untracked: true},
		{name: "configuration persistence", failure: "database"},
		{name: "stale configuration deletion", failure: "del", stale: true},
		{name: "stale membership deletion", failure: "srem", stale: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			cache, source, storage, pool := newRecordingContractCache([]*model.MonitorRecordRule{{Name: "before:up", Expr: "up", Enable: 1}})
			client := &recordingPublicationRedis{refreshRedis: storage, members: map[string]bool{}}
			configs := &recordingPublicationConfigs{rows: map[string]*model.MonitorConfig{}}
			cache.redis = client
			cache.batchManager = NewBatchConfigManager(configs, cache.logger)
			if err := cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			configKey := buildRedisKeyRecordRule("127.0.0.1")
			hashKey := recordPublicationKey(pool.ID)
			previousConfig, previousHash := storage.values[configKey], storage.values[hashKey]
			if previousConfig == "" || previousHash == "" || configs.rows["127.0.0.1"] == nil || !client.members["127.0.0.1"] {
				t.Fatal("initial publication did not establish independent last-good states")
			}
			if test.untracked {
				delete(client.members, "127.0.0.1")
			}
			if test.stale {
				client.members["127.0.0.2"] = true
				storage.values[buildRedisKeyRecordRule("127.0.0.2")] = previousConfig
			}
			source.rows = []*model.MonitorRecordRule{{Name: "after:up", Expr: "sum(up)", Enable: 1, Labels: model.StringList{"team=after"}}}
			client.failure = test.failure
			configs.failure = test.failure == "database"
			// A failed publication may be logged or returned, but cannot be acknowledged.
			_ = cache.GenerateMainConfig(ctx)
			if !strings.HasPrefix(storage.values[hashKey], "pending:") {
				t.Fatal("failed publication retained a trusted completion marker")
			}
			if test.failure == "set" && storage.values[configKey] != previousConfig {
				t.Fatal("failed configuration write deleted or replaced current last-good")
			}
			if test.failure == "database" && configs.rows["127.0.0.1"].ConfigContent != previousConfig {
				t.Fatal("failed persistence replaced the stored last-good configuration")
			}
			client.failure, configs.failure = "", false
			if err := cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			rules := parseRecordingContract(t, storage.values[configKey])
			if len(rules) != 1 || rules[0].Record != "after:up" || rules[0].Labels["team"] != "after" {
				t.Fatalf("recovered publication did not retry the same snapshot: %+v", rules)
			}
			if storage.values[hashKey] == previousHash || !client.members["127.0.0.1"] {
				t.Fatal("recovered publication did not acknowledge configuration and membership")
			}
			if configs.rows["127.0.0.1"] == nil || configs.rows["127.0.0.1"].ConfigContent != storage.values[configKey] {
				t.Fatal("recovered publication did not persist the generated configuration")
			}
			if test.stale && (client.members["127.0.0.2"] || storage.values[buildRedisKeyRecordRule("127.0.0.2")] != "") {
				t.Fatal("recovered publication did not finish stale configuration cleanup")
			}
			writes, saves := storage.writes, configs.attempts
			if err := cache.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if storage.writes != writes || configs.attempts != saves {
				t.Fatal("acknowledged unchanged snapshot was published again")
			}
		})
	}
}
