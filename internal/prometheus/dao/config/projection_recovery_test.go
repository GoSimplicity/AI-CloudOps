package config

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestRecoveryEnumerationOnlyCurrentCommittedAndFixedHighWater(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	reader := NewProjectionRecoveryJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	commit := func(n byte, pool int) ProjectionReceipt {
		t.Helper()
		request := journalBehaviorRequest(n)
		request.PoolID = pool
		ref, err := dao.ReservePoolOperation(ctx, request)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := dao.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
		if err != nil {
			t.Fatal(err)
		}
		return receipt
	}
	old := commit(1, 7)
	current := commit(2, 7)
	projected := commit(3, 8)
	proof, _ := projectionAckProof(projected)
	if ok, err := dao.MarkProjectionCompleteIfCurrent(ctx, projected, proof); !ok || err != nil {
		t.Fatal(err)
	}
	request := journalBehaviorRequest(4)
	request.PoolID = 9
	if _, err := dao.ReservePoolOperation(ctx, request); err != nil {
		t.Fatal(err)
	}
	high, err := reader.CurrentProjectionRecoveryHighWater(ctx)
	if err != nil {
		t.Fatal(err)
	}
	late := commit(5, 10)
	rows, err := reader.ListCurrentUnprojectedPoolOperations(ctx, 0, high, 100)
	if err != nil || len(rows) != 1 || rows[0] != current.Reservation {
		t.Fatal("history/projected/reserved/late entered snapshot", rows, err)
	}
	rows, err = reader.ListCurrentUnprojectedPoolOperations(ctx, current.Reservation.OperationID, late.Reservation.OperationID, 1)
	if err != nil || len(rows) != 1 || rows[0] != late.Reservation {
		t.Fatal("keyset lost next current operation", err)
	}
	// Corrupt payload is retained as a poison locator, then rejected by its loader;
	// it must not prevent a later pool from being enumerated.
	if err := db.Model(&model.MonitorConfigPoolOperation{}).Where("operation_id = ?", current.Reservation.OperationID).Update("manifest", "{broken").Error; err != nil {
		t.Fatal(err)
	}
	rows, err = reader.ListCurrentUnprojectedPoolOperations(ctx, 0, late.Reservation.OperationID, 100)
	if err != nil || len(rows) != 2 {
		t.Fatal("poison payload blocked enumeration", err)
	}
	if _, err := dao.LoadLastCommittedProjection(ctx, 4, 7); err == nil {
		t.Fatal("poison payload acquired receipt")
	}
	if old.Reservation.OperationID >= current.Reservation.OperationID {
		t.Fatal("fixture ordering")
	}
}

func TestRecoveryExpectedLastCommittedGuardsReserveAndCommit(t *testing.T) {
	for _, mode := range []string{"reserve", "commit", "known committed retry"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			commit := func(n byte) ProjectionReceipt {
				t.Helper()
				ref, err := dao.ReservePoolOperation(ctx, journalBehaviorRequest(n))
				if err != nil {
					t.Fatal(err)
				}
				receipt, err := dao.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
				if err != nil {
					t.Fatal(err)
				}
				return receipt
			}
			first := commit(1)
			guard := first.Reservation.OperationID
			request := journalBehaviorRequest(2)
			request.RecoveryOfOperationID = &guard
			request.ExpectedLastCommittedOperationID = &guard
			if mode == "reserve" {
				second := commit(3)
				if _, err := dao.ReservePoolOperation(ctx, request); !errors.Is(err, ErrJournalConflict) {
					t.Fatal("stale recovery reserved after newer commit", err)
				}
				last, _ := dao.LoadLastCommittedProjection(ctx, 4, 7)
				if last == nil || last.Reservation != second.Reservation {
					t.Fatal("stale guard moved current")
				}
				return
			}
			ref, err := dao.ReservePoolOperation(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			plan := v2RecordPlan(ref)
			plan.ExpectedLastCommittedOperationID = &guard
			if mode == "commit" {
				// Model a changed control pointer while leaving this reserved attempt intact,
				// isolating the commit guard from CurrentAttempt's independent protection.
				if err := db.Model(&model.MonitorConfigPoolControl{}).Where("family=4 AND pool_id=7").Update("last_committed_operation_id", nil).Error; err != nil {
					t.Fatal(err)
				}
				if _, err := dao.CommitGeneratedProjectionPlan(ctx, plan); !errors.Is(err, ErrJournalConflict) {
					t.Fatal("commit ignored expected last-good pointer", err)
				}
				var op model.MonitorConfigPoolOperation
				if err := db.First(&op, "operation_id = ?", ref.OperationID).Error; err != nil || op.Manifest != nil {
					t.Fatal("failed transition issued manifest", err)
				}
				return
			}
			receipt, err := dao.CommitGeneratedProjectionPlan(ctx, plan)
			if err != nil {
				t.Fatal(err)
			}
			again, err := dao.CommitGeneratedProjectionPlan(ctx, plan)
			if err != nil || !equalProjectionPayload(receipt, again) {
				t.Fatal("known committed retry rejected own advanced lastCommitted", err)
			}
		})
	}
}
func TestRecoverySinglePoolSourceReadersAndInvalidPages(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	reader := NewProjectionRecoveryJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	if err := db.Create(&model.MonitorConfigPoolControl{Family: 4, PoolID: 7, ReservedGeneration: 9}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.MonitorConfigPoolControl{Family: 2, PoolID: 7, ReservedGeneration: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.MonitorConfigPoolControl{Family: 5, PoolID: 7, ReservedGeneration: 15}).Error; err != nil {
		t.Fatal(err)
	}
	if version, err := reader.ReadPoolProjectionSourceVersion(ctx, 4, 7); err != nil || version != 9 {
		t.Fatal(version, err)
	}
	if version, err := reader.ReadPoolProjectionSourceVersion(ctx, 3, 7); err != nil || version != 0 {
		t.Fatal(version, err)
	}
	if pair, err := reader.ReadAMPoolSourceVersion(ctx, 7); err != nil || pair.FullCounter != 12 || pair.SubsetCounter != 15 {
		t.Fatal(pair, err)
	}
	for _, size := range []int{0, -1, 101} {
		if _, err := reader.ListCurrentUnprojectedPoolOperations(ctx, 0, 100, size); !errors.Is(err, ErrJournalInvalid) {
			t.Fatal("bad page accepted", size, err)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reader.ReadAMPoolSourceVersion(canceled, 7); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation lost", err)
	}
}
