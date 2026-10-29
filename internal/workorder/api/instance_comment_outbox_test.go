package api

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommentHTTPRejectsSpoofedSystemFlagAndActor(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "comment-http.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderNotificationEvent{}); err != nil {
		t.Fatal(err)
	}
	instance := model.WorkorderInstance{Title: "fixture", SerialNumber: "comment-http", ProcessID: 1, Status: model.InstanceStatusDraft, FormData: model.JSONMap{"hidden": "form-canary"}}
	if err := db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	instanceDAO := dao.NewWorkorderInstanceDAO(db, zap.NewNop())
	notifier := service.NewWorkorderNotificationService(nil, nil, zap.NewNop(), instanceDAO, nil)
	comments := service.NewInstanceCommentService(dao.NewWorkorderInstanceCommentDAO(db, zap.NewNop()), instanceDAO, notifier, zap.NewNop())
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "trusted"}); c.Next() })
	NewInstanceCommentHandler(comments).RegisterRouters(engine)
	body := `{"instance_id":1,"operator_id":999,"operator_name":"forged","content":"comment fixture","type":"system","is_system":1}`
	req := httptest.NewRequest("POST", "/api/workorder/instance/comment/create", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, req)
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 200 || result.Code != 0 {
		t.Fatalf("request failed: %d", response.Code)
	}
	var comment model.WorkorderInstanceComment
	if err := db.First(&comment).Error; err != nil || comment.OperatorID != 7 || comment.OperatorName != "trusted" || comment.IsSystem != 2 || comment.Type != model.CommentTypeNormal {
		t.Fatal("untrusted actor or system flag accepted")
	}
	var event model.WorkorderNotificationEvent
	if err := db.First(&event).Error; err != nil || event.SenderID != 7 || event.EventType != model.EventTypeInstanceCommented || strings.Contains(string(event.Snapshot), "form-canary") {
		t.Fatal("comment event lacks safe trusted snapshot")
	}
}
