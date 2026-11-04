package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

func projectionCodecCheck(t *testing.T, handle *PreparedProjection) {
	t.Helper()
	if reflect.TypeOf(projectionWire{}).NumField() != 14 {
		t.Fatal("canonical wire fields changed without codec review")
	}
	wire := handle.wire
	wire.Mode = "prepare"
	wire.Target = ""
	compact, err := marshalProjectionWire(wire, handle.keys)
	if err != nil {
		t.Fatal(err)
	}
	var top []json.RawMessage
	if err := json.Unmarshal(compact, &top); err != nil || len(top) != 15 {
		t.Fatal("encoder did not emit positional top15", err)
	}
	var tag string
	if err := json.Unmarshal(top[0], &tag); err != nil || tag != "pw2" {
		t.Fatal("encoder did not emit pw2", err)
	}
	restored, err := decodeProjectionTestWire(string(compact), handle.keys)
	if err != nil || !reflect.DeepEqual(restored, wire) {
		t.Fatal("compact changed a canonical field", err)
	}
	// Keep an independent legacy object path rather than converting every test
	// input to compact and accidentally losing old-format compatibility coverage.
	object, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	old, err := decodeProjectionTestWire(string(object), handle.keys)
	if err != nil || !reflect.DeepEqual(old, wire) {
		t.Fatal("legacy canonical object changed", err)
	}
}
func TestProjectionCompactWireActualSQLFamiliesAndEmpty(t *testing.T) {
	for _, family := range []int8{1, 2, 3, 4, 5} {
		t.Run(fmt.Sprint(family), func(t *testing.T) {
			journal, _, _ := projectionStoreSQL(t)
			ctx := context.Background()
			ref, err := journal.ReservePoolOperation(ctx, configdao.PoolOperationRequest{Family: family, PoolID: 7, Nonce: [16]byte{1}})
			if err != nil {
				t.Fatal(err)
			}
			patterns := map[int8]string{1: ConfigNamePrometheus, 2: ConfigNameAlertManager, 3: ConfigNameAlertRule, 4: ConfigNameRecordRule}
			items := []configdao.GeneratedPlanItem{}
			if family != 5 {
				row := &model.MonitorConfig{PoolID: 7, ConfigType: family, InstanceIP: "2001:DB8::1", Name: fmt.Sprintf(patterns[family], 7, "2001:DB8::1"), ConfigContent: "groups: []\n", Status: 1}
				items = append(items, configdao.GeneratedPlanItem{SourceID: 1, Action: model.ConfigProjectionRender, Desired: row, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: family, InstanceIP: row.InstanceIP}})
			}
			if family == 2 || family == 5 {
				group := 3
				row := &model.MonitorConfig{PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", Name: "webhook_codec_3_127.0.0.1", ConfigContent: "{}", Status: 1}
				items = append(items, configdao.GeneratedPlanItem{SourceID: 2, Action: model.ConfigProjectionRender, Desired: row, Binding: &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.0.1", GroupID: &group}})
			}
			request := projectionFullRequest(ref, items...)
			if family == 2 {
				request.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
				request.Controls = append(request.Controls, configdao.ProjectionControl{ControlID: 3, Kind: configdao.ProjectionWebhookIntents, Intents: &configdao.WebhookIntentsControl{RequireEmpty: true}})
			}
			if family == 5 {
				request.Controls = []configdao.ProjectionControl{{ControlID: 9, Kind: configdao.ProjectionACKWebhookSubset, Ack: &configdao.ProjectionAckControl{ArtifactSourceIDs: []uint64{2}}}}
			}
			receipt, err := journal.CommitGeneratedProjectionPlan(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
				snapshot, err := lease.Snapshot()
				if err != nil {
					return err
				}
				handle, err := compileProjection(nil, lease, snapshot)
				if err != nil {
					return err
				}
				projectionCodecCheck(t, handle)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("empty full", func(t *testing.T) {
		journal, _, _ := projectionStoreSQL(t)
		ctx := context.Background()
		ref := projectionReserve(t, journal, 4, 7, 1)
		receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref))
		if err != nil {
			t.Fatal(err)
		}
		err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
			snapshot, err := lease.Snapshot()
			if err != nil {
				return err
			}
			handle, err := compileProjection(nil, lease, snapshot)
			if err != nil {
				return err
			}
			projectionCodecCheck(t, handle)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
func TestProjectionCompactWireOrdinaryActualSQL(t *testing.T) {
	journal, ordinary, db := projectionStoreSQL(t)
	ctx := context.Background()
	ref := projectionReserve(t, journal, 4, 7, 1)
	item := projectionRecordItem(7, 1, "127.0.0.1", "groups: []\n")
	if _, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item)); err != nil {
		t.Fatal(err)
	}
	if err := ordinary.DeleteMonitorConfig(ctx, item.Desired.ID); err != nil {
		t.Fatal(err)
	}
	var event model.MonitorConfigArtifactEvent
	if err := db.Order("event_id DESC").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	artifact := configdao.ArtifactReceipt{EventID: event.EventID, ConfigType: event.ConfigType, InstanceKey: string(event.InstanceKey), Action: event.Action, ProjectionPolicy: event.ProjectionPolicy, ConfigRowID: event.ConfigRowID, OldOwnerDigest: event.OldOwnerDigest, NewOwnerDigest: event.NewOwnerDigest, BodyDigest: event.BodyDigest, BindingEventID: event.BindingEventID}
	err := journal.WithArtifactProjectionLease(ctx, artifact, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(nil, lease, snapshot)
		if err != nil {
			return err
		}
		projectionCodecCheck(t, handle)
		if handle.wire.PoolFence != 0 || handle.wire.Ack != 0 || len(handle.wire.Controls) != 0 {
			t.Fatal("ordinary codec gained full controls")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestProjectionCompactWireRejectsNilAndKeyElisionMismatch(t *testing.T) {
	err := batchGoBoundaryFixture(t, []string{"groups: []\n"}, func(handle *PreparedProjection, _ []uint64, _ *batchGoBoundaryHook) error {
		for _, mode := range []string{"roles", "fences", "artifacts", "controls", "desired", "add", "remove", "role key"} {
			t.Run(mode, func(t *testing.T) {
				raw, _ := json.Marshal(handle.wire)
				var wire projectionWire
				if err := json.Unmarshal(raw, &wire); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "roles":
					wire.Roles = nil
				case "fences":
					wire.Fences = nil
				case "artifacts":
					wire.Artifacts = nil
				case "controls":
					wire.Controls = nil
				case "desired":
					wire.Controls[0].Desired = nil
				case "add":
					wire.Controls[0].Add = nil
				case "remove":
					wire.Controls[0].Remove = nil
				case "role key":
					wire.Roles[0].Key += "wrong"
				}
				if _, err := marshalProjectionWire(wire, handle.keys); !errors.Is(err, ErrProjectionStore) {
					t.Fatal("encoder hid malformed canonical input", err)
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestProjectionCompactWireDefaultSenderAndLargeIDs(t *testing.T) {
	err := batchGoBoundaryFixture(t, []string{"groups: []\n", "groups: []\n"}, func(handle *PreparedProjection, ids []uint64, hook *batchGoBoundaryHook) error {
		if err := handle.ApplyArtifacts(context.Background(), ids); !errors.Is(err, ErrProjectionStore) || hook.calls != 1 {
			t.Fatal("send-boundary fixture did not observe one EVAL error", err)
		}
		n := hook.args[2].(int)
		raw := hook.args[n+3].(string)
		if !strings.HasPrefix(raw, `["pw2",`) {
			t.Fatal("actual default sender still emitted an object")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	journal, _, db := projectionStoreSQL(t)
	ctx := context.Background()
	anchor := model.MonitorConfigArtifactEvent{EventID: 9007199254740992, ConfigType: 4, InstanceKey: []byte("203.0.113.250"), Action: model.ConfigProjectionInvalidate, ProjectionPolicy: model.ConfigProjectionNotApplicable, DeliveryState: model.ConfigProjectionApplied}
	if err := db.Create(&anchor).Error; err != nil {
		t.Fatal(err)
	}
	ref := projectionReserve(t, journal, 4, 7, 1)
	item := projectionRecordItem(7, ^uint64(0), "127.0.0.1", "groups: []\n")
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, projectionFullRequest(ref, item))
	if err != nil {
		t.Fatal(err)
	}
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(nil, lease, snapshot)
		if err != nil {
			return err
		}
		projectionCodecCheck(t, handle)
		raw, err := marshalProjectionWire(handle.wire, handle.keys)
		if err != nil {
			return err
		}
		decoded, err := decodeProjectionTestWire(string(raw), handle.keys)
		if err != nil {
			return err
		}
		if decoded.Artifacts[0].ID != strconv.FormatUint(receipt.Artifacts[0].EventID, 10) || !strings.Contains(decoded.AckReceipt, strconv.FormatUint(^uint64(0), 10)) {
			t.Fatal("uint64 text lost precision")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestProjectionCompactWireRealAMOpaqueIntentBytes(t *testing.T) {
	journal, _, _ := projectionStoreSQL(t)
	ctx := context.Background()
	reserve := func(n byte) configdao.PoolReservation {
		t.Helper()
		ref, err := journal.ReservePoolOperation(ctx, configdao.PoolOperationRequest{Family: 2, PoolID: 7, Nonce: [16]byte{n}})
		if err != nil {
			t.Fatal(err)
		}
		return ref
	}
	request := func(ref configdao.PoolReservation, item configdao.GeneratedPlanItem) configdao.GeneratedProjectionPlanRequest {
		req := projectionFullRequest(ref, item)
		req.Controls[1].Ack.RequiredControlIDs = []uint64{1, 3}
		req.Controls = append(req.Controls, configdao.ProjectionControl{ControlID: 3, Kind: configdao.ProjectionWebhookIntents, Intents: &configdao.WebhookIntentsControl{RequireEmpty: true}})
		return req
	}
	group := 3
	binding := &configdao.ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.0.1", GroupID: &group}
	item := configdao.GeneratedPlanItem{SourceID: 1, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", Name: "webhook_中文<&>_3_127.0.0.1", ConfigContent: "{}", Status: 1}, Binding: binding}
	if _, err := journal.CommitGeneratedProjectionPlan(ctx, request(reserve(1), item)); err != nil {
		t.Fatal(err)
	}
	owner := configdao.GeneratedConfigOwnerOf(item.Desired)
	retire := configdao.GeneratedPlanItem{SourceID: 2, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: binding}
	req := request(reserve(2), retire)
	change := configdao.IntentOwnerChange{Owner: owner, Proof: configdao.ProjectionProofRef{SourceID: 2}}
	req.Controls[2].Intents.Add = []configdao.IntentOwnerChange{change}
	req.Controls[2].Intents.Remove = []configdao.IntentOwnerChange{change}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		handle, err := compileProjection(nil, lease, snapshot)
		if err != nil {
			return err
		}
		projectionCodecCheck(t, handle)
		expected, err := json.Marshal(owner)
		if err != nil {
			return err
		}
		found := false
		for _, control := range handle.wire.Controls {
			if control.Kind == "webhook_intents" {
				found = true
				if !control.RequireEmpty || len(control.Add) != 1 || len(control.Remove) != 1 || control.Add[0] != string(expected) || control.Remove[0] != string(expected) {
					t.Fatal("actual issued locator bytes lost")
				}
				if !strings.Contains(control.Add[0], `\u003c`) || !strings.Contains(control.Add[0], "中文") {
					t.Fatal("fixture did not exercise opaque escaped/UTF8 JSON")
				}
			}
		}
		if !found {
			t.Fatal("actual AM fixture omitted intents")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
