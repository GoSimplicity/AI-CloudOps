package cache

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"
)

type alertArtifactReads struct {
	*alertContractRedis
	failKey   string
	cancelKey string
	cancel    context.CancelFunc
	reads     []string
	sets      []string
}

func (r *alertArtifactReads) Get(ctx context.Context, key string) *redis.StringCmd {
	r.reads = append(r.reads, key)
	if key == r.failKey {
		return redis.NewStringResult("", errors.New("PRIVATE_ARTIFACT_READ_ERROR"))
	}
	if key == r.cancelKey && r.cancel != nil {
		r.cancel()
	}
	return r.alertContractRedis.Get(ctx, key)
}
func (r *alertArtifactReads) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) *redis.StatusCmd {
	r.sets = append(r.sets, key)
	return r.alertContractRedis.Set(ctx, key, value, ttl)
}

func TestAMArtifactReconcileMissingAndOverwrittenBytes(t *testing.T) {
	for _, kind := range []string{"main", "webhook"} {
		for _, drift := range []string{"missing", "overwritten"} {
			t.Run(kind+"/"+drift, func(t *testing.T) {
				c, _, r, db, pool, logs := alertContractCache(t)
				ctx := context.Background()
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				key := buildRedisKeyAlertManagerMain("127.0.0.1")
				if kind == "webhook" {
					key = buildRedisKeyWebhookFile("127.0.0.1", 3)
				}
				expected := r.values[key]
				rows := copyAlertRows(db.rows)
				ack := r.values[alertPublicationStateKey(pool.ID)]
				if drift == "missing" {
					delete(r.values, key)
				} else {
					r.values[key] = "fixture wrong artifact"
				}
				c = rebuildAlertRetirementABA(c)
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				if r.values[key] != expected || r.values[alertPublicationStateKey(pool.ID)] != ack || !reflect.DeepEqual(rows, db.rows) {
					t.Fatal("artifact repair changed source version/metadata or failed to restore native bytes")
				}
				writes := r.writes
				if err := c.GenerateMainConfig(ctx); err != nil || r.writes != writes {
					t.Fatal("healthy same-ACK publication no longer takes zero-write path")
				}
				assertAlertLogsSafe(t, logs)
			})
		}
	}
}
func TestAMArtifactReadErrorsAndCancellationPerformZeroWrites(t *testing.T) {
	for _, kind := range []string{"main", "webhook"} {
		for _, mode := range []string{"error", "cancel"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				c, _, r, db, _, logs := alertContractCache(t)
				if err := c.GenerateMainConfig(context.Background()); err != nil {
					t.Fatal(err)
				}
				key := buildRedisKeyAlertManagerMain("127.0.0.1")
				if kind == "webhook" {
					key = buildRedisKeyWebhookFile("127.0.0.1", 3)
				}
				r.values[key] = "fixture drift"
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				wrapped := &alertArtifactReads{alertContractRedis: r}
				if mode == "error" {
					wrapped.failKey = key
				} else {
					wrapped.cancelKey = key
					wrapped.cancel = cancel
				}
				c.redis = wrapped
				rows, values := copyAlertRows(db.rows), copyAlertRedis(r.values)
				writes := r.writes
				err := c.GenerateMainConfig(ctx)
				if err == nil || r.writes != writes || !sameAlertRetirementState(values, r.values, rows, db.rows) {
					t.Fatal("artifact read failure wrote a partial publication")
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal("artifact-read cancellation identity lost")
				}
				if strings.Contains(err.Error(), "PRIVATE_ARTIFACT_READ_ERROR") {
					t.Fatal("raw dependency error leaked")
				}
				assertAlertLogsSafe(t, logs)
			})
		}
	}
}
func TestAMArtifactRepairPendingFailuresRecoverAcrossSourceAndRename(t *testing.T) {
	for _, stage := range []string{"pending write", "main cache", "webhook cache", "hash write"} {
		for _, resolution := range []string{"A", "B", "rename B"} {
			t.Run(stage+"/"+resolution, func(t *testing.T) {
				c, g, r, db, pool, _ := alertContractCache(t)
				ctx := context.Background()
				originalName, originalGroup := pool.Name, g.rows[0].Name
				addressA := "https://fixture.invalid/api/v1/alerts/receive?token=SECRET_A"
				addressB := "https://fixture.invalid/api/v1/alerts/receive?token=SECRET_B"
				c.alertWebhookAddr = addressA
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				oldACK := r.values[alertPublicationStateKey(pool.ID)]
				// Source B changes both rendered main paths and real webhook URL bytes.
				pool.Name = "source-B"
				g.rows[0].Name = "group-B"
				c.alertWebhookAddr = addressB
				r.failure = stage
				if err := c.GenerateMainConfig(ctx); err == nil {
					t.Fatal("injected required-write failure reported success")
				}
				ack := r.values[alertPublicationStateKey(pool.ID)]
				if stage == "pending write" {
					if ack != oldACK {
						t.Fatal("failed pending write replaced old ACK")
					}
				} else if !strings.HasPrefix(ack, "pending:") {
					t.Fatal("partial required writes retained acknowledged publication")
				}
				r.failure = ""
				if resolution == "A" {
					pool.Name = originalName
					g.rows[0].Name = originalGroup
					c.alertWebhookAddr = addressA
				} else if resolution == "rename B" {
					pool.Name = "renamed-B"
				}
				c = rebuildAlertRetirementABA(c)
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				expectedWebhook := c.alertWebhookAddr + "&send_group_id=3"
				for _, ip := range pool.AlertManagerInstances {
					body := r.values[buildRedisKeyAlertManagerMain(ip)]
					parsed := parseAlertContract(t, body)
					if parsed.Route.Receiver != g.rows[0].Name || !strings.Contains(body, "webhook_"+pool.Name+"_3.yml") {
						t.Fatal("recovered main used stale source or rename")
					}
					if r.values[buildRedisKeyWebhookFile(ip, 3)] != expectedWebhook {
						t.Fatal("recovered native webhook URL was stale or redacted")
					}
					matching := false
					for _, row := range db.rows {
						if row.ConfigType == model.ConfigTypeAlertManager && row.InstanceIP == ip && row.ConfigContent == body {
							matching = true
						}
					}
					if !matching {
						t.Fatal("recovered Redis body and owned metadata disagree")
					}
				}
				if strings.HasPrefix(r.values[alertPublicationStateKey(pool.ID)], "pending:") {
					t.Fatal("recovery did not acknowledge complete required writes")
				}
			})
		}
	}
}
func TestAMArtifactRepairPreservesNativeSecretWithoutLoggingIt(t *testing.T) {
	c, _, r, _, _, logs := alertContractCache(t)
	const secret = "SYNTHETIC_SELF_HEAL_SECRET"
	c.alertWebhookAddr = "https://fixture.invalid/api/v1/alerts/receive?token=" + secret
	ctx := context.Background()
	if err := c.GenerateMainConfig(ctx); err != nil {
		t.Fatal(err)
	}
	key := buildRedisKeyWebhookFile("127.0.0.1", 3)
	expected := r.values[key]
	if !strings.Contains(expected, secret) {
		t.Fatal("fixture did not contain operational URL bytes")
	}
	delete(r.values, key)
	if err := c.GenerateMainConfig(ctx); err != nil || r.values[key] != expected {
		t.Fatal("self-heal failed to retain native operational URL")
	}
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), secret) {
			t.Fatal("self-heal exposed operational URL secret")
		}
	}
}
func TestAMArtifactReadScopePreservesDisabledBlackholeAndDeletedEmptyDesired(t *testing.T) {
	for _, mode := range []string{"disabled blackhole", "deleted empty desired"} {
		t.Run(mode, func(t *testing.T) {
			c, g, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if mode == "disabled blackhole" {
				g.rows[0].Enable = 2
			}
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			wrapped := &alertArtifactReads{alertContractRedis: r}
			c.redis = wrapped
			if mode == "disabled blackhole" {
				key := buildRedisKeyAlertManagerMain("127.0.0.1")
				delete(r.values, key)
				if err := c.GenerateMainConfig(ctx); err != nil {
					t.Fatal(err)
				}
				if parseAlertContract(t, r.values[key]).Route.Receiver != "cloudops-disabled" {
					t.Fatal("blackhole main contract changed")
				}
				for _, key := range append(wrapped.reads, wrapped.sets...) {
					if strings.HasPrefix(key, "monitor:webhook_file:") {
						t.Fatal("disabled group produced/read desired webhook")
					}
				}
			} else {
				c.alertPoolDAO = &deletedAMInventory{pool: pool}
				markAMDeleted(pool)
				g.fail = true
				wrapped.failKey = buildRedisKeyAlertManagerMain("127.0.0.1")
				if err := c.RetireDeletedAlertManagerPool(ctx, pool.ID); err != nil {
					t.Fatal(err)
				}
				assertDeletedAMRetired(t, r, db)
			}
		})
	}
}

func TestAMArtifactSameSourceRepairFailureRetainsRetryAcrossRebuild(t *testing.T) {
	for _, tc := range []struct{ kind, stage string }{{"main", "pending write"}, {"main", "main cache"}, {"webhook", "webhook cache"}, {"webhook", "hash write"}} {
		t.Run(tc.kind+"/"+tc.stage, func(t *testing.T) {
			c, _, r, db, pool, _ := alertContractCache(t)
			ctx := context.Background()
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			key := buildRedisKeyAlertManagerMain("127.0.0.1")
			if tc.kind == "webhook" {
				key = buildRedisKeyWebhookFile("127.0.0.1", 3)
			}
			expected := r.values[key]
			ack := r.values[alertPublicationStateKey(pool.ID)]
			rows := copyAlertRows(db.rows)
			delete(r.values, key)
			r.failure = tc.stage
			if err := c.GenerateMainConfig(ctx); err == nil {
				t.Fatal("same-source repair ignored required write failure")
			}
			marker := r.values[alertPublicationStateKey(pool.ID)]
			if tc.stage == "pending write" {
				if marker != ack {
					t.Fatal("failed pending replaced old ACK")
				}
			} else if !strings.HasPrefix(marker, "pending:") {
				t.Fatal("partial same-source repair was acknowledged")
			}
			r.failure = ""
			c = rebuildAlertRetirementABA(c)
			if err := c.GenerateMainConfig(ctx); err != nil {
				t.Fatal(err)
			}
			if r.values[key] != expected || r.values[alertPublicationStateKey(pool.ID)] != ack || !reflect.DeepEqual(rows, db.rows) {
				t.Fatal("same-source repair did not converge to native bytes/version after rebuild")
			}
		})
	}
}
