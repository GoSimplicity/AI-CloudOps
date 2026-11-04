package config

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestIndependentProjectionLeaseCallbackScope(t *testing.T) {
	_, db := journalContractFixture(t)
	j := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	ref, err := j.ReservePoolOperation(ctx, journalContractRequest(0xd1))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := j.CommitGeneratedProjectionPlan(ctx, rootEmptyProjectionPlan(ref))
	if err != nil {
		t.Fatal(err)
	}
	var escaped ProjectionLease
	err = j.WithPoolProjectionLease(ctx, receipt, func(l ProjectionLease) error {
		escaped = l
		s, err := l.Snapshot()
		if err != nil || s.Pool == nil {
			t.Fatal("snapshot unavailable", err)
		}
		s.Pool.Controls = nil
		if err := l.RecheckCurrent(ctx); err != nil {
			t.Fatal("snapshot mutation escaped", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal("single-connection lease failed", err)
	}
	if _, err := escaped.Snapshot(); err == nil {
		t.Fatal("escaped snapshot authority survived callback")
	}
	if err := escaped.RecheckCurrent(ctx); err == nil {
		t.Fatal("escaped check authority survived callback")
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	err = j.WithPoolProjectionLease(cancelCtx, receipt, func(l ProjectionLease) error {
		cancel()
		if _, e := l.Snapshot(); !errors.Is(e, context.Canceled) {
			t.Fatal("canceled lease remained usable", e)
		}
		return context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("cancel classification lost", err)
	}
}

func TestIndependentProjectionPersistedIntentSourceProof(t *testing.T) {
	_, db := journalContractFixture(t)
	ctx := context.Background()
	ordinary := NewMonitorConfigDAO(zap.NewNop(), db)
	j := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	row := &model.MonitorConfig{Name: "webhook_synthetic_3_127.0.0.1", PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", ConfigContent: "synthetic", Status: 2}
	if err := ordinary.CreateMonitorConfig(ctx, row); err != nil {
		t.Fatal(err)
	}
	expected := GeneratedConfigOwnerOf(row)
	group := 3
	hash := [32]byte{0xb3}
	ref, err := j.ReservePoolOperation(ctx, PoolOperationRequest{Family: 2, PoolID: 7, Nonce: [16]byte{0xd2}})
	if err != nil {
		t.Fatal(err)
	}
	request := GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: []GeneratedPlanItem{{SourceID: 71, Action: model.ConfigProjectionRender, Expected: &expected, Desired: row, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 5, InstanceIP: "127.0.0.1", GroupID: &group}}}}, Controls: []ProjectionControl{
		{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{}}},
		{ControlID: 2, Kind: ProjectionWebhookIntents, Intents: &WebhookIntentsControl{Add: []IntentOwnerChange{{Owner: expected, Proof: ProjectionProofRef{SourceID: 71}}}, Remove: []IntentOwnerChange{}, RequireEmpty: true}},
		{ControlID: 3, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: []uint64{71}, RequiredControlIDs: []uint64{1, 2}}},
	}}
	issued, err := j.CommitGeneratedProjectionPlan(ctx, request)
	if err != nil {
		t.Fatal("valid plan rejected", err)
	}
	if err := j.WithPoolProjectionLease(ctx, issued, func(l ProjectionLease) error { return l.RecheckCurrent(ctx) }); err != nil {
		t.Fatal("valid source proof cannot lease", err)
	}
	var op model.MonitorConfigPoolOperation
	if db.First(&op, "operation_id=?", ref.OperationID).Error != nil {
		t.Fatal("op unavailable")
	}
	envelope, err := decodeProjectionManifest(op.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	envelope.Controls[1].Intents.Add[0].ProofSourceID = 999 // no such source in issued artifact vector
	envelope.Controls[1].Digest = issuedControlDigest(envelope.Controls[1])
	envelope.Controls[2].Ack.RequiredControls[1].Digest = envelope.Controls[1].Digest
	envelope.Controls[2].Digest = issuedControlDigest(envelope.Controls[2])
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	if db.Model(&op).Update("manifest", string(raw)).Error != nil {
		t.Fatal("corrupt persistence fixture failed")
	}
	loaded, err := j.LoadLastCommittedProjection(ctx, 2, 7)
	if err != nil {
		return
	} // rejected malformed stored source mapping
	callback := false
	leaseErr := j.WithPoolProjectionLease(ctx, *loaded, func(ProjectionLease) error { callback = true; return nil })
	if callback || leaseErr == nil {
		t.Fatal("persistent intent sourceID absent from manifest was accepted as complete projection authority")
	}
}

func TestIndependentProjectionOrdinaryWriteInvalidatesPoolLease(t *testing.T) {
	_, db := journalContractFixture(t)
	ctx := context.Background()
	j := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ref, err := j.ReservePoolOperation(ctx, journalContractRequest(0xd3))
	if err != nil {
		t.Fatal(err)
	}
	row, base := journalContractRender(ref)
	hash := base.SourceHash
	req := GeneratedProjectionPlanRequest{GeneratedPlanRequest: base, Controls: []ProjectionControl{
		{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: []string{row.InstanceIP}}},
		{ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: []uint64{71}, RequiredControlIDs: []uint64{1}}},
	}}
	receipt, err := j.CommitGeneratedProjectionPlan(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	ordinary := NewMonitorConfigDAO(zap.NewNop(), db)
	row.ConfigContent = "ordinary synthetic edit"
	if err := ordinary.UpdateMonitorConfig(ctx, row); err != nil {
		t.Fatal(err)
	}
	callback := false
	err = j.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { callback = true; return nil })
	if err == nil || callback {
		t.Fatal("ordinary same-namespace mutation did not revoke old pool lease")
	}
	var event model.MonitorConfigArtifactEvent
	if db.Order("event_id DESC").First(&event).Error != nil {
		t.Fatal("ordinary event unavailable")
	}
	artifact, err := artifactReceipt(event, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = j.WithArtifactProjectionLease(ctx, artifact, func(l ProjectionLease) error {
		snapshot, err := l.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Pool != nil || len(snapshot.Artifacts) != 1 || len(snapshot.Artifacts[0].Body) != 0 || snapshot.Artifacts[0].Binding.ConfigType != 4 {
			t.Fatal("ordinary invalidation acquired render/pool authority")
		}
		return nil
	})
	if err != nil {
		t.Fatal("valid ordinary lineage cannot acquire scoped invalidation lease", err)
	}
	// Any MarkComplete error seals the handle, including invalid caller proofs.
	// A fresh lease remains available; a rejected handle must not be reused.
	err = j.WithArtifactProjectionLease(ctx, artifact, func(l ProjectionLease) error {
		if ok, err := l.MarkComplete(ProjectionAckProof{}); err == nil || ok {
			t.Fatal("ordinary invalidation acquired pool completion authority")
		}
		if _, err := l.Snapshot(); err == nil {
			t.Fatal("rejected pool completion left ordinary lease usable")
		}
		return nil
	})
	if err == nil {
		t.Fatal("callback swallowed a sealed lease error")
	}
	if err := j.WithArtifactProjectionLease(ctx, artifact, func(l ProjectionLease) error { return l.RecheckCurrent(ctx) }); err != nil {
		t.Fatal("sealed handle prevented an independent valid lease retry", err)
	}
}
