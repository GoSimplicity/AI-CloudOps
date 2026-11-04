//go:build integration

package cache

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

type rootProjectionValue struct {
	Present bool
	Dump    string
}

func rootProjectionValues(t *testing.T, client *redis.Client, keys []string) map[string]rootProjectionValue {
	t.Helper()
	values := make(map[string]rootProjectionValue, len(keys))
	for _, key := range keys {
		raw, err := client.Dump(context.Background(), key).Result()
		if err == redis.Nil {
			values[key] = rootProjectionValue{}
			continue
		}
		if err != nil {
			t.Fatal("cannot inspect dedicated projection fixture")
		}
		values[key] = rootProjectionValue{Present: true, Dump: raw}
	}
	return values
}

func TestProjectionStoreReviewQueuedFullPlanCannotRevertHigherPrepare(t *testing.T) {
	client := projectionRedisNative(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	journal, _, _ := projectionStoreSQL(t)
	ctx := context.Background()
	const pool = 961
	firstRef := projectionReserve(t, journal, 4, pool, 0xa1)
	firstItem := projectionRecordItem(pool, 71, "127.0.96.1", "groups: [] # A synthetic\n")
	first, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(firstRef, firstItem))
	if err != nil {
		t.Fatal(err)
	}
	queued := make(map[string]string)
	var oldKeys []string
	var observableKeys []string
	var oldBody string
	err = journal.WithPoolProjectionLease(ctx, first, func(lease configdao.ProjectionLease) error {
		handle, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		oldKeys = append([]string(nil), handle.keys...)
		for _, role := range handle.wire.Roles {
			if role.Role != "pool_fence" && role.Role != "namespace_fence" {
				observableKeys = append(observableKeys, role.Key)
			}
		}
		oldBody = firstItem.Desired.ConfigContent
		for _, mode := range []string{"prepare", "artifact", "members", "ack"} {
			wire := handle.wire
			wire.Mode = mode
			if mode == "artifact" {
				wire.Target = strconv.FormatUint(first.Artifacts[0].EventID, 10)
			}
			if mode == "members" {
				wire.Target = "1"
			}
			raw, err := json.Marshal(wire)
			if err != nil {
				return err
			}
			queued[mode] = string(raw)
		}
		if err := handle.ApplyArtifact(ctx, first.Artifacts[0].EventID); err != nil {
			return err
		}
		if err := handle.ApplyMembers(ctx, 1); err != nil {
			return err
		}
		proof, err := handle.Acknowledge(ctx)
		if err != nil {
			return err
		}
		_, err = lease.MarkComplete(proof)
		return err
	})
	if err != nil {
		t.Fatal("first complete publication failed", err)
	}
	expected := configdao.GeneratedConfigOwnerOf(firstItem.Desired)
	secondRef := projectionReserve(t, journal, 4, pool, 0xb2)
	secondItem := projectionRecordItem(pool, 72, firstItem.Binding.InstanceIP, "groups: [] # B synthetic\n")
	secondItem.Expected = &expected
	second, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(secondRef, secondItem))
	if err != nil {
		t.Fatal(err)
	}
	err = journal.WithPoolProjectionLease(ctx, second, func(lease configdao.ProjectionLease) error {
		beforePrepare := rootProjectionValues(t, client, observableKeys)
		handle, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		if !reflect.DeepEqual(rootProjectionValues(t, client, observableKeys), beforePrepare) {
			t.Fatal("prepare changed the raw body, members, intent or old ACK")
		}
		allKeys := append(append([]string(nil), oldKeys...), handle.keys...)
		baseline := rootProjectionValues(t, client, allKeys)
		for _, mode := range []string{"prepare", "artifact", "members", "ack"} {
			// The original command enters Redis only now, after the next SQL commit
			// and higher fence prepare. No expired Go lease is used to short-circuit it.
			result, err := client.Eval(ctx, projectionFenceLua, oldKeys, queued[mode], oldBody).Text()
			if err != nil || result != "STALE" {
				t.Fatalf("late %s did not reject lower generation: %v", mode, err)
			}
			if !reflect.DeepEqual(rootProjectionValues(t, client, allKeys), baseline) {
				t.Fatalf("late %s changed an artifact, member, ACK or fence", mode)
			}
		}
		if client.Get(ctx, buildRedisKeyRecordRule(firstItem.Binding.InstanceIP)).Val() != firstItem.Desired.ConfigContent {
			t.Fatal("prepare or rejected commands changed the previous raw body")
		}
		if err := handle.ApplyArtifact(ctx, second.Artifacts[0].EventID); err != nil {
			return err
		}
		if err := handle.ApplyMembers(ctx, 1); err != nil {
			return err
		}
		proof, err := handle.Acknowledge(ctx)
		if err != nil {
			return err
		}
		_, err = lease.MarkComplete(proof)
		return err
	})
	if err != nil {
		t.Fatal("higher generation publication failed", err)
	}
	if client.Get(ctx, buildRedisKeyRecordRule(firstItem.Binding.InstanceIP)).Val() != secondItem.Desired.ConfigContent {
		t.Fatal("higher generation raw bytes were not preserved")
	}
	if client.Get(ctx, recordPublicationKey(pool)).Val() != hex.EncodeToString(second.SourceHash[:]) {
		t.Fatal("higher generation legacy SHA was not preserved")
	}
	loaded, err := journal.LoadLastProjectedProjection(ctx, 4, pool)
	if err != nil || loaded == nil || loaded.Reservation != secondRef {
		t.Fatal("higher generation SQL ACK was lost", err)
	}
}
