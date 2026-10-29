package service

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

func TestCurrentStepAndActionsDoNotMutateInstance(t *testing.T) {
	db := eventTestDB(t)
	form := model.WorkorderFormDesign{Name: "fixture-form", Schema: model.JSONMap{"fields": []any{}}, Status: model.FormDesignStatusPublished, OperatorID: 7}
	if err := db.Create(&form).Error; err != nil {
		t.Fatal(err)
	}
	process := model.WorkorderProcess{Name: "fixture-process", FormDesignID: form.ID, Definition: model.JSONMap{"steps": []any{map[string]any{"id": "start", "type": "start"}}, "connections": []any{}}, Status: model.ProcessStatusPublished, OperatorID: 7}
	if err := db.Create(&process).Error; err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Title: "fixture", SerialNumber: "read-only-fixture", ProcessID: process.ID, Status: model.InstanceStatusDraft, OperatorID: 7}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	svc := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), processDao: workdao.NewProcessDAO(db, zap.NewNop()), logger: zap.NewNop()}
	step, err := svc.GetCurrentStep(context.Background(), instance.ID)
	if err != nil || step == nil || step.ID != "start" {
		t.Fatalf("fallback step: %v", err)
	}
	if _, err := svc.GetAvailableActions(context.Background(), instance.ID, 7); err != nil {
		t.Fatal(err)
	}
	var actual model.WorkorderInstance
	if err := db.First(&actual, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if actual.CurrentStepID != nil {
		t.Fatal("read-only query modified stored step")
	}
	if !actual.UpdatedAt.Equal(instance.UpdatedAt) {
		t.Fatal("read-only query modified update timestamp")
	}
}
