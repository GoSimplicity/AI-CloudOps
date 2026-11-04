package cache

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Legacy maps remain fixture input/assertion views only. Namespace ownership,
// atomic business writes, journal receipts and lease authority come from SQLite.
type recordProjectionFixture struct {
	configdao.MonitorConfigDAO
	db              *gorm.DB
	journal         configdao.RecordProjectionJournalDAO
	history         configdao.GeneratedConfigHistoryReader
	seeds           func() []*model.MonitorConfig
	observe         func([]*model.MonitorConfig)
	failure         func() string
	write           func()
	seeding         bool
	observationKind int8
	commits         atomic.Uint64
}

func newRecordProjectionFixture(t *testing.T, seeds func() []*model.MonitorConfig, observe func([]*model.MonitorConfig), failure func() string, write func()) *recordProjectionFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
		t.Fatal(err)
	}
	log := zap.NewNop()
	f := &recordProjectionFixture{MonitorConfigDAO: configdao.NewMonitorConfigDAO(log, db), db: db, journal: configdao.NewRecordProjectionJournalDAO(log, db), history: configdao.NewGeneratedConfigHistoryReader(log, db), seeds: seeds, observe: observe, failure: failure, write: write}
	inject := func(mode string) func(*gorm.DB) {
		return func(tx *gorm.DB) {
			if tx.Statement.SkipHooks || tx.Statement.Table != "cl_monitor_configs" {
				return
			}
			if f.write != nil {
				f.write()
			}
			if f.failure != nil {
				fault := f.failure()
				if fault == mode || fault == "database" {
					tx.AddError(errors.New("fixture metadata write failure"))
				}
			}
		}
	}
	if db.Callback().Create().Before("gorm:create").Register("record_fixture:create", inject("db create")) != nil || db.Callback().Update().Before("gorm:update").Register("record_fixture:update", inject("db update")) != nil || db.Callback().Delete().Before("gorm:delete").Register("record_fixture:delete", inject("db delete")) != nil {
		t.Fatal("SQL fault callbacks unavailable")
	}
	if db.Callback().Query().Before("gorm:query").Register("record_fixture:journal_read", func(tx *gorm.DB) {
		if !tx.Statement.SkipHooks && tx.Statement.Table == "cl_monitor_config_pool_operations" && f.failure != nil && f.failure() == "hash read" {
			tx.AddError(errors.New("fixture durable publication read failure"))
		}
	}) != nil {
		t.Fatal("journal read fault unavailable")
	}
	wrapped := &recordFixturePool{DB: pool, fixture: f}
	db.ConnPool = wrapped
	db.Statement.ConnPool = wrapped
	return f
}
func (f *recordProjectionFixture) seed() error {
	if f.seeds == nil {
		return nil
	}
	f.seeding = true
	defer func() { f.seeding = false }()
	for _, input := range f.seeds() {
		if input == nil {
			continue
		}
		row := *input
		var actual model.MonitorConfig
		err := f.db.Unscoped().First(&actual, row.ID).Error
		if err == nil && reflect.DeepEqual(actual, row) {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// Direct SQL seeding models persisted historical/corrupt data. It does not
		// manufacture a journal event or generated permission for that data.
		if err := f.db.Session(&gorm.Session{SkipHooks: true}).Unscoped().Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
func (f *recordProjectionFixture) refresh() error {
	if f.observe == nil {
		return nil
	}
	var rows []*model.MonitorConfig
	kind := f.observationKind
	if kind == 0 {
		kind = model.ConfigTypeRecordRule
	}
	if err := f.db.Where("config_type = ?", kind).Order("id").Find(&rows).Error; err != nil {
		return err
	}
	f.observe(rows)
	return nil
}
func (f *recordProjectionFixture) GetGeneratedConfigPoolCandidates(ctx context.Context, req configdao.GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error) {
	if f.failure != nil && f.failure() == "db read" {
		return nil, 0, errors.New("fixture metadata read failure")
	}
	if err := f.seed(); err != nil {
		return nil, 0, err
	}
	return f.MonitorConfigDAO.GetGeneratedConfigPoolCandidates(ctx, req)
}
func (f *recordProjectionFixture) GetGeneratedConfigNamespacePage(ctx context.Context, req configdao.GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error) {
	return f.MonitorConfigDAO.GetGeneratedConfigNamespacePage(ctx, req)
}
func (f *recordProjectionFixture) attach(c *recordRuleConfigCache) {
	c.configDAO = f
	c.batchManager = NewBatchConfigManager(f, c.logger)
	c.journal = f.journal
	c.history = f.history
	c.projection = &recordFixtureRunner{fixture: f, store: c.redis}
}

type recordFixtureRunner struct {
	fixture *recordProjectionFixture
	store   redis.Cmdable
}

func (r *recordFixtureRunner) Project(ctx context.Context, lease configdao.ProjectionLease) (result error) {
	if lease == nil {
		return errors.New("fixture requires real sealed lease")
	}
	if err := lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	snapshot, err := lease.Snapshot()
	if err != nil {
		return err
	}
	if snapshot.Pool == nil || !snapshot.Pool.CompleteControls {
		return errors.New("fixture missing durable full plan")
	}
	family := snapshot.Pool.Reservation.Family
	if family != model.ConfigTypeAlertRule && family != model.ConfigTypeRecordRule {
		return errors.New("fixture only accepts sealed scrape-rule families")
	}

	if recordFixtureRedisFailure(r.store) == "pending" || recordFixtureRedisFailure(r.store) == "sadd" {
		return errors.New("fixture prepare failure")
	}
	for _, material := range snapshot.Artifacts {
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		key, err := projectionRawKey(material.Binding)
		if err != nil {
			return err
		}
		if material.Artifact.Action == model.ConfigProjectionRender {
			err = r.store.Set(ctx, key, string(material.Body), 0).Err()
		} else {
			err = r.store.Del(ctx, key).Err()
		}
		if err != nil {
			return err
		}
	}
	for _, control := range snapshot.Pool.Controls {
		if control.Members == nil {
			continue
		}
		key, err := projectionMemberKey(family, snapshot.Pool.Reservation.PoolID)
		if err != nil {
			return err
		}
		old, err := r.store.SMembers(ctx, key).Result()
		if err != nil {
			return err
		}
		desired := map[string]bool{}
		for _, ip := range control.Members.DesiredIPs {
			desired[ip] = true
			if err := r.store.SAdd(ctx, key, ip).Err(); err != nil {
				return err
			}
		}
		for _, ip := range old {
			if !desired[ip] {
				if err := r.store.SRem(ctx, key, ip).Err(); err != nil {
					return err
				}
			}
		}
	}
	if err := lease.RecheckCurrent(ctx); err != nil {
		return err
	}
	var proof configdao.ProjectionAckProof
	found := false
	for _, control := range snapshot.Pool.Controls {
		if control.Ack != nil {
			proof = configdao.ProjectionAckProof{Reservation: snapshot.Pool.Reservation, SourceHash: snapshot.Pool.SourceHash, PlanDigest: snapshot.Pool.PlanDigest, ArtifactVector: append([]configdao.ProjectionVectorEntry(nil), control.Ack.ArtifactVector...), RequiredControls: append([]configdao.ProjectionControlDigest(nil), control.Ack.RequiredControls...)}
			found = true
		}
	}
	if !found {
		return errors.New("fixture missing full ACK control")
	}
	legacyKey, err := projectionLegacyKey(family, proof.Reservation.PoolID)
	if err != nil {
		return err
	}
	if err := r.store.Set(ctx, legacyKey, hex.EncodeToString(proof.SourceHash[:]), 0).Err(); err != nil {
		return err
	}
	// Store persists its prepared wire proof before Acknowledge returns a
	// detached proof. Empty wire vectors are JSON arrays, not null slices.
	wireProof := proof
	wireProof.ArtifactVector = append([]configdao.ProjectionVectorEntry{}, proof.ArtifactVector...)
	wireProof.RequiredControls = append([]configdao.ProjectionControlDigest{}, proof.RequiredControls...)
	raw, err := json.Marshal(wireProof)
	if err != nil {
		return err
	}
	if err := r.store.Set(ctx, projectionPoolKey(family, proof.Reservation.PoolID, "ack"), string(raw), 0).Err(); err != nil {
		return err
	}
	marked, err := lease.MarkComplete(proof)
	if err != nil {
		return err
	}
	if !marked {
		return errors.New("fixture failed current SQL mark")
	}
	return nil
}
func recordFixtureRedisFailure(store redis.Cmdable) string {
	switch s := store.(type) {
	case *recordReconcileRedis:
		return s.failure
	case *recordingPublicationRedis:
		return s.failure
	case *supportRedis:
		return s.failure
	case *alertRulePubRedis:
		return s.failure
	}
	return ""
}

func recordFixturePending(c *recordRuleConfigCache) bool {
	runner, ok := c.projection.(*recordFixtureRunner)
	if !ok {
		return false
	}
	var control model.MonitorConfigPoolControl
	if runner.fixture.db.First(&control, "family=4 AND pool_id=7").Error != nil || control.CurrentAttemptOperationID == nil {
		return false
	}
	var op model.MonitorConfigPoolOperation
	if runner.fixture.db.First(&op, "operation_id=?", *control.CurrentAttemptOperationID).Error != nil || op.Phase == model.ConfigPublicationProjected {
		return false
	}
	raw, err := c.redis.Get(context.Background(), projectionPoolKey(4, 7, "ack")).Result()
	if err == redis.Nil {
		return true
	}
	if err != nil {
		return false
	}
	var proof configdao.ProjectionAckProof
	return json.Unmarshal([]byte(raw), &proof) == nil && proof.Reservation.OperationID != op.OperationID
}

type recordFixturePool struct {
	*sql.DB
	fixture *recordProjectionFixture
}

func (p *recordFixturePool) BeginTx(ctx context.Context, options *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.DB.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &recordFixtureTx{Tx: tx, fixture: p.fixture}, nil
}

type recordFixtureTx struct {
	*sql.Tx
	fixture *recordProjectionFixture
}

func (tx *recordFixtureTx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return err
	}
	if tx.fixture.seeding {
		return nil
	}
	tx.fixture.commits.Add(1)
	return tx.fixture.refresh()
}

func attachRecordingProjection(t *testing.T, c *recordRuleConfigCache, configs *recordingPublicationConfigs) {
	t.Helper()
	f := newRecordProjectionFixture(t, configs.targetedRows, func(rows []*model.MonitorConfig) {
		for key, row := range configs.rows {
			if row.ConfigType == 4 {
				delete(configs.rows, key)
			}
		}
		for _, row := range rows {
			copy := *row
			configs.rows[row.InstanceIP] = &copy
			if row.ID > configs.next {
				configs.next = row.ID
			}
		}
	}, func() string {
		if configs.failure {
			return "database"
		}
		return ""
	}, func() { configs.attempts++ })
	f.attach(c)
}
func attachRecordReconcileProjection(t *testing.T, c *recordRuleConfigCache, db *recordReconcileDB) {
	t.Helper()
	f := newRecordProjectionFixture(t, func() []*model.MonitorConfig { return targetedFakeMapRows(db.rows) }, func(rows []*model.MonitorConfig) {
		for id, row := range db.rows {
			if row.ConfigType == 4 {
				delete(db.rows, id)
			}
		}
		for _, row := range rows {
			copy := *row
			db.rows[row.ID] = &copy
			if row.ID > db.next {
				db.next = row.ID
			}
		}
	}, func() string { return db.failure }, func() { db.writes++ })
	db.projection = f
	f.attach(c)
}

// Exported only in the test binary so external-package HTTP/DAO fixtures use
// exactly the same real journal and sealed-lease runner, without production API.
func RecordProjectionDependenciesForTest(db *gorm.DB, store redis.Cmdable) (configdao.RecordProjectionJournalDAO, configdao.GeneratedConfigHistoryReader, RecordProjectionRunner) {
	log := zap.NewNop()
	f := &recordProjectionFixture{db: db, journal: configdao.NewRecordProjectionJournalDAO(log, db), history: configdao.NewGeneratedConfigHistoryReader(log, db)}
	return f.journal, f.history, &recordFixtureRunner{fixture: f, store: store}
}

// Interleaving delegates the exact typed business transaction. It never issues
// a synthetic permit/receipt or bypasses the journal's namespace/history guard.
type recordFixtureCommitHook struct {
	configdao.RecordProjectionJournalDAO
	before func()
}

func (h *recordFixtureCommitHook) CommitRecordProjectionPlan(ctx context.Context, req configdao.RecordProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	if h.before != nil {
		h.before()
	}
	return h.RecordProjectionJournalDAO.CommitRecordProjectionPlan(ctx, req)
}
