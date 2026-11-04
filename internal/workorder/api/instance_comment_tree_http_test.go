package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCommentTreeHTTPPreservesNestedRepliesAndParentBoundary(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "comment-tree-http.sqlite")), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&model.WorkorderInstance{}, &model.WorkorderProcess{}, &model.WorkorderFormDesign{}, &model.WorkorderCategory{}, &model.WorkorderInstanceComment{}, &model.WorkorderInstanceFlow{}, &model.WorkorderInstanceTimeline{}, &model.WorkorderNotificationEvent{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{7, 8} {
		if db.Create(&model.WorkorderInstance{Model: model.Model{ID: id}, Title: "owned", SerialNumber: fmt.Sprintf("tree-%d", id), ProcessID: 1, OperatorID: 7, Status: model.InstanceStatusDraft}).Error != nil {
			t.Fatal("instance fixture failed")
		}
	}
	instanceDAO := dao.NewWorkorderInstanceDAO(db, zap.NewNop())
	notifier := service.NewWorkorderNotificationService(nil, nil, zap.NewNop(), instanceDAO, nil)
	comments := service.NewInstanceCommentService(dao.NewWorkorderInstanceCommentDAO(db, zap.NewNop()), instanceDAO, notifier, zap.NewNop())
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "trusted-owner"}); c.Next() })
	NewInstanceCommentHandler(comments).RegisterRouters(engine)
	call := func(method, path string, body any) (int, json.RawMessage) {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(method, path, bytes.NewReader(raw))
		if method == "POST" {
			request.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, request)
		var envelope struct {
			Code int             `json:"code"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(response.Body.Bytes(), &envelope) != nil {
			t.Fatal("invalid HTTP envelope")
		}
		return envelope.Code, envelope.Data
	}
	parent := 0
	ids := []int{}
	for depth := 1; depth <= 5; depth++ {
		body := map[string]any{"instance_id": 7, "content": fmt.Sprintf("level-%d", depth), "operator_id": 999, "operator_name": "forged", "type": "system", "is_system": 1}
		if parent != 0 {
			body["parent_id"] = parent
		}
		if code, _ := call("POST", "/api/workorder/instance/comment/create", body); code != 0 {
			t.Fatal("legal nested HTTP creation failed")
		}
		var row model.WorkorderInstanceComment
		if db.Where("content=?", fmt.Sprintf("level-%d", depth)).First(&row).Error != nil || row.OperatorID != 7 || row.OperatorName != "trusted-owner" || row.Type != model.CommentTypeNormal || row.IsSystem != 2 {
			t.Fatal("created comment lost trusted actor or system boundary")
		}
		ids = append(ids, row.ID)
		parent = row.ID
		if depth == 3 || depth == 5 {
			code, data := call("GET", "/api/workorder/instance/comment/list?instance_id=7&page=1&size=100", nil)
			var flat struct {
				Items []model.WorkorderInstanceComment `json:"items"`
				Total int64                            `json:"total"`
			}
			if code != 0 || json.Unmarshal(data, &flat) != nil || len(flat.Items) != depth || flat.Total != int64(depth) {
				t.Fatal("flat list lost successfully created comments")
			}
			code, data = call("GET", "/api/workorder/instance/comment/tree/7", nil)
			var roots []model.WorkorderInstanceComment
			if code != 0 || json.Unmarshal(data, &roots) != nil || len(roots) != 1 {
				t.Fatal("tree HTTP root missing")
			}
			node := &roots[0]
			for level := 0; level < depth; level++ {
				if node.ID != ids[level] {
					t.Fatal("tree changed a comment ID or parent association")
				}
				if level < depth-1 {
					if len(node.Children) != 1 {
						t.Fatalf("HTTP tree omitted accepted level %d", level+2)
					}
					node = &node.Children[0]
				}
			}
		}
	}
	before := int64(0)
	db.Model(&model.WorkorderInstanceComment{}).Count(&before)
	if code, _ := call("POST", "/api/workorder/instance/comment/create", map[string]any{"instance_id": 8, "content": "cross-instance", "parent_id": ids[0]}); code == 0 {
		t.Fatal("cross-instance parent accepted")
	}
	var after, events int64
	db.Model(&model.WorkorderInstanceComment{}).Count(&after)
	db.Model(&model.WorkorderNotificationEvent{}).Count(&events)
	if after != before || events != 5 {
		t.Fatal("rejected parent wrote a comment/event or nested comments lost their outbox events")
	}
}
