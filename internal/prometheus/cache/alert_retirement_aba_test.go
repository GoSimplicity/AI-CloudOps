package cache

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func rebuildAlertRetirementABA(c *alertManagerConfigCache) *alertManagerConfigCache {
	return &alertManagerConfigCache{redis: c.redis, logger: c.logger, alertWebhookAddr: c.alertWebhookAddr, webhookFileDir: c.webhookFileDir, serverPort: c.serverPort, alertPoolDAO: c.alertPoolDAO, alertSendDAO: c.alertSendDAO, configDAO: c.configDAO, batchManager: c.batchManager, journal: c.journal, history: c.history, projection: c.projection}
}
func TestAlertRetirementSecondABAWithOldIntent(t *testing.T) {
	for _, firstFailure := range []string{"webhook delete", "intent remove"} {
		for _, replacement := range []string{"owned active", "owned inactive", "foreign", "custom inactive"} {
			t.Run(firstFailure+"/"+replacement, func(t *testing.T) {
				c, g, r, db, pool, _ := alertContractCache(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				key := "127.0.0.2_3"
				redisKey := buildRedisKeyWebhookFile("127.0.0.2", 3)
				oldID := 0
				for _, row := range db.rows {
					if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == key {
						oldID = row.ID
					}
				}
				if oldID == 0 {
					t.Fatal("baseline webhook missing")
				}
				wrapped := &alertRetirementRedis{alertContractRedis: r}
				c.redis = wrapped
				pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
				if firstFailure == "webhook delete" {
					r.failure = "webhook delete"
				} else {
					wrapped.failIntentRemove = true
				}
				if err := c.GenerateMainConfig(ctx); err == nil {
					t.Fatal("first retirement fault not exercised")
				}
				if db.rows[oldID] != nil || len(r.members[alertRetirementIntentKey(pool.ID)]) == 0 {
					t.Fatal("old tombstone or durable intent missing")
				}
				r.failure = ""
				wrapped.failIntentRemove = true
				pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
				c = rebuildAlertRetirementABA(c)
				if err := c.GenerateMainConfig(ctx); err == nil {
					t.Fatal("restored A must retain failed SREM")
				}
				var restored *model.MonitorConfig
				for _, row := range db.rows {
					if row.ConfigType == model.ConfigTypeWebhookFile && row.InstanceIP == key {
						restored = row
					}
				}
				if restored == nil || restored.ID == oldID || restored.Status != model.ConfigStatusActive || r.values[redisKey] == "" {
					t.Fatal("A did not create a distinct new owned row")
				}
				wrapped.failIntentRemove = false
				if replacement == "foreign" {
					restored.PoolID = 99
					restored.Name = "foreign-preserved"
				}
				if replacement == "owned inactive" || replacement == "custom inactive" {
					restored.Status = model.ConfigStatusInactive
				}
				if replacement == "custom inactive" {
					restored.Name = "operator-custom"
				}
				g.rows[0].Enable = 2
				c = rebuildAlertRetirementABA(c)
				beforeRedis, beforeRows := copyAlertRedis(r.values), copyAlertRows(db.rows)
				rw, dw := r.writes, db.writes
				err := c.GenerateMainConfig(ctx)
				t.Logf("replacement=%s oldID=%d restoredID=%d error=%v intentCount=%d", replacement, oldID, restored.ID, err != nil, len(r.members[alertRetirementIntentKey(pool.ID)]))
				if replacement == "owned active" || replacement == "owned inactive" {
					if err != nil {
						t.Fatal("new canonical active owned row is permanently blocked by its obsolete retirement intent")
					}
					if len(r.members[alertRetirementIntentKey(pool.ID)]) != 0 || r.values[redisKey] != "" {
						t.Fatal("off did not converge")
					}
				} else {
					if err == nil || r.writes != rw || db.writes != dw {
						t.Fatal("foreign or custom inactive replacement was not rejected before writes")
					}
					if !sameAlertRetirementState(beforeRedis, r.values, beforeRows, db.rows) {
						t.Fatal("protected replacement state changed")
					}
				}
			})
		}
	}
}

func sameAlertRetirementState(a, b map[string]string, c, d map[int]*model.MonitorConfig) bool {
	return reflect.DeepEqual(a, b) && reflect.DeepEqual(c, d)
}

// A failed SQL retirement must preserve the original live row and last-good
// projection. A later rename can succeed without inventing a retirement intent;
// a subsequent committed retirement still needs durable retry after Redis fails.
func TestAlertRetirementRenamedIntentTombstoneRetry(t *testing.T) {
	c, g, r, db, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	wrapped := &alertRetirementRedis{alertContractRedis: r}
	c.redis = wrapped
	beforeRows, beforeRedis := copyAlertRows(db.rows), copyAlertRedis(r.values)
	beforeWrites := r.writes
	pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
	db.failure = "webhook DB delete"
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("initial DB fault not exercised")
	}
	if r.writes != beforeWrites || !sameAlertRetirementState(beforeRedis, r.values, beforeRows, db.rows) {
		t.Fatal("failed SQL retirement changed last-good metadata or Redis")
	}
	db.failure = ""
	pool.Name = "renamed"
	pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
	c = rebuildAlertRetirementABA(c)
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal("restore and rename after rolled-back retirement failed", err)
	}
	if r.values[buildRedisKeyWebhookFile("127.0.0.2", 3)] == "" || len(r.members[alertRetirementIntentKey(pool.ID)]) != 0 {
		t.Fatal("restore lost its webhook or invented a retirement intent")
	}
	g.rows[0].Enable = 2
	r.failure = "webhook delete"
	c = rebuildAlertRetirementABA(c)
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("renamed retirement deletion fault not exercised")
	}
	r.failure = ""
	c = rebuildAlertRetirementABA(c)
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	if len(r.members[alertRetirementIntentKey(pool.ID)]) != 0 {
		t.Fatal("renamed tombstone intents not drained")
	}
}
