package config

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func alertDAOPlan(ref PoolReservation, items ...GeneratedPlanItem) ScrapeRuleProjectionPlanRequest {
	hash := [32]byte{byte(ref.PoolID)}
	ips := []string{}
	ids := []uint64{}
	for _, item := range items {
		ids = append(ids, item.SourceID)
		if item.Action == model.ConfigProjectionRender {
			ips = append(ips, item.Desired.InstanceIP)
		}
	}
	return ScrapeRuleProjectionPlanRequest{GeneratedProjectionPlanRequest: GeneratedProjectionPlanRequest{GeneratedPlanRequest: GeneratedPlanRequest{Reservation: ref, SourceHash: hash, Items: items}, Controls: []ProjectionControl{{ControlID: 1, Kind: ProjectionMembersFull, Members: &FullMembersControl{DesiredIPs: ips}}, {ControlID: 2, Kind: ProjectionACKFull, Ack: &ProjectionAckControl{LegacySHA: &hash, ArtifactSourceIDs: ids, RequiredControlIDs: []uint64{1}}}}}}
}
func TestAlertRuleScopedHistoryCurrentOwnerAndObservedABA(t *testing.T) {
	for _, change := range []bool{false, true} {
		t.Run(map[bool]string{false: "observed deleted foreign", true: "changed observation rejects"}[change], func(t *testing.T) {
			ordinary, _, db := journalBehaviorFixture(t)
			journal := NewAlertRuleProjectionJournalDAO(zap.NewNop(), db)
			ctx := context.Background()
			row := &model.MonitorConfig{PoolID: 7, ConfigType: 3, InstanceIP: "127.0.0.1", Name: "prometheus_alert_rule_7_127.0.0.1.yaml", ConfigContent: "groups: []", Status: 1}
			if err := ordinary.CreateMonitorConfig(ctx, row); err != nil {
				t.Fatal(err)
			}
			foreign := &model.MonitorConfig{Model: model.Model{DeletedAt: 7}, PoolID: 99, ConfigType: 3, InstanceIP: row.InstanceIP, Name: "old custom", ConfigContent: "observed", Status: 2}
			if db.Create(foreign).Error != nil {
				t.Fatal("history fixture failed")
			}
			var history []*model.MonitorConfig
			db.Unscoped().Where("config_type = ?", 3).Find(&history)
			guard, err := ScrapeRuleHistoryGuardOf(ScrapeAlertRule, row.InstanceIP, history)
			if err != nil {
				t.Fatal(err)
			}
			ref, err := journal.ReserveAlertRulePoolOperation(ctx, PoolOperationRequest{Family: 3, PoolID: 7, Nonce: [16]byte{1}}, 0)
			if err != nil {
				t.Fatal(err)
			}
			owner := GeneratedConfigOwnerOf(row)
			request := alertDAOPlan(ref, GeneratedPlanItem{SourceID: 11, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 3, InstanceIP: row.InstanceIP}})
			request.History = []ScrapeRuleHistoryGuard{guard}
			if change {
				db.Unscoped().Model(&model.MonitorConfig{}).Where("id = ?", foreign.ID).Update("config_content", "changed")
			}
			receipt, err := journal.CommitAlertRuleProjectionPlan(ctx, request)
			if change {
				if !errors.Is(err, ErrJournalConflict) || receipt.Reservation.OperationID != 0 {
					t.Fatal("changed history bypassed guard", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			again, err := journal.CommitAlertRuleProjectionPlan(ctx, request)
			if err != nil || !equalProjectionPayload(receipt, again) {
				t.Fatal("self-mutated guard invalidated signed retry", err)
			}
			versions, err := journal.ReadAlertRuleSourceVersions(ctx)
			if err != nil || len(versions) != 1 || versions[0].PoolID != 7 {
				t.Fatal("family3 source versions incorrect", err)
			}
			recordVersions, err := NewRecordProjectionJournalDAO(zap.NewNop(), db).ReadRecordSourceVersions(ctx)
			if err != nil || len(recordVersions) != 0 {
				t.Fatal("family3 control leaked to family4", err)
			}
		})
	}
}
func TestAlertRuleLegacySideProofUsesOwnCodecAndRejectsCrossScope(t *testing.T) {
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewAlertRuleProjectionJournalDAO(zap.NewNop(), db)
	history := NewAlertRuleHistoryReader(zap.NewNop(), db)
	ctx := context.Background()
	a := &model.MonitorConfig{PoolID: 7, ConfigType: 3, InstanceIP: "127.0.0.1", Name: "prometheus_alert_rule_7_127.0.0.1.yaml", Status: 1}
	if err := ordinary.CreateMonitorConfig(ctx, a); err != nil {
		t.Fatal(err)
	}
	owner := GeneratedConfigOwnerOf(a)
	if err := ordinary.DeleteMonitorConfig(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	bRef, err := journal.ReserveAlertRulePoolOperation(ctx, PoolOperationRequest{Family: 3, PoolID: 99, Nonce: [16]byte{9}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	bRow := &model.MonitorConfig{PoolID: 99, ConfigType: 3, InstanceIP: a.InstanceIP, Name: "prometheus_alert_rule_99_127.0.0.1.yaml", ConfigContent: "B exact bytes", Status: 1}
	b, err := journal.CommitAlertRuleProjectionPlan(ctx, alertDAOPlan(bRef, GeneratedPlanItem{SourceID: 9, Action: model.ConfigProjectionRender, Desired: bRow, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 3, InstanceIP: bRow.InstanceIP}}))
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := history.GetLegacySupersededAlertRuleArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proofs) != 0 {
		t.Fatal("unprojected replacement granted proof", err)
	}
	ack, _ := projectionAckProof(b)
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, b, ack); err != nil || !ok {
		t.Fatal(err)
	}
	proofs, err = history.GetLegacySupersededAlertRuleArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proofs) != 1 {
		t.Fatal("family3 projected replacement not recognized", err)
	}
	aRef, err := journal.ReserveAlertRulePoolOperation(ctx, PoolOperationRequest{Family: 3, PoolID: 7, Nonce: [16]byte{1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	request := alertDAOPlan(aRef)
	request.Superseded = proofs
	recovered, err := journal.CommitAlertRuleProjectionPlan(ctx, request)
	if err != nil || len(recovered.Artifacts) != 0 {
		t.Fatal("family3 control-only adoption failed", err)
	}
	var op model.MonitorConfigPoolOperation
	db.First(&op, "operation_id = ?", aRef.OperationID)
	if op.Manifest == nil || !strings.Contains(*op.Manifest, `"rule_superseded"`) || strings.Contains(*op.Manifest, `"record_superseded"`) {
		t.Fatal("family3 serialized as family4")
	}
	resolved, err := history.GetProjectedAlertRuleRetirements(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(resolved) != 1 {
		t.Fatal("family3 side proof not durable", err)
	}
	corrupt := strings.Replace(*op.Manifest, `"rule_superseded"`, `"record_superseded"`, 1)
	db.Model(&op).Update("manifest", corrupt)
	if _, err := journal.LoadLastCommittedProjection(ctx, 3, 7); err == nil {
		t.Fatal("cross-scope side payload was accepted")
	}
	if _, err := journal.ReserveAlertRulePoolOperation(ctx, PoolOperationRequest{Family: 4, PoolID: 8, Nonce: [16]byte{8}}, 0); !errors.Is(err, ErrJournalInvalid) {
		t.Fatal("Alert facade reserved Record scope", err)
	}
}
