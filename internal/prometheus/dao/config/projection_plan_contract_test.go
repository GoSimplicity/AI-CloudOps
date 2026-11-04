package config

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func rootEmptyProjectionPlan(ref PoolReservation) GeneratedProjectionPlanRequest {
	hash := [32]byte{0xa4}
	return GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash}, Controls: []ProjectionControl{
		{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}},
		{ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, RequiredControlIDs: []uint64{1}}},
	}}
}
func TestProjectionPlanContractV1RemainsHistoryWithoutFullControls(t *testing.T) {
	old, db := journalContractFixture(t)
	ctx := context.Background()
	ref, err := old.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.CommitGeneratedPlan(ctx, GeneratedPlanRequest{Reservation: ref, SourceHash: [32]byte{1}}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := NewProjectionPlanJournalDAO(zap.NewNop(), db).LoadProjectionByNonce(ctx, ref.Nonce)
	if err != nil || snapshot.Projection == nil || snapshot.Projection.ManifestVersion != 1 || snapshot.Projection.CompleteControls || len(snapshot.Projection.Controls) != 0 {
		t.Fatal("legacy empty manifest manufactured complete projection authority", err)
	}
}
func TestProjectionPlanContractEmptyV2SurvivesRebuildAndLegacyMarkCannotBypass(t *testing.T) {
	old, db := journalContractFixture(t)
	ctx := context.Background()
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ref, err := journal.ReservePoolOperation(ctx, journalContractRequest(1))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := journal.CommitGeneratedProjectionPlan(ctx, rootEmptyProjectionPlan(ref))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ManifestVersion != 2 || !receipt.CompleteControls || len(receipt.Artifacts) != 0 || len(receipt.Controls) != 2 {
		t.Fatal("empty v2 operation lost its durable controls")
	}
	if marked, err := old.MarkPoolProjectedIfCurrent(ctx, receipt.PoolReceipt); err == nil || marked {
		t.Fatal("legacy artifact-only marker bypassed v2 control completion")
	}
	rebuilt := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	loaded, err := rebuilt.LoadLastCommittedProjection(ctx, ref.Family, ref.PoolID)
	if err != nil || loaded == nil || !reflect.DeepEqual(*loaded, receipt) {
		t.Fatal("rebuild did not load the exact issued control manifest", err)
	}
	var record model.MonitorConfigPoolOperation
	if db.First(&record, "operation_id=?", ref.OperationID).Error != nil || record.Manifest == nil || *record.Manifest == "[]" {
		t.Fatal("v2 controls were kept only in caller memory")
	}
}
func TestProjectionPlanContractRejectsIncompleteAndCrossFamilyControls(t *testing.T) {
	for _, mode := range []string{"missing members", "missing control proof", "wrong legacy hash", "webhook subset touches full members"} {
		t.Run(mode, func(t *testing.T) {
			_, db := journalContractFixture(t)
			ctx := context.Background()
			journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			reserve := journalContractRequest(1)
			if mode == "webhook subset touches full members" {
				reserve.Family = model.ConfigPublicationFamilyAlertManagerWebhook
			}
			ref, err := journal.ReservePoolOperation(ctx, reserve)
			if err != nil {
				t.Fatal(err)
			}
			request := rootEmptyProjectionPlan(ref)
			switch mode {
			case "missing members":
				request.Controls = request.Controls[1:]
			case "missing control proof":
				request.Controls[1].Ack.RequiredControlIDs = nil
			case "wrong legacy hash":
				bad := [32]byte{0xff}
				request.Controls[1].Ack.LegacySHA = &bad
			case "webhook subset touches full members":
				request.Controls[1].Kind = ProjectionACKWebhookSubset
				request.Controls[1].Ack.LegacySHA = nil
			}
			if _, err := journal.CommitGeneratedProjectionPlan(ctx, request); err == nil {
				t.Fatal("invalid projection control plan was committed")
			}
			snapshot, err := journal.LoadOperationByNonce(ctx, ref.Nonce)
			if err != nil || snapshot.Receipt != nil || snapshot.SourceHash != nil {
				t.Fatal("rejected controls left a committed manifest", err)
			}
		})
	}
}
