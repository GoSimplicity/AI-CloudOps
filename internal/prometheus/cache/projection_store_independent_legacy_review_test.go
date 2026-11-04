//go:build integration

package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

func TestIndependentProjectionStoreLegacySHAIsEqualPayload(t *testing.T) {
	client := projectionRedisNative(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	journal, _, _ := projectionStoreSQL(t)
	ctx := context.Background()
	ref := projectionReserve(t, journal, 4, 973, 0xe1)
	item := projectionRecordItem(973, 71, "127.0.97.3", "groups: [] # independent synthetic\n")
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
	if err != nil {
		t.Fatal(err)
	}
	// A complete, successful public projection proves this is not merely a
	// missing-artifact/member acknowledgement rejection.
	projectionNativeApply(t, client, store, journal, receipt)
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		handle, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		if _, err := handle.Acknowledge(ctx); err != nil {
			t.Fatal("legal ACK retry rejected", err)
		}
		before := rootProjectionValues(t, client, handle.keys)
		wire := handle.wire
		wire.Mode = "ack"
		wire.LegacySHA = strings.Repeat("f", 64)
		if wire.LegacySHA == handle.wire.LegacySHA {
			wire.LegacySHA = strings.Repeat("0", 64)
		}
		raw, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		result, err := store.client.Eval(ctx, projectionFenceLua, handle.keys, string(raw), "").Text()
		if err != nil {
			t.Fatal("native script failed; test requires a protocol rejection", err)
		}
		after := rootProjectionValues(t, client, handle.keys)
		if result == "OK" || !reflect.DeepEqual(before, after) {
			t.Fatalf("same generation/nonce/receipt accepted changed legacy SHA or wrote state: result=%s", result)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIndependentProjectionStoreEqualPlanCannotRedirectArtifactBinding(t *testing.T) {
	client := projectionRedisNative(t)
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	journal, _, _ := projectionStoreSQL(t)
	ctx := context.Background()
	ref := projectionReserve(t, journal, 4, 974, 0xe2)
	item := projectionRecordItem(974, 72, "127.0.97.4", "groups: [] # independent original binding\n")
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
	if err != nil {
		t.Fatal(err)
	}
	projectionNativeApply(t, client, store, journal, receipt)
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		handle, err := store.Prepare(ctx, lease)
		if err != nil {
			return err
		}
		projectionTrack(t, client, handle)
		if _, err := handle.Acknowledge(ctx); err != nil {
			t.Fatal("legal ACK retry rejected", err)
		}
		encoded, err := json.Marshal(handle.wire)
		if err != nil {
			t.Fatal(err)
		}
		var wire projectionWire
		if json.Unmarshal(encoded, &wire) != nil {
			t.Fatal("wire clone failed")
		}
		keys := append([]string(nil), handle.keys...)
		artifact := wire.Artifacts[0]
		const redirectedIP = "127.0.97.5"
		redirectedRaw := buildRedisKeyRecordRule(redirectedIP)
		redirectedNamespace := projectionNamespaceKey(4, redirectedIP)
		if client.Exists(ctx, redirectedRaw, redirectedNamespace).Val() != 0 {
			t.Fatal("redirect namespace must be unseen")
		}
		keys[artifact.Raw-1] = redirectedRaw
		keys[artifact.Fence-1] = redirectedNamespace
		wire.Roles[artifact.Raw-1].Key = redirectedRaw
		wire.Roles[artifact.Raw-1].IP = redirectedIP
		wire.Roles[artifact.Fence-1].Key = redirectedNamespace
		wire.Roles[artifact.Fence-1].IP = redirectedIP
		t.Cleanup(func() { client.Del(context.Background(), redirectedRaw, redirectedNamespace) })
		allKeys := append(append([]string(nil), handle.keys...), redirectedRaw, redirectedNamespace)
		before := rootProjectionValues(t, client, allKeys)
		// Preserve the issued pool generation/nonce/signature, ACK receipt and all
		// controls. Only redirect one artifact/raw+namespace role pair to a new IP.
		wire.Mode = "prepare"
		command, err := json.Marshal(wire)
		if err != nil {
			t.Fatal(err)
		}
		prepared, err := store.client.Eval(ctx, projectionFenceLua, keys, string(command), "").Text()
		if err != nil {
			t.Fatal("native script error instead of protocol rejection", err)
		}
		applied := "not attempted"
		if prepared == "OK" {
			wire.Mode = "artifact"
			wire.Target = artifact.ID
			command, err = json.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			applied, err = store.client.Eval(ctx, projectionFenceLua, keys, string(command), string(handle.materials[artifact.ID])).Text()
			if err != nil {
				t.Fatal("redirected artifact script failed", err)
			}
		}
		after := rootProjectionValues(t, client, allKeys)
		if prepared == "OK" || !reflect.DeepEqual(before, after) {
			t.Fatalf("equal plan admitted changed artifact binding or wrote state: prepare=%s artifact=%s", prepared, applied)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
