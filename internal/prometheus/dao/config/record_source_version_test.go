package config

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestRecordSourceVersionCASRejectsOlderSnapshotWithoutMintingGeneration(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewRecordProjectionJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	before, err := journal.ReadRecordSourceVersions(ctx)
	if err != nil || len(before) != 0 {
		t.Fatal(err)
	}
	bRequest := journalBehaviorRequest(2)
	b, err := journal.ReserveRecordPoolOperation(ctx, bRequest, 0)
	if err != nil || b.Generation != 1 {
		t.Fatal(err)
	}
	old, err := journal.ReserveRecordPoolOperation(ctx, journalBehaviorRequest(1), 0)
	if !errors.Is(err, ErrJournalConflict) || old.OperationID != 0 {
		t.Fatal("old source minted a newer generation", err)
	}
	var count int64
	db.Model(&model.MonitorConfigPoolOperation{}).Count(&count)
	if count != 1 {
		t.Fatal("rejected source created an operation")
	}
	after, err := journal.ReadRecordSourceVersions(ctx)
	if err != nil || len(after) != 1 || after[0].Counter != 1 || after[0].PoolID != 7 {
		t.Fatal("rejected source changed durable version", err)
	}
	again, err := journal.ReserveRecordPoolOperation(ctx, bRequest, 0)
	if err != nil || again != b {
		t.Fatal("same nonce retry minted another slot", err)
	}
	fresh, err := journal.ReserveRecordPoolOperation(ctx, journalBehaviorRequest(3), after[0].Counter)
	if err != nil || fresh.Generation != 2 {
		t.Fatal("fresh source could not advance", err)
	}
}
func TestRecordSourceVersionsReadEveryPageAndKeepOtherFamiliesSeparate(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewRecordProjectionJournalDAO(zap.NewNop(), db)
	controls := []model.MonitorConfigPoolControl{}
	for pool := 1; pool <= 105; pool++ {
		controls = append(controls, model.MonitorConfigPoolControl{Family: 4, PoolID: pool, ReservedGeneration: uint64(pool)})
	}
	controls = append(controls, model.MonitorConfigPoolControl{Family: 2, PoolID: 7, ReservedGeneration: 999})
	if db.Create(&controls).Error != nil {
		t.Fatal("version fixture failed")
	}
	versions, err := journal.ReadRecordSourceVersions(context.Background())
	if err != nil || len(versions) != 105 {
		t.Fatal("version pages incomplete", err)
	}
	for i, version := range versions {
		if version.PoolID != i+1 || version.Counter != uint64(i+1) {
			t.Fatal("version snapshot scope/order changed")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := journal.ReadRecordSourceVersions(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("version cancellation lost", err)
	}
}
