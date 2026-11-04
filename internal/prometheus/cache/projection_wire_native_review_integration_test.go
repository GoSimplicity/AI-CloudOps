//go:build integration

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"github.com/redis/go-redis/v9"
)

func wireReviewEncode(t *testing.T, p *PreparedProjection, mode, target string, compact bool) []byte {
	t.Helper()
	wire := p.wire
	wire.Mode = mode
	wire.Target = target
	var raw []byte
	var err error
	if compact {
		raw, err = marshalProjectionWire(wire, p.keys)
	} else {
		raw, err = json.Marshal(wire)
	}
	if err != nil {
		t.Fatal("wire review encoding failed")
	}
	return raw
}
func wireReviewEval(t *testing.T, c *redis.Client, keys []string, raw []byte, tail ...string) string {
	t.Helper()
	args := []interface{}{string(raw)}
	for _, v := range tail {
		args = append(args, v)
	}
	result, err := c.Eval(context.Background(), projectionFenceLua, keys, args...).Text()
	if err != nil {
		t.Fatal("wire review requires a definite Lua reply")
	}
	return result
}
func wireReviewOK(t *testing.T, c *redis.Client, p *PreparedProjection, mode, target string, compact bool, tail ...string) {
	t.Helper()
	if wireReviewEval(t, c, p.keys, wireReviewEncode(t, p, mode, target, compact), tail...) != "OK" {
		t.Fatal("valid cross-codec operation rejected")
	}
}
func wireReviewEmptyObjects(t *testing.T, raw []byte) []byte {
	t.Helper()
	var top []json.RawMessage
	if json.Unmarshal(raw, &top) != nil || len(top) != 15 {
		t.Fatal("compact empty fixture shape changed")
	}
	var artifacts []json.RawMessage
	if json.Unmarshal(top[7], &artifacts) != nil || len(artifacts) != 0 {
		t.Fatal("fixture artifacts are not empty")
	}
	top[7] = json.RawMessage(`{}`)
	var controls [][]json.RawMessage
	if json.Unmarshal(top[8], &controls) != nil {
		t.Fatal("control fixture decode failed")
	}
	for _, row := range controls {
		for _, index := range []int{3, 4, 5} {
			var list []string
			if json.Unmarshal(row[index], &list) != nil || len(list) != 0 {
				t.Fatal("fixture list is not empty")
			}
			row[index] = json.RawMessage(`{}`)
		}
	}
	top[8], _ = json.Marshal(controls)
	result, err := json.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func wireReviewPositiveFlow(t *testing.T, c *redis.Client, oldPrepare, oldActions, emptyObjects bool) {
	t.Helper()
	n := 2
	if emptyObjects {
		n = 0
	}
	f := newBatchReviewNativeFixture(t, c, n, "groups: [] # wire bytes\n", 4)
	err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		p, err := compileProjection(f.store, lease, snapshot)
		if err != nil {
			return err
		}
		f.track(t, p)
		prepare := wireReviewEncode(t, p, "prepare", "", !oldPrepare)
		if wireReviewEval(t, c, p.keys, prepare, "") != "OK" {
			return errors.New("initial codec prepare rejected")
		}
		before := batchReviewNativeDump(t, c, p.keys)
		alternate := wireReviewEncode(t, p, "prepare", "", oldPrepare)
		if emptyObjects {
			alternate = wireReviewEmptyObjects(t, alternate)
		}
		if wireReviewEval(t, c, p.keys, alternate, "") != "OK" || !reflect.DeepEqual(before, batchReviewNativeDump(t, c, p.keys)) {
			return errors.New("same canonical prepare changed state across codecs")
		}
		if n > 0 {
			ids, bodies := []string{}, []string{}
			for _, a := range p.wire.Artifacts {
				ids = append(ids, a.ID)
				bodies = append(bodies, string(p.materials[a.ID]))
			}
			selection, _ := json.Marshal(ids)
			tail := append([]string{string(selection)}, bodies...)
			wireReviewOK(t, c, p, "artifact_batch", "", !oldActions, tail...)
		}
		for _, ctrl := range p.wire.Controls {
			if ctrl.Kind == string(configdao.ProjectionMembersFull) {
				raw := wireReviewEncode(t, p, "members", ctrl.ID, !oldActions)
				if emptyObjects {
					raw = wireReviewEmptyObjects(t, raw)
				}
				if wireReviewEval(t, c, p.keys, raw, "") != "OK" {
					return errors.New("cross-codec members rejected")
				}
			}
		}
		wireReviewOK(t, c, p, "ack", "", !oldActions, "")
		if got, err := c.Get(context.Background(), projectionPoolKey(4, f.receipt.Reservation.PoolID, "ack")).Result(); err != nil || got != p.wire.AckReceipt {
			return errors.New("canonical ACK bytes changed")
		}
		if got, err := c.Get(context.Background(), recordPublicationKey(f.receipt.Reservation.PoolID)).Result(); err != nil || got != p.wire.LegacySHA {
			return errors.New("legacy SHA changed")
		}
		for _, item := range f.items {
			if body, err := c.Get(context.Background(), buildRedisKeyRecordRule(item.Binding.InstanceIP)).Result(); err != nil || body != item.Desired.ConfigContent {
				return errors.New("raw artifact bytes changed")
			}
		}
		stable := batchReviewNativeDump(t, c, p.keys)
		wireReviewOK(t, c, p, "ack", "", oldActions, "")
		if !reflect.DeepEqual(stable, batchReviewNativeDump(t, c, p.keys)) {
			return errors.New("same canonical ACK was not idempotent across codecs")
		}
		// Mark only the public Store result: Acknowledge normalizes empty
		// proof vectors for SQL, while p.proof is private compilation state.
		proof, err := p.Acknowledge(context.Background())
		if err != nil {
			return err
		}
		marked, err := lease.MarkComplete(proof)
		if err == nil && !marked {
			return errors.New("SQL completion marker rejected actual ACK")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func runProjectionWireIndependentNativeReview(t *testing.T, c *redis.Client) {
	t.Run("old prepare compact actions", func(t *testing.T) { wireReviewPositiveFlow(t, c, true, false, false) })
	t.Run("compact prepare old actions", func(t *testing.T) { wireReviewPositiveFlow(t, c, false, true, false) })
	t.Run("empty collection objects equal arrays", func(t *testing.T) { wireReviewPositiveFlow(t, c, true, false, true) })
	for _, mode := range []string{"last artifact extra", "last artifact missing", "last role null", "last role number", "false empty group", "unknown codec", "top extra", "top missing", "missing control bool", "null control bool", "tuple object"} {
		t.Run(mode, func(t *testing.T) {
			f := newBatchReviewNativeFixture(t, c, 2, "groups: []\n", 4)
			err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
				p, err := f.store.Prepare(context.Background(), lease)
				if err != nil {
					return err
				}
				f.track(t, p)
				raw := wireReviewEncode(t, p, "artifact_batch", "", true)
				var top []json.RawMessage
				if json.Unmarshal(raw, &top) != nil || len(top) != 15 {
					return errors.New("pw2 shape unavailable")
				}
				var artifacts, roles, controls [][]json.RawMessage
				if json.Unmarshal(top[7], &artifacts) != nil || json.Unmarshal(top[5], &roles) != nil || json.Unmarshal(top[8], &controls) != nil {
					return errors.New("tuple fixtures unavailable")
				}
				switch mode {
				case "last artifact extra":
					artifacts[len(artifacts)-1] = append(artifacts[len(artifacts)-1], json.RawMessage(`"ignored"`))
					top[7], _ = json.Marshal(artifacts)
				case "last artifact missing":
					artifacts[len(artifacts)-1] = artifacts[len(artifacts)-1][:5]
					top[7], _ = json.Marshal(artifacts)
				case "last role null":
					roles[len(roles)-1][1] = json.RawMessage(`null`)
					top[5], _ = json.Marshal(roles)
				case "last role number":
					roles[len(roles)-1][1] = json.RawMessage(`7`)
					top[5], _ = json.Marshal(roles)
				case "false empty group":
					roles[len(roles)-1][4] = json.RawMessage(`false`)
					top[5], _ = json.Marshal(roles)
				case "unknown codec":
					top[0] = json.RawMessage(`"pw1"`)
				case "top extra":
					top = append(top, json.RawMessage(`0`))
				case "top missing":
					top = top[:14]
				case "missing control bool":
					controls[len(controls)-1] = controls[len(controls)-1][:7]
					top[8], _ = json.Marshal(controls)
				case "null control bool":
					controls[len(controls)-1][7] = json.RawMessage(`null`)
					top[8], _ = json.Marshal(controls)
				case "tuple object":
					rows := []json.RawMessage{}
					for _, row := range artifacts {
						b, _ := json.Marshal(row)
						rows = append(rows, b)
					}
					rows[len(rows)-1] = json.RawMessage(`{"1":"not an array"}`)
					top[7], _ = json.Marshal(rows)
				}
				raw, err = json.Marshal(top)
				if err != nil {
					return err
				}
				ids, bodies := []string{}, []string{}
				for _, a := range p.wire.Artifacts {
					ids = append(ids, a.ID)
					bodies = append(bodies, string(p.materials[a.ID]))
				}
				selection, _ := json.Marshal(ids)
				tail := append([]string{string(selection)}, bodies...)
				before := batchReviewNativeDump(t, c, p.keys)
				if result := wireReviewEval(t, c, p.keys, raw, tail...); result == "OK" {
					t.Error("malformed compact input was accepted")
				}
				if !reflect.DeepEqual(before, batchReviewNativeDump(t, c, p.keys)) {
					t.Error("malformed compact input wrote before rejection")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, mode := range []string{"redirect last KEY", "unselected raw TYPE", "unselected newer namespace"} {
		t.Run(mode, func(t *testing.T) {
			f := newBatchReviewNativeFixture(t, c, 3, "groups: []\n", 4)
			err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
				p, err := f.store.Prepare(context.Background(), lease)
				if err != nil {
					return err
				}
				f.track(t, p)
				raw := wireReviewEncode(t, p, "artifact_batch", "", true)
				last := p.wire.Artifacts[2]
				keys := append([]string(nil), p.keys...)
				all := append([]string(nil), keys...)
				want := ""
				switch mode {
				case "redirect last KEY":
					keys[last.Raw-1] += ":redirect"
					extra := keys[last.Raw-1]
					all = append(all, extra)
					t.Cleanup(func() { c.Del(context.Background(), extra) })
					want = "ROLE"
				case "unselected raw TYPE":
					if c.SAdd(context.Background(), keys[last.Raw-1], "wrong TYPE fixture").Err() != nil {
						return errors.New("TYPE setup failed")
					}
					want = "TYPE"
				case "unselected newer namespace":
					event, err := strconv.ParseUint(last.ID, 10, 64)
					if err != nil {
						return err
					}
					if c.HSet(context.Background(), keys[last.Fence-1], "version", projectionVersion(event+1)).Err() != nil {
						return errors.New("newer namespace setup failed")
					}
					want = "STALE"
				}
				ids := []string{p.wire.Artifacts[0].ID, p.wire.Artifacts[1].ID}
				selection, _ := json.Marshal(ids)
				before := batchReviewNativeDump(t, c, all)
				got := wireReviewEval(t, c, keys, raw, string(selection), string(p.materials[ids[0]]), string(p.materials[ids[1]]))
				if got != want || !reflect.DeepEqual(before, batchReviewNativeDump(t, c, all)) {
					t.Errorf("compact key/vector preflight failed expected %s got %s", want, got)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("large IDs stay opaque strings", func(t *testing.T) { wireReviewLargeNative(t, c) })
	t.Run("AM opaque intent bytes across codecs", func(t *testing.T) { wireReviewAMNative(t, c) })
	t.Run("subset has no full controls", func(t *testing.T) { wireReviewSubsetNative(t, c) })
	t.Run("ordinary has no render or ACK authority", func(t *testing.T) { wireReviewOrdinaryNative(t, c) })
}

func wireReviewLargeNative(t *testing.T, c *redis.Client) {
	wire, keys := wireReviewCanonical(t)
	p := &PreparedProjection{wire: wire, keys: keys}
	t.Cleanup(func() { c.Del(context.Background(), keys...) })
	wireReviewOK(t, c, p, "prepare", "", false, "")
	ids := []string{}
	bodies := []string{}
	for _, a := range wire.Artifacts {
		ids = append(ids, a.ID)
		bodies = append(bodies, "groups: []\n")
	}
	if ids[0] != "9007199254740993" || !strings.Contains(wire.AckReceipt, "18446744073709551615") {
		t.Fatal("large integer control fixture not established")
	}
	selection, _ := json.Marshal(ids)
	wireReviewOK(t, c, p, "artifact_batch", "", true, append([]string{string(selection)}, bodies...)...)
	for _, ctrl := range wire.Controls {
		if ctrl.Kind == string(configdao.ProjectionMembersFull) {
			wireReviewOK(t, c, p, "members", ctrl.ID, true, "")
		}
	}
	wireReviewOK(t, c, p, "ack", "", true, "")
	if raw, err := c.Get(context.Background(), projectionPoolKey(4, 7607, "ack")).Result(); err != nil || raw != wire.AckReceipt {
		t.Fatal("large IDs changed in the actual Redis ACK")
	}
}

func wireReviewAMNative(t *testing.T, c *redis.Client) {
	journal, ordinary, _ := projectionStoreSQL(t)
	const pool = 7632
	ip := "198.18.232.1"
	group := 3
	ref := projectionReserve(t, journal, 2, pool, 121)
	item := configdao.GeneratedPlanItem{SourceID: 1, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: pool, ConfigType: 5, InstanceIP: ip + "_3", Name: "webhook_wire_双引号\"&<tag>_3_" + ip, ConfigContent: "{}", Status: 1}, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: ip, GroupID: &group}}
	amPlan := func(ref configdao.PoolReservation, items ...configdao.GeneratedPlanItem) configdao.GeneratedProjectionPlanRequest {
		req := projectionFullRequest(ref, items...)
		req.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
		req.Controls = append(req.Controls, configdao.ProjectionControl{ControlID: 3, Kind: configdao.ProjectionWebhookIntents, Intents: &configdao.WebhookIntentsControl{RequireEmpty: true}})
		return req
	}
	if _, err := journal.CommitGeneratedProjectionPlan(context.Background(), amPlan(ref, item)); err != nil {
		t.Fatal(err)
	}
	oldOwner := configdao.GeneratedConfigOwnerOf(item.Desired)
	renamed := *item.Desired
	renamed.Name = "webhook_wire_new_3_" + ip
	if ordinary.UpdateMonitorConfig(context.Background(), &renamed) != nil {
		t.Fatal("same-ID rename fixture failed")
	}
	owner := configdao.GeneratedConfigOwnerOf(&renamed)
	retire := configdao.GeneratedPlanItem{SourceID: 2, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: item.Binding}
	request := amPlan(projectionReserve(t, journal, 2, pool, 122), retire)
	request.Controls[2].Intents.Add = []configdao.IntentOwnerChange{{Owner: owner, Proof: configdao.ProjectionProofRef{SourceID: 2}}}
	request.Controls[2].Intents.Remove = []configdao.IntentOwnerChange{{Owner: owner, Proof: configdao.ProjectionProofRef{SourceID: 2}}, {Owner: oldOwner, Proof: configdao.ProjectionProofRef{SourceID: 2}, AliasCurrentOwner: &owner}}
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewProjectionStore(c)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		p, err := compileProjection(store, lease, snapshot)
		if err != nil {
			return err
		}
		projectionTrack(t, c, p)
		oldJSON, _ := json.Marshal(oldOwner)
		newJSON, _ := json.Marshal(owner)
		key := alertRetirementIntentKey(pool)
		if c.SAdd(context.Background(), key, string(oldJSON)).Err() != nil {
			return errors.New("old locator seed failed")
		}
		wireReviewOK(t, c, p, "prepare", "", false, "")
		wireReviewOK(t, c, p, "intent_add", "3", true, "")
		entries, err := c.SMembers(context.Background(), key).Result()
		if err != nil {
			return err
		}
		expected := map[string]bool{string(oldJSON): true, string(newJSON): true}
		if len(entries) != 2 || !expected[entries[0]] || !expected[entries[1]] {
			return errors.New("compact codec changed opaque owner JSON bytes")
		}
		for _, a := range p.wire.Artifacts {
			wireReviewOK(t, c, p, "artifact", a.ID, false, "")
		}
		wireReviewOK(t, c, p, "members", "1", true, "")
		wireReviewOK(t, c, p, "intent_remove", "3", false, "")
		if count, err := c.SCard(context.Background(), key).Result(); err != nil || count != 0 {
			return errors.New("old object could not remove compact codec locators")
		}
		proof, err := p.Acknowledge(context.Background())
		if err != nil {
			return err
		}
		if got, err := c.Get(context.Background(), projectionPoolKey(2, pool, "ack")).Result(); err != nil || got != p.wire.AckReceipt {
			return errors.New("AM ACK bytes changed")
		}
		marked, err := lease.MarkComplete(proof)
		if err == nil && !marked {
			return errors.New("AM SQL completion failed")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func wireReviewSubsetNative(t *testing.T, c *redis.Client) {
	f := newBatchReviewNativeFixture(t, c, 2, "subset compact bytes", 5)
	pool := f.receipt.Reservation.PoolID
	fullKeys := []string{alertPublicationStateKey(pool), buildRedisSetKeyAlertManagerMainPoolIPs(pool), buildRedisKeyAlertManagerMain(f.items[0].Binding.InstanceIP)}
	if c.Set(context.Background(), fullKeys[0], "preserved full SHA", 0).Err() != nil || c.SAdd(context.Background(), fullKeys[1], "preserved full member").Err() != nil || c.Set(context.Background(), fullKeys[2], "preserved main body", 0).Err() != nil {
		t.Fatal("full state sentinel failed")
	}
	t.Cleanup(func() { c.Del(context.Background(), fullKeys...) })
	before := batchReviewNativeDump(t, c, fullKeys)
	err := f.journal.WithPoolProjectionLease(context.Background(), f.receipt, func(lease configdao.ProjectionLease) error {
		p, err := f.store.Prepare(context.Background(), lease)
		if err != nil {
			return err
		}
		f.track(t, p)
		ids := []uint64{}
		for _, a := range f.receipt.Artifacts {
			ids = append(ids, a.EventID)
		}
		if err := p.ApplyArtifacts(context.Background(), ids); err != nil {
			return err
		}
		proof, err := p.Acknowledge(context.Background())
		if err != nil {
			return err
		}
		if p.wire.Legacy != 0 || len(p.wire.Controls) != 1 {
			return errors.New("compact subset gained full controls")
		}
		marked, err := lease.MarkComplete(proof)
		if err == nil && !marked {
			return errors.New("subset SQL completion failed")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, batchReviewNativeDump(t, c, fullKeys)) {
		t.Fatal("compact subset changed full AM state")
	}
}

func wireReviewOrdinaryNative(t *testing.T, c *redis.Client) {
	f := newBatchReviewNativeFixture(t, c, 1, "groups: []\n", 4)
	row := *f.items[0].Desired
	row.ConfigContent = "groups: [] # ordinary\n"
	if f.ordinary.UpdateMonitorConfig(context.Background(), &row) != nil {
		t.Fatal("ordinary source update failed")
	}
	var event model.MonitorConfigArtifactEvent
	if f.db.Order("event_id DESC").First(&event).Error != nil {
		t.Fatal("ordinary event missing")
	}
	artifact := configdao.ArtifactReceipt{EventID: event.EventID, ConfigType: event.ConfigType, InstanceKey: string(event.InstanceKey), Action: event.Action, ProjectionPolicy: event.ProjectionPolicy, ConfigRowID: event.ConfigRowID, OldOwnerDigest: event.OldOwnerDigest, NewOwnerDigest: event.NewOwnerDigest, BodyDigest: event.BodyDigest, BindingEventID: event.BindingEventID}
	rawKey := buildRedisKeyRecordRule(row.InstanceIP)
	if c.Set(context.Background(), rawKey, "old projected body", 0).Err() != nil {
		t.Fatal("ordinary raw setup failed")
	}
	err := f.journal.WithArtifactProjectionLease(context.Background(), artifact, func(lease configdao.ProjectionLease) error {
		p, err := f.store.Prepare(context.Background(), lease)
		if err != nil {
			return err
		}
		f.track(t, p)
		if p.wire.PoolFence != 0 || p.wire.Ack != 0 || len(p.wire.Controls) != 0 {
			return errors.New("compact ordinary acquired pool authority")
		}
		if _, err := p.Acknowledge(context.Background()); err == nil {
			return errors.New("ordinary acquired ACK")
		}
		return p.ApplyArtifact(context.Background(), artifact.EventID)
	})
	if err != nil {
		t.Fatal(err)
	}
	if exists, err := c.Exists(context.Background(), rawKey).Result(); err != nil || exists != 0 {
		t.Fatal("ordinary invalidation did not apply without rendering its SQL body")
	}
}
