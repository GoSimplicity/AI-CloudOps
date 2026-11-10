package dao

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestYamlOwnershipIsEnforced(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.K8sYamlTask{}, &model.K8sYamlTemplate{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tasks := NewYamlTaskDAO(db, zap.NewNop())
	templates := NewYamlTemplateDAO(db, zap.NewNop())
	template := &model.K8sYamlTemplate{Name: "owned", Content: "kind: ConfigMap", ClusterID: 9, UserID: 7}
	if err := templates.CreateYamlTemplate(ctx, template); err != nil {
		t.Fatal(err)
	}
	task := &model.K8sYamlTask{Name: "rollout", TemplateID: template.ID, ClusterID: 9, UserID: 7, Status: "Pending"}
	if err := tasks.CreateYamlTask(ctx, task); err != nil {
		t.Fatal(err)
	}

	if _, err := tasks.GetOwnedYamlTask(ctx, task.ID, 9, 8); !errors.Is(err, constants.ErrNotResourceOwner) {
		t.Fatalf("other user task = %v", err)
	}
	if _, err := tasks.GetOwnedYamlTask(ctx, task.ID, 9, 0); !errors.Is(err, constants.ErrInvalidParam) {
		t.Fatalf("zero user task = %v", err)
	}
	owned, err := tasks.GetOwnedYamlTask(ctx, task.ID, 9, 7)
	if err != nil || owned.ID != task.ID || owned.UserID != 7 {
		t.Fatalf("owner task = %+v %v", owned, err)
	}

	if err := templates.DeleteYamlTemplate(ctx, template.ID, 9, 8); !errors.Is(err, constants.ErrNotResourceOwner) {
		t.Fatalf("other user template = %v", err)
	}
	kept, err := templates.GetYamlTemplateByID(ctx, template.ID, 9)
	if err != nil || kept.UserID != 7 {
		t.Fatalf("template deleted for non-owner: %+v %v", kept, err)
	}
	if err := templates.DeleteYamlTemplate(ctx, template.ID, 9, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := templates.GetYamlTemplateByID(ctx, template.ID, 9); err == nil {
		t.Fatal("owner delete left the template")
	}
}
