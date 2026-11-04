package config

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestAMIntentAliasCancellationUsesCurrentRowProof(t *testing.T) {
	for _, mode := range []string{"multiple old names", "alias cannot add", "different row cannot alias", "different namespace cannot alias"} {
		t.Run(mode, func(t *testing.T) {
			_, _, db := journalBehaviorFixture(t)
			dao := NewProjectionPlanJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			reserve := journalBehaviorRequest(1)
			reserve.Family = 2
			ref, err := dao.ReservePoolOperation(ctx, reserve)
			if err != nil {
				t.Fatal(err)
			}
			item := webhookPlanItem()
			if _, err := dao.CommitGeneratedProjectionPlan(ctx, v2AMPlan(ref, item)); err != nil {
				t.Fatal(err)
			}
			old := GeneratedConfigOwnerOf(item.Desired)
			row := *item.Desired
			row.Name = "webhook_current_3_127.0.0.1"
			item.Expected = &old
			item.Desired = &row
			current := GeneratedConfigOwnerOf(&row)
			reserve.Nonce = [16]byte{2}
			ref, err = dao.ReservePoolOperation(ctx, reserve)
			if err != nil {
				t.Fatal(err)
			}
			request := v2AMPlan(ref, item)
			change := IntentOwnerChange{Owner: old, AliasCurrentOwner: &current, Proof: ProjectionProofRef{SourceID: item.SourceID}}
			switch mode {
			case "alias cannot add":
				request.Controls[2].Intents.Add = []IntentOwnerChange{change}
			case "different row cannot alias":
				change.AliasCurrentOwner.ID++
				request.Controls[2].Intents.Remove = []IntentOwnerChange{change}
			case "different namespace cannot alias":
				change.AliasCurrentOwner.InstanceIP = "127.0.0.2_3"
				request.Controls[2].Intents.Remove = []IntentOwnerChange{change}
			default:
				another := change
				another.Owner.Name = "webhook_earlier_3_127.0.0.1"
				request.Controls[2].Intents.Remove = []IntentOwnerChange{change, another}
			}
			receipt, err := dao.CommitGeneratedProjectionPlan(ctx, request)
			if mode != "multiple old names" {
				if err == nil {
					t.Fatal("unproven alias acquired issued control")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(receipt.Controls[2].Intents.Remove) != 2 {
				t.Fatal("old canonical aliases lost")
			}
			if err := dao.WithPoolProjectionLease(ctx, receipt, func(lease ProjectionLease) error { return lease.RecheckCurrent(ctx) }); err != nil {
				t.Fatal(err)
			}
			loaded, err := dao.LoadLastCommittedProjection(ctx, 2, 7)
			if err != nil || loaded == nil || len(loaded.Controls[2].Intents.Remove) != 2 {
				t.Fatal("alias cancellation not durable", err)
			}
		})
	}
}
