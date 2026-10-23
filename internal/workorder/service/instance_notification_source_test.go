package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workdao "github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	databaseLogger "github.com/GoSimplicity/AI-CloudOps/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInstanceNotificationSourceSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "source.db")), &gorm.Config{Logger: databaseLogger.NewGORMLogger(zap.NewNop())})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	assertInstanceNotificationSource(t, db)
}

func assertInstanceNotificationSource(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := db.AutoMigrate(&model.WorkorderCategory{}, &model.WorkorderFormDesign{}, &model.WorkorderProcess{}, &model.WorkorderTemplate{}, &model.WorkorderInstance{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}); err != nil {
			t.Fatal("source migration failed")
		}
	}
	formCategory, templateCategory, processCategory := 11, 22, 33
	for _, id := range []int{11, 22, 33, 99} {
		if err := db.Create(&model.WorkorderCategory{Model: model.Model{ID: id}, Name: fmt.Sprintf("owned-category-%d", id), Status: 1}).Error; err != nil {
			t.Fatal("category fixture failed")
		}
	}
	form := &model.WorkorderFormDesign{Name: "source-form", Status: 2, CategoryID: &formCategory, Schema: model.JSONMap{"fields": []any{map[string]any{"id": "owned_field", "name": "owned_field", "type": "text", "label": "owned", "required": 0}}}}
	process := &model.WorkorderProcess{Name: "source-process", Status: 2, CategoryID: &processCategory, Definition: model.JSONMap{"steps": []any{}, "connections": []any{}}}
	if err := db.Create(form).Error; err != nil {
		t.Fatal("form fixture failed")
	}
	process.FormDesignID = form.ID
	if err := db.Create(process).Error; err != nil {
		t.Fatal("process fixture failed")
	}
	template := &model.WorkorderTemplate{Name: "source-template", Status: 1, ProcessID: process.ID, FormDesignID: form.ID, CategoryID: &templateCategory}
	if err := db.Create(template).Error; err != nil {
		t.Fatal("template fixture failed")
	}
	service := &instanceService{dao: workdao.NewWorkorderInstanceDAO(db, zap.NewNop()), processDao: workdao.NewProcessDAO(db, zap.NewNop()), formDesignDao: workdao.NewWorkorderFormDesignDAO(db, zap.NewNop()), templateDao: workdao.NewTemplateDAO(db, zap.NewNop()), flowDao: workdao.NewInstanceFlowDAO(db, zap.NewNop()), timelineDao: workdao.NewInstanceTimeLineDAO(db, zap.NewNop()), logger: zap.NewNop()}
	// JSON中的伪造来源不属于请求DTO，服务只使用已验证的模板/表单
	var from model.CreateWorkorderInstanceFromTemplateReq
	if err := json.Unmarshal([]byte(`{"title":"from-template-owned","priority":2,"form_data":{},"template_id":99999,"category_id":99999}`), &from); err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInstanceFromTemplate(ctx, template.ID, &from); err != nil {
		t.Fatal(err)
	}
	var saved model.WorkorderInstance
	if err := db.Where("title = ?", from.Title).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.TemplateID == nil || *saved.TemplateID != template.ID || saved.CategoryID == nil || *saved.CategoryID != templateCategory {
		t.Fatal("trusted template source not persisted")
	}
	var normal model.CreateWorkorderInstanceReq
	if err := json.Unmarshal([]byte(`{"title":"normal-owned","priority":2,"status":1,"form_data":{},"template_id":99999,"category_id":99999}`), &normal); err != nil {
		t.Fatal(err)
	}
	normal.ProcessID = process.ID
	if err := service.CreateInstance(ctx, &normal); err != nil {
		t.Fatal(err)
	}
	var ordinary model.WorkorderInstance
	if err := db.Where("title = ?", normal.Title).First(&ordinary).Error; err != nil {
		t.Fatal(err)
	}
	if ordinary.TemplateID != nil || ordinary.CategoryID == nil || *ordinary.CategoryID != formCategory {
		t.Fatal("normal form category source wrong or template spoofed")
	}
	// 修改来源记录以及普通实例更新均不能改写快照
	wantTemplateCategory := *saved.CategoryID
	changed := 99
	db.Model(template).Update("category_id", changed)
	db.Model(form).Update("category_id", changed)
	if err := service.dao.UpdateInstance(ctx, &model.WorkorderInstance{Model: model.Model{ID: saved.ID}, Title: "renamed-owned", TemplateID: &changed, CategoryID: &changed}); err != nil {
		t.Fatal(err)
	}
	var after model.WorkorderInstance
	if err := db.First(&after, saved.ID).Error; err != nil {
		t.Fatal(err)
	}
	if *after.TemplateID != template.ID || *after.CategoryID != wantTemplateCategory {
		t.Fatal("immutable source overwritten")
	}
	legacy := &model.WorkorderInstance{Title: "unknown-owned", SerialNumber: "unknown-owned", ProcessID: process.ID, Status: 1}
	if err := db.Create(legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.WorkorderInstance{}); err != nil {
		t.Fatal(err)
	}
	var unknown model.WorkorderInstance
	db.First(&unknown, legacy.ID)
	if unknown.TemplateID != nil || unknown.CategoryID != nil {
		t.Fatal("historical unknown source guessed from mutable process")
	}
	if notificationMatchesInstanceScope(&model.WorkorderNotification{TemplateID: &template.ID}, &unknown) || notificationMatchesInstanceScope(&model.WorkorderNotification{CategoryID: &formCategory}, &unknown) {
		t.Fatal("unknown matched concrete scope")
	}
	templateWithoutCategory := &model.WorkorderTemplate{Name: "source-uncategorized", Status: 1, ProcessID: process.ID, FormDesignID: form.ID}
	if err := db.Create(templateWithoutCategory).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.CreateInstanceFromTemplate(ctx, templateWithoutCategory.ID, &model.CreateWorkorderInstanceFromTemplateReq{Title: "uncategorized-owned", Priority: 2}); err != nil {
		t.Fatal(err)
	}
	var uncategorized model.WorkorderInstance
	db.Where("title = ?", "uncategorized-owned").First(&uncategorized)
	if uncategorized.CategoryID != nil {
		t.Fatal("nil template category fell back to unrelated category")
	}
}
