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

type alertRuleProjectionFixture struct {
	base    *recordProjectionFixture
	journal configdao.AlertRuleProjectionJournalDAO
	history configdao.AlertRuleHistoryReader
	observe func([]*model.MonitorConfig)
}

func newAlertRuleProjectionFixture(t *testing.T, view *alertRulePubDB, shared *recordProjectionFixture) *alertRuleProjectionFixture {
	t.Helper()
	observe := func(rows []*model.MonitorConfig) {
		for id, row := range view.rows {
			if row.ConfigType == 3 {
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
	}
	base := shared
	if base == nil {
		base = newRecordProjectionFixture(t, func() []*model.MonitorConfig { return targetedFakeMapRows(view.rows) }, observe, func() string { return view.failure }, func() { view.writes++ })
		base.observationKind = 3
	}
	f := &alertRuleProjectionFixture{base: base, journal: configdao.NewAlertRuleProjectionJournalDAO(zap.NewNop(), base.db), history: configdao.NewAlertRuleHistoryReader(zap.NewNop(), base.db), observe: observe}
	view.projection = f
	return f
}
func (f *alertRuleProjectionFixture) attach(c *alertRuleConfigCache) {
	c.configDAO = f.base
	c.batchManager = NewBatchConfigManager(f.base, c.logger)
	c.journal = f.journal
	c.history = f.history
	c.projection = &recordFixtureRunner{fixture: f.base, store: c.redis}
}

// Shared Support runs both families against one SQL database. Refresh family3
// only after its outer call releases the sealed transaction (pool size is one).
func (f *alertRuleProjectionFixture) refresh() error {
	var rows []*model.MonitorConfig
	if err := f.base.db.Where("config_type=3").Order("id").Find(&rows).Error; err != nil {
		return err
	}
	f.observe(rows)
	return nil
}
func alertFixturePending(c *alertRuleConfigCache) bool {
	runner, ok := c.projection.(*recordFixtureRunner)
	if !ok {
		return false
	}
	var control model.MonitorConfigPoolControl
	if runner.fixture.db.First(&control, "family=3 AND pool_id=7").Error != nil || control.CurrentAttemptOperationID == nil {
		return false
	}
	var op model.MonitorConfigPoolOperation
	if runner.fixture.db.First(&op, "operation_id=?", *control.CurrentAttemptOperationID).Error != nil || op.Phase == model.ConfigPublicationProjected {
		return false
	}
	raw, err := c.redis.Get(context.Background(), projectionPoolKey(3, 7, "ack")).Result()
	if err == redis.Nil {
		return true
	}
	if err != nil {
		return false
	}
	var proof configdao.ProjectionAckProof
	return json.Unmarshal([]byte(raw), &proof) == nil && proof.Reservation.OperationID != op.OperationID
}
func AlertRuleProjectionDependenciesForTest(db *gorm.DB, store redis.Cmdable) (configdao.AlertRuleProjectionJournalDAO, configdao.AlertRuleHistoryReader, AlertRuleProjectionRunner) {
	log := zap.NewNop()
	base := &recordProjectionFixture{db: db}
	return configdao.NewAlertRuleProjectionJournalDAO(log, db), configdao.NewAlertRuleHistoryReader(log, db), &recordFixtureRunner{fixture: base, store: store}
}

func attachAlertRuleProjection(t *testing.T, c *alertRuleConfigCache, view *alertRulePubDB, shared *recordProjectionFixture) *alertRuleProjectionFixture {
	f := newAlertRuleProjectionFixture(t, view, shared)
	f.attach(c)
	return f
}
