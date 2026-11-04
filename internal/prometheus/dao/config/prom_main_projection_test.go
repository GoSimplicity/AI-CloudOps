package config

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

func promDAOPlan(ref PoolReservation, items ...GeneratedPlanItem) PromMainProjectionPlanRequest {
	return alertDAOPlan(ref, items...)
}
func TestPromMainScopedHistoryAndPublicRuleScopeRemainSeparate(t *testing.T) {
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewPromMainProjectionJournalDAO(zap.NewNop(), db)
	ctx := context.Background()
	row := &model.MonitorConfig{PoolID: 7, ConfigType: 1, InstanceIP: "127.0.0.1", Name: "prometheus_scrape_pool_7_127.0.0.1.yaml", ConfigContent: "global: {}", Status: 1}
	if err := ordinary.CreateMonitorConfig(ctx, row); err != nil {
		t.Fatal(err)
	}
	old := &model.MonitorConfig{Model: model.Model{DeletedAt: 7}, PoolID: 99, ConfigType: 1, InstanceIP: row.InstanceIP, Name: "old custom main config", ConfigContent: "observed", Status: 2}
	if db.Create(old).Error != nil {
		t.Fatal("history fixture failed")
	}
	var rows []*model.MonitorConfig
	db.Unscoped().Where("config_type = ?", 1).Order("id").Find(&rows)
	guard, err := PromMainHistoryGuardOf(row.InstanceIP, rows)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ScrapeRuleHistoryGuardOf(ScrapeRuleScope(1), row.InstanceIP, rows); !errors.Is(err, ErrJournalInvalid) {
		t.Fatal("published rule API silently accepted main-config scope", err)
	}
	ref, err := journal.ReservePromMainPoolOperation(ctx, PoolOperationRequest{Family: 1, PoolID: 7, Nonce: [16]byte{1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	owner := GeneratedConfigOwnerOf(row)
	request := promDAOPlan(ref, GeneratedPlanItem{SourceID: 11, Action: model.ConfigProjectionRetire, Expected: &owner, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 1, InstanceIP: row.InstanceIP}})
	request.History = []PromMainHistoryGuard{guard}
	receipt, err := journal.CommitPromMainProjectionPlan(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	again, err := journal.CommitPromMainProjectionPlan(ctx, request)
	if err != nil || !equalProjectionPayload(receipt, again) {
		t.Fatal("own soft-delete broke signed retry", err)
	}
	versions, err := journal.ReadPromMainSourceVersions(ctx)
	if err != nil || len(versions) != 1 || versions[0].PoolID != 7 {
		t.Fatal("main source versions incorrect", err)
	}
	records, err := NewRecordProjectionJournalDAO(zap.NewNop(), db).ReadRecordSourceVersions(ctx)
	if err != nil || len(records) != 0 {
		t.Fatal("main control leaked to Record", err)
	}
	if _, err := NewRecordProjectionJournalDAO(zap.NewNop(), db).CommitRecordProjectionPlan(ctx, request); !errors.Is(err, ErrJournalInvalid) {
		t.Fatal("Record facade committed Prom", err)
	}
	if _, err := NewAlertRuleProjectionJournalDAO(zap.NewNop(), db).CommitAlertRuleProjectionPlan(ctx, request); !errors.Is(err, ErrJournalInvalid) {
		t.Fatal("Alert facade committed Prom", err)
	}
}
func TestPromMainLegacyDispositionHasDedicatedCodecAndSurvivesNewOwnerChanges(t *testing.T) {
	ordinary, _, db := journalBehaviorFixture(t)
	journal := NewPromMainProjectionJournalDAO(zap.NewNop(), db)
	history := NewPromMainHistoryReader(zap.NewNop(), db)
	ctx := context.Background()
	a := &model.MonitorConfig{PoolID: 7, ConfigType: 1, InstanceIP: "127.0.0.1", Name: "prometheus_scrape_pool_7_127.0.0.1.yaml", Status: 1}
	if err := ordinary.CreateMonitorConfig(ctx, a); err != nil {
		t.Fatal(err)
	}
	owner := GeneratedConfigOwnerOf(a)
	if err := ordinary.DeleteMonitorConfig(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	bRef, err := journal.ReservePromMainPoolOperation(ctx, PoolOperationRequest{Family: 1, PoolID: 99, Nonce: [16]byte{9}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	bRow := &model.MonitorConfig{PoolID: 99, ConfigType: 1, InstanceIP: a.InstanceIP, Name: "prometheus_scrape_pool_99_127.0.0.1.yaml", ConfigContent: "global: {} # B exact bytes", Status: 1}
	b, err := journal.CommitPromMainProjectionPlan(ctx, promDAOPlan(bRef, GeneratedPlanItem{SourceID: 9, Action: model.ConfigProjectionRender, Desired: bRow, Binding: &ProjectionBinding{Version: 1, Role: "artifact", ConfigType: 1, InstanceIP: bRow.InstanceIP}}))
	if err != nil {
		t.Fatal(err)
	}
	proof, err := history.GetLegacySupersededPromMainArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proof) != 0 {
		t.Fatal("unprojected B granted disposition", err)
	}
	ack, _ := projectionAckProof(b)
	if ok, err := journal.MarkProjectionCompleteIfCurrent(ctx, b, ack); err != nil || !ok {
		t.Fatal(err)
	}
	proof, err = history.GetLegacySupersededPromMainArtifacts(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(proof) != 1 {
		t.Fatal("projected B proof absent", err)
	}
	ref, err := journal.ReservePromMainPoolOperation(ctx, PoolOperationRequest{Family: 1, PoolID: 7, Nonce: [16]byte{1}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	request := promDAOPlan(ref)
	request.Superseded = proof
	recovered, err := journal.CommitPromMainProjectionPlan(ctx, request)
	if err != nil || len(recovered.Artifacts) != 0 {
		t.Fatal("control-only main adoption failed", err)
	}
	var op model.MonitorConfigPoolOperation
	db.First(&op, "operation_id = ?", ref.OperationID)
	if op.Manifest == nil || !strings.Contains(*op.Manifest, `"prom_superseded"`) || strings.Contains(*op.Manifest, `"record_superseded"`) || strings.Contains(*op.Manifest, `"rule_superseded"`) {
		t.Fatal("main side data used another family's codec")
	}
	changed := *bRow
	changed.ConfigContent = "later B body"
	if err := ordinary.UpdateMonitorConfig(ctx, &changed); err != nil {
		t.Fatal(err)
	}
	resolved, err := history.GetProjectedPromMainRetirements(ctx, 7, []GeneratedConfigOwner{owner})
	if err != nil || len(resolved) != 1 {
		t.Fatal("later B body revoked immutable main disposition", err)
	}
	corrupt := strings.Replace(*op.Manifest, `"prom_superseded"`, `"rule_superseded"`, 1)
	db.Model(&op).Update("manifest", corrupt)
	if _, err := journal.LoadLastCommittedProjection(ctx, 1, 7); err == nil {
		t.Fatal("cross-family side proof accepted")
	}
}
