package cache

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"strings"
	"sync"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// The harness shares only external storage when rebuilding a cache. No mutex or
// process-local publication flag is copied into the replacement instance.
type publicationAckHarness struct {
	poolID         int
	pending        func() bool
	valueA, valueB string
	publish        func() error
	setValue       func(string)
	setName        func(string)
	setFailure     func(string)
	legacyOnly     func() string
	legacyMarker   func() string
	restart        func()
	state          func() (redisValue, databaseValue, marker string)
	writeCounts    func() (artifactCalls, databaseCalls int)
}

func recordPublicationAckHarness(t *testing.T) publicationAckHarness {
	t.Helper()
	cache, source, storage, pool := newRecordingContractCache(t, []*model.MonitorRecordRule{{Name: "fixture:up", Expr: "up", Enable: 1}})
	client := &recordingPublicationRedis{refreshRedis: storage, members: map[string]bool{}}
	database := &recordingPublicationConfigs{rows: map[string]*model.MonitorConfig{}}
	cache.redis = client
	cache.batchManager = NewBatchConfigManager(database, cache.logger)
	attachRecordingProjection(t, cache, database)
	return publicationAckHarness{
		poolID: pool.ID, pending: func() bool { return recordFixturePending(cache) },
		valueA: "up", valueB: "sum(up)",
		publish:  func() error { return cache.GenerateMainConfig(context.Background()) },
		setValue: func(value string) { source.rows[0].Expr = value },
		setName:  func(name string) { pool.Name = name },
		legacyOnly: func() string {
			ack := storage.values[recordPublicationKey(pool.ID)]
			delete(storage.values, recordPublicationKey(pool.ID))
			storage.values[buildRedisHashKeyRecordRule(pool.Name)] = ack
			return ack
		},
		legacyMarker: func() string { return storage.values[buildRedisHashKeyRecordRule(pool.Name)] },
		setFailure: func(stage string) {
			client.failure = stage
			database.failure = stage == "database"
		},
		restart: func() {
			cache = &recordRuleConfigCache{logger: cache.logger, redis: client,
				scrapePoolDAO: cache.scrapePoolDAO, alertRecordDAO: source,
				batchManager: NewBatchConfigManager(database, cache.logger), configDAO: cache.configDAO, journal: cache.journal, history: cache.history, projection: cache.projection}
		},
		state: func() (string, string, string) {
			redisRows := parseRecordingContract(t, storage.values[buildRedisKeyRecordRule("127.0.0.1")])
			dbRows := parseRecordingContract(t, database.rows["127.0.0.1"].ConfigContent)
			if len(redisRows) != 1 || len(dbRows) != 1 {
				t.Fatal("recording fixture lost its rule")
			}
			return redisRows[0].Expr, dbRows[0].Expr, storage.values[recordPublicationKey(pool.ID)]
		},
		writeCounts: func() (int, int) { return client.artifactCalls, database.attempts },
	}
}

func promPublicationAckHarness(t *testing.T) publicationAckHarness {
	t.Helper()
	fixture := newPromPublicationFixture()
	return publicationAckHarness{
		poolID: fixture.pool.ID,
		valueA: "/metrics", valueB: "/metrics-b",
		publish:  func() error { return fixture.cache.GenerateMainConfig(context.Background()) },
		setValue: func(value string) { fixture.jobs.jobs[0].MetricsPath = value },
		setName:  func(name string) { fixture.pool.Name = name },
		legacyOnly: func() string {
			ack := fixture.storage.values[promPublicationKey(fixture.pool.ID)]
			delete(fixture.storage.values, promPublicationKey(fixture.pool.ID))
			fixture.storage.values[buildRedisHashKeyPrometheus(fixture.pool.Name)] = ack
			return ack
		},
		legacyMarker: func() string { return fixture.storage.values[buildRedisHashKeyPrometheus(fixture.pool.Name)] },
		setFailure: func(stage string) {
			fixture.storage.failure = stage
			fixture.configs.failure = stage == "database"
		},
		restart: func() {
			fixture.cache = &prometheusConfigCache{logger: fixture.cache.logger, redis: fixture.storage,
				scrapePoolDAO: fixture.cache.scrapePoolDAO, scrapeJobDAO: fixture.jobs,
				batchManager: NewBatchConfigManager(fixture.configs, fixture.cache.logger)}
		},
		state: func() (string, string, string) {
			redisConfig := parsePublishedPromConfig(t, fixture.storage.values[buildRedisKeyPrometheusMain("127.0.0.1")])
			dbConfig := parsePublishedPromConfig(t, fixture.configs.rows["127.0.0.1"].ConfigContent)
			if len(redisConfig.ScrapeConfigs) != 1 || len(dbConfig.ScrapeConfigs) != 1 {
				t.Fatal("Prometheus fixture lost its scrape job")
			}
			return redisConfig.ScrapeConfigs[0].MetricsPath, dbConfig.ScrapeConfigs[0].MetricsPath,
				fixture.storage.values[promPublicationKey(fixture.pool.ID)]
		},
		writeCounts: func() (int, int) { return fixture.storage.artifactCalls, fixture.configs.attempts },
	}
}

func TestPublicationAckRebuildRepairsCurrentSource(t *testing.T) {
	for _, generator := range []struct {
		name string
		make func(*testing.T) publicationAckHarness
	}{{"record", recordPublicationAckHarness}, {"prometheus", promPublicationAckHarness}, {"alert", alertRulePublicationAckHarness}} {
		for _, stage := range []string{"set", "sadd", "database", "hash"} {
			for _, resume := range []string{"A", "B"} {
				t.Run(generator.name+"/"+stage+"/resume-"+resume, func(t *testing.T) {
					fixture := generator.make(t)
					if err := fixture.publish(); err != nil {
						t.Fatal(err)
					}
					fixture.setValue(fixture.valueB)
					fixture.setFailure(stage)
					if err := fixture.publish(); err == nil {
						t.Fatal("incomplete B publication reported success")
					}
					_, _, pending := fixture.state()
					if fixture.pending != nil && !fixture.pending() || fixture.pending == nil && !strings.HasPrefix(pending, "pending:") {
						t.Fatal("a partially written generation retained a trusted ACK")
					}
					wanted := fixture.valueB
					if resume == "A" {
						wanted = fixture.valueA
					}
					fixture.setValue(wanted)
					fixture.setFailure("")
					fixture.restart()
					if err := fixture.publish(); err != nil {
						t.Fatal(err)
					}
					redisValue, databaseValue, marker := fixture.state()
					if redisValue != wanted || databaseValue != wanted || marker == "" || strings.HasPrefix(marker, "pending:") {
						t.Fatalf("recovery disagrees with current source: Redis=%q DB=%q", redisValue, databaseValue)
					}
					artifacts, saves := fixture.writeCounts()
					if err := fixture.publish(); err != nil {
						t.Fatal(err)
					}
					if nextArtifacts, nextSaves := fixture.writeCounts(); nextArtifacts != artifacts || nextSaves != saves {
						t.Fatal("a recovered unchanged generation was published again")
					}
				})
			}
		}
	}
}

func TestPublicationAckStablePoolIdentityAcrossRename(t *testing.T) {
	for _, generator := range []struct {
		name string
		make func(*testing.T) publicationAckHarness
	}{{"record", recordPublicationAckHarness}, {"prometheus", promPublicationAckHarness}, {"alert", alertRulePublicationAckHarness}} {
		for _, stage := range []string{"sadd", "hash"} {
			for _, resume := range []string{"A", "B"} {
				t.Run(generator.name+"/"+stage+"/resume-"+resume, func(t *testing.T) {
					fixture := generator.make(t)
					fixture.setName("name-a")
					if err := fixture.publish(); err != nil {
						t.Fatal(err)
					}
					fixture.setName("name-b")
					fixture.setValue(fixture.valueB)
					fixture.setFailure(stage)
					if err := fixture.publish(); err == nil {
						t.Fatal("renamed B publication did not fail")
					}
					_, _, marker := fixture.state()
					if fixture.pending != nil && !fixture.pending() || fixture.pending == nil && !strings.HasPrefix(marker, "pending:") {
						t.Fatal("renaming changed the publication identity")
					}
					fixture.setName("name-a")
					wanted := fixture.valueB
					if resume == "A" {
						wanted = fixture.valueA
					}
					fixture.setValue(wanted)
					fixture.setFailure("")
					fixture.restart()
					if err := fixture.publish(); err != nil {
						t.Fatal(err)
					}
					redisValue, databaseValue, marker := fixture.state()
					if redisValue != wanted || databaseValue != wanted || strings.HasPrefix(marker, "pending:") {
						t.Fatal("returning to the old name trusted a stale name-keyed ACK")
					}
				})
			}
		}
	}
}

func TestPublicationLegacyNameHashCannotImpersonateStableIdentity(t *testing.T) {
	for _, generator := range []struct {
		name string
		make func(*testing.T) publicationAckHarness
	}{{"record", recordPublicationAckHarness}, {"prometheus", promPublicationAckHarness}, {"alert", alertRulePublicationAckHarness}} {
		t.Run(generator.name, func(t *testing.T) {
			fixture := generator.make(t)
			fixture.setName(fmt.Sprintf("pool:%d", fixture.poolID))
			if err := fixture.publish(); err != nil {
				t.Fatal(err)
			}
			legacy := fixture.legacyOnly()
			if len(legacy) != 64 {
				t.Fatal("legacy fixture is not a completed SHA")
			}
			artifacts, _ := fixture.writeCounts()
			fixture.restart()
			if err := fixture.publish(); err != nil {
				t.Fatal(err)
			}
			redisValue, databaseValue, marker := fixture.state()
			if redisValue != fixture.valueA || databaseValue != fixture.valueA || marker == "" || strings.HasPrefix(marker, "pending:") {
				t.Fatal("stable identity was not established independently of the legacy key")
			}
			if nextArtifacts, _ := fixture.writeCounts(); nextArtifacts <= artifacts {
				t.Fatal("a matching legacy name SHA incorrectly skipped first stable publication")
			}
			if fixture.legacyMarker() != legacy {
				t.Fatal("legacy name key was modified or deleted")
			}
		})
	}
}

func TestPublicationPendingFailureMakesNoArtifactWrites(t *testing.T) {
	for _, generator := range []struct {
		name string
		make func(*testing.T) publicationAckHarness
	}{{"record", recordPublicationAckHarness}, {"prometheus", promPublicationAckHarness}, {"alert", alertRulePublicationAckHarness}} {
		t.Run(generator.name, func(t *testing.T) {
			fixture := generator.make(t)
			if err := fixture.publish(); err != nil {
				t.Fatal(err)
			}
			_, _, previousAck := fixture.state()
			artifacts, saves := fixture.writeCounts()
			fixture.setValue(fixture.valueB)
			fixture.setFailure("pending")
			if err := fixture.publish(); err == nil {
				t.Error("pending write failure reported success")
			}
			redisValue, databaseValue, marker := fixture.state()
			if fixture.pending != nil {
				if redisValue != fixture.valueA || databaseValue != fixture.valueB || marker != previousAck || !fixture.pending() {
					t.Error("failed projection prepare lost committed SQL/pending state or advanced raw/ACK")
				}
				if nextArtifacts, _ := fixture.writeCounts(); nextArtifacts != artifacts {
					t.Error("failed prepare changed Redis artifacts")
				}
			} else {
				if redisValue != fixture.valueA || databaseValue != fixture.valueA || marker != previousAck {
					t.Error("failed pending write changed an acknowledged generation")
				}
				if nextArtifacts, nextSaves := fixture.writeCounts(); nextArtifacts != artifacts || nextSaves != saves {
					t.Error("artifacts were touched before a durable pending marker")
				}
			}
			fixture.setFailure("")
			fixture.restart()
			if err := fixture.publish(); err != nil {
				t.Fatal(err)
			}
			redisValue, databaseValue, marker = fixture.state()
			if redisValue != fixture.valueB || databaseValue != fixture.valueB || strings.HasPrefix(marker, "pending:") {
				t.Fatal("pending failure could not recover to B")
			}
		})
	}
}

func TestPublicationAckSerializesOneCacheInstance(t *testing.T) {
	for _, generator := range []struct {
		name string
		make func(*testing.T) publicationAckHarness
	}{{"record", recordPublicationAckHarness}, {"prometheus", promPublicationAckHarness}, {"alert", alertRulePublicationAckHarness}} {
		t.Run(generator.name, func(t *testing.T) {
			fixture := generator.make(t)
			const callers = 4
			start := make(chan struct{})
			failures := make(chan error, callers)
			var group sync.WaitGroup
			for range callers {
				group.Add(1)
				go func() { defer group.Done(); <-start; failures <- fixture.publish() }()
			}
			close(start)
			group.Wait()
			close(failures)
			for err := range failures {
				if err != nil {
					t.Fatal(err)
				}
			}
			redisValue, databaseValue, marker := fixture.state()
			if redisValue != fixture.valueA || databaseValue != fixture.valueA || strings.HasPrefix(marker, "pending:") {
				t.Fatal("serialized publication did not establish A")
			}
			if _, saves := fixture.writeCounts(); saves != 1 {
				t.Fatal("unchanged concurrent calls repeated persistence")
			}
		})
	}
}

func alertRulePublicationAckHarness(t *testing.T) publicationAckHarness {
	t.Helper()
	cache, source, store, db, pool := newAlertRulePublication(t)
	// Match the existing one-artifact harness: one actual row write is one
	// first publication, so the unchanged concurrent-call assertion stays exact.
	pool.PrometheusInstances = model.StringList{"127.0.0.1"}
	return publicationAckHarness{poolID: pool.ID, pending: func() bool { return alertFixturePending(cache) }, valueA: "up", valueB: "sum(up)", publish: func() error { return cache.GenerateMainConfig(context.Background()) }, setValue: func(value string) { source.rows[0].Expr = value }, setName: func(name string) { pool.Name = name }, setFailure: func(stage string) {
		store.failure = stage
		if stage == "hash" {
			store.failure = "ack"
		}
		db.failure = ""
		if stage == "database" {
			db.failure = "database"
		}
	}, legacyOnly: func() string {
		ack := store.values[alertRulePublicationKey(pool.ID)]
		delete(store.values, alertRulePublicationKey(pool.ID))
		store.values[buildRedisHashKeyAlertRule(pool.Name)] = ack
		return ack
	}, legacyMarker: func() string { return store.values[buildRedisHashKeyAlertRule(pool.Name)] }, restart: func() {
		cache = NewAlertRuleConfigCache(zap.NewNop(), cache.scrapePoolDAO, source, cache.configDAO, cache.batchManager, store, cache.journal, cache.history, cache.projection).(*alertRuleConfigCache)
	}, state: func() (string, string, string) {
		raw := alertingContractParse(t, store.values[buildRedisKeyAlertRule("127.0.0.1")])
		var saved *model.MonitorConfig
		for _, row := range db.rows {
			if row.InstanceIP == "127.0.0.1" && row.ConfigType == 3 {
				saved = row
			}
		}
		if saved == nil {
			t.Fatal("Alert SQL view missing")
		}
		stored := alertingContractParse(t, saved.ConfigContent)
		if len(raw.Groups) != 1 || len(stored.Groups) != 1 || len(raw.Groups[0].Rules) != 1 || len(stored.Groups[0].Rules) != 1 {
			t.Fatal("Alert fixture lost its rule")
		}
		return raw.Groups[0].Rules[0].Expr, stored.Groups[0].Rules[0].Expr, store.values[alertRulePublicationKey(pool.ID)]
	}, writeCounts: func() (int, int) { return store.writes, db.writes }}
}
