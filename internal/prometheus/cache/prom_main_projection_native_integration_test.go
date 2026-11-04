//go:build integration

package cache

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	appconfig "github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"go.uber.org/zap"
)

type rootPromJobSnapshotHook struct {
	scrapedao.ScrapeJobDAO
	afterSnapshot func()
	once          sync.Once
}

func (d *rootPromJobSnapshotHook) GetMonitorScrapeJobsByPoolId(ctx context.Context, pool int) ([]*model.MonitorScrapeJob, error) {
	rows, err := d.ScrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool)
	if err == nil {
		d.once.Do(d.afterSnapshot)
	}
	return rows, err
}

func TestPromMainProjectionNativeOlderSourceCannotReplaceNewer(t *testing.T) {
	client := projectionRedisNative(t)
	for index, mode := range []string{"pool snapshot", "job snapshot"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			_, metadata, db := projectionStoreSQL(t)
			if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}); err != nil {
				t.Fatal(err)
			}
			logger := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
			jobs := scrapedao.NewScrapeJobDAO(db, logger)
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 9761 + index}, Name: "source-before", ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5, PrometheusInstances: model.StringList{"127.0.97." + strconv.Itoa(index+1)}}
			if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
				t.Fatal(err)
			}
			job := &model.MonitorScrapeJob{PoolID: pool.ID, Name: "native_source", Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "127.0.0.1", Port: 9101, MetricsPath: "/metrics", Scheme: "http", ScrapeInterval: 30, ScrapeTimeout: 10}
			if err := jobs.CreateMonitorScrapeJob(ctx, job); err != nil {
				t.Fatal(err)
			}
			journal := configdao.NewPromMainProjectionJournalDAO(logger, db)
			history := configdao.NewPromMainHistoryReader(logger, db)
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			create := func(sourcePools scrapedao.ScrapePoolDAO, sourceJobs scrapedao.ScrapeJobDAO) PrometheusConfigCache {
				publisher, err := NewPrometheusConfigCache(&appconfig.Config{}, logger, sourcePools, sourceJobs, metadata, NewBatchConfigManager(metadata, logger), client, journal, history, NewPromMainProjectionRunner(store))
				if err != nil {
					t.Fatal(err)
				}
				return publisher
			}
			newer := create(pools, jobs)
			var accepted *configdao.ProjectionReceipt
			var keys []string
			var baseline map[string]rootProjectionValue
			called := false
			publishNewSource := func() {
				called = true
				if mode == "pool snapshot" {
					req := &model.UpdateMonitorScrapePoolReq{ID: pool.ID, Name: "source-after", PrometheusInstances: append(model.StringList(nil), pool.PrometheusInstances...), ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5}
					if err := pools.UpdateMonitorScrapePool(ctx, req); err != nil {
						t.Fatal(err)
					}
				} else {
					changed := *job
					changed.Port = 9102
					if err := jobs.UpdateMonitorScrapeJob(ctx, &changed, nil); err != nil {
						t.Fatal(err)
					}
				}
				if err := newer.GenerateMainConfig(ctx); err != nil {
					t.Fatal("new PromMain source did not publish", err)
				}
				var err error
				accepted, err = journal.LoadLastProjectedProjection(ctx, 1, pool.ID)
				if err != nil || accepted == nil {
					t.Fatal("new source not durably projected", err)
				}
				if err := journal.WithPoolProjectionLease(ctx, *accepted, func(lease configdao.ProjectionLease) error {
					snapshot, err := lease.Snapshot()
					if err != nil {
						return err
					}
					handle, err := compileProjection(store, lease, snapshot)
					if err != nil {
						return err
					}
					projectionTrack(t, client, handle)
					keys = append([]string(nil), handle.keys...)
					baseline = rootProjectionValues(t, client, keys)
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}
			var older PrometheusConfigCache
			if mode == "pool snapshot" {
				older = create(&rootRecordPoolSnapshotHook{ScrapePoolDAO: pools, afterSnapshot: publishNewSource}, jobs)
			} else {
				older = create(pools, &rootPromJobSnapshotHook{ScrapeJobDAO: jobs, afterSnapshot: publishNewSource})
			}
			_ = older.GenerateMainConfig(ctx)
			if !called {
				t.Fatal("actual PromMain source-read barrier not reached")
			}
			for _, load := range []func(context.Context, int8, int) (*configdao.ProjectionReceipt, error){journal.LoadLastCommittedProjection, journal.LoadLastProjectedProjection} {
				current, err := load(ctx, 1, pool.ID)
				if err != nil || current == nil || current.Reservation != accepted.Reservation || current.SourceHash != accepted.SourceHash {
					t.Fatal("old PromMain source replaced a completed newer source", err)
				}
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, keys), baseline) {
				t.Fatal("old PromMain source changed newer raw, membership, SHA, ACK or fences")
			}
		})
	}
}

func TestPromMainProjectionNativeCredentialRotationUsesOperationalBytes(t *testing.T) {
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}); err != nil {
		t.Fatal(err)
	}
	logger := zap.NewNop()
	pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
	jobs := scrapedao.NewScrapeJobDAO(db, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	const ip = "127.0.98.1"
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 9861}, Name: "native-credential-rotation", ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5, PrometheusInstances: model.StringList{ip}}
	if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// These markers are synthetic test data, not credentials for any service.
	tokenA, tokenB := "synthetic-fixture-marker-A", "synthetic-fixture-marker-B"
	job := &model.MonitorScrapeJob{PoolID: pool.ID, Name: "credential_probe", Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "127.0.0.1", Port: 9101, MetricsPath: "/metrics", Scheme: "http", ScrapeInterval: 30, ScrapeTimeout: 10, BearerToken: tokenA}
	publicA := *job
	publicB := *job
	publicB.BearerToken = tokenB
	redactedA, err := json.Marshal(publicA)
	if err != nil {
		t.Fatal(err)
	}
	redactedB, err := json.Marshal(publicB)
	if err != nil || !bytes.Equal(redactedA, redactedB) {
		t.Fatal("fixture public JSON should not contain token differences")
	}
	if err := jobs.CreateMonitorScrapeJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	journal := configdao.NewPromMainProjectionJournalDAO(logger, db)
	history := configdao.NewPromMainHistoryReader(logger, db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	publisher, err := NewPrometheusConfigCache(&appconfig.Config{}, logger, pools, jobs, metadata, NewBatchConfigManager(metadata, logger), client, journal, history, NewPromMainProjectionRunner(store))
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal("initial operational credential publication failed", err)
	}
	first, err := journal.LoadLastProjectedProjection(ctx, 1, pool.ID)
	if err != nil || first == nil {
		t.Fatal(err)
	}
	if err := journal.WithPoolProjectionLease(ctx, *first, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	before := publisher.GetConfigByIP(ip)
	if !strings.Contains(before, tokenA) || strings.Contains(before, tokenB) {
		t.Fatal("first operational YAML did not retain its synthetic marker")
	}
	changed := *job
	if err := jobs.UpdateMonitorScrapeJob(ctx, &changed, &tokenB); err != nil {
		t.Fatal("actual credential-only DAO rotation failed", err)
	}
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal("rotated operational credential publication failed", err)
	}
	after := publisher.GetConfigByIP(ip)
	if !strings.Contains(after, tokenB) || strings.Contains(after, tokenA) || before == after {
		t.Fatal("credential-only change was hidden by redacted source serialization")
	}
	current, err := journal.LoadLastProjectedProjection(ctx, 1, pool.ID)
	if err != nil || current == nil || current.SourceHash == first.SourceHash || current.Reservation.Generation <= first.Reservation.Generation {
		t.Fatal("credential-only change did not alter the accepted operational version", err)
	}
	rows, err := metadata.GetMonitorConfigsByInstances(ctx, []string{ip}, 1)
	if err != nil || len(rows) != 1 || rows[0].ConfigContent != after || rows[0].ConfigHash != calculateConfigHash(after) {
		t.Fatal("operational metadata/body/hash did not agree after rotation", err)
	}
}

type rootPromJobFailure struct {
	scrapedao.ScrapeJobDAO
	fail  bool
	calls int
}

func (d *rootPromJobFailure) GetMonitorScrapeJobsByPoolId(ctx context.Context, pool int) ([]*model.MonitorScrapeJob, error) {
	d.calls++
	if d.fail {
		return nil, errors.New("deleted pool must not read old jobs")
	}
	return d.ScrapeJobDAO.GetMonitorScrapeJobsByPoolId(ctx, pool)
}

func TestPromMainProjectionNativeDeletedPoolRetiresWithoutJobRead(t *testing.T) {
	client := projectionRedisNative(t)
	_, metadata, db := projectionStoreSQL(t)
	if err := db.AutoMigrate(&model.MonitorScrapePool{}, &model.MonitorScrapeJob{}); err != nil {
		t.Fatal(err)
	}
	logger := zap.NewNop()
	pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
	jobs := scrapedao.NewScrapeJobDAO(db, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	const ip = "127.0.99.1"
	pool := &model.MonitorScrapePool{Model: model.Model{ID: 9961}, Name: "native-deleted-pool", ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5, PrometheusInstances: model.StringList{ip}}
	if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
		t.Fatal(err)
	}
	job := &model.MonitorScrapeJob{PoolID: pool.ID, Name: "native_deleted_job", Enable: 1, ServiceDiscoveryType: model.ServiceDiscoveryTypeStatic, IpAddress: "127.0.0.1", Port: 9101, MetricsPath: "/metrics", Scheme: "http", ScrapeInterval: 30, ScrapeTimeout: 10}
	if err := jobs.CreateMonitorScrapeJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	journal := configdao.NewPromMainProjectionJournalDAO(logger, db)
	history := configdao.NewPromMainHistoryReader(logger, db)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	guardedJobs := &rootPromJobFailure{ScrapeJobDAO: jobs}
	publisher, err := NewPrometheusConfigCache(&appconfig.Config{}, logger, pools, guardedJobs, metadata, NewBatchConfigManager(metadata, logger), client, journal, history, NewPromMainProjectionRunner(store))
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	first, err := journal.LoadLastProjectedProjection(ctx, 1, pool.ID)
	if err != nil || first == nil {
		t.Fatal(err)
	}
	if err := journal.WithPoolProjectionLease(ctx, *first, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	guardedJobs.fail = true
	calls := guardedJobs.calls
	if db.WithContext(ctx).Delete(pool).Error != nil {
		t.Fatal("could not prepare real SQL pool tombstone")
	}
	if err := publisher.RetireDeletedScrapePool(ctx, pool.ID); err != nil {
		t.Fatal("deleted pool retirement failed", err)
	}
	if guardedJobs.calls != calls {
		t.Fatal("deleted pool consulted unavailable jobs")
	}
	if client.Exists(ctx, buildRedisKeyPrometheusMain(ip)).Val() != 0 || client.SCard(ctx, buildRedisSetKeyPrometheusMainPoolIPs(pool.ID)).Val() != 0 {
		t.Fatal("deleted pool raw/member state remained")
	}
	final, err := journal.LoadLastProjectedProjection(ctx, 1, pool.ID)
	if err != nil || final == nil || len(final.Artifacts) != 1 || final.Artifacts[0].Action != model.ConfigProjectionRetire || final.Reservation.Generation <= first.Reservation.Generation {
		t.Fatal("deleted pool did not complete its durable retirement", err)
	}
}
