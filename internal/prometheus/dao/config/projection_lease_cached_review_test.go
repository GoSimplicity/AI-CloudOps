package config

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func cachedReviewRecord(t *testing.T) (*monitorConfigDAO, ProjectionPlanJournalDAO, ProjectionReceipt, *model.MonitorConfig) {
	t.Helper()
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewProjectionPlanJournalDAO(zap.NewNop(), db)
	ref, err := journal.ReservePoolOperation(context.Background(), journalBehaviorRequest(91))
	if err != nil {
		t.Fatal(err)
	}
	item := journalBehaviorRender(17, "127.0.0.191")
	receipt, err := journal.CommitGeneratedProjectionPlan(context.Background(), v2RecordPlan(ref, *item))
	if err != nil {
		t.Fatal(err)
	}
	return ordinary, journal, receipt, item.Desired
}
func assertCachedReviewSealed(t *testing.T, lease ProjectionLease, proof ProjectionAckProof) {
	t.Helper()
	if _, err := lease.Snapshot(); err == nil {
		t.Error("sealed lease exposed cached snapshot")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := lease.RecheckCurrent(ctx); err == nil {
		t.Error("new context renewed sealed cached lease")
	}
	if marked, err := lease.MarkComplete(proof); marked || err == nil {
		t.Error("sealed cached lease marked SQL projected")
	}
}

func TestCachedLeaseReviewOriginalTransactionEndCannotBorrowHealthyPool(t *testing.T) {
	ordinary, journal, receipt, row := cachedReviewRecord(t)
	proof, err := projectionAckProof(receipt)
	if err != nil {
		t.Fatal(err)
	}
	entered, detected := false, false
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(handle ProjectionLease) error {
		entered = true
		lease := handle.(*sqlProjectionLease)
		// End the actual original sql.Tx, releasing this max-open=1 connection.
		// A global-DB ping would now succeed, while the sealed original tx is dead.
		if lease.tx.Rollback().Error != nil {
			return errors.New("cannot end original review transaction")
		}
		changed, err := ordinary.GetMonitorConfigByID(context.Background(), row.ID)
		if err != nil {
			return err
		}
		changed.ConfigContent = "groups: [] # SQL committed after original tx ended\n"
		if err := ordinary.UpdateMonitorConfig(context.Background(), changed); err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = handle.RecheckCurrent(ctx)
		detected = err != nil
		if !detected {
			t.Error("live global pool substituted for the ended original transaction")
		}
		assertCachedReviewSealed(t, handle, proof)
		return err
	})
	if !entered || !detected || err == nil {
		t.Fatal("ended original transaction was not detected and sealed")
	}
	actual, err := ordinary.GetMonitorConfigByID(context.Background(), row.ID)
	if err != nil || actual.ConfigContent != "groups: [] # SQL committed after original tx ended\n" {
		t.Fatal("failed original lease changed the subsequent ordinary SQL write")
	}
}

func TestCachedLeaseReviewCallerMutationExitAndNewLeaseStayIndependent(t *testing.T) {
	ordinary, journal, receipt, row := cachedReviewRecord(t)
	pristine := cloneProjectionReceipt(receipt)
	proof, err := projectionAckProof(pristine)
	if err != nil {
		t.Fatal(err)
	}
	var escaped ProjectionLease
	err = journal.WithPoolProjectionLease(context.Background(), receipt, func(lease ProjectionLease) error {
		escaped = lease
		want, err := lease.Snapshot()
		if err != nil {
			return err
		}
		caller, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if len(caller.Artifacts) != 1 || len(caller.Artifacts[0].Body) == 0 {
			return errors.New("nonempty review material unavailable")
		}
		caller.Artifacts[0].Body[0] ^= 0xff
		caller.Artifacts[0].Binding.InstanceIP = "127.0.0.199"
		caller.Pool.Controls[0].Members.DesiredIPs[0] = "127.0.0.198"
		caller.Pool.Artifacts[0].BodyDigest[0] ^= 0xff
		receipt.SourceHash[0] ^= 0xff
		receipt.Artifacts[0].Binding.InstanceIP = "127.0.0.197"
		if err := lease.RecheckCurrent(context.Background()); err != nil {
			return err
		}
		got, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(want, got) {
			t.Error("caller altered the cached private material through a returned clone or input receipt")
		}
		return nil
	})
	if err != nil || escaped == nil {
		t.Fatal("valid private lease failed", err)
	}
	assertCachedReviewSealed(t, escaped, proof)
	changed, err := ordinary.GetMonitorConfigByID(context.Background(), row.ID)
	if err != nil {
		t.Fatal(err)
	}
	changed.ConfigContent = "groups: [] # new namespace revision\n"
	if ordinary.UpdateMonitorConfig(context.Background(), changed) != nil {
		t.Fatal("ordinary revision fixture failed")
	}
	called := false
	if err := journal.WithPoolProjectionLease(context.Background(), pristine, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
		t.Fatal("new acquisition inherited prior lease cache and skipped its stale initial proof")
	}
}

func TestCachedLeaseReviewOrdinaryCannotRenderOrCompleteAndErrorSeals(t *testing.T) {
	ordinary, journal, receipt, row := cachedReviewRecord(t)
	proof, err := projectionAckProof(receipt)
	if err != nil {
		t.Fatal(err)
	}
	row.ConfigContent = "groups: [] # ordinary owned update\n"
	if err := ordinary.UpdateMonitorConfig(context.Background(), row); err != nil {
		t.Fatal(err)
	}
	var event model.MonitorConfigArtifactEvent
	if ordinary.db.Order("event_id DESC").First(&event).Error != nil {
		t.Fatal("ordinary revision missing")
	}
	artifact, err := artifactReceipt(event, 0)
	if err != nil {
		t.Fatal(err)
	}
	entered, denied := false, false
	err = journal.WithArtifactProjectionLease(context.Background(), artifact, func(lease ProjectionLease) error {
		entered = true
		snapshot, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if snapshot.Pool != nil || len(snapshot.Artifacts) != 1 || len(snapshot.Artifacts[0].Body) != 0 || snapshot.Artifacts[0].Artifact.Action != model.ConfigProjectionInvalidate {
			return errors.New("ordinary acquisition acquired generated body or render authority")
		}
		snapshot.Artifacts[0].Body = []byte("caller supplied unauthorized body")
		snapshot.Artifacts[0].Artifact.Action = model.ConfigProjectionRender
		if err := lease.RecheckCurrent(context.Background()); err != nil {
			return err
		}
		again, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if len(again.Artifacts[0].Body) != 0 || again.Artifacts[0].Artifact.Action != model.ConfigProjectionInvalidate {
			t.Error("ordinary caller promoted cached private material")
		}
		marked, markErr := lease.MarkComplete(proof)
		denied = !marked && markErr != nil
		assertCachedReviewSealed(t, lease, proof)
		return markErr
	})
	if !entered || !denied || err == nil {
		t.Fatal("ordinary cached lease bypassed completion error sealing")
	}
}

func TestCachedLeaseReviewAMPeerChangeCannotReusePriorPairValidation(t *testing.T) {
	_, _, db := journalBehaviorFixture(t)
	journal := NewAMProjectionJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	request := journalBehaviorRequest(92)
	request.Family = 2
	ref, err := journal.ReserveAMPoolOperation(ctx, request, AMSourceVersion{PoolID: 7})
	if err != nil {
		t.Fatal(err)
	}
	full, err := journal.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: v2AMPlan(ref), SourceVersion: AMSourceVersion{PoolID: 7}})
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.WithPoolProjectionLease(ctx, full, func(lease ProjectionLease) error {
		before, err := lease.Snapshot()
		if err != nil {
			return err
		}
		clone, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if clone.Pool.AMControls == nil || clone.Pool.AMControls.FullOperationID == nil {
			return errors.New("AM pair proof absent")
		}
		*clone.Pool.AMControls.FullOperationID++
		if err := lease.RecheckCurrent(ctx); err != nil {
			return err
		}
		after, err := lease.Snapshot()
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(before, after) {
			t.Error("caller altered the cached AM control pair")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	source, err := NewProjectionRecoveryJournalDAO(zap.NewNop(), db).ReadAMPoolSourceVersion(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	request = journalBehaviorRequest(93)
	request.Family = 5
	subsetRef, err := journal.ReserveAMPoolOperation(ctx, request, source)
	if err != nil {
		t.Fatal(err)
	}
	subsetPlan := GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: subsetRef, Items: []GeneratedPlanItem{}}, Controls: []ProjectionControl{{ControlID: 3, Kind: ProjectionACKWebhookSubset, Ack: &ProjectionAckControl{ArtifactSourceIDs: []uint64{}, RequiredControlIDs: []uint64{}}}}}
	subset, err := journal.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: subsetPlan, SourceVersion: source})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	if err := journal.WithPoolProjectionLease(ctx, full, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
		t.Fatal("fresh full acquisition reused earlier cached peer validation")
	}
	if err := journal.WithPoolProjectionLease(ctx, subset, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) }); err != nil {
		t.Fatal("fresh current subset rejected because another lease had cached or failed", err)
	}
}
