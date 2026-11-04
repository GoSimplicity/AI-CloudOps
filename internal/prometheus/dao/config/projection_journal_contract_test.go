package config

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func journalContractFixture(t *testing.T) (ProjectionJournalDAO, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.MonitorConfig{}, &model.MonitorConfigNamespaceGuard{}, &model.MonitorConfigPoolControl{}, &model.MonitorConfigPoolOperation{}, &model.MonitorConfigArtifactEvent{}); err != nil {
		t.Fatal(err)
	}
	return NewProjectionJournalDAO(zap.NewNop(), db), db
}
func journalContractRequest(n byte) PoolOperationRequest {
	return PoolOperationRequest{Family: model.ConfigPublicationFamilyRecordRule, PoolID: 7, Nonce: [16]byte{n}}
}
func TestJournalContractNonceIsReservationIdentity(t *testing.T) {
	for _, mode := range []string{"same request", "different pool", "different family", "different recovery"} {
		t.Run(mode, func(t *testing.T) {
			dao, db := journalContractFixture(t)
			ctx := context.Background()
			request := journalContractRequest(1)
			first, err := dao.ReservePoolOperation(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			if first.OperationID == 0 || first.Generation != 1 {
				t.Fatal("first reservation lacks a durable identity")
			}
			next := request
			switch mode {
			case "different pool":
				next.PoolID++
			case "different family":
				next.Family = model.ConfigPublicationFamilyPrometheusMain
			case "different recovery":
				old := first.OperationID
				next.RecoveryOfOperationID = &old
			}
			second, err := dao.ReservePoolOperation(ctx, next)
			if mode == "same request" {
				if err != nil || second != first {
					t.Fatal("nonce retry allocated a different reservation", err)
				}
			} else if err == nil {
				t.Fatal("nonce was reused for a different request")
			}
			var count int64
			if db.Model(&model.MonitorConfigPoolOperation{}).Count(&count).Error != nil || count != 1 {
				t.Fatal("nonce retry appended an extra operation")
			}
			var control model.MonitorConfigPoolControl
			if db.Where("family=? AND pool_id=?", first.Family, first.PoolID).First(&control).Error != nil || control.ReservedGeneration != 1 {
				t.Fatal("nonce conflict consumed a generation")
			}
			if control.LastCommittedOperationID != nil || control.LastProjectedOperationID != nil {
				t.Fatal("reservation manufactured a last-good receipt")
			}
		})
	}
}
func TestJournalContractEmptyLastGoodSurvivesFailedNewAttemptAndRebuild(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	hashA := [32]byte{0xa1}
	hashB := [32]byte{0xb2}
	first, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	var reserved model.MonitorConfigPoolOperation
	if db.First(&reserved, "operation_id=?", first.OperationID).Error != nil || reserved.Manifest != nil || reserved.CommittedAt != nil {
		t.Fatal("reserved manifest is not SQL NULL")
	}
	receipt, err := dao.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: first, SourceHash: hashA})
	if err != nil {
		t.Fatal(err)
	}
	if len(receipt.Artifacts) != 0 {
		t.Fatal("empty operation invented an artifact")
	}
	var committed model.MonitorConfigPoolOperation
	if db.First(&committed, "operation_id=?", first.OperationID).Error != nil || committed.Manifest == nil || committed.CommittedAt == nil {
		t.Fatal("empty operation lost its committed proof")
	}
	var items []json.RawMessage
	if json.Unmarshal([]byte(*committed.Manifest), &items) != nil || items == nil || len(items) != 0 {
		t.Fatal("committed empty manifest is not JSON []")
	}
	// An empty operation has no artifact deliveries. This internal call represents
	// the later caller's confirmed ACK; it does not pretend to exercise Redis.
	marked, err := dao.MarkPoolProjectedIfCurrent(ctx, receipt)
	if err != nil || !marked {
		t.Fatal("empty committed operation cannot record caller-confirmed projection", err)
	}
	next, err := dao.ReservePoolOperation(ctx, journalContractRequest(2))
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := dao.CommitGeneratedPlan(cancelled, GeneratedPlanRequest{Reservation: next, SourceHash: hashB}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled next attempt lost its cancellation identity")
	}
	rebuilt := NewProjectionJournalDAO(zap.NewNop(), db)
	for _, load := range []func(context.Context, int8, int) (*PoolReceipt, error){rebuilt.LoadLastCommitted, rebuilt.LoadLastProjected} {
		last, err := load(ctx, first.Family, first.PoolID)
		if err != nil || last == nil || last.Reservation.OperationID != first.OperationID || last.SourceHash != hashA || len(last.Artifacts) != 0 {
			t.Fatal("failed new attempt erased the previous empty last-good receipt", err)
		}
	}
	snapshot, err := rebuilt.LoadOperationByNonce(ctx, next.Nonce)
	if err != nil || snapshot.Receipt != nil || snapshot.SourceHash != nil || snapshot.Phase != model.ConfigPublicationReserved {
		t.Fatal("failed attempt was falsely committed/projected", err)
	}
	recovery := journalContractRequest(3)
	id := first.OperationID
	recovery.RecoveryOfOperationID = &id
	third, err := rebuilt.ReservePoolOperation(ctx, recovery)
	if err != nil || third.Generation != 3 {
		t.Fatal("recovery rewound or skipped the durable counter", err)
	}
	restored, err := rebuilt.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: third, SourceHash: hashA})
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := rebuilt.MarkPoolProjectedIfCurrent(ctx, restored); err != nil || !ok {
		t.Fatal("fresh recovery receipt cannot advance last-good", err)
	}
	last, err := rebuilt.LoadLastProjected(ctx, first.Family, first.PoolID)
	if err != nil || last == nil || last.Reservation.OperationID != third.OperationID || last.SourceHash != hashA {
		t.Fatal("recovery did not retain the chosen historical source")
	}
}
func TestJournalContractCommittedPayloadCannotBeRewritten(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	r, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	first, err := dao.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: r, SourceHash: [32]byte{1}})
	if err != nil {
		t.Fatal(err)
	}
	var before model.MonitorConfigPoolOperation
	if db.First(&before, "operation_id=?", r.OperationID).Error != nil {
		t.Fatal("committed operation missing")
	}
	_, err = dao.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: r, SourceHash: [32]byte{2}})
	if err == nil {
		t.Fatal("committed receipt accepted a different source payload")
	}
	var after model.MonitorConfigPoolOperation
	if db.First(&after, "operation_id=?", r.OperationID).Error != nil {
		t.Fatal("operation disappeared")
	}
	if !reflect.DeepEqual(before.SourceHash, after.SourceHash) || before.Manifest == nil || after.Manifest == nil || *before.Manifest != *after.Manifest {
		t.Fatal("conflicting retry rewrote committed evidence")
	}
	loaded, err := dao.LoadLastCommitted(ctx, r.Family, r.PoolID)
	if err != nil || loaded == nil || loaded.SourceHash != first.SourceHash {
		t.Fatal("conflict displaced last committed payload")
	}
}

func journalContractRender(ref PoolReservation) (*model.MonitorConfig, GeneratedPlanRequest) {
	row := &model.MonitorConfig{PoolID: 7, InstanceIP: "192.0.2.7", ConfigType: model.ConfigTypeRecordRule, Name: "prometheus_record_rule_7_192.0.2.7.yaml", ConfigContent: "groups: []\n", Status: model.ConfigStatusActive, LastGeneratedTime: 1}
	request := GeneratedPlanRequest{Reservation: ref, SourceHash: [32]byte{0xa7}, Items: []GeneratedPlanItem{{SourceID: 71, Action: model.ConfigProjectionRender, Desired: row, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: model.ConfigTypeRecordRule, InstanceIP: row.InstanceIP}}}}
	return row, request
}
func TestJournalContractCreateReceiptRetryRetainsAssignedIdentity(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	ref, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	input, request := journalContractRender(ref)
	first, err := dao.CommitGeneratedPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if input.ID <= 0 || len(first.Artifacts) != 1 || first.Artifacts[0].ConfigRowID == nil || *first.Artifacts[0].ConfigRowID != input.ID || first.Artifacts[0].SourceID != 71 {
		t.Fatal("committed create did not return its durable SourceID/row identity")
	}
	second, err := dao.CommitGeneratedPlan(ctx, request)
	if err != nil || len(second.Artifacts) != 1 || second.Artifacts[0].EventID != first.Artifacts[0].EventID {
		t.Fatal("retry with committed input ID changed the immutable receipt", err)
	}
	for _, table := range []interface{}{&model.MonitorConfig{}, &model.MonitorConfigArtifactEvent{}} {
		var count int64
		if db.Model(table).Count(&count).Error != nil || count != 1 {
			t.Fatal("committed retry duplicated metadata or events")
		}
	}
}
func TestJournalContractOrdinaryChangeInvalidatesOldArtifactReceipt(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	ref, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	input, request := journalContractRender(ref)
	receipt, err := dao.CommitGeneratedPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if err := NewMonitorConfigDAO(zap.NewNop(), db).UpdateMonitorConfig(ctx, &model.MonitorConfig{Model: model.Model{ID: input.ID}, Name: "operator-owned"}); err != nil {
		t.Fatal(err)
	}
	var latest model.MonitorConfigArtifactEvent
	if db.Order("event_id DESC").First(&latest).Error != nil || latest.EventID <= receipt.Artifacts[0].EventID || latest.ProjectionPolicy != model.ConfigProjectionLineageOnly || latest.BindingEventID == nil || *latest.BindingEventID != receipt.Artifacts[0].EventID {
		t.Fatal("ordinary mutation did not retain committed binding lineage")
	}
	marked, err := dao.MarkArtifactDeliveryIfCurrent(ctx, receipt.Artifacts[0], model.ConfigProjectionApplied)
	if marked || err != nil && !errors.Is(err, ErrJournalConflict) {
		t.Fatal("obsolete artifact receipt was accepted or returned an unrelated error", err)
	}
	var old model.MonitorConfigArtifactEvent
	if db.First(&old, "event_id=?", receipt.Artifacts[0].EventID).Error != nil || old.DeliveryState == model.ConfigProjectionApplied {
		t.Fatal("stale receipt marked an older artifact applied")
	}
}

func TestJournalContractBusinessAndEventsRollbackTogether(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	ref, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	// This fixture constraint forces the second otherwise-valid create to fail
	// after the first metadata row and its event were actually inserted.
	if db.Exec("CREATE UNIQUE INDEX fixture_one_config_per_pool ON cl_monitor_configs(pool_id)").Error != nil {
		t.Fatal("fixture constraint failed")
	}
	insertedMetadata, insertedEvents := int64(0), int64(0)
	if err := db.Callback().Create().After("gorm:create").Register("fixture:journal-insert-observer", func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Schema == nil {
			return
		}
		switch tx.Statement.Schema.Table {
		case "cl_monitor_configs":
			insertedMetadata += tx.RowsAffected
		case "cl_monitor_config_artifact_events":
			insertedEvents += tx.RowsAffected
		}
	}); err != nil {
		t.Fatal(err)
	}
	first, request := journalContractRender(ref)
	second := *first
	second.InstanceIP = "192.0.2.8"
	second.Name = "prometheus_record_rule_7_192.0.2.8.yaml"
	request.Items = append(request.Items, GeneratedPlanItem{SourceID: 72, Action: model.ConfigProjectionRender, Desired: &second, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: model.ConfigTypeRecordRule, InstanceIP: second.InstanceIP}})
	firstBefore, secondBefore := *first, second
	receipt, err := dao.CommitGeneratedPlan(ctx, request)
	if err == nil {
		t.Fatal("real second SQL failure was accepted")
	}
	if insertedMetadata != 1 || insertedEvents != 1 {
		t.Fatal("fixture did not reach successful first metadata/event inserts before the real second SQL failure")
	}
	if receipt.Reservation.OperationID != 0 || len(receipt.Artifacts) != 0 || !reflect.DeepEqual(firstBefore, *first) || !reflect.DeepEqual(secondBefore, second) {
		t.Fatal("rolled-back plan leaked receipt, IDs or timestamps")
	}
	for _, table := range []interface{}{&model.MonitorConfig{}, &model.MonitorConfigArtifactEvent{}, &model.MonitorConfigNamespaceGuard{}} {
		var count int64
		if db.Model(table).Count(&count).Error != nil || count != 0 {
			t.Fatal("metadata, events or newly created guards survived rollback")
		}
	}
	var control model.MonitorConfigPoolControl
	if db.Where("family=? AND pool_id=?", ref.Family, ref.PoolID).First(&control).Error != nil || control.LastCommittedOperationID != nil || control.LastProjectedOperationID != nil {
		t.Fatal("failed business plan advanced last-good pointers")
	}
	snapshot, err := dao.LoadOperationByNonce(ctx, ref.Nonce)
	if err != nil || snapshot.Receipt != nil || snapshot.SourceHash != nil || snapshot.Phase != model.ConfigPublicationReserved {
		t.Fatal("failed business plan manufactured committed evidence")
	}
}

func TestJournalContractArtifactSourceIDMustMatchCommittedManifest(t *testing.T) {
	dao, db := journalContractFixture(t)
	ctx := context.Background()
	ref, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	_, request := journalContractRender(ref)
	receipt, err := dao.CommitGeneratedPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	forged := receipt.Artifacts[0]
	forged.SourceID++
	ok, err := dao.MarkArtifactDeliveryIfCurrent(ctx, forged, model.ConfigProjectionApplied)
	if ok || err != nil && !errors.Is(err, ErrJournalConflict) {
		t.Fatal("artifact receipt accepted a SourceID absent from its committed manifest", err)
	}
	var event model.MonitorConfigArtifactEvent
	if db.First(&event, "event_id=?", forged.EventID).Error != nil || event.DeliveryState != model.ConfigProjectionPending {
		t.Fatal("forged receipt mutated delivery state")
	}
}

func TestJournalContractCreateRejectsUnsupportedStatus(t *testing.T) {
	for _, status := range []int8{-1, 3} {
		t.Run(map[int8]string{-1: "negative", 3: "outside enum"}[status], func(t *testing.T) {
			dao, db := journalContractFixture(t)
			ctx := context.Background()
			ref, err := dao.ReservePoolOperation(ctx, journalContractRequest(1))
			if err != nil {
				t.Fatal(err)
			}
			input, request := journalContractRender(ref)
			input.Status = status
			_, err = dao.CommitGeneratedPlan(ctx, request)
			if err == nil {
				t.Fatal("unsupported create status obtained a generated receipt")
			}
			var rows, events int64
			if db.Model(&model.MonitorConfig{}).Count(&rows).Error != nil || db.Model(&model.MonitorConfigArtifactEvent{}).Count(&events).Error != nil || rows != 0 || events != 0 || input.ID != 0 {
				t.Fatal("invalid create status changed metadata or events")
			}
			snapshot, err := dao.LoadOperationByNonce(ctx, ref.Nonce)
			if err != nil || snapshot.Receipt != nil || snapshot.SourceHash != nil {
				t.Fatal("invalid create status manufactured a committed manifest")
			}
		})
	}
}
