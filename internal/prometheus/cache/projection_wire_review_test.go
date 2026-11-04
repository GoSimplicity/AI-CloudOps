package cache

import (
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
)

func wireReviewCanonical(t *testing.T) (projectionWire, []string) {
	t.Helper()
	journal, _, db := projectionStoreSQL(t)
	anchor := model.MonitorConfigArtifactEvent{EventID: 9007199254740992, ConfigType: 4, InstanceKey: []byte("198.18.249.1"), Action: model.ConfigProjectionInvalidate, ProjectionPolicy: model.ConfigProjectionNotApplicable, DeliveryState: model.ConfigProjectionApplied}
	if db.Create(&anchor).Error != nil {
		t.Fatal("large-ID fixture unavailable")
	}
	ref := projectionReserve(t, journal, 4, 7607, 119)
	items := []configdao.GeneratedPlanItem{projectionRecordItem(7607, 1, "198.18.249.2", "groups: []\n"), projectionRecordItem(7607, ^uint64(0), "198.18.249.3", "groups: []\n")}
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), projectionFullRequest(ref, items...))
	if err != nil {
		t.Fatal(err)
	}
	var wire projectionWire
	var keys []string
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease configdao.ProjectionLease) error {
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		p, err := compileProjection(nil, lease, snapshot)
		if err != nil {
			return err
		}
		wire = p.wire
		keys = append([]string(nil), p.keys...)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	wire.Mode = "artifact"
	wire.Target = wire.Artifacts[0].ID
	return wire, keys
}

func TestWireReviewCompactTransportPreservesCanonicalOpaqueValues(t *testing.T) {
	wire, keys := wireReviewCanonical(t)
	raw, err := marshalProjectionWire(wire, keys)
	if err != nil {
		t.Fatal(err)
	}
	var tuple []json.RawMessage
	if json.Unmarshal(raw, &tuple) != nil || len(tuple) != 15 || string(tuple[0]) != `"pw2"` {
		t.Fatal("compact transport marker or exact top arity changed")
	}
	var roles [][]json.RawMessage
	if json.Unmarshal(tuple[5], &roles) != nil || len(roles) != len(keys) {
		t.Fatal("compact role cardinality changed")
	}
	for _, row := range roles {
		if len(row) != 5 {
			t.Fatal("role tuple did not elide exactly the duplicated key")
		}
	}
	decoded, err := decodeProjectionTestWire(string(raw), keys)
	if err != nil || !reflect.DeepEqual(wire, decoded) {
		t.Fatal("canonical wire changed during Go codec roundtrip")
	}
	if decoded.Artifacts[0].ID != strconv.FormatUint(9007199254740993, 10) || !strings.Contains(decoded.AckReceipt, strconv.FormatUint(^uint64(0), 10)) {
		t.Fatal("opaque IDs lost exact string/JSON representation")
	}
	legacy, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	old, err := decodeProjectionTestWire(string(legacy), keys)
	if err != nil || !reflect.DeepEqual(old, decoded) {
		t.Fatal("legacy and compact Go views have different canonical behavior")
	}
	// This test checks transport data only. Real Lua/fingerprint proof is provided
	// separately to Root's owned integration gate, not inferred from this decoder.
}

func TestWireReviewEncoderRejectsMismatchedKeyElision(t *testing.T) {
	for _, mode := range []string{"missing KEY", "extra KEY", "changed role key"} {
		t.Run(mode, func(t *testing.T) {
			wire, keys := wireReviewCanonical(t)
			switch mode {
			case "missing KEY":
				keys = keys[:len(keys)-1]
			case "extra KEY":
				keys = append(keys, "unissued physical key")
			case "changed role key":
				wire.Roles[len(wire.Roles)-1].Key += "-changed"
			}
			if _, err := marshalProjectionWire(wire, keys); err == nil {
				t.Fatal("key elision concealed a role/key mismatch")
			}
		})
	}
}
