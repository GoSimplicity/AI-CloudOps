package cache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

type alertRetirementRedis struct {
	*alertContractRedis
	failIntentAdd, failIntentRemove bool
	onPending                       func()
}

func (r *alertRetirementRedis) Set(ctx context.Context, k string, v any, ttl time.Duration) *redis.StatusCmd {
	if strings.HasPrefix(v.(string), "pending:") && r.onPending != nil {
		f := r.onPending
		r.onPending = nil
		f()
	}
	return r.alertContractRedis.Set(ctx, k, v, ttl)
}
func (r *alertRetirementRedis) SAdd(ctx context.Context, k string, v ...any) *redis.IntCmd {
	if r.failIntentAdd && strings.Contains(k, ":retirement:") {
		return redis.NewIntResult(0, errors.New("injected"))
	}
	return r.alertContractRedis.SAdd(ctx, k, v...)
}
func (r *alertRetirementRedis) SRem(ctx context.Context, k string, v ...any) *redis.IntCmd {
	if r.failIntentRemove && strings.Contains(k, ":retirement:") {
		return redis.NewIntResult(0, errors.New("injected"))
	}
	return r.alertContractRedis.SRem(ctx, k, v...)
}
func TestAlertRetirementRetryAnchors(t *testing.T) {
	for _, mode := range []string{"rebuild", "ABA", "rename ABA", "rename ABA index remove retry", "active rename ABA SQL rollback", "index remove retry", "index add failure", "main DB retired", "foreign replacement", "changed tombstone"} {
		t.Run(mode, func(t *testing.T) {
			c, g, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			originalName := pool.Name
			hookKey := buildRedisKeyWebhookFile("127.0.0.2", 3)
			var oldOwner configdao.GeneratedConfigOwner
			for _, row := range db.rows {
				if row.ConfigType == 5 && row.InstanceIP == "127.0.0.2_3" {
					oldOwner = configdao.GeneratedConfigOwnerOf(row)
				}
			}
			baselineRows, baselineRedis := copyAlertRows(db.rows), copyAlertRedis(r.values)
			baselineWrites := r.writes
			wrapped := &alertRetirementRedis{alertContractRedis: r}
			c.redis = wrapped
			pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
			if mode == "index add failure" {
				wrapped.failIntentAdd = true
			} else {
				r.failure = "webhook delete"
			}
			if mode == "main DB retired" {
				r.failure = "main delete"
			}
			if strings.HasPrefix(mode, "active ") {
				db.failure = "webhook DB delete"
			}
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("retirement failure accepted")
			}
			if strings.HasPrefix(mode, "active ") {
				// SQL now commits before Redis projection: a failed delete rolls
				// back the whole business transaction and creates no new intent.
				if r.writes != baselineWrites || !sameAlertRetirementState(baselineRedis, r.values, baselineRows, db.rows) {
					t.Fatal("failed SQL retirement changed last-good metadata or Redis")
				}
			}
			if !amFixturePending(c, 2, pool.ID) {
				t.Fatal("failed retirement ACKed")
			}
			if mode == "index add failure" {
				if !amFixturePending(c, 2, pool.ID) {
					t.Fatal("failed external intent add lost the SQL retirement plan")
				}
				wrapped.failIntentAdd = false
			} else if mode != "main DB retired" && !strings.HasPrefix(mode, "active ") {
				if db.rows[oldOwner.ID] != nil || r.values[hookKey] == "" || len(r.members[alertRetirementIntentKey(pool.ID)]) != 1 {
					t.Fatal("retired DB lost retry locator or cached content")
				}
			}
			if mode == "main DB retired" {
				for _, row := range db.rows {
					if row.ConfigType == 2 && row.InstanceIP == "127.0.0.2" {
						t.Fatal("retired main remains active")
					}
				}
			}
			r.failure = ""
			db.failure = ""
			if mode == "foreign replacement" {
				db.next++
				db.rows[db.next] = &model.MonitorConfig{Model: model.Model{ID: db.next}, PoolID: 99, ConfigType: 5, InstanceIP: oldOwner.InstanceIP, Name: "custom", ConfigContent: "custom"}
			}
			if mode == "changed tombstone" {
				owner := db.retiredOwners[oldOwner.ID]
				owner.Name = "changed"
				db.retiredOwners[oldOwner.ID] = owner
			}
			if strings.Contains(mode, "ABA") {
				pool.AlertManagerInstances = model.StringList{"127.0.0.1", "127.0.0.2"}
				if strings.Contains(mode, "rename ABA") {
					pool.Name = "renamed"
				}
			}
			// Reconstruct the cache object: no process-local retirement ledger is required.
			rebuilt := alertManagerConfigCache{redis: c.redis, logger: c.logger, alertWebhookAddr: c.alertWebhookAddr, webhookFileDir: c.webhookFileDir, serverPort: c.serverPort, alertPoolDAO: c.alertPoolDAO, alertSendDAO: c.alertSendDAO, configDAO: c.configDAO, batchManager: c.batchManager, journal: c.journal, history: c.history, projection: c.projection}
			if strings.Contains(mode, "index remove retry") {
				wrapped.failIntentRemove = true
			}
			err := rebuilt.GenerateMainConfig(ctx)
			if mode == "foreign replacement" || mode == "changed tombstone" {
				if err == nil || r.values[hookKey] == "" {
					t.Fatal("foreign retry was destructive or acknowledged")
				}
				return
			}
			if strings.Contains(mode, "index remove retry") {
				if err == nil || len(r.members[alertRetirementIntentKey(pool.ID)]) == 0 {
					t.Fatal("failed SREM lost retry anchor")
				}
				wrapped.failIntentRemove = false
				err = rebuilt.GenerateMainConfig(ctx)
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(r.members[alertRetirementIntentKey(pool.ID)]) != 0 {
				t.Fatal("completed retirement retained intent")
			}
			if strings.Contains(mode, "ABA") {
				if r.values[hookKey] == "" {
					t.Fatal("ABA removed restored webhook")
				}
				found := false
				for _, row := range db.rows {
					if row.InstanceIP == oldOwner.InstanceIP && row.ConfigType == 5 {
						found = true
						if strings.Contains(mode, "rename ABA") && strings.Contains(row.Name, originalName) {
							t.Fatal("rename not restored")
						}
					}
				}
				if !found {
					t.Fatal("ABA metadata not restored")
				}
			} else if r.values[hookKey] != "" {
				t.Fatal("retired webhook not cleaned")
			}
			if strings.HasPrefix(mode, "active ") {
				g.rows[0].Enable = 2
				if err := rebuilt.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				if r.values[hookKey] != "" || len(r.members[alertRetirementIntentKey(pool.ID)]) != 0 {
					t.Fatal("restored SQL-rollback owner did not retire after disable")
				}
			}
		})
	}
}
func TestAlertRetirementOwnerChangeAfterPreflight(t *testing.T) {
	for _, kind := range []int8{model.ConfigTypeAlertManager, model.ConfigTypeWebhookFile} {
		t.Run(string(rune('0'+kind)), func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			ip := "127.0.0.2"
			key := buildRedisKeyAlertManagerMain(ip)
			if kind == 5 {
				ip += "_3"
				key = buildRedisKeyWebhookFile("127.0.0.2", 3)
			}
			var target *model.MonitorConfig
			for _, row := range db.rows {
				if row.ConfigType == kind && row.InstanceIP == ip {
					target = row
				}
			}
			before := r.values[key]
			called := false
			wrapped := &alertRetirementRedis{alertContractRedis: r, onPending: func() {
				called = true
				row, err := db.projection.MonitorConfigDAO.GetMonitorConfigByID(ctx, target.ID)
				if err != nil {
					t.Fatal("actual owner fixture missing", err)
				}
				row.Name = "operator-custom"
				if err := db.projection.MonitorConfigDAO.UpdateMonitorConfig(ctx, row); err != nil {
					t.Fatal("actual ordinary owner change failed", err)
				}
			}}
			c.redis = wrapped
			pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("changed owner accepted")
			}
			if !called {
				t.Fatal("actual typed AM commit interleaving was not reached")
			}
			actual, err := db.projection.MonitorConfigDAO.GetMonitorConfigByID(ctx, target.ID)
			if err != nil || actual.ID != target.ID || actual.Name != "operator-custom" || actual.ConfigType != kind || r.values[key] != before {
				t.Fatal("CAS rejection followed destructive cleanup or lost the actual changed owner")
			}
		})
	}
}
func TestAlertRetirementMalformedIntentZeroWrites(t *testing.T) {
	for _, raw := range []string{"not-json", `{"ID":1}`, `{"ID":1,"PoolID":99,"ConfigType":5,"InstanceIP":"127.0.0.2_3","Name":"webhook_x_3_127.0.0.2"}`} {
		c, _, r, db, pool, _ := alertContractCache(t)
		if err := c.GenerateMainConfig(context.Background()); err != nil {
			t.Fatal(err)
		}
		r.members[alertRetirementIntentKey(pool.ID)] = map[string]bool{raw: true}
		before := r.writes
		rows, values := copyAlertRows(db.rows), copyAlertRedis(r.values)
		versions, err := db.projection.journal.ReadAMSourceVersions(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		err = c.GenerateMainConfig(context.Background())
		if err == nil || r.writes != before {
			t.Fatalf("invalid intent rejected=%v publication write delta=%d", err != nil, r.writes-before)
		}
		afterVersions, err := db.projection.journal.ReadAMSourceVersions(context.Background())
		if err != nil || !reflect.DeepEqual(versions, afterVersions) || !sameAlertRetirementState(values, r.values, rows, db.rows) {
			t.Fatal("invalid intent changed a source reservation or last-good business state")
		}
	}
}
func TestAlertRetirementIntentContainsOnlyOwner(t *testing.T) {
	c, _, r, _, pool, _ := alertContractCache(t)
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	pool.AlertManagerInstances = model.StringList{"127.0.0.1"}
	r.failure = "webhook delete"
	if c.GenerateMainConfig(ctx) == nil {
		t.Fatal("missing failure")
	}
	for encoded := range r.members[alertRetirementIntentKey(pool.ID)] {
		var data map[string]any
		if json.Unmarshal([]byte(encoded), &data) != nil || len(data) != 5 {
			t.Fatal("intent does not contain exactly owner fields")
		}
		if strings.Contains(encoded, "Content") || strings.Contains(encoded, "http") {
			t.Fatal("intent leaked body")
		}
	}
}
