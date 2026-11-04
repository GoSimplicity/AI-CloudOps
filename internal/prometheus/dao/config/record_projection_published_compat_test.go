package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type publishedRecordPayload struct {
	PlanDigest string
	Manifest   string
	Receipt    json.RawMessage
}

func publishedRecordPlan(ref PoolReservation, items ...GeneratedPlanItem) GeneratedProjectionPlanRequest {
	hash := [32]byte{0xb5}
	ips := []string{}
	ids := []uint64{}
	for _, item := range items {
		ids = append(ids, item.SourceID)
		if item.Action == model.ConfigProjectionRender {
			ips = append(ips, item.Binding.InstanceIP)
		}
	}
	return GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: ips}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: ids, RequiredControlIDs: []uint64{1}}}}}
}
func publishedRecordRender(pool int, source uint64) GeneratedPlanItem {
	const ip = "127.0.45.1"
	return GeneratedPlanItem{SourceID: source, Action: model.ConfigProjectionRender, Desired: &model.MonitorConfig{PoolID: pool, ConfigType: 4, InstanceIP: ip, Name: fmt.Sprintf("prometheus_record_rule_%d_%s.yaml", pool, ip), ConfigContent: fmt.Sprintf("groups: [] # published pool %d\n", pool), Status: model.ConfigStatusActive, LastGeneratedTime: 100}, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: ip}}
}
func TestRecordPublishedProjectionPayloadCompatibility(t *testing.T) {
	snapshots := map[string]publishedRecordPayload{}
	for _, scenario := range []string{"empty", "render", "live owner with deleted foreign history", "accepted superseded", "legacy superseded"} {
		t.Run(scenario, func(t *testing.T) {
			_, db := journalContractFixture(t)
			ctx := context.Background()
			journal := NewRecordProjectionJournalDAO(zap.NewNop(), db)
			reserve := func(pool int, n byte, recovery *uint64) PoolReservation {
				t.Helper()
				ref, err := journal.ReservePoolOperation(ctx, PoolOperationRequest{Family: 4, PoolID: pool, Nonce: [16]byte{n}, RecoveryOfOperationID: recovery})
				if err != nil {
					t.Fatal(err)
				}
				return ref
			}
			commit := func(request RecordProjectionPlanRequest) ProjectionReceipt {
				t.Helper()
				receipt, err := journal.CommitRecordProjectionPlan(ctx, request)
				if err != nil {
					t.Fatal(err)
				}
				return receipt
			}
			var receipt ProjectionReceipt
			switch scenario {
			case "empty":
				receipt = commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(7, 1, nil))})
			case "render":
				receipt = commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(7, 1, nil), publishedRecordRender(7, 11))})
			case "live owner with deleted foreign history":
				own := *publishedRecordRender(7, 11).Desired
				own.ID = 1
				foreign := own
				foreign.ID = 2
				foreign.PoolID = 99
				foreign.Name = "custom historical owner"
				foreign.DeletedAt = 123
				if db.Create(&own).Error != nil || db.Create(&foreign).Error != nil {
					t.Fatal("published owner fixture failed")
				}
				guard, err := RecordHistoryGuardOf(own.InstanceIP, []*model.MonitorConfig{&own, &foreign})
				if err != nil {
					t.Fatal(err)
				}
				expected := GeneratedConfigOwnerOf(&own)
				retire := GeneratedPlanItem{SourceID: 11, Action: model.ConfigProjectionRetire, Expected: &expected, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: own.InstanceIP}}
				receipt = commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(7, 1, nil), retire), History: []RecordHistoryGuard{guard}})
			default:
				var accepted ProjectionReceipt
				var oldOwner GeneratedConfigOwner
				if scenario == "accepted superseded" {
					item := publishedRecordRender(7, 11)
					accepted = commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(7, 1, nil), item)})
					oldOwner = GeneratedConfigOwnerOf(item.Desired)
					if err := NewMonitorConfigDAO(zap.NewNop(), db).DeleteMonitorConfig(ctx, item.Desired.ID); err != nil {
						t.Fatal(err)
					}
				} else {
					old := *publishedRecordRender(7, 11).Desired
					old.ID = 1
					old.DeletedAt = 123
					digest := sha256.Sum256([]byte(old.ConfigContent))
					old.ConfigHash = hex.EncodeToString(digest[:])
					if db.Create(&old).Error != nil {
						t.Fatal("published legacy fixture failed")
					}
					oldOwner = GeneratedConfigOwnerOf(&old)
				}
				if db.Unscoped().Model(&model.MonitorConfig{}).Where("id = ?", oldOwner.ID).Update("deleted_at", uint64(123)).Error != nil {
					t.Fatal("fixed historical deletion time failed")
				}
				replacement := commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(8, 2, nil), publishedRecordRender(8, 12))})
				proof, err := projectionAckProof(replacement)
				if err != nil {
					t.Fatal(err)
				}
				if marked, err := journal.MarkProjectionCompleteIfCurrent(ctx, replacement, proof); err != nil || !marked {
					t.Fatal("SQL-only compatibility completion fixture failed", err)
				}
				history := NewGeneratedConfigHistoryReader(zap.NewNop(), db)
				var released []RecordSupersededArtifact
				var recovery *uint64
				if scenario == "accepted superseded" {
					released, err = history.GetSupersededRecordArtifacts(ctx, accepted)
					recovery = &accepted.Reservation.OperationID
				} else {
					released, err = history.GetLegacySupersededRecordArtifacts(ctx, 7, []GeneratedConfigOwner{oldOwner})
				}
				if err != nil || len(released) != 1 {
					t.Fatal("published disposition fixture failed", err)
				}
				receipt = commit(RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: publishedRecordPlan(reserve(7, 3, recovery)), Superseded: released})
			}
			var op model.MonitorConfigPoolOperation
			if db.First(&op, "operation_id = ?", receipt.Reservation.OperationID).Error != nil || op.Manifest == nil {
				t.Fatal("published manifest unavailable")
			}
			raw, err := json.Marshal(receipt)
			if err != nil {
				t.Fatal(err)
			}
			snapshots[scenario] = publishedRecordPayload{PlanDigest: hex.EncodeToString(receipt.PlanDigest[:]), Manifest: *op.Manifest, Receipt: raw}
		})
	}
	if t.Failed() {
		return
	}
	actual, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if capture := os.Getenv("CLOUDOPS_CAPTURE_RECORD_PUBLISHED_PAYLOAD"); capture != "" {
		parent, err := os.Stat(filepath.Dir(capture))
		if err != nil || parent.Mode().Perm() != 0700 {
			t.Fatal("capture requires an owned private directory")
		}
		f, err := os.OpenFile(capture, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal("capture target must not already exist")
		}
		if _, err := f.Write(append(actual, '\n')); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile("testdata/record_projection_db65.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]publishedRecordPayload
	if json.Unmarshal(raw, &expected) != nil {
		t.Fatal("invalid published compatibility fixture")
	}
	for name, value := range expected {
		// Indentation belongs to the fixture file, not the captured receipt.
		// Compact preserves property order, string encoding and numeric values.
		var compact bytes.Buffer
		if json.Compact(&compact, value.Receipt) != nil {
			t.Fatal("invalid captured receipt")
		}
		value.Receipt = append(json.RawMessage(nil), compact.Bytes()...)
		expected[name] = value
	}
	if !reflect.DeepEqual(snapshots, expected) {
		t.Fatal("published db65 Record manifest, receipt or fingerprint changed")
	}
}
