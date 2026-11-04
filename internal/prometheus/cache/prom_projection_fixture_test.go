package cache

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type promMainProjectionFixture struct {
	base    *recordProjectionFixture
	journal configdao.PromMainProjectionJournalDAO
	history configdao.PromMainHistoryReader
}

func newPromMainProjectionFixture(t *testing.T, seeds func() []*model.MonitorConfig, observe func([]*model.MonitorConfig), failure func() string, write func()) *promMainProjectionFixture {
	t.Helper()
	base := newRecordProjectionFixture(t, seeds, observe, failure, write)
	base.observationKind = 1
	return &promMainProjectionFixture{base: base, journal: configdao.NewPromMainProjectionJournalDAO(zap.NewNop(), base.db), history: configdao.NewPromMainHistoryReader(zap.NewNop(), base.db)}
}
func (f *promMainProjectionFixture) attach(c *prometheusConfigCache) {
	c.configDAO = f.base
	c.batchManager = NewBatchConfigManager(f.base, c.logger)
	c.journal = f.journal
	c.history = f.history
	c.projection = &recordFixtureRunner{fixture: f.base, store: c.redis}
}
func attachPromReconcileProjection(t *testing.T, c *prometheusConfigCache, view *promReconcileDB) {
	t.Helper()
	f := newPromMainProjectionFixture(t, func() []*model.MonitorConfig { return targetedFakeMapRows(view.rows) }, func(rows []*model.MonitorConfig) {
		for id, row := range view.rows {
			if row.ConfigType == 1 {
				delete(view.rows, id)
			}
		}
		for _, row := range rows {
			copy := *row
			view.rows[row.ID] = &copy
			if row.ID > view.next {
				view.next = row.ID
			}
		}
	}, func() string { return view.failure }, func() { view.writes++ })
	view.projection = f
	f.attach(c)
}
func attachPromPublicationProjection(t *testing.T, c *prometheusConfigCache, view *recordingPublicationConfigs, store *promPublicationRedis) *promMainProjectionFixture {
	t.Helper()
	f := newPromMainProjectionFixture(t, view.targetedRows, func(rows []*model.MonitorConfig) {
		for key, row := range view.rows {
			if row.ConfigType == 1 {
				delete(view.rows, key)
			}
		}
		for _, row := range rows {
			copy := *row
			view.rows[row.InstanceIP] = &copy
			if row.ID > view.next {
				view.next = row.ID
			}
		}
	}, func() string {
		if view.failure {
			return "database"
		}
		if store.failure == "get" {
			return "hash read"
		}
		return ""
	}, func() { view.attempts++ })
	f.attach(c)
	return f
}
func promFixturePending(c *prometheusConfigCache, pool int) bool {
	runner, ok := c.projection.(*recordFixtureRunner)
	if !ok {
		return false
	}
	var control model.MonitorConfigPoolControl
	if runner.fixture.db.First(&control, "family=1 AND pool_id=?", pool).Error != nil || control.CurrentAttemptOperationID == nil {
		return false
	}
	var op model.MonitorConfigPoolOperation
	if runner.fixture.db.First(&op, "operation_id=?", *control.CurrentAttemptOperationID).Error != nil || op.Phase == model.ConfigPublicationProjected {
		return false
	}
	raw, err := c.redis.Get(context.Background(), projectionPoolKey(1, pool, "ack")).Result()
	if err == redis.Nil {
		return true
	}
	if err != nil {
		return false
	}
	var proof configdao.ProjectionAckProof
	return json.Unmarshal([]byte(raw), &proof) == nil && proof.Reservation.OperationID != op.OperationID
}
func PromMainProjectionDependenciesForTest(db *gorm.DB, store redis.Cmdable) (configdao.PromMainProjectionJournalDAO, configdao.PromMainHistoryReader, PromMainProjectionRunner) {
	log := zap.NewNop()
	base := &recordProjectionFixture{db: db}
	return configdao.NewPromMainProjectionJournalDAO(log, db), configdao.NewPromMainHistoryReader(log, db), &recordFixtureRunner{fixture: base, store: store}
}

type promMainFixtureCommitHook struct {
	configdao.PromMainProjectionJournalDAO
	before func()
}

func (h *promMainFixtureCommitHook) CommitPromMainProjectionPlan(ctx context.Context, req configdao.PromMainProjectionPlanRequest) (configdao.ProjectionReceipt, error) {
	if h.before != nil {
		h.before()
	}
	return h.PromMainProjectionJournalDAO.CommitPromMainProjectionPlan(ctx, req)
}
func (f *recordReuseFixture) cacheWithCommitHook(id int, before func()) scrapeRuleTestPublisher {
	cache := f.cache(id)
	switch c := cache.(type) {
	case *prometheusConfigCache:
		journal, ok := c.journal.(configdao.PromMainProjectionJournalDAO)
		if !ok {
			f.test.Fatal("real main journal missing")
		}
		c.journal = &promMainFixtureCommitHook{PromMainProjectionJournalDAO: journal, before: before}
	case *alertRuleConfigCache:
		journal, ok := c.journal.(configdao.AlertRuleProjectionJournalDAO)
		if !ok {
			f.test.Fatal("real alert journal missing")
		}
		c.journal = &alertRuleFixtureCommitHook{AlertRuleProjectionJournalDAO: journal, before: before}
	case *recordRuleConfigCache:
		journal, ok := c.journal.(configdao.RecordProjectionJournalDAO)
		if !ok {
			f.test.Fatal("real record journal missing")
		}
		c.journal = &recordFixtureCommitHook{RecordProjectionJournalDAO: journal, before: before}
	default:
		f.test.Fatal("unsupported sealed publisher")
	}
	return cache
}
