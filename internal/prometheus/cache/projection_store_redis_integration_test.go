//go:build integration

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

type projectionRedisConnection struct {
	Address    string `json:"address"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	OwnerKey   string `json:"owner_key"`
	OwnerToken string `json:"owner_token"`
}

func projectionRedisNative(t *testing.T) *redis.Client {
	t.Helper()
	path := os.Getenv("CLOUDOPS_PROJECTION_STORE_REDIS_CONNECTION_FILE")
	if path == "" {
		t.Skip("private projection Store Redis fixture not configured")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 16384 {
		t.Fatal("connection requires small private regular file")
	}
	parent, err := os.Lstat(filepath.Dir(path))
	if err != nil || !parent.IsDir() || parent.Mode().Perm() != 0700 {
		t.Fatal("connection requires private regular parent directory")
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal("private connection unavailable")
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		t.Fatal("private connection changed")
	}
	var cfg projectionRedisConnection
	decoder := json.NewDecoder(io.LimitReader(file, 16385))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cfg) != nil || decoder.Decode(new(interface{})) != io.EOF {
		t.Fatal("invalid private connection format")
	}
	host, port, err := net.SplitHostPort(cfg.Address)
	n, portErr := strconv.Atoi(port)
	if err != nil || host != "127.0.0.1" || portErr != nil || n < 1 || n > 65535 || n == 16379 || cfg.Username == "" || cfg.Password == "" || cfg.OwnerKey != "cloudops:test:projection-store:owner" || cfg.OwnerToken == "" {
		t.Fatal("connection is not a dedicated local projection fixture")
	}
	client := redis.NewClient(&redis.Options{Addr: cfg.Address, Username: cfg.Username, Password: cfg.Password, DB: 0, MaxRetries: -1})
	t.Cleanup(func() { client.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	marker, err := client.Get(ctx, cfg.OwnerKey).Result()
	if err != nil || marker != cfg.OwnerToken {
		t.Fatal("dedicated Redis ownership marker mismatch")
	}
	return client
}
func projectionTrack(t *testing.T, client *redis.Client, handle *PreparedProjection) {
	t.Helper()
	keys := append([]string(nil), handle.keys...)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Del(ctx, keys...).Err(); err != nil {
			t.Error("owned fixture keys could not be removed")
		}
	})
}
func projectionNativeApply(t *testing.T, client *redis.Client, store *ProjectionStore, journal configdao.ProjectionPlanJournalDAO, receipt configdao.ProjectionReceipt) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		handle, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		for _, control := range receipt.Controls {
			if control.Intents != nil {
				if err := handle.AddIntents(ctx, control.ControlID); err != nil {
					return err
				}
			}
		}
		for _, artifact := range receipt.Artifacts {
			if err := handle.ApplyArtifact(ctx, artifact.EventID); err != nil {
				return err
			}
		}
		for _, control := range receipt.Controls {
			if control.Members != nil {
				if err := handle.ApplyMembers(ctx, control.ControlID); err != nil {
					return err
				}
			}
			if control.Intents != nil {
				if err := handle.RemoveIntents(ctx, control.ControlID); err != nil {
					return err
				}
			}
		}
		proof, err := handle.Acknowledge(ctx)
		if err != nil {
			return err
		}
		marked, err := lease.MarkComplete(proof)
		if err == nil && !marked {
			return fmt.Errorf("current projection not marked")
		}
		return err
	})
	if err != nil {
		t.Fatal("projection protocol failed", err)
	}
}
func TestProjectionStoreNativeRedis(t *testing.T) {
	client := projectionRedisNative(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	t.Run("native bytes full ACK rebuild and empty controls", func(t *testing.T) {
		journal, _, _ := projectionStoreSQL(t)
		ref := projectionReserve(t, journal, 4, 701, 1)
		item := projectionRecordItem(701, 11, "127.0.7.1", "groups: [] # synthetic 秘密 bytes\n")
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
		if err != nil {
			t.Fatal(err)
		}
		projectionNativeApply(t, client, store, journal, receipt)
		body, err := client.Get(ctx, buildRedisKeyRecordRule(item.Binding.InstanceIP)).Result()
		if err != nil || body != item.Desired.ConfigContent {
			t.Fatal("native content was altered")
		}
		loaded, err := journal.LoadLastCommittedProjection(ctx, 4, 701)
		if err != nil || loaded == nil || loaded.Phase != model.ConfigPublicationProjected {
			t.Fatal("projected SQL receipt missing", err)
		}
		projectionNativeApply(t, client, store, journal, *loaded)
		emptyRef := projectionReserve(t, journal, 4, 702, 2)
		empty, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(emptyRef))
		if err != nil {
			t.Fatal(err)
		}
		projectionNativeApply(t, client, store, journal, empty)
		if client.Exists(ctx, projectionPoolKey(4, 702, "ack")).Val() != 1 || client.Get(ctx, recordPublicationKey(702)).Val() == "" {
			t.Fatal("empty publication lost ACK")
		}
	})
	t.Run("prepare TYPE failure has zero partial fences or raw writes", func(t *testing.T) {
		journal, _, _ := projectionStoreSQL(t)
		ref := projectionReserve(t, journal, 4, 703, 1)
		item := projectionRecordItem(703, 11, "127.0.7.3", "new synthetic")
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
		if err != nil {
			t.Fatal(err)
		}
		key := buildRedisKeyRecordRule(item.Binding.InstanceIP)
		if client.SAdd(ctx, key, "preserve existing wrong type").Err() != nil {
			t.Fatal("wrong type fixture failed")
		}
		t.Cleanup(func() { client.Del(ctx, key) })
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			compiled, err := compileProjection(store, lease, snapshot)
			if err != nil {
				return err
			}
			projectionTrack(t, client, compiled)
			if _, err := store.Prepare(ctx, lease); err == nil {
				t.Fatal("wrong TYPE prepare accepted")
			}
			for _, fence := range compiled.wire.Fences {
				if client.Exists(ctx, compiled.keys[fence.Key-1]).Val() != 0 {
					t.Fatal("partial prepare fence survived rejected TYPE")
				}
			}
			if client.SCard(ctx, key).Val() != 1 || client.Exists(ctx, recordPublicationKey(703)).Val() != 0 {
				t.Fatal("TYPE failure mutated raw/legacy state")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("same version conflicting nonce and invalid roles reject without writes", func(t *testing.T) {
		journal, _, _ := projectionStoreSQL(t)
		ref := projectionReserve(t, journal, 4, 704, 1)
		item := projectionRecordItem(704, 11, "127.0.7.4", "synthetic")
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
		if err != nil {
			t.Fatal(err)
		}
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			handle, err := store.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			baseline := client.HGetAll(ctx, handle.keys[handle.wire.PoolFence-1]).Val()
			raw, _ := json.Marshal(handle.wire)
			var altered projectionWire
			json.Unmarshal(raw, &altered)
			altered.Mode = "prepare"
			altered.Fences[0].Nonce = strings.Repeat("f", 32)
			command, _ := json.Marshal(altered)
			result, err := store.client.Eval(ctx, projectionFenceLua, handle.keys, string(command), "").Text()
			if err != nil || result != "CONFLICT" {
				t.Fatal("equal version conflict not rejected", err)
			}
			altered = handle.wire
			altered.Mode = "prepare"
			altered.Roles = append([]projectionRole(nil), altered.Roles...)
			altered.Roles[0].Role = "artifact"
			command, _ = json.Marshal(altered)
			result, err = store.client.Eval(ctx, projectionFenceLua, handle.keys, string(command), "").Text()
			if err != nil || result != "ROLE" {
				t.Fatal("invalid key role not rejected", err)
			}
			after := client.HGetAll(ctx, handle.keys[handle.wire.PoolFence-1]).Val()
			if projectionDigest(baseline) != projectionDigest(after) || client.Exists(ctx, buildRedisKeyRecordRule(item.Binding.InstanceIP)).Val() != 0 {
				t.Fatal("rejected command wrote state")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	for _, mode := range []string{"queued render", "queued retire"} {
		t.Run(mode+" above double precision", func(t *testing.T) {
			journal, ordinary, db := projectionStoreSQL(t)
			anchor := model.MonitorConfigArtifactEvent{
				EventID: 9007199254740992, ConfigType: 4, InstanceKey: []byte("127.0.99.99"),
				Action: model.ConfigProjectionInvalidate, ProjectionPolicy: model.ConfigProjectionNotApplicable,
				DeliveryState: model.ConfigProjectionApplied,
			}
			if err := db.Create(&anchor).Error; err != nil {
				t.Fatal("real SQLite event anchor fixture failed", err)
			}
			ref := projectionReserve(t, journal, 4, 705, 1)
			item := projectionRecordItem(705, 11, "127.0.7.5", "A original synthetic body")
			first, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
			if err != nil {
				t.Fatal(err)
			}
			if first.Artifacts[0].EventID <= 9007199254740992 {
				t.Fatal("real event IDs did not exceed 2^53")
			}
			projectionNativeApply(t, client, store, journal, first)
			expected := configdao.GeneratedConfigOwnerOf(item.Desired)
			next := projectionReserve(t, journal, 4, 705, 2)
			var changed configdao.GeneratedPlanItem
			if mode == "queued render" {
				changed = projectionRecordItem(705, 12, item.Binding.InstanceIP, "A changed synthetic body")
				changed.Expected = &expected
			} else {
				changed = configdao.GeneratedPlanItem{SourceID: 12, Action: model.ConfigProjectionRetire, Expected: &expected, Binding: item.Binding}
			}
			old, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(next, changed))
			if err != nil {
				t.Fatal(err)
			}
			var keys []string
			var command, body string
			err = journal.WithPoolProjectionLease(ctx, old, func(lease configdao.ProjectionLease) error {
				handle, err := store.Prepare(ctx, lease)
				if err != nil {
					return err
				}
				projectionTrack(t, client, handle)
				wire := handle.wire
				wire.Mode = "artifact"
				wire.Target = strconv.FormatUint(old.Artifacts[0].EventID, 10)
				raw, _ := json.Marshal(wire)
				command = string(raw)
				keys = append([]string(nil), handle.keys...)
				body = string(handle.materials[wire.Target])
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			if mode == "queued render" {
				if err := ordinary.DeleteMonitorConfig(ctx, item.Desired.ID); err != nil {
					t.Fatal(err)
				}
			}
			newerRef := projectionReserve(t, journal, 4, 706, 3)
			newerItem := projectionRecordItem(706, 13, item.Binding.InstanceIP, "B newer synthetic body")
			newer, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(newerRef, newerItem))
			if err != nil {
				t.Fatal(err)
			}
			projectionNativeApply(t, client, store, journal, newer)
			result, err := store.client.Eval(ctx, projectionFenceLua, keys, command, body).Text()
			if err != nil || result != "STALE" {
				t.Fatal("actually delayed old Redis command was not rejected", err)
			}
			if client.Get(ctx, buildRedisKeyRecordRule(item.Binding.InstanceIP)).Val() != newerItem.Desired.ConfigContent || client.Get(ctx, recordPublicationKey(706)).Val() == "" {
				t.Fatal("old command damaged higher owner body/ACK")
			}
		})
	}
	t.Run("ACK requires every durable control and no foreign intent residue", func(t *testing.T) {
		journal, ordinary, _ := projectionStoreSQL(t)
		group := 3
		row := &model.MonitorConfig{PoolID: 707, ConfigType: 5, InstanceIP: "127.0.7.7_3", Name: "webhook_synthetic_3_127.0.7.7", ConfigContent: "before", Status: 2}
		if err := ordinary.CreateMonitorConfig(ctx, row); err != nil {
			t.Fatal(err)
		}
		owner := configdao.GeneratedConfigOwnerOf(row)
		ref := projectionReserve(t, journal, 2, 707, 1)
		item := configdao.GeneratedPlanItem{SourceID: 31, Action: model.ConfigProjectionRender, Expected: &owner, Desired: row, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.7.7", GroupID: &group}}
		request := projectionFullRequest(ref, item)
		request.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
		change := configdao.IntentOwnerChange{Owner: owner, Proof: configdao.ProjectionProofRef{SourceID: 31}}
		request.Controls = append(request.Controls, configdao.ProjectionControl{ControlID: 3, Kind: configdao.ProjectionWebhookIntents, Intents: &configdao.WebhookIntentsControl{Add: []configdao.IntentOwnerChange{change}, Remove: []configdao.IntentOwnerChange{change}, RequireEmpty: true}})
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			handle, err := store.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			if _, err := handle.Acknowledge(ctx); err == nil {
				t.Fatal("ACK skipped artifact/member/intents")
			}
			if err := handle.AddIntents(ctx, 3); err != nil {
				return err
			}
			if err := handle.ApplyArtifact(ctx, receipt.Artifacts[0].EventID); err != nil {
				return err
			}
			if err := handle.ApplyMembers(ctx, 1); err != nil {
				return err
			}
			if _, err := handle.Acknowledge(ctx); err == nil {
				t.Fatal("ACK ignored nonempty required intent")
			}
			if err := handle.RemoveIntents(ctx, 3); err != nil {
				return err
			}
			if err := handle.AddIntents(ctx, 3); err != nil {
				return err
			}
			if client.SCard(ctx, alertRetirementIntentKey(707)).Val() != 0 {
				t.Fatal("late equal Add resurrected completed intents")
			}
			if client.SAdd(ctx, alertRetirementIntentKey(707), "foreign preserved residue").Err() != nil {
				t.Fatal("residue fixture failed")
			}
			if _, err := handle.Acknowledge(ctx); err == nil {
				t.Fatal("ACK ignored unknown intent residue")
			}
			if client.SRem(ctx, alertRetirementIntentKey(707), "foreign preserved residue").Err() != nil {
				t.Fatal("residue cleanup failed")
			}
			proof, err := handle.Acknowledge(ctx)
			if err != nil {
				return err
			}
			_, err = lease.MarkComplete(proof)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("webhook subset and ordinary lineage never gain full AM controls", func(t *testing.T) {
		journal, ordinary, db := projectionStoreSQL(t)
		group := 5
		ref := projectionReserve(t, journal, 5, 708, 1)
		row := &model.MonitorConfig{PoolID: 708, ConfigType: 5, InstanceIP: "127.0.7.8_5", Name: "webhook_subset_5_127.0.7.8", ConfigContent: "subset native body", Status: 1}
		item := configdao.GeneratedPlanItem{SourceID: 51, Action: model.ConfigProjectionRender, Desired: row, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.7.8", GroupID: &group}}
		request := configdao.GeneratedProjectionPlanRequest{GeneratedPlanRequest: configdao.GeneratedPlanRequest{Reservation: ref, Items: []configdao.GeneratedPlanItem{item}}, Controls: []configdao.ProjectionControl{{ControlID: 9, Kind: configdao.ProjectionACKWebhookSubset, Ack: &configdao.ProjectionAckControl{ArtifactSourceIDs: []uint64{51}}}}}
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		if client.Set(ctx, alertPublicationStateKey(708), "preserved full AM SHA", 0).Err() != nil || client.SAdd(ctx, buildRedisSetKeyAlertManagerMainPoolIPs(708), "preserved full member").Err() != nil {
			t.Fatal("full AM control fixtures failed")
		}
		t.Cleanup(func() { client.Del(ctx, alertPublicationStateKey(708), buildRedisSetKeyAlertManagerMainPoolIPs(708)) })
		projectionNativeApply(t, client, store, journal, receipt)
		changed := *row
		changed.ConfigContent = "ordinary SQL body cannot render"
		if err := ordinary.UpdateMonitorConfig(ctx, &changed); err != nil {
			t.Fatal(err)
		}
		var event model.MonitorConfigArtifactEvent
		if db.Order("event_id DESC").First(&event).Error != nil {
			t.Fatal("lineage event unavailable")
		}
		artifact := configdao.ArtifactReceipt{EventID: event.EventID, ConfigType: event.ConfigType, InstanceKey: string(event.InstanceKey), Action: event.Action, ProjectionPolicy: event.ProjectionPolicy, ConfigRowID: event.ConfigRowID, OldOwnerDigest: event.OldOwnerDigest, NewOwnerDigest: event.NewOwnerDigest, BodyDigest: event.BodyDigest, BindingEventID: event.BindingEventID}
		err = journal.WithArtifactProjectionLease(ctx, artifact, func(lease configdao.ProjectionLease) error {
			handle, err := store.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			if handle.wire.PoolFence != 0 || handle.wire.Ack != 0 || handle.wire.Legacy != 0 || len(handle.wire.Controls) != 0 {
				t.Fatal("ordinary handle acquired pool controls")
			}
			if _, err := handle.Acknowledge(ctx); err == nil {
				t.Fatal("ordinary handle acquired ACK")
			}
			return handle.ApplyArtifact(ctx, artifact.EventID)
		})
		if err != nil {
			t.Fatal(err)
		}
		if client.Exists(ctx, buildRedisKeyWebhookFile("127.0.7.8", 5)).Val() != 0 || client.Get(ctx, alertPublicationStateKey(708)).Val() != "preserved full AM SHA" || client.SCard(ctx, buildRedisSetKeyAlertManagerMainPoolIPs(708)).Val() != 1 {
			t.Fatal("subset/ordinary changed full AM control state")
		}
	})
	t.Run("lost raw response does not retry and durable receipt recovers", func(t *testing.T) {
		journal, _, _ := projectionStoreSQL(t)
		ref := projectionReserve(t, journal, 4, 710, 1)
		item := projectionRecordItem(710, 11, "127.0.7.10", "actually written before reply loss")
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
		if err != nil {
			t.Fatal(err)
		}
		var drop atomic.Bool
		var dropped, dials atomic.Int32
		options := *client.Options()
		options.PoolSize = 1
		options.MinIdleConns = 0
		options.Dialer = func(ctx context.Context, network, address string) (net.Conn, error) {
			dials.Add(1)
			conn, err := new(net.Dialer).DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return &projectionDropReadConn{Conn: conn, drop: &drop, dropped: &dropped}, nil
		}
		base := redis.NewClient(&options)
		defer base.Close()
		uncertain, err := NewProjectionStore(base)
		if err != nil {
			t.Fatal(err)
		}
		defer uncertain.Close()
		entered := false
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			handle, err := uncertain.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			drop.Store(true)
			entered = true
			err = handle.ApplyArtifact(ctx, receipt.Artifacts[0].EventID)
			if !errors.Is(err, ErrProjectionStore) {
				t.Fatal("unknown raw result was silently accepted", err)
			}
			if dropped.Load() != 1 || dials.Load() != 1 {
				t.Fatal("unknown raw command transparently retried or was not actually sent")
			}
			return err
		})
		if !entered || err == nil {
			t.Fatal("unknown response not surfaced")
		}
		if client.Get(ctx, buildRedisKeyRecordRule(item.Binding.InstanceIP)).Val() != item.Desired.ConfigContent || client.Exists(ctx, recordPublicationKey(710)).Val() != 0 {
			t.Fatal("unknown result fixture did not observe real partial raw success")
		}
		loaded, err := journal.LoadLastCommittedProjection(ctx, 4, 710)
		if err != nil || loaded == nil || loaded.Phase != model.ConfigPublicationCommitted {
			t.Fatal("unknown raw result lost durable committed SQL state", err)
		}
		projectionNativeApply(t, client, store, journal, *loaded)
	})

	t.Run("pool generations above double precision reject a delayed empty ACK", func(t *testing.T) {
		journal, _, db := projectionStoreSQL(t)
		if db.Create(&model.MonitorConfigPoolControl{Family: 4, PoolID: 711, ReservedGeneration: 9007199254740992}).Error != nil {
			t.Fatal("real SQL pool counter anchor failed")
		}
		firstRef := projectionReserve(t, journal, 4, 711, 1)
		if firstRef.Generation != 9007199254740993 {
			t.Fatal("real SQL generation lost precision")
		}
		first, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(firstRef))
		if err != nil {
			t.Fatal(err)
		}
		var queued string
		var keys []string
		err = journal.WithPoolProjectionLease(ctx, first, func(lease configdao.ProjectionLease) error {
			handle, err := store.Prepare(ctx, lease)
			if err != nil {
				return err
			}
			projectionTrack(t, client, handle)
			if err := handle.ApplyMembers(ctx, 1); err != nil {
				return err
			}
			wire := handle.wire
			wire.Mode = "ack"
			raw, _ := json.Marshal(wire)
			queued = string(raw)
			keys = append([]string(nil), handle.keys...)
			proof, err := handle.Acknowledge(ctx)
			if err != nil {
				return err
			}
			_, err = lease.MarkComplete(proof)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		secondRef := projectionReserve(t, journal, 4, 711, 2)
		if secondRef.Generation != firstRef.Generation+1 {
			t.Fatal("generation allocator did not increment")
		}
		second, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(secondRef))
		if err != nil {
			t.Fatal(err)
		}
		projectionNativeApply(t, client, store, journal, second)
		before := client.Get(ctx, projectionPoolKey(4, 711, "ack")).Val()
		result, err := store.client.Eval(ctx, projectionFenceLua, keys, queued, "").Text()
		if err != nil || result != "STALE" || client.Get(ctx, projectionPoolKey(4, 711, "ack")).Val() != before {
			t.Fatal("delayed high generation ACK was not zero-write rejected", err)
		}
	})

}

// Discard exactly one real Redis response after the server has replied. This
// preserves real Lua execution while making its result unknown to the caller.
type projectionDropReadConn struct {
	net.Conn
	drop    *atomic.Bool
	dropped *atomic.Int32
}

func (c *projectionDropReadConn) Read(buffer []byte) (int, error) {
	n, err := c.Conn.Read(buffer)
	if n > 0 && c.drop.CompareAndSwap(true, false) {
		c.dropped.Add(1)
		return 0, io.EOF
	}
	return n, err
}
