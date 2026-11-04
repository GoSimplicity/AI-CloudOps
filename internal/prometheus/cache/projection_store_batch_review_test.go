package cache

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// This hook captures Go commands without any Redis I/O. It does not execute Lua
// or prove fencing/TYPE/atomicity. Native adversarial cases below require Root's
// explicitly owned client and are never called by these default unit tests.
type batchReviewCommand struct {
	wire      projectionWire
	selection []string
	bodies    int
}
type batchReviewHook struct {
	calls    []batchReviewCommand
	fail     error
	reply    string
	parseErr error
}

func (*batchReviewHook) DialHook(redis.DialHook) redis.DialHook {
	return func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("review hook forbids network")
	}
}
func (h *batchReviewHook) ProcessHook(redis.ProcessHook) redis.ProcessHook {
	return func(_ context.Context, cmd redis.Cmder) error {
		if cmd.Name() != "eval" {
			return errors.New("unexpected non-EVAL command")
		}
		args := cmd.Args()
		n, ok := args[2].(int)
		if !ok || len(args) < n+5 {
			h.parseErr = errors.New("unexpected EVAL envelope")
			return h.parseErr
		}
		raw, ok := args[n+3].(string)
		if !ok {
			h.parseErr = errors.New("non-string frame")
			return h.parseErr
		}
		entry := batchReviewCommand{}
		if json.Unmarshal([]byte(raw), &entry.wire) != nil {
			h.parseErr = errors.New("invalid frame")
			return h.parseErr
		}
		if entry.wire.Mode == "artifact_batch" {
			selected, ok := args[n+4].(string)
			if !ok || json.Unmarshal([]byte(selected), &entry.selection) != nil {
				h.parseErr = errors.New("invalid selection")
				return h.parseErr
			}
			entry.bodies = len(args) - (n + 5)
		} else {
			entry.bodies = 1
		}
		h.calls = append(h.calls, entry)
		if h.fail != nil {
			return h.fail
		}
		result, ok := cmd.(*redis.Cmd)
		if !ok {
			return errors.New("unexpected EVAL result type")
		}
		reply := h.reply
		if reply == "" {
			reply = "OK"
		}
		result.SetVal(reply)
		return nil
	}
}
func (*batchReviewHook) ProcessPipelineHook(redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(context.Context, []redis.Cmder) error { return errors.New("review does not permit pipelines") }
}
func batchReviewItems(n int, body string) []configdao.GeneratedPlanItem {
	result := []configdao.GeneratedPlanItem{}
	for i := 0; i < n; i++ {
		result = append(result, projectionRecordItem(7007, uint64(i+1), fmt.Sprintf("198.19.%d.%d", i/250, i%250+1), body))
	}
	return result
}
func batchReviewPrepared(t *testing.T, n int, body string, callback func(*PreparedProjection, []uint64, *batchReviewHook)) {
	t.Helper()
	journal, _, _ := projectionStoreSQL(t)
	ref := projectionReserve(t, journal, 4, 7007, 111)
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, batchReviewItems(n, body)...))
	if err != nil {
		t.Fatal(err)
	}
	base := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = base.Close() })
	store, err := NewProjectionStore(base)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	hook := &batchReviewHook{}
	store.client.AddHook(hook)
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
		prepared, err := store.Prepare(context.Background(), lease)
		if err != nil {
			return err
		}
		ids := []uint64{}
		for _, a := range receipt.Artifacts {
			ids = append(ids, a.EventID)
		}
		hook.calls = nil
		callback(prepared, ids, hook)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if hook.parseErr != nil {
		t.Fatal(hook.parseErr)
	}
	current, err := journal.LoadLastCommittedProjection(context.Background(), 4, 7007)
	if err != nil || current == nil || current.Phase != model.ConfigPublicationCommitted {
		t.Fatal("fake transport was mistaken for durable external completion")
	}
}
func TestBatchReviewInvalidLastSelectionFailsBeforeAnyCommand(t *testing.T) {
	for _, mode := range []string{"unknown ID", "duplicate ID", "missing material", "material without issued ID", "changed material body", "last artifact role"} {
		t.Run(mode, func(t *testing.T) {
			batchReviewPrepared(t, 3, "groups: []\n", func(p *PreparedProjection, ids []uint64, h *batchReviewHook) {
				selected := append([]uint64(nil), ids...)
				switch mode {
				case "unknown ID":
					selected[len(selected)-1] += 1000000
				case "duplicate ID":
					selected[len(selected)-1] = selected[0]
				case "missing material":
					delete(p.materials, strconv.FormatUint(selected[len(selected)-1], 10))
				case "material without issued ID":
					id := selected[len(selected)-1] + 1000000
					p.materials[strconv.FormatUint(id, 10)] = []byte("unissued material")
					selected[len(selected)-1] = id
				case "changed material body":
					key := strconv.FormatUint(selected[len(selected)-1], 10)
					p.materials[key] = append(append([]byte(nil), p.materials[key]...), []byte("altered")...)
				case "last artifact role":
					artifact := p.wire.Artifacts[len(p.wire.Artifacts)-1]
					p.wire.Roles[artifact.Raw-1].Role = "members"
				}
				if err := p.ApplyArtifacts(context.Background(), selected); err == nil || len(h.calls) != 0 {
					t.Fatal("invalid last entry sent an earlier valid artifact")
				}
			})
		})
	}
}
func TestBatchReviewTransportOrFenceErrorNeverFallsBack(t *testing.T) {
	for _, mode := range []string{"unknown reply", "STALE", "TYPE"} {
		t.Run(mode, func(t *testing.T) {
			batchReviewPrepared(t, 32, "groups: []\n", func(p *PreparedProjection, ids []uint64, h *batchReviewHook) {
				if mode == "unknown reply" {
					h.fail = io.ErrUnexpectedEOF
				} else {
					h.reply = mode
				}
				err := p.ApplyArtifacts(context.Background(), ids)
				if err == nil || len(h.calls) != 1 || h.calls[0].wire.Mode != "artifact_batch" {
					t.Fatal("EVAL failure retried, fell back or sent another batch")
				}
				if mode == "unknown reply" && !errors.Is(err, ErrProjectionStore) {
					t.Fatal("unknown result lost its Store error classification")
				}
				if mode != "unknown reply" && !errors.Is(err, ErrProjectionFence) {
					t.Fatal("negative Lua response lost its fence classification")
				}
			})
		})
	}
}
func TestBatchReviewBoundedCountAndFullRequestBudget(t *testing.T) {
	t.Run("sixteen then original single", func(t *testing.T) {
		batchReviewPrepared(t, 17, "groups: []\n", func(p *PreparedProjection, ids []uint64, h *batchReviewHook) {
			if err := p.ApplyArtifacts(context.Background(), ids); err != nil {
				t.Fatal(err)
			}
			if len(h.calls) != 2 || h.calls[0].wire.Mode != "artifact_batch" || len(h.calls[0].selection) != 16 || h.calls[0].bodies != 16 || h.calls[1].wire.Mode != "artifact" {
				t.Fatal("bounded groups changed count or the existing single-item path")
			}
			for i, id := range ids[:16] {
				if h.calls[0].selection[i] != strconv.FormatUint(id, 10) {
					t.Fatal("batch reordered selection or lost exact decimal IDs")
				}
			}
		})
	})
	t.Run("body budget fits but whole EVAL cannot", func(t *testing.T) {
		const limit = 1 << 20
		bodySize := (limit - len(projectionFenceLua)/2) / 2
		if 2*bodySize >= limit || 2*bodySize+len(projectionFenceLua) <= limit {
			t.Fatal("RESP lower-bound fixture is invalid")
		}
		// The two bodies alone fit. Script bytes alone prove the whole request does
		// not, without mirroring the production RESP-size arithmetic.
		batchReviewPrepared(t, 2, strings.Repeat("x", bodySize), func(p *PreparedProjection, ids []uint64, h *batchReviewHook) {
			if err := p.ApplyArtifacts(context.Background(), ids); err != nil {
				t.Fatal(err)
			}
			if len(h.calls) != 2 || h.calls[0].wire.Mode != "artifact" || h.calls[1].wire.Mode != "artifact" {
				t.Fatal("body-only admission ignored full EVAL/KEYS/ARGV overhead")
			}
		})
	})
}
func TestBatchReviewEmptySelectionStillRequiresLiveOriginalLease(t *testing.T) {
	var escaped *PreparedProjection
	var hook *batchReviewHook
	batchReviewPrepared(t, 0, "", func(p *PreparedProjection, _ []uint64, h *batchReviewHook) {
		escaped, hook = p, h
		if err := p.ApplyArtifacts(context.Background(), nil); err != nil || len(h.calls) != 0 {
			t.Fatal("valid empty artifact list sent work or failed")
		}
	})
	if err := escaped.ApplyArtifacts(context.Background(), nil); err == nil || len(hook.calls) != 0 {
		t.Fatal("empty list bypassed an expired SQL lease")
	}
}

func TestBatchReviewSelectionRetainsUnselectedNamespaceVector(t *testing.T) {
	batchReviewPrepared(t, 3, "groups: []\n", func(p *PreparedProjection, ids []uint64, h *batchReviewHook) {
		if err := p.ApplyArtifacts(context.Background(), ids[:2]); err != nil {
			t.Fatal(err)
		}
		if len(h.calls) != 1 || h.calls[0].wire.Mode != "artifact_batch" || len(h.calls[0].wire.Artifacts) != 3 || len(h.calls[0].wire.Fences) != 4 {
			t.Fatal("selected batch discarded an unselected namespace from the full behavior frame")
		}
		found := false
		for _, a := range h.calls[0].wire.Artifacts {
			found = found || a.ID == strconv.FormatUint(ids[2], 10)
		}
		if !found {
			t.Fatal("unselected namespace lost its original issued event")
		}
	})
}

type batchReviewNativeFixture struct {
	journal  configdao.ProjectionPlanJournalDAO
	ordinary configdao.MonitorConfigDAO
	db       *gorm.DB
	store    *ProjectionStore
	client   *redis.Client
	receipt  configdao.ProjectionReceipt
	items    []configdao.GeneratedPlanItem
}

func newBatchReviewNativeFixture(t *testing.T, client *redis.Client, n int, body string, family int8) *batchReviewNativeFixture {
	t.Helper()
	journal, ordinary, db := projectionStoreSQL(t)
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal("native fixture identity unavailable")
	}
	pool := 1000000 + int(binary.BigEndian.Uint32(random[:])%1000000)
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	ref, err := journal.ReservePoolOperation(context.Background(), configdao.PoolOperationRequest{Family: family, PoolID: pool, Nonce: nonce})
	if err != nil {
		t.Fatal(err)
	}
	items := []configdao.GeneratedPlanItem{}
	for i := 0; i < n; i++ {
		ip := fmt.Sprintf("198.19.250.%d", i+1)
		item := projectionRecordItem(pool, uint64(i+1), ip, body)
		if family == 5 {
			group := 9
			item.Desired.ConfigType = 5
			item.Desired.InstanceIP = ip + "_9"
			item.Desired.Name = "webhook_batch_review_9_" + ip
			item.Binding = &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: ip, GroupID: &group}
		}
		items = append(items, item)
	}
	request := projectionFullRequest(ref, items...)
	if family == 5 {
		sources := []uint64{}
		for _, item := range items {
			sources = append(sources, item.SourceID)
		}
		request.Controls = []configdao.ProjectionControl{{ControlID: 9, Kind: configdao.ProjectionACKWebhookSubset, Ack: &configdao.ProjectionAckControl{ArtifactSourceIDs: sources, RequiredControlIDs: []uint64{}}}}
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewProjectionStore(client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return &batchReviewNativeFixture{journal: journal, ordinary: ordinary, db: db, store: store, client: client, receipt: receipt, items: items}
}
func (f *batchReviewNativeFixture) track(t *testing.T, p *PreparedProjection) {
	t.Helper()
	keys := append([]string(nil), p.keys...)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if f.client.Del(ctx, keys...).Err() != nil {
			t.Error("independent batch fixture cleanup failed")
		}
	})
}
func batchReviewNativeDump(t *testing.T, c *redis.Client, keys []string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, key := range keys {
		raw, err := c.Dump(context.Background(), key).Result()
		if err == redis.Nil {
			result[key] = "absent"
			continue
		}
		if err != nil {
			t.Fatal("native batch DUMP failed")
		}
		result[key] = "present:" + raw
	}
	return result
}
func batchReviewNativeWire(t *testing.T, p *PreparedProjection) (projectionWire, []string, []string) {
	t.Helper()
	raw, err := json.Marshal(p.wire)
	if err != nil {
		t.Fatal(err)
	}
	var wire projectionWire
	if json.Unmarshal(raw, &wire) != nil {
		t.Fatal("cannot clone native frame")
	}
	wire.Mode = "artifact_batch"
	wire.Target = ""
	selected := []string{}
	bodies := []string{}
	for _, a := range wire.Artifacts {
		selected = append(selected, a.ID)
		bodies = append(bodies, string(p.materials[a.ID]))
	}
	return wire, selected, bodies
}
func batchReviewNativeReject(t *testing.T, c *redis.Client, p *PreparedProjection, wire projectionWire, selection string, bodies []string, keys []string) {
	t.Helper()
	if keys == nil {
		keys = p.keys
	}
	before := batchReviewNativeDump(t, c, keys)
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	args := []interface{}{string(raw), selection}
	for _, body := range bodies {
		args = append(args, body)
	}
	result, err := c.Eval(context.Background(), projectionFenceLua, keys, args...).Text()
	if err != nil || result == "OK" {
		t.Fatal("adversarial batch was not rejected by a definite Lua response")
	}
	if after := batchReviewNativeDump(t, c, keys); !reflect.DeepEqual(before, after) {
		t.Fatal("rejected batch mutated at least one complete Redis value")
	}
}

// Root invokes this only from its owned integration wrapper. There is no
// default-skip native Test and no connection discovery or Docker in this file.
func runProjectionBatchIndependentNativeReview(t *testing.T, client *redis.Client) {
	for _, mode := range []string{"selection object", "missing body", "extra body", "seventeen items", "body budget", "unselected raw TYPE", "same-version legacy SHA", "same-version ACK body", "last role"} {
		t.Run(mode, func(t *testing.T) {
			count, body := 3, "groups: []\n"
			if mode == "seventeen items" {
				count = 17
			}
			if mode == "body budget" {
				count = 2
				body = strings.Repeat("x", 600000)
			}
			f := newBatchReviewNativeFixture(t, client, count, body, 4)
			err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
				p, err := f.store.Prepare(context.Background(), lease)
				if err != nil {
					return err
				}
				f.track(t, p)
				wire, selected, bodies := batchReviewNativeWire(t, p)
				if mode == "unselected raw TYPE" {
					selected = selected[:2]
					bodies = bodies[:2]
					last := wire.Artifacts[2]
					if client.SAdd(context.Background(), p.keys[last.Raw-1], "deliberately bad type").Err() != nil {
						return errors.New("native wrong TYPE setup failed")
					}
				}
				selection, err := json.Marshal(selected)
				if err != nil {
					return err
				}
				switch mode {
				case "selection object":
					selection = []byte(`{"1":"` + selected[0] + `"}`)
				case "missing body":
					bodies = bodies[:len(bodies)-1]
				case "extra body":
					bodies = append(bodies, "")
				case "same-version legacy SHA":
					wire.LegacySHA = strings.Repeat("a", 64)
				case "same-version ACK body":
					wire.AckReceipt += " "
				case "last role":
					wire.Roles[wire.Artifacts[len(wire.Artifacts)-1].Raw-1].Role = "members"
				}
				batchReviewNativeReject(t, client, p, wire, string(selection), bodies, nil)
				return nil
			})
			if err != nil {
				t.Fatal("independent native setup failed", err)
			}
		})
	}
	t.Run("unselected namespace actually taken by another SQL pool", func(t *testing.T) {
		f := newBatchReviewNativeFixture(t, client, 3, "groups: [] # old owner\n", 4)
		var late *PreparedProjection
		err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
			var err error
			late, err = f.store.Prepare(context.Background(), lease)
			if err == nil {
				f.track(t, late)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		if f.ordinary.DeleteMonitorConfig(context.Background(), f.items[2].Desired.ID) != nil {
			t.Fatal("old namespace owner did not actually retire in SQL")
		}
		pool := f.receipt.Reservation.PoolID + 1
		ref := projectionReserve(t, f.journal, 4, pool, 212)
		item := projectionRecordItem(pool, 101, f.items[2].Binding.InstanceIP, "groups: [] # new owner\n")
		newer, err := f.journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, item))
		if err != nil {
			t.Fatal(err)
		}
		allKeys := append([]string(nil), late.keys...)
		err = f.journal.WithPoolProjectionLease(context.Background(), newer, func(lease configdao.ProjectionLease) error {
			p, err := f.store.Prepare(context.Background(), lease)
			if err != nil {
				return err
			}
			f.track(t, p)
			allKeys = append(allKeys, p.keys...)
			if err := p.ApplyArtifact(context.Background(), newer.Artifacts[0].EventID); err != nil {
				return err
			}
			for _, control := range newer.Controls {
				if control.Members != nil {
					if err := p.ApplyMembers(context.Background(), control.ControlID); err != nil {
						return err
					}
				}
			}
			proof, err := p.Acknowledge(context.Background())
			if err != nil {
				return err
			}
			marked, err := lease.MarkComplete(proof)
			if err == nil && !marked {
				return errors.New("replacement was not actually marked")
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		wire, selected, bodies := batchReviewNativeWire(t, late)
		selection, _ := json.Marshal(selected[:2])
		// Send the old already-issued wire directly, as a delayed TCP request would.
		// The changed namespace is deliberately outside the selected batch.
		before := batchReviewNativeDump(t, client, allKeys)
		batchReviewNativeReject(t, client, late, wire, string(selection), bodies[:2], nil)
		if !reflect.DeepEqual(before, batchReviewNativeDump(t, client, allKeys)) {
			t.Fatal("late batch changed the replacement pool")
		}
	})
	t.Run("ordinary cannot turn invalidation into render", func(t *testing.T) {
		f := newBatchReviewNativeFixture(t, client, 1, "groups: []\n", 4)
		row := *f.items[0].Desired
		row.ConfigContent = "groups: [] # legitimate ordinary update\n"
		if f.ordinary.UpdateMonitorConfig(context.Background(), &row) != nil {
			t.Fatal("ordinary SQL update failed")
		}
		var event model.MonitorConfigArtifactEvent
		if f.db.Order("event_id DESC").First(&event).Error != nil {
			t.Fatal("ordinary event unavailable")
		}
		artifact := configdao.ArtifactReceipt{EventID: event.EventID, ConfigType: event.ConfigType, InstanceKey: string(event.InstanceKey), Action: event.Action, ProjectionPolicy: event.ProjectionPolicy, ConfigRowID: event.ConfigRowID, OldOwnerDigest: event.OldOwnerDigest, NewOwnerDigest: event.NewOwnerDigest, BodyDigest: event.BodyDigest, BindingEventID: event.BindingEventID}
		err := f.journal.WithArtifactProjectionLease(context.Background(), artifact, func(lease configdao.ProjectionLease) error {
			p, err := f.store.Prepare(context.Background(), lease)
			if err != nil {
				return err
			}
			f.track(t, p)
			wire, selected, bodies := batchReviewNativeWire(t, p)
			wire.Artifacts[0].Action = model.ConfigProjectionRender
			selection, _ := json.Marshal(selected)
			batchReviewNativeReject(t, client, p, wire, string(selection), bodies, nil)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("subset cannot acquire a main artifact role", func(t *testing.T) {
		f := newBatchReviewNativeFixture(t, client, 2, "subset synthetic bytes", 5)
		err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
			p, err := f.store.Prepare(context.Background(), lease)
			if err != nil {
				return err
			}
			f.track(t, p)
			wire, selected, bodies := batchReviewNativeWire(t, p)
			wire.Roles[wire.Artifacts[len(wire.Artifacts)-1].Raw-1].Kind = "2"
			selection, _ := json.Marshal(selected)
			batchReviewNativeReject(t, client, p, wire, string(selection), bodies, nil)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
