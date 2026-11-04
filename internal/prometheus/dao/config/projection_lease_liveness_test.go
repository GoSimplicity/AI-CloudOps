package config

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestLeaseRecheckUsesOnlyItsOriginalTransactionAfterInitialValidation(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	item := journalBehaviorRender(1, "127.0.0.1")
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref, *item))
	if err != nil {
		t.Fatal(err)
	}
	selects := 0
	probes := 0
	var original gorm.ConnPool
	count := func(tx *gorm.DB) {
		sql := strings.TrimSpace(tx.Statement.SQL.String())
		if strings.HasPrefix(sql, "SELECT ") {
			selects++
		}
		if sql == "SELECT 1" {
			probes++
			if original != nil && tx.Statement.ConnPool != original {
				t.Error("liveness borrowed a different connection")
			}
		}
	}
	if err := db.Callback().Query().After("gorm:query").Register("lease_liveness:queries", count); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Row().After("gorm:row").Register("lease_liveness:rows", count); err != nil {
		t.Fatal(err)
	}
	var escaped ProjectionLease
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		escaped = lease
		actual := lease.(*sqlProjectionLease)
		if !actual.initialized || selects <= 1 {
			t.Fatal("lease skipped complete initial proof")
		}
		original = actual.tx.Statement.ConnPool
		before := selects
		for i := 0; i < 8; i++ {
			if err := lease.RecheckCurrent(ctx); err != nil {
				return err
			}
		}
		if selects-before != 8 || probes != 8 {
			t.Fatal("recheck rebuilt material or omitted real SQL", selects-before, probes)
		}
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		snapshot.Artifacts[0].Body[0] = 'X'
		second, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if string(second.Artifacts[0].Body) != item.Desired.ConfigContent {
			t.Fatal("validated material lost deep-copy isolation")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if probes != 9 {
		t.Fatal("lease exit omitted original-transaction liveness probe", probes)
	}
	if _, err := escaped.Snapshot(); err == nil {
		t.Fatal("callback exit retained authority")
	}
}

func TestLeaseCanceledRecheckCannotBeRenewedWithANewContext(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, v2RecordPlan(ref))
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = journal.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error {
		called = true
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		if err := lease.RecheckCurrent(canceled); !errors.Is(err, context.Canceled) {
			t.Fatal("child cancellation lost", err)
		}
		if _, err := lease.Snapshot(); err == nil {
			t.Fatal("canceled recheck retained cached snapshot authority")
		}
		if err := lease.RecheckCurrent(context.Background()); err == nil {
			t.Fatal("new context revived sealed lease")
		}
		proof, _ := projectionAckProof(receipt)
		if _, err := lease.MarkComplete(proof); err == nil {
			t.Fatal("sealed lease issued completion")
		}
		return nil
	})
	if err == nil || !called {
		t.Fatal("canceled callback lease was accepted", err)
	}
}
