package api

import (
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAndDeleteEventsUseAuthenticatedOperator(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "actor.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	if err = db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderNotificationEvent{}); err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Title: "owned", SerialNumber: "owned", ProcessID: 1, Status: model.InstanceStatusDraft, FormData: model.JSONMap{"secret": "private-fixture"}}
	if err = db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	instanceDAO := dao.NewWorkorderInstanceDAO(db, zap.NewNop())
	notifier := service.NewWorkorderNotificationService(nil, nil, zap.NewNop(), instanceDAO, nil)
	svc := service.NewInstanceService(instanceDAO, nil, nil, nil, nil, nil, nil, notifier, zap.NewNop())
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	NewInstanceHandler(svc).RegisterRouters(engine)
	for _, call := range []struct{ method, path, body string }{{"PUT", "/api/workorder/instance/update/1", `{"id":999,"operator_id":999,"title":"updated"}`}, {"DELETE", "/api/workorder/instance/delete/1", ""}} {
		req := httptest.NewRequest(call.method, call.path, strings.NewReader(call.body))
		if call.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		var response struct {
			Code int `json:"code"`
		}
		if err = json.Unmarshal(rec.Body.Bytes(), &response); err != nil || rec.Code != 200 || response.Code != 0 {
			t.Fatalf("request failed: %d %s", rec.Code, rec.Body.String())
		}
	}
	var events []model.WorkorderNotificationEvent
	if err = db.Order("id").Find(&events).Error; err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events=%d", len(events))
	}
	for _, event := range events {
		if event.SenderID != 7 || event.InstanceID != instance.ID {
			t.Fatal("event actor or instance was spoofed")
		}
		if strings.Contains(string(event.Snapshot), "private-fixture") {
			t.Fatal("unused form data persisted in event snapshot")
		}
	}
}
