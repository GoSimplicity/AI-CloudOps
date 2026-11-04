package config

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func TestAMJournalPairSourceAndCommittedVector(t *testing.T) {
	for _, mode := range []string{"peer committed invalidates source", "peer reserved preserves last good", "both directions source CAS"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			dao := NewAMProjectionJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			initial := AMSourceVersion{PoolID: 7}
			reserve := journalBehaviorRequest(1)
			reserve.Family = 2
			full, err := dao.ReserveAMPoolOperation(ctx, reserve, initial)
			if err != nil {
				t.Fatal(err)
			}
			receipt, err := dao.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: v2AMPlan(full), SourceVersion: initial})
			if err != nil {
				t.Fatal(err)
			}
			versions, err := dao.ReadAMSourceVersions(ctx)
			if err != nil || len(versions) != 1 {
				t.Fatal("pair version absent", err)
			}
			version := versions[0]
			reserve.Nonce = [16]byte{2}
			reserve.Family = 5
			subset, err := dao.ReserveAMPoolOperation(ctx, reserve, version)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "peer reserved preserves last good" {
				err = dao.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) })
				if err != nil {
					t.Fatal("failed/reserved peer starved committed full", err)
				}
				return
			}
			request := GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: subset}, Controls: []ProjectionControl{{ControlID: 3, Kind: ProjectionACKWebhookSubset, Ack: &ProjectionAckControl{ArtifactSourceIDs: []uint64{}, RequiredControlIDs: []uint64{}}}}}
			peer, err := dao.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: request, SourceVersion: version})
			if err != nil {
				t.Fatal(err)
			}
			called := false
			if err := dao.WithPoolProjectionLease(ctx, receipt, func(ProjectionLease) error { called = true; return nil }); err == nil || called {
				t.Fatal("old full ignored committed subset")
			}
			if err := dao.WithPoolProjectionLease(ctx, peer, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) }); err != nil {
				t.Fatal(err)
			}
			reserve.Nonce = [16]byte{3}
			reserve.Family = 2
			if _, err := dao.ReserveAMPoolOperation(ctx, reserve, version); err == nil {
				t.Fatal("stale pair snapshot reserved newer full")
			}
			if mode == "both directions source CAS" {
				versions, err = dao.ReadAMSourceVersions(ctx)
				if err != nil {
					t.Fatal(err)
				}
				reserve.Nonce = [16]byte{4}
				latest, err := dao.ReserveAMPoolOperation(ctx, reserve, versions[0])
				if err != nil {
					t.Fatal(err)
				}
				if _, err = dao.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: v2AMPlan(latest), SourceVersion: versions[0]}); err != nil {
					t.Fatal(err)
				}
				reserve.Nonce = [16]byte{5}
				reserve.Family = 5
				if _, err := dao.ReserveAMPoolOperation(ctx, reserve, versions[0]); err == nil {
					t.Fatal("stale pair snapshot reserved newer subset")
				}
			}
		})
	}
}

func TestAMSubsetCannotAcquireFullRetirementControls(t *testing.T) {
	for _, mode := range []string{"full controls", "retire"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			dao := NewAMProjectionJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			version := AMSourceVersion{PoolID: 7}
			reserve := journalBehaviorRequest(1)
			reserve.Family = 5
			ref, err := dao.ReserveAMPoolOperation(ctx, reserve, version)
			if err != nil {
				t.Fatal(err)
			}
			request := v2AMPlan(ref)
			if mode == "retire" {
				owner := GeneratedConfigOwner{ID: 1, PoolID: 7, ConfigType: 5, InstanceIP: "127.0.0.1_3", Name: "webhook_original_3_127.0.0.1"}
				binding, err := ProjectionBindingForNamespace(5, owner.InstanceIP)
				if err != nil {
					t.Fatal(err)
				}
				request = GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, Items: []GeneratedPlanItem{{SourceID: 1, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: &binding}}}, Controls: []ProjectionControl{{ControlID: 3, Kind: ProjectionACKWebhookSubset, Ack: &ProjectionAckControl{ArtifactSourceIDs: []uint64{1}, RequiredControlIDs: []uint64{}}}}}
			}
			if _, err := dao.CommitAMProjectionPlan(ctx, AMProjectionPlanRequest{GeneratedProjectionPlanRequest: request, SourceVersion: version}); err == nil {
				t.Fatal("subset acquired full/retire authority")
			}
			last, err := dao.LoadLastCommittedProjection(ctx, 5, 7)
			if err != nil || last != nil {
				t.Fatal("rejected subset persisted committed operation", err)
			}
		})
	}
}
