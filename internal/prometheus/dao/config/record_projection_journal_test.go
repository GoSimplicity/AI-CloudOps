package config

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func recordGuardFixture(t *testing.T, tombstone bool) (RecordProjectionJournalDAO, *monitorConfigDAO, RecordProjectionPlanRequest) {
	t.Helper()
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewRecordProjectionJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	row := journalBehaviorRender(11, "127.0.0.1").Desired
	if err := ordinary.CreateMonitorConfig(ctx, row); err != nil {
		t.Fatal(err)
	}
	if tombstone {
		if err := ordinary.DeleteMonitorConfig(ctx, row.ID); err != nil {
			t.Fatal(err)
		}
	}
	var history []*model.MonitorConfig
	if db.Unscoped().Order("id").Find(&history).Error != nil {
		t.Fatal("history fixture unreadable")
	}
	proof, err := RecordHistoryGuardOf(row.InstanceIP, history)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	owner := GeneratedConfigOwnerOf(row)
	item := GeneratedPlanItem{SourceID: 11, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: row.InstanceIP}}
	hash := [32]byte{1}
	request := RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: []GeneratedPlanItem{item}}, Controls: []ProjectionControl{
		{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}},
		{ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: []uint64{11}, RequiredControlIDs: []uint64{1}}},
	}}, History: []RecordHistoryGuard{proof}}
	return journal, ordinary, request
}
func TestRecordHistoryGuardRejectsForeignCreateDeleteABAInSameTransaction(t *testing.T) {
	for _, tombstone := range []bool{false, true} {
		t.Run(map[bool]string{false: "active snapshot", true: "tombstone snapshot"}[tombstone], func(t *testing.T) {
			journal, ordinary, request := recordGuardFixture(t, tombstone)
			ctx := context.Background()
			db := ordinary.db
			if !tombstone {
				if err := ordinary.DeleteMonitorConfig(ctx, request.Items[0].Expected.ID); err != nil {
					t.Fatal(err)
				}
			}
			foreign := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: "127.0.0.1", Name: "custom foreign", Status: 2}
			if err := ordinary.CreateMonitorConfig(ctx, foreign); err != nil {
				t.Fatal(err)
			}
			if err := ordinary.DeleteMonitorConfig(ctx, foreign.ID); err != nil {
				t.Fatal(err)
			}
			var before int64
			if db.Model(&model.MonitorConfigArtifactEvent{}).Count(&before).Error != nil {
				t.Fatal("event count unavailable")
			}
			receipt, err := journal.CommitRecordProjectionPlan(ctx, request)
			if !errors.Is(err, ErrJournalConflict) || receipt.Reservation.OperationID != 0 {
				t.Fatal("foreign create/delete ABA crossed history guard", err)
			}
			var after int64
			db.Model(&model.MonitorConfigArtifactEvent{}).Count(&after)
			if after != before {
				t.Fatal("rejected history emitted a new artifact event")
			}
			snapshot, err := journal.LoadOperationByNonce(ctx, request.Reservation.Nonce)
			if err != nil || snapshot.Receipt != nil || snapshot.SourceHash != nil {
				t.Fatal("rejected history committed manifest", err)
			}
		})
	}
}
func TestRecordHistoryGuardIdempotentReceiptPrecedesSelfMutationCheck(t *testing.T) {
	journal, ordinary, request := recordGuardFixture(t, false)
	ctx := context.Background()
	first, err := journal.CommitRecordProjectionPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := journal.CommitRecordProjectionPlan(ctx, request)
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("own soft-delete invalidated same signed payload", err)
	}
	changed := request
	changed.History = append([]RecordHistoryGuard(nil), request.History...)
	changed.History[0].Rows = append([]GeneratedConfigHistoryRow(nil), request.History[0].Rows...)
	changed.History[0].Rows[0].DeletedAt = 1
	if _, err := journal.CommitRecordProjectionPlan(ctx, changed); !errors.Is(err, ErrJournalConflict) {
		t.Fatal("same nonce changed history identity", err)
	}
	nextRef, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(2))
	if err != nil {
		t.Fatal(err)
	}
	request.Reservation = nextRef
	if _, err := journal.CommitRecordProjectionPlan(ctx, request); !errors.Is(err, ErrJournalConflict) {
		t.Fatal("new operation ignored stale deletion state", err)
	}
	var rows []*model.MonitorConfig
	ordinary.db.Unscoped().Find(&rows)
	if len(rows) != 1 || rows[0].DeletedAt == 0 {
		t.Fatal("retirement tombstone was restored or duplicated")
	}
}

func TestRecordCompletedRetirementProofSurvivesLaterEmptyOperationAndNewOwner(t *testing.T) {
	journal, ordinary, request := recordGuardFixture(t, false)
	ctx := context.Background()
	history := NewGeneratedConfigHistoryReader(zap.NewNop(), ordinary.db)
	owner := *request.Items[0].Expected
	receipt, err := journal.CommitRecordProjectionPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(completed) != 0 {
		t.Fatal("unprojected retirement became complete", err)
	}
	proof, err := projectionAckProof(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, receipt, proof); err != nil || !ok {
		t.Fatal("caller-confirmed retirement not marked", err)
	}
	nextRef, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(2))
	if err != nil {
		t.Fatal(err)
	}
	hash := [32]byte{2}
	empty, err := journal.CommitRecordProjectionPlan(ctx, RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: nextRef, SourceHash: hash}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, RequiredControlIDs: []uint64{1}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	proof, _ = projectionAckProof(empty)
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, empty, proof); err != nil || !ok {
		t.Fatal(err)
	}
	foreign := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: owner.InstanceIP, Name: "new foreign owner", ConfigContent: "must preserve", Status: 2}
	if err := ordinary.CreateMonitorConfig(ctx, foreign); err != nil {
		t.Fatal(err)
	}
	completed, err = history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(completed) != 1 || completed[0] != owner {
		t.Fatal("completed historical owner was rediscovered as pending", err)
	}
	wrong := owner
	wrong.Name = "different owner snapshot"
	completed, err = history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{wrong})
	if err != nil || len(completed) != 0 {
		t.Fatal("wrong owner digest borrowed completion proof", err)
	}
	current, err := ordinary.GetMonitorConfigByID(ctx, foreign.ID)
	if err != nil || current.ConfigContent != "must preserve" {
		t.Fatal("read-only proof lookup changed current owner", err)
	}
}

func TestRecordAcceptedRetirementSupersededByProjectedReplacementPersistsAcrossRecovery(t *testing.T) {
	for _, mode := range []string{"retire", "pending render", "projected render"} {
		t.Run(mode, func(t *testing.T) {
			journal, ordinary, request := recordGuardFixture(t, false)
			ctx := context.Background()
			if mode != "retire" {
				row, err := ordinary.GetMonitorConfigByID(ctx, request.Items[0].Expected.ID)
				if err != nil {
					t.Fatal(err)
				}
				row.ConfigContent = "A accepted render"
				row.ConfigHash = ""
				request.Items[0].Action = model.ConfigProjectionRender
				request.Items[0].Desired = row
				request.History = nil
				request.Controls[0].Members.DesiredIPs = []string{row.InstanceIP}
			}
			history := NewGeneratedConfigHistoryReader(zap.NewNop(), ordinary.db)
			pending, err := journal.CommitRecordProjectionPlan(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			if mode != "retire" {
				if mode == "projected render" {
					proof, _ := projectionAckProof(pending)
					if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, pending, proof); err != nil || !ok {
						t.Fatal(err)
					}
					loaded, err := journal.LoadLastCommittedProjection(ctx, 4, 7)
					if err != nil || loaded == nil {
						t.Fatal(err)
					}
					pending = *loaded
				}
				if err := ordinary.DeleteMonitorConfig(ctx, request.Items[0].Expected.ID); err != nil {
					t.Fatal(err)
				}
			}
			bRef, err := journal.ReservePoolOperation(ctx, PoolOperationRequest{Family: 4, PoolID: 99, Nonce: [16]byte{9}})
			if err != nil {
				t.Fatal(err)
			}
			bRow := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: "127.0.0.1", Name: "prometheus_record_rule_99_127.0.0.1.yaml", Status: 1, ConfigContent: "B original"}
			bHash := [32]byte{9}
			bRequest := RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: bRef, SourceHash: bHash, Items: []GeneratedPlanItem{{SourceID: 9, Action: model.ConfigProjectionRender, Desired: bRow, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: bRow.InstanceIP}}}}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{bRow.InstanceIP}}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &bHash, ArtifactSourceIDs: []uint64{9}, RequiredControlIDs: []uint64{1}}}}}}
			b, err := journal.CommitRecordProjectionPlan(ctx, bRequest)
			if err != nil {
				t.Fatal(err)
			}
			proofs, err := history.GetSupersededRecordArtifacts(ctx, pending)
			if err != nil || len(proofs) != 0 {
				t.Fatal("unprojected B received supersede permission", err)
			}
			ack, _ := projectionAckProof(b)
			if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, b, ack); err != nil || !ok {
				t.Fatal(err)
			}
			proofs, err = history.GetSupersededRecordArtifacts(ctx, pending)
			if err != nil || len(proofs) != 1 {
				t.Fatal("projected B did not resolve accepted A retirement", err)
			}
			oldID := pending.Reservation.OperationID
			recoveryRef, err := journal.ReservePoolOperation(ctx, PoolOperationRequest{Family: 4, PoolID: 7, Nonce: [16]byte{2}, RecoveryOfOperationID: &oldID})
			if err != nil {
				t.Fatal(err)
			}
			recoveryRequest := RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: recoveryRef, SourceHash: pending.SourceHash}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &pending.SourceHash, RequiredControlIDs: []uint64{1}}}}}, Superseded: proofs}
			recovered, err := journal.CommitRecordProjectionPlan(ctx, recoveryRequest)
			if err != nil || len(recovered.Artifacts) != 0 {
				t.Fatal("control-only recovery not committed", err)
			}
			oldOwner := *request.Items[0].Expected
			resolved, err := history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{oldOwner})
			if err != nil || len(resolved) != 1 {
				t.Fatal("committed recovery lost raw-retirement resolution before ACK", err)
			}
			// B can change normally after the decision. Historical immutable completion
			// remains valid; resolving A must not pin B's row/body forever.
			changed := *bRow
			changed.ConfigContent = "B normal later update"
			if err := ordinary.UpdateMonitorConfig(ctx, &changed); err != nil {
				t.Fatal(err)
			}
			rebuilt := NewRecordProjectionJournalDAO(zap.NewNop(), ordinary.db)
			loaded, err := rebuilt.LoadLastCommittedProjection(ctx, 4, 7)
			if err != nil || loaded == nil || len(loaded.Artifacts) != 0 {
				t.Fatal("rebuild forgot accepted supersede after B changed", err)
			}
			ack, _ = projectionAckProof(*loaded)
			if ok, err := rebuilt.MarkProjectionCompleteIfCurrent(ctx, *loaded, ack); err != nil || !ok {
				t.Fatal("A recovery ACK became dependent on B's future body", err)
			}
			resolved, err = history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{oldOwner})
			if err != nil || len(resolved) != 1 {
				t.Fatal("future A cycle forgot durable supersede", err)
			}
			current, err := ordinary.GetMonitorConfigByID(ctx, bRow.ID)
			if err != nil || current.ConfigContent != changed.ConfigContent {
				t.Fatal("A recovery modified B", err)
			}

		})
	}
}

func TestRecordLegacyOwnerAdoptionHasNoInventedAcceptedIdentity(t *testing.T) {
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewRecordProjectionJournalDAO(zap.NewNop(), db)
	history := NewGeneratedConfigHistoryReader(zap.NewNop(), db)
	ctx := context.Background()
	a := journalBehaviorRender(11, "127.0.0.1").Desired
	if err := ordinary.CreateMonitorConfig(ctx, a); err != nil {
		t.Fatal(err)
	}
	owner := GeneratedConfigOwnerOf(a)
	if err := ordinary.DeleteMonitorConfig(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&model.MonitorConfigPoolOperation{}).Count(&count)
	if count != 0 {
		t.Fatal("legacy owner fixture accidentally had a pool journal operation")
	}
	bRef, err := journal.ReservePoolOperation(ctx, PoolOperationRequest{Family: 4, PoolID: 99, Nonce: [16]byte{9}})
	if err != nil {
		t.Fatal(err)
	}
	row := &model.MonitorConfig{PoolID: 99, ConfigType: 4, InstanceIP: owner.InstanceIP, Name: "prometheus_record_rule_99_127.0.0.1.yaml", ConfigContent: "new B", Status: 1}
	hash := [32]byte{9}
	b, err := journal.CommitRecordProjectionPlan(ctx, RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: bRef, SourceHash: hash, Items: []GeneratedPlanItem{{SourceID: 9, Action: model.ConfigProjectionRender, Desired: row, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 4, InstanceIP: row.InstanceIP}}}}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{row.InstanceIP}}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: []uint64{9}, RequiredControlIDs: []uint64{1}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := history.GetLegacySupersededRecordArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proofs) != 0 {
		t.Fatal("unprojected B authorized legacy adoption", err)
	}
	proof, _ := projectionAckProof(b)
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, b, proof); err != nil || !ok {
		t.Fatal(err)
	}
	proofs, err = history.GetLegacySupersededRecordArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proofs) != 1 || proofs[0].Origin != RecordSupersededLegacyOwner || proofs[0].AcceptedOperationID != 0 || proofs[0].AcceptedEventID != 0 || proofs[0].AcceptedAction != "" {
		t.Fatal("legacy proof invented accepted identity", err)
	}
	ref, err := journal.ReservePoolOperation(ctx, journalBehaviorRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	off := [32]byte{1}
	recovered, err := journal.CommitRecordProjectionPlan(ctx, RecordProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: off}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &off, RequiredControlIDs: []uint64{1}}}}}, Superseded: proofs})
	if err != nil || len(recovered.Artifacts) != 0 {
		t.Fatal("first journal adoption did not sign control-only state", err)
	}
	resolved, err := history.GetProjectedRecordRetirements(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(resolved) != 1 {
		t.Fatal("legacy adoption without old event was not durably indexed", err)
	}
	changed := *row
	changed.ConfigContent = "later B change"
	if err := ordinary.UpdateMonitorConfig(ctx, &changed); err != nil {
		t.Fatal(err)
	}
	loaded, err := journal.LoadLastCommittedProjection(ctx, 4, 7)
	if err != nil || loaded == nil {
		t.Fatal("legacy proof was revoked by B's later update", err)
	}
	proof, _ = projectionAckProof(*loaded)
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, *loaded, proof); err != nil || !ok {
		t.Fatal(err)
	}
}

func TestRecordLiveOwnerRetiresWhileDeletedForeignHistoryRemainsObserved(t *testing.T) {
	for _, mutate := range []bool{false, true} {
		t.Run(map[bool]string{false: "unchanged deleted observation", true: "changed deleted observation"}[mutate], func(t *testing.T) {
			journal, ordinary, request := recordGuardFixture(t, false)
			ctx := context.Background()
			foreign := &model.MonitorConfig{Model: model.Model{DeletedAt: 7}, PoolID: 99, ConfigType: 4, InstanceIP: request.Items[0].Expected.InstanceIP, Name: "deleted custom historical owner", ConfigContent: "must remain", Status: 2}
			if ordinary.db.Create(foreign).Error != nil {
				t.Fatal("historical fixture failed")
			}
			var rows []*model.MonitorConfig
			ordinary.db.Unscoped().Order("id ASC").Find(&rows)
			guard, err := RecordHistoryGuardOf(foreign.InstanceIP, rows)
			if err != nil {
				t.Fatal(err)
			}
			request.History = []RecordHistoryGuard{guard}
			if mutate {
				if ordinary.db.Unscoped().Model(&model.MonitorConfig{}).Where("id = ?", foreign.ID).Update("config_content", "changed after snapshot").Error != nil {
					t.Fatal("history mutation failed")
				}
			}
			receipt, err := journal.CommitRecordProjectionPlan(ctx, request)
			if mutate {
				if !errors.Is(err, ErrJournalConflict) || receipt.Reservation.OperationID != 0 {
					t.Fatal("changed observation bypassed full history guard", err)
				}
				return
			}
			if err != nil {
				t.Fatal("current unique canonical owner could not retire", err)
			}
			var retained model.MonitorConfig
			if ordinary.db.Unscoped().First(&retained, foreign.ID).Error != nil || retained.DeletedAt != 7 || retained.ConfigContent != "must remain" || retained.Name != foreign.Name {
				t.Fatal("retirement modified observed foreign history")
			}
		})
	}
}
