package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type recoveryDispatchFunc func(context.Context, configdao.PoolReservation) error

func (f recoveryDispatchFunc) Recover(ctx context.Context, ref configdao.PoolReservation) error {
	return f(ctx, ref)
}

// Empty generated plans are real committed operations, not mock permissions.
// They exercise the durable queue independently of rendering and Redis.
func commitRecoveryEmpty(t *testing.T, f *recordProjectionFixture, family int8, pool int) configdao.ProjectionReceipt {
	t.Helper()
	journal := configdao.NewProjectionPlanJournalDAO(zap.NewNop(), f.db)
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	ref, err := journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: family, PoolID: pool, Nonce: nonce})
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte("recovery-empty-fixture"))
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), configdao.GeneratedProjectionPlanRequest{
		GeneratedPlanRequest: configdao.GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: []configdao.GeneratedPlanItem{}},
		Controls: []configdao.ProjectionControl{
			{ControlID: 1, Kind: configdao.ProjectionMembersFull, Members: &configdao.FullMembersControl{DesiredIPs: []string{}}},
			{ControlID: 2, Kind: configdao.ProjectionACKFull, Ack: &configdao.ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: []uint64{}, RequiredControlIDs: []uint64{1}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}
func recoveryOptions() ProjectionRecoveryOptions {
	options := DefaultProjectionRecoveryOptions()
	options.PageSize = 2
	options.MaxPerCycle = 100
	return options
}

func TestProjectionRecoveryWorkerUsesRealCurrentQueueAndHighWater(t *testing.T) {
	f := newRecordProjectionFixture(t, nil, nil, nil, nil)
	first := commitRecoveryEmpty(t, f, 4, 1)
	stale := commitRecoveryEmpty(t, f, 3, 2)
	latest := commitRecoveryEmpty(t, f, 3, 2)
	third := commitRecoveryEmpty(t, f, 1, 3)
	// A newer reservation is not a replacement for a committed last-good plan.
	var nonce [16]byte
	nonce[0] = 99
	if _, err := f.journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: 4, PoolID: 1, Nonce: nonce}); err != nil {
		t.Fatal(err)
	}
	var seen []uint64
	var appended configdao.ProjectionReceipt
	dispatcher := recoveryDispatchFunc(func(_ context.Context, ref configdao.PoolReservation) error {
		seen = append(seen, ref.OperationID)
		if appended.Reservation.OperationID == 0 {
			appended = commitRecoveryEmpty(t, f, 4, 4)
		}
		return nil
	})
	worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), dispatcher, zap.NewNop(), recoveryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := []uint64{first.Reservation.OperationID, latest.Reservation.OperationID, third.Reservation.OperationID}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("current queue = %v, want %v; superseded operation %d and round-appended operation must not run", seen, want, stale.Reservation.OperationID)
	}
	seen = nil
	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, id := range seen {
		found = found || id == appended.Reservation.OperationID
	}
	if !found {
		t.Fatal("next bounded round never reached the newly committed operation")
	}
}

func TestProjectionRecoveryWorkerBadPoolDoesNotStarveBoundedCyclesOrLeakErrors(t *testing.T) {
	f := newRecordProjectionFixture(t, nil, nil, nil, nil)
	for id := 1; id <= 5; id++ {
		commitRecoveryEmpty(t, f, 4, id)
	}
	core, logs := observer.New(zap.DebugLevel)
	var seen []int
	const canary = "private-recovery-error-canary"
	dispatcher := recoveryDispatchFunc(func(_ context.Context, ref configdao.PoolReservation) error {
		seen = append(seen, ref.PoolID)
		if ref.PoolID == 1 {
			return errors.New(canary)
		}
		return nil
	})
	options := recoveryOptions()
	options.MaxPerCycle = 2
	worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), dispatcher, zap.New(core), options)
	if err != nil {
		t.Fatal(err)
	}
	for round := 0; round < 3; round++ {
		_ = worker.RunOnce(context.Background())
	}
	for id := 1; id <= 5; id++ {
		found := false
		for _, got := range seen {
			found = found || got == id
		}
		if !found {
			t.Fatalf("bad first pool starved pool %d: calls %v", id, seen)
		}
	}
	for _, entry := range logs.All() {
		raw, err := json.Marshal(entry.ContextMap())
		if err != nil {
			t.Fatal("observer context could not be encoded", err)
		}
		if strings.Contains(entry.Message, canary) || strings.Contains(string(raw), canary) {
			t.Fatal("worker logged an untrusted underlying error")
		}
	}
}

func TestProjectionRecoveryWorkerRunIsIndependentAndCancellationClosesInFlightItem(t *testing.T) {
	f := newRecordProjectionFixture(t, nil, nil, nil, nil)
	commitRecoveryEmpty(t, f, 4, 7)
	reached := make(chan struct{})
	var once sync.Once
	dispatcher := recoveryDispatchFunc(func(ctx context.Context, _ configdao.PoolReservation) error {
		once.Do(func() { close(reached) })
		<-ctx.Done()
		return ctx.Err()
	})
	options := recoveryOptions()
	options.Interval = time.Hour
	options.ItemTimeout = time.Minute
	worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), dispatcher, zap.NewNop(), options)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- worker.Run(ctx) }()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-reached:
	case <-timer.C:
		t.Fatal("worker did not recover accepted SQL work without a normal generator or Cron trigger")
	}
	cancel()
	select {
	case err := <-finished:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-timer.C:
		t.Fatal("worker retained a canceled in-flight recovery")
	}
}

func TestProjectionRecoveryWorkerRestoresCommittedScrapeFamiliesWithoutNormalGeneration(t *testing.T) {
	for _, family := range []int8{1, 3, 4} {
		t.Run(fmt.Sprint(family), func(t *testing.T) {
			f := newRecoverySourceFixture(t, family)
			f.store.failure = "pending"
			if err := f.publisher(t).GenerateMainConfig(context.Background()); err == nil {
				t.Fatal("projection fault was not reached")
			}
			pending := f.last(t)
			if pending.Phase != model.ConfigPublicationCommitted {
				t.Fatal("fault did not leave a real committed unprojected operation")
			}
			f.store.failure = ""
			recoverer := f.recoverer(t)
			worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.base.db), recoveryDispatchFunc(recoverer.RecoverCommittedPool), zap.NewNop(), recoveryOptions())
			if err != nil {
				t.Fatal(err)
			}
			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			current := f.last(t)
			if current.Phase != model.ConfigPublicationProjected {
				t.Fatal("independent worker did not complete durable SQL operation")
			}
			if f.store.values[projectionPoolKey(family, 7, "ack")] == "" {
				t.Fatal("completed recovery has no actual matching projection ACK")
			}
			values, sets := copyAlertRedis(f.store.values), copySupportSets(f.store.sets)
			if err := worker.RunOnce(context.Background()); err != nil {
				t.Fatal(err)
			}
			if again := f.last(t); again.Reservation != current.Reservation || !reflect.DeepEqual(values, f.store.values) || !reflect.DeepEqual(sets, f.store.sets) {
				t.Fatal("completed recovery changed generation or projection on the next scan")
			}
		})
	}
}

func TestProjectionRecoveryWorkerExpectedLastCommittedGuardRejectsPostEntryReplacement(t *testing.T) {
	f := newRecoverySourceFixture(t, 4)
	f.store.failure = "pending"
	a := f.publisher(t).(*recordRuleConfigCache)
	if a.GenerateMainConfig(context.Background()) == nil {
		t.Fatal("initial projection failure not reached")
	}
	old := f.last(t)
	f.store.failure = ""
	// Reserve B for real, but stop before its business transaction. A will read
	// B's already-incremented source counter while lastCommitted still points A.
	f.changeSource(t, 2)
	b := f.publisher(t).(*recordRuleConfigCache)
	var captured *configdao.RecordProjectionPlanRequest
	b.journal = &recoveryRecordJournalHook{RecordProjectionJournalDAO: f.base.journal, ProjectionRecoveryJournalDAO: configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.base.db), commit: func(_ context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
		captured = &req
		return configdao.ProjectionReceipt{}, errors.New("deliberate pre-commit barrier")
	}}
	if b.GenerateMainConfig(context.Background()) == nil || captured == nil {
		t.Fatal("actual B reservation/typed commit barrier was not reached")
	}
	if current := f.last(t); current.Reservation != old.Reservation {
		t.Fatal("captured B unexpectedly replaced last committed A")
	}
	f.changeSource(t, 3)
	called := false
	var want configdao.ProjectionReceipt
	var values map[string]string
	var sets map[string]map[string]bool
	var rows []model.MonitorConfig
	a.journal = &recoveryRecordJournalHook{RecordProjectionJournalDAO: f.base.journal, ProjectionRecoveryJournalDAO: configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.base.db), beforeReserve: func() {
		called = true
		// All A entry/current/source checks have finished. Commit and deliver the
		// existing B reservation without incrementing the counter A just read.
		receipt, err := f.base.journal.CommitRecordProjectionPlan(context.Background(), *captured)
		if err != nil {
			t.Fatal("pre-reserved B actual SQL commit failed", err)
		}
		err = f.base.journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
			return (&recordFixtureRunner{fixture: f.base, store: f.store}).Project(context.Background(), lease)
		})
		if err != nil {
			t.Fatal("pre-reserved B actual sealed projection failed", err)
		}
		want = f.last(t)
		values, sets = copyAlertRedis(f.store.values), copySupportSets(f.store.sets)
		if err := f.base.db.Unscoped().Order("id").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
	}}
	_ = a.RecoverCommittedPool(context.Background(), old.Reservation)
	if !called {
		t.Fatal("actual recovery narrow Reserve boundary was not reached")
	}
	got := f.last(t)
	var after []model.MonitorConfig
	if err := f.base.db.Unscoped().Order("id").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if got.Reservation != want.Reservation || got.Phase != want.Phase || !reflect.DeepEqual(values, f.store.values) || !reflect.DeepEqual(sets, f.store.sets) || !reflect.DeepEqual(rows, after) {
		t.Fatal("old recovery re-signed after a completed replacement; source-counter check is insufficient")
	}
}

func TestProjectionRecoveryWorkerAMStalePeerRebuildsCurrentCohortWithoutPingPong(t *testing.T) {
	for _, family := range []int8{2, 5} {
		t.Run(fmt.Sprint(family), func(t *testing.T) {
			f := newAMPairTestFixture(t)
			ctx := context.Background()
			first := f.cache(t, "http://127.0.0.1:8891", nil, nil)
			f.store.failure = "pending write"
			if family == 2 {
				if first.GenerateMainConfig(ctx) == nil {
					t.Fatal("full projection failure not reached")
				}
			} else {
				routes, receivers := first.GenerateRouteConfigForPool(ctx, f.locator(t))
				if routes != nil || receivers != nil {
					t.Fatal("subset projection failure not reached")
				}
			}
			before, err := f.journal.LoadLastCommittedProjection(ctx, family, 7)
			if err != nil || before == nil || before.Phase != model.ConfigPublicationCommitted {
				t.Fatal("missing AM committed operation", err)
			}
			f.store.failure = ""
			if err := f.sql.db.Model(&model.MonitorSendGroup{}).Where("id=3").Update("repeat_interval", "2h").Error; err != nil {
				t.Fatal(err)
			}
			peer := f.cache(t, "http://127.0.0.1:8892", nil, nil)
			if family == 2 {
				routes, receivers := peer.GenerateRouteConfigForPool(ctx, f.locator(t))
				if len(routes) == 0 || len(receivers) == 0 {
					t.Fatal("new peer subset did not publish")
				}
			} else if err := peer.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			recovery := f.cache(t, "http://127.0.0.1:8892", nil, nil)
			worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.sql.db), recoveryDispatchFunc(recovery.RecoverCommittedPool), zap.NewNop(), recoveryOptions())
			if err != nil {
				t.Fatal(err)
			}
			if err := worker.RunOnce(ctx); err != nil {
				t.Fatal(err)
			}
			after, err := f.journal.LoadLastCommittedProjection(ctx, family, 7)
			if err != nil || after == nil || after.Phase != model.ConfigPublicationProjected {
				t.Fatal("stale AM peer receipt did not converge to current full/subset cohort", err)
			}
			stable := f.state(t)
			for round := 0; round < 3; round++ {
				if err := worker.RunOnce(ctx); err != nil {
					t.Fatal(err)
				}
				f.assertState(t, stable)
			}
		})
	}
}

func TestProjectionRecoveryWorkerCorruptSQLPoolDoesNotBlockHealthyPublisher(t *testing.T) {
	f := newRecoverySourceFixture(t, 4)
	bad := commitRecoveryEmpty(t, f.base, 4, 6)
	const canary = "private-worker-manifest-canary"
	// Persisted corruption is fixture input, not a journal-issued permission.
	if err := f.base.db.Model(&model.MonitorConfigPoolOperation{}).Where("operation_id=?", bad.Reservation.OperationID).UpdateColumn("manifest", `{"invalid":"`+canary+`"}`).Error; err != nil {
		t.Fatal(err)
	}
	f.store.failure = "pending"
	if f.publisher(t).GenerateMainConfig(context.Background()) == nil {
		t.Fatal("healthy pool did not leave its actual pending operation")
	}
	f.store.failure = ""
	core, logs := observer.New(zap.DebugLevel)
	recoverer := f.recoverer(t)
	worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.base.db), recoveryDispatchFunc(recoverer.RecoverCommittedPool), zap.New(core), recoveryOptions())
	if err != nil {
		t.Fatal(err)
	}
	if worker.RunOnce(context.Background()) == nil {
		t.Fatal("corrupt durable pool was silently reported complete")
	}
	if receipt := f.last(t); receipt.Phase != model.ConfigPublicationProjected {
		t.Fatal("corrupt earlier pool starved the healthy actual publisher")
	}
	var row model.MonitorConfigPoolOperation
	if err := f.base.db.First(&row, "operation_id=?", bad.Reservation.OperationID).Error; err != nil {
		t.Fatal(err)
	}
	if row.Phase != model.ConfigPublicationCommitted || row.ProjectedAt != nil {
		t.Fatal("corrupt operation was falsely marked projected")
	}
	for _, entry := range logs.All() {
		raw, err := json.Marshal(entry.ContextMap())
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(entry.Message, canary) || strings.Contains(string(raw), canary) {
			t.Fatal("corrupt manifest leaked into worker logs")
		}
	}
}
