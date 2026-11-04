package config

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestLeaseInitialCurrentOperationIsReadOnce(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	items := []GeneratedPlanItem{}
	for i := 1; i <= 8; i++ {
		items = append(items, *journalBehaviorRender(uint64(i), fmt.Sprintf("127.0.0.%d", i)))
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, items...))
	if err != nil {
		t.Fatal(err)
	}
	operations := 0
	if err := db.Callback().Query().After("gorm:query").Register("lease_initial:operation_reads", func(tx *gorm.DB) {
		if tx.Statement.Table == (model.MonitorConfigPoolOperation{}).TableName() {
			operations++
		}
	}); err != nil {
		t.Fatal(err)
	}
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		if operations != 1 {
			t.Fatal("each artifact reloaded its own already-verified operation", operations)
		}
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if len(snapshot.Artifacts) != len(items) {
			t.Fatal("initial validation omitted a material")
		}
		for i, material := range snapshot.Artifacts {
			if string(material.Body) != items[i].Desired.ConfigContent {
				t.Fatal("initial material body changed")
			}
		}
		return lease.RecheckCurrent(ctx)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLeaseOperationIndexRejectsAnInvalidLaterEntryOrControl(t *testing.T) {
	for _, mode := range []string{"later digest", "duplicate event", "control digest"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			makeReceipt := func(n byte, firstIP string) ProjectionReceipt {
				t.Helper()
				ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(n))
				if err != nil {
					t.Fatal(err)
				}
				items := []GeneratedPlanItem{*journalBehaviorRender(1, firstIP), *journalBehaviorRender(2, firstIP+"0")}
				receipt, err := journal.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, items...))
				if err != nil {
					t.Fatal(err)
				}
				return receipt
			}
			good := makeReceipt(1, "127.0.0.1")
			bad := makeReceipt(2, "127.0.0.2")
			var op model.MonitorConfigPoolOperation
			if err := db.First(&op, "operation_id = ?", bad.Reservation.OperationID).Error; err != nil {
				t.Fatal(err)
			}
			envelope, err := decodeProjectionManifest(op.Manifest)
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "later digest":
				envelope.Artifacts[1].PlanDigest = "invalid"
			case "duplicate event":
				envelope.Artifacts[1].EventID = envelope.Artifacts[0].EventID
			case "control digest":
				envelope.Controls[0].Digest[0] ^= 1
			}
			raw, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&op).Update("manifest", string(raw)).Error; err != nil {
				t.Fatal(err)
			}
			err = db.Transaction(func(tx *gorm.DB) error {
				reader := newLeaseInitialOperations(tx)
				facts, err := reader.operation(good.Reservation.OperationID)
				if err != nil {
					return err
				}
				if facts == nil || !facts.containsEvent(good.Artifacts[0].EventID) {
					t.Fatal("valid op did not establish membership")
				}
				facts, err = reader.operation(bad.Reservation.OperationID)
				if err != nil {
					return err
				}
				if facts != nil {
					t.Fatal("warm different operation or an earlier matching entry hid malformed full manifest")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			called := false
			if err := journal.WithPoolProjectionLease(ctx, bad, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
				t.Fatal("malformed operation entered callback")
			}
		})
	}
}
