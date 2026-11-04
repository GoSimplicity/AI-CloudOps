//go:build integration

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func TestProjectionBatchNativeRejectsBeforeAnyArtifactWrite(t *testing.T) {
	client := projectionRedisNative(t)
	for index, mode := range []string{"last body", "last unknown id", "last duplicate id", "last raw TYPE", "changed prepared SHA", "valid batch"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			journal, _, _ := projectionStoreSQL(t)
			ref := projectionReserve(t, journal, 4, 23001+index, 1)
			items := []configdao.GeneratedPlanItem{
				projectionRecordItem(ref.PoolID, 1, "127.0.230.1", "groups: [] # first\n"),
				projectionRecordItem(ref.PoolID, 2, "127.0.230.2", "groups: [] # second\n"),
				projectionRecordItem(ref.PoolID, 3, "127.0.230.3", "groups: [] # last\n"),
			}
			receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			if err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
				prepared, err := store.Prepare(ctx, lease)
				if err != nil {
					return err
				}
				projectionTrack(t, client, prepared)
				wire := prepared.wire
				wire.Mode, wire.Target = "artifact_batch", ""
				ids := []string{}
				bodies := []string{}
				for _, artifact := range receipt.Artifacts {
					id := strconv.FormatUint(artifact.EventID, 10)
					ids = append(ids, id)
					bodies = append(bodies, string(prepared.materials[id]))
				}
				switch mode {
				case "last body":
					bodies[len(bodies)-1] += "corrupt"
				case "last unknown id":
					ids[len(ids)-1] = "18446744073709551615"
				case "last duplicate id":
					ids[len(ids)-1] = ids[0]
				case "last raw TYPE":
					key := buildRedisKeyRecordRule(items[len(items)-1].Desired.InstanceIP)
					if err := client.HSet(ctx, key, "poison", "owned synthetic TYPE").Err(); err != nil {
						return err
					}
				case "changed prepared SHA":
					wire.LegacySHA = strings.Repeat("f", 64)
				}
				frame, err := json.Marshal(wire)
				if err != nil {
					return err
				}
				selection, err := json.Marshal(ids)
				if err != nil {
					return err
				}
				args := []interface{}{string(frame), string(selection)}
				for _, body := range bodies {
					args = append(args, body)
				}
				before := rootProjectionValues(t, client, prepared.keys)
				result, err := store.client.Eval(ctx, projectionFenceLua, prepared.keys, args...).Text()
				if err != nil {
					return err
				}
				if mode == "valid batch" {
					if result != "OK" {
						t.Fatal("the real Lua batch path rejected a complete valid batch")
					}
					for _, item := range items {
						body, err := client.Get(ctx, buildRedisKeyRecordRule(item.Desired.InstanceIP)).Result()
						if err != nil || body != item.Desired.ConfigContent {
							t.Fatal("valid batch did not write all accepted bodies")
						}
					}
					if err := prepared.ApplyMembers(ctx, 1); err != nil {
						return err
					}
					proof, err := prepared.Acknowledge(ctx)
					if err != nil {
						return err
					}
					marked, err := lease.MarkComplete(proof)
					if err != nil {
						return err
					}
					if !marked {
						t.Fatal("valid batch did not complete its SQL operation")
					}
					return nil
				}
				if result == "OK" {
					t.Fatal("invalid final batch item or altered frame was accepted")
				}
				if !reflect.DeepEqual(before, rootProjectionValues(t, client, prepared.keys)) {
					t.Fatal("batch validation failed after mutating an earlier artifact or control")
				}
				return nil
			}); err != nil {
				t.Fatal("owned batch preflight fixture failed", err)
			}
		})
	}
}

func TestProjectionBatchNativeExpiredLeaseSendsNoEval(t *testing.T) {
	client := projectionRedisNative(t)
	for index, mode := range []string{"rolled back original transaction", "callback exited"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			journal, _, db := projectionStoreSQL(t)
			ref := projectionReserve(t, journal, 4, 23201+index, 1)
			items := []configdao.GeneratedPlanItem{projectionRecordItem(ref.PoolID, 1, "127.0.232.1", "groups: []\n"), projectionRecordItem(ref.PoolID, 2, "127.0.232.2", "groups: []\n")}
			receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
			if err != nil {
				t.Fatal(err)
			}
			store, err := NewProjectionStore(client)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			var original interface{ Rollback() error }
			if db.Callback().Query().After("gorm:query").Register("batch_review:original_tx", func(tx *gorm.DB) {
				if current, ok := tx.Statement.ConnPool.(interface{ Rollback() error }); ok {
					original = current
				}
			}) != nil {
				t.Fatal("original SQL transaction observer unavailable")
			}
			ids := []uint64{}
			for _, artifact := range receipt.Artifacts {
				ids = append(ids, artifact.EventID)
			}
			var saved *PreparedProjection
			assertDenied := func(prepared *PreparedProjection) {
				before := rootProjectionValues(t, client, prepared.keys)
				calls, _ := rootProjectionEvalStats(t, ctx, client)
				if err := prepared.ApplyArtifacts(ctx, ids); err == nil {
					t.Fatal("expired SQL authority accepted a new batch")
				}
				after, _ := rootProjectionEvalStats(t, ctx, client)
				if after != calls || !reflect.DeepEqual(before, rootProjectionValues(t, client, prepared.keys)) {
					t.Fatal("expired SQL lease sent an EVAL or changed Redis")
				}
			}
			err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
				prepared, err := store.Prepare(ctx, lease)
				if err != nil {
					return err
				}
				projectionTrack(t, client, prepared)
				saved = prepared
				if mode == "rolled back original transaction" {
					if original == nil || original.Rollback() != nil {
						t.Fatal("did not roll back actual original SQL transaction")
					}
					assertDenied(prepared)
					return errors.New("owned transaction deliberately ended")
				}
				return nil
			})
			if mode == "callback exited" {
				if err != nil {
					t.Fatal(err)
				}
				assertDenied(saved)
			} else if err == nil {
				t.Fatal("rolled-back lease falsely completed callback")
			}
		})
	}
}

func TestProjectionBatchNativeLostReplyRecoversWithoutRetry(t *testing.T) {
	client := projectionRedisNative(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	journal, _, _ := projectionStoreSQL(t)
	ref := projectionReserve(t, journal, 4, 23101, 1)
	items := []configdao.GeneratedPlanItem{
		projectionRecordItem(ref.PoolID, 1, "127.0.231.1", "groups: [] # batch first\n"),
		projectionRecordItem(ref.PoolID, 2, "127.0.231.2", "groups: [] # batch second\n"),
		projectionRecordItem(ref.PoolID, 3, "127.0.231.3", "groups: [] # batch third\n"),
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
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
	ids := []uint64{}
	for _, artifact := range receipt.Artifacts {
		ids = append(ids, artifact.EventID)
	}
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		prepared, err := uncertain.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, prepared)
		drop.Store(true)
		err = prepared.ApplyArtifacts(ctx, ids)
		if !errors.Is(err, ErrProjectionStore) {
			t.Fatal("unknown batch result was accepted", err)
		}
		if dropped.Load() != 1 || dials.Load() != 1 {
			t.Fatal("batch reply loss was not exercised or caused a transparent retry")
		}
		return err
	})
	if err == nil {
		t.Fatal("unknown batch result falsely completed SQL")
	}
	for _, item := range items {
		if body, err := client.Get(ctx, buildRedisKeyRecordRule(item.Desired.InstanceIP)).Result(); err != nil || body != item.Desired.ConfigContent {
			t.Fatal("server did not execute the complete batch before dropping its reply")
		}
	}
	if n, err := client.Exists(ctx, projectionPoolKey(4, ref.PoolID, "ack")).Result(); err != nil || n != 0 {
		t.Fatal("unknown batch result produced a false ACK")
	}
	loaded, err := journal.LoadLastCommittedProjection(ctx, 4, ref.PoolID)
	if err != nil || loaded == nil || loaded.Phase != model.ConfigPublicationCommitted {
		t.Fatal("unknown batch result lost durable pending state", err)
	}
	normal, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer normal.Close()
	if err := journal.WithPoolProjectionLease(ctx, *loaded, func(lease configdao.ProjectionLease) error {
		return NewRecordProjectionRunner(normal).Project(ctx, lease)
	}); err != nil {
		t.Fatal("rebuilt batch projection did not reconcile durable receipt", err)
	}
	projected, err := journal.LoadLastProjectedProjection(ctx, 4, ref.PoolID)
	if err != nil || projected == nil || projected.Reservation != ref {
		t.Fatal("batch recovery did not durably complete original operation", err)
	}
}

func TestProjectionBatchIndependentNative(t *testing.T) {
	runProjectionBatchIndependentNativeReview(t, projectionRedisNative(t))
}

func TestProjectionBatchNativeRawBodyBytesRemainExact(t *testing.T) {
	client := projectionRedisNative(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	journal, _, _ := projectionStoreSQL(t)
	ref := projectionReserve(t, journal, 4, 23301, 1)
	items := []configdao.GeneratedPlanItem{projectionRecordItem(ref.PoolID, 1, "127.0.233.1", "synthetic\x00\xff\xc3\x28\n"), projectionRecordItem(ref.PoolID, 2, "127.0.233.2", "groups: [] # raw body\n")}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, items...))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		prepared, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, prepared)
		ids := []uint64{}
		for _, artifact := range receipt.Artifacts {
			ids = append(ids, artifact.EventID)
		}
		if err := prepared.ApplyArtifacts(ctx, ids); err != nil {
			return err
		}
		for _, item := range items {
			body, err := client.Get(ctx, buildRedisKeyRecordRule(item.Desired.InstanceIP)).Result()
			if err != nil || body != item.Desired.ConfigContent {
				t.Fatal("batch changed raw source bytes through text encoding")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
