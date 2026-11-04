//go:build integration

package cache

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	appconfig "github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	scrapedao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func rootRecoveryWorker(t *testing.T, journal configdao.ProjectionRecoveryJournalDAO, families map[int8]PoolProjectionRecoverer) *ProjectionRecoveryWorker {
	t.Helper()
	options := DefaultProjectionRecoveryOptions()
	options.PageSize, options.MaxPerCycle = 1, 1
	options.Interval = time.Hour
	options.ItemTimeout, options.CycleTimeout = 10*time.Second, 20*time.Second
	// Select only the real publishers exercised by this fixture. The dispatcher
	// and worker are the production implementations, with no fake delivery path.
	worker, err := NewProjectionRecoveryWorker(journal, &projectionRecoveryDispatcher{families: families}, zap.NewNop(), options)
	if err != nil {
		t.Fatal(err)
	}
	return worker
}

func rootRecoveryDelayedStore(t *testing.T, client *redis.Client) (*ProjectionStore, *rootRecordQueue) {
	t.Helper()
	queue := rootNewRecordQueue(t, client.Options().Addr)
	options := *client.Options()
	options.Addr = queue.listener.Addr().String()
	options.Protocol, options.PoolSize, options.MinIdleConns = 2, 1, 0
	proxy := redis.NewClient(&options)
	t.Cleanup(func() { proxy.Close() })
	store, err := NewProjectionStore(proxy)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store, queue
}

func rootRecoveryInterrupted(t *testing.T, ctx context.Context, queue *rootRecordQueue, publish func() error) {
	t.Helper()
	finished := make(chan error, 1)
	go func() { finished <- publish() }()
	select {
	case <-queue.seen:
	case err := <-finished:
		if !queue.held.Load() {
			t.Fatal("publisher returned before actual queued artifact request", err)
		}
		if err == nil {
			t.Fatal("unknown Redis result falsely completed publication")
		}
		return
	case <-ctx.Done():
		t.Fatal("publisher did not send its actual artifact request")
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("unknown Redis result falsely completed publication")
		}
	case <-ctx.Done():
		t.Fatal("publisher did not release its SQL lease after EOF")
	}
}

func rootRecoveryAMInterrupted(t *testing.T, f *rootAMNativeFixture, subset bool, webhook string) *rootRecordQueue {
	t.Helper()
	store, queue := rootRecoveryDelayedStore(t, f.client)
	delayed := *f
	delayed.store = store
	pub := delayed.publisher(webhook)
	rootRecoveryInterrupted(t, f.ctx, queue, func() error {
		if !subset {
			return pub.GenerateMainConfig(f.ctx)
		}
		routes, receivers := pub.GenerateRouteConfigForPool(f.ctx, f.pool)
		if routes == nil && receivers == nil {
			return errors.New("subset delivery incomplete")
		}
		return nil
	})
	return queue
}

func TestProjectionRecoveryNativeAMPeerConverges(t *testing.T) {
	for index, oldSubset := range []bool{false, true} {
		t.Run(map[bool]string{false: "full pending then subset", true: "subset pending then full"}[oldSubset], func(t *testing.T) {
			f := rootAMNative(t, 10001+index)
			const currentURL = "http://native-recovery-current.invalid"
			pub := f.publisher(currentURL)
			renameSource := func() {
				if err := f.db.WithContext(f.ctx).Model(&model.MonitorAlertManagerPool{}).Where("id = ?", f.pool.ID).Update("name", "native-recovery-renamed").Error; err != nil {
					t.Fatal(err)
				}
				current, err := f.pools.GetAlertPoolByID(f.ctx, f.pool.ID)
				if err != nil {
					t.Fatal(err)
				}
				f.pool = current
			}
			var queue *rootRecordQueue
			var family int8 = 2
			if oldSubset {
				if err := pub.GenerateMainConfig(f.ctx); err != nil {
					t.Fatal(err)
				}
				queue = rootRecoveryAMInterrupted(t, f, true, "http://native-recovery-obsolete.invalid")
				family = 5
				renameSource()
				if err := pub.GenerateMainConfig(f.ctx); err != nil {
					t.Fatal(err)
				}
			} else {
				queue = rootRecoveryAMInterrupted(t, f, false, currentURL)
				renameSource()
				if r, s := pub.GenerateRouteConfigForPool(f.ctx, f.pool); len(r) == 0 || len(s) == 0 {
					t.Fatal("newer subset did not complete")
				}
			}
			pending, err := f.journal.LoadLastCommittedProjection(f.ctx, family, f.pool.ID)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted {
				t.Fatal("pending peer operation missing", err)
			}
			ip := f.pool.AlertManagerInstances[0]
			fullKeys := []string{buildRedisKeyAlertManagerMain(ip), buildRedisSetKeyAlertManagerMainPoolIPs(f.pool.ID), alertRetirementIntentKey(f.pool.ID), alertPublicationStateKey(f.pool.ID), projectionPoolKey(2, f.pool.ID, "ack"), projectionPoolKey(2, f.pool.ID, "fence")}
			fullBefore := rootProjectionValues(t, f.client, fullKeys)
			recoverer := pub.(PoolProjectionRecoverer)
			worker := rootRecoveryWorker(t, configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), map[int8]PoolProjectionRecoverer{2: recoverer, 5: recoverer})
			if err := worker.RunOnce(f.ctx); err != nil {
				t.Fatal("independent worker did not recover AM stale peer", err)
			}
			full, subset := f.projected(2), f.projected(5)
			if !strings.Contains(pub.GetConfigByIP(ip), "webhook_native-recovery-renamed_") {
				t.Fatal("worker combined an old main filename with a newly renamed subset")
			}
			if f.projected(family).Reservation.Generation <= pending.Reservation.Generation {
				t.Fatal("stale peer vector was not replaced by fresh guarded recovery")
			}
			if oldSubset && !reflect.DeepEqual(fullBefore, rootProjectionValues(t, f.client, fullKeys)) {
				t.Fatal("subset recovery changed full state")
			}
			for i := 0; i < 3; i++ {
				if err := worker.RunOnce(f.ctx); err != nil {
					t.Fatal(err)
				}
			}
			if f.projected(2).Reservation != full.Reservation || f.projected(5).Reservation != subset.Reservation {
				t.Fatal("recovery generated AM peer ping-pong")
			}
			keys := append(append([]string(nil), fullKeys...), buildRedisKeyWebhookFile(ip, f.group.ID), projectionPoolKey(5, f.pool.ID, "ack"), projectionPoolKey(5, f.pool.ID, "fence"))
			before := rootProjectionValues(t, f.client, keys)
			queue.unblock()
			select {
			case result := <-queue.result:
				if result.err != nil || result.response != "STALE" {
					t.Fatal("old AM artifact was not fenced after recovery", result.err)
				}
			case <-f.ctx.Done():
				t.Fatal("old artifact was not evaluated")
			}
			if !reflect.DeepEqual(before, rootProjectionValues(t, f.client, keys)) {
				t.Fatal("old AM packet changed recovered state")
			}
		})
	}
}

type rootRecoveryNotify struct {
	next ProjectionRecoveryDispatcher
	done chan error
}

func (d *rootRecoveryNotify) Recover(ctx context.Context, ref configdao.PoolReservation) error {
	err := d.next.Recover(ctx, ref)
	select {
	case d.done <- err:
	default:
	}
	return err
}

func TestProjectionRecoveryNativeScrapeFamiliesRestart(t *testing.T) {
	for _, kind := range []int8{1, 3, 4} {
		t.Run(map[int8]string{1: "prom", 3: "alert", 4: "record"}[kind], func(t *testing.T) {
			client := projectionRedisNative(t)
			_, metadata, db := projectionStoreSQL(t)
			if err := db.AutoMigrate(&model.MonitorScrapePool{}); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			logger := zap.NewNop()
			pools := scrapedao.NewScrapePoolDAO(db, logger, nil)
			pool := &model.MonitorScrapePool{Model: model.Model{ID: 10100 + int(kind)}, Name: "recovery-native", PrometheusInstances: model.StringList{"127.0.101.1"}, SupportAlert: 1, SupportRecord: 1, ScrapeInterval: 30, ScrapeTimeout: 10, RemoteTimeoutSeconds: 5}
			if err := pools.CreateMonitorScrapePool(ctx, pool); err != nil {
				t.Fatal(err)
			}
			journal := rootNativeScrapeRuleJournal(kind, db)
			makePublisher := func(store *ProjectionStore) rootNativeScrapeRulePublisher {
				track := func(next AMProjectionRunner) *rootAMTrackingRunner {
					return &rootAMTrackingRunner{t: t, client: client, store: store, next: next}
				}
				switch kind {
				case 1:
					pub, err := NewPrometheusConfigCache(&appconfig.Config{}, logger, pools, rootPromNativeJobs{pool: pool.ID, version: "recovery"}, metadata, NewBatchConfigManager(metadata, logger), client, journal, configdao.NewPromMainHistoryReader(logger, db), track(NewPromMainProjectionRunner(store)))
					if err != nil {
						t.Fatal(err)
					}
					return pub
				case 3:
					return NewAlertRuleConfigCache(logger, pools, rootAlertNativeRules{pool: pool.ID, expr: "sum(up)"}, metadata, NewBatchConfigManager(metadata, logger), client, journal, configdao.NewAlertRuleHistoryReader(logger, db), track(NewAlertRuleProjectionRunner(store)))
				default:
					return NewRecordRuleConfigCache(logger, pools, rootRecordNativeRules{pool: pool.ID, expr: "sum(up)"}, metadata, NewBatchConfigManager(metadata, logger), client, journal, configdao.NewGeneratedConfigHistoryReader(logger, db), track(NewRecordProjectionRunner(store)))
				}
			}
			delayed, queue := rootRecoveryDelayedStore(t, client)
			old := makePublisher(delayed)
			rootRecoveryInterrupted(t, ctx, queue, func() error { return old.GenerateMainConfig(ctx) })
			pending, err := journal.LoadLastCommittedProjection(ctx, kind, pool.ID)
			if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted {
				t.Fatal("durable pending operation missing", err)
			}
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { store.Close() })
			rebuilt := makePublisher(store)
			worker := rootRecoveryWorker(t, configdao.NewProjectionRecoveryJournalDAO(logger, db), map[int8]PoolProjectionRecoverer{kind: rebuilt.(PoolProjectionRecoverer)})
			// Only the new worker runs. No manager refresh or Cron is started.
			notify := &rootRecoveryNotify{next: worker.dispatcher, done: make(chan error, 1)}
			worker.dispatcher = notify
			runCtx, stop := context.WithCancel(ctx)
			done := make(chan error, 1)
			exited := make(chan struct{})
			go func() { defer close(exited); done <- worker.Run(runCtx) }()
			t.Cleanup(func() {
				stop()
				select {
				case <-exited:
				case <-time.After(5 * time.Second):
					t.Error("recovery worker did not exit before fixture cleanup")
				}
			})
			select {
			case err := <-notify.done:
				if err != nil {
					stop()
					t.Fatal("worker restart did not recover", err)
				}
			case <-ctx.Done():
				stop()
				t.Fatal("worker did not scan immediately")
			}
			stop()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("worker did not stop after cancellation")
			}
			got, err := journal.LoadLastProjectedProjection(ctx, kind, pool.ID)
			if err != nil || got == nil || got.Reservation != pending.Reservation {
				t.Fatal("valid pending receipt was not idempotently recovered", err)
			}
			if raw := rebuilt.GetConfigByIP(pool.PrometheusInstances[0]); raw == "" {
				t.Fatal("recovery did not serve native raw bytes")
			}
			keys := []string{rootNativeScrapeRuleRaw(kind, pool.PrometheusInstances[0]), projectionNamespaceKey(kind, pool.PrometheusInstances[0]), rootNativeScrapeRuleLegacy(kind, pool.ID), rootNativeScrapeRuleMembers(kind, pool.ID), projectionPoolKey(kind, pool.ID, "ack"), projectionPoolKey(kind, pool.ID, "fence")}
			before := rootProjectionValues(t, client, keys)
			queue.unblock()
			select {
			case result := <-queue.result:
				if result.err != nil || result.response != "OK" && result.response != "STALE" {
					t.Fatal("old packet was not safely reconciled", result.err)
				}
			case <-ctx.Done():
				t.Fatal("old packet was not evaluated")
			}
			if !reflect.DeepEqual(before, rootProjectionValues(t, client, keys)) {
				t.Fatal("old packet changed recovered raw, members, ACK or fences")
			}
		})
	}
}

func TestProjectionRecoveryNativePoisonDoesNotBlockLaterPool(t *testing.T) {
	f := rootAMNative(t, 10201)
	const webhook = "http://native-recovery-poison.invalid"
	_ = rootRecoveryAMInterrupted(t, f, false, webhook)
	bad, err := f.journal.LoadLastCommittedProjection(f.ctx, 2, f.pool.ID)
	if err != nil || bad == nil {
		t.Fatal("poison fixture has no committed operation", err)
	}
	if err := f.db.WithContext(f.ctx).Model(&model.MonitorConfigPoolOperation{}).Where("operation_id = ?", bad.Reservation.OperationID).Update("manifest", "not-json").Error; err != nil {
		t.Fatal(err)
	}
	good := *f
	pool := *f.pool
	pool.Model = model.Model{ID: 10202}
	pool.Name = "native-recovery-good"
	pool.AlertManagerInstances = model.StringList{"127.0.102.2"}
	group := *f.group
	group.Model = model.Model{ID: 10202}
	group.PoolID = pool.ID
	good.pool, good.group = &pool, &group
	if err := f.pools.CreateMonitorAlertManagerPool(f.ctx, &pool); err != nil {
		t.Fatal(err)
	}
	if err := f.groups.CreateMonitorSendGroup(f.ctx, &group); err != nil {
		t.Fatal(err)
	}
	queue := rootRecoveryAMInterrupted(t, &good, false, webhook)
	pending, err := f.journal.LoadLastCommittedProjection(f.ctx, 2, pool.ID)
	if err != nil || pending == nil || pending.Phase != model.ConfigPublicationCommitted {
		t.Fatal("healthy later pending operation missing", err)
	}
	pub := good.publisher(webhook).(PoolProjectionRecoverer)
	worker := rootRecoveryWorker(t, configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), map[int8]PoolProjectionRecoverer{2: pub, 5: pub})
	if err := worker.RunOnce(f.ctx); err == nil {
		t.Fatal("malformed first operation was silently accepted")
	}
	before, err := f.journal.LoadLastProjectedProjection(f.ctx, 2, pool.ID)
	if err != nil || before != nil {
		t.Fatal("one-item first cycle unexpectedly processed later pool", err)
	}
	if err := worker.RunOnce(f.ctx); err != nil {
		t.Fatal("poison candidate blocked the next page", err)
	}
	if got := good.projected(2); got.Reservation != pending.Reservation {
		t.Fatal("healthy pool did not recover its original valid plan")
	}
	if got, err := f.journal.LoadLastProjected(f.ctx, 2, f.pool.ID); err != nil || got != nil {
		t.Fatal("malformed operation was falsely marked projected", err)
	}
	ip := pool.AlertManagerInstances[0]
	keys := []string{buildRedisKeyAlertManagerMain(ip), buildRedisKeyWebhookFile(ip, group.ID), buildRedisSetKeyAlertManagerMainPoolIPs(pool.ID), alertPublicationStateKey(pool.ID), projectionPoolKey(2, pool.ID, "ack"), projectionPoolKey(2, pool.ID, "fence")}
	stable := rootProjectionValues(t, f.client, keys)
	queue.unblock()
	select {
	case result := <-queue.result:
		if result.err != nil || result.response != "OK" && result.response != "STALE" {
			t.Fatal("old healthy packet was not safely reconciled", result.err)
		}
	case <-f.ctx.Done():
		t.Fatal("old healthy packet was not evaluated")
	}
	if !reflect.DeepEqual(stable, rootProjectionValues(t, f.client, keys)) {
		t.Fatal("old packet changed healthy recovered pool")
	}
}
