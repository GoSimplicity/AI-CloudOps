package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInboxHTTPRequiresRBACAndAlwaysUsesAuthenticatedUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "inbox-http.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderInboxMessage{}); err != nil {
		t.Fatal(err)
	}
	first := model.WorkorderInboxMessage{EventID: 1, NotificationID: 1, UserID: 7, InstanceID: 12, EventType: "created", Subject: "own-subject", Content: "own-body"}
	second := model.WorkorderInboxMessage{EventID: 1, NotificationID: 1, UserID: 8, InstanceID: 12, EventType: "created", Subject: "other-secret", Content: "other-body"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	svc := service.NewWorkorderNotificationService(nil, nil, zap.NewNop(), dao.NewWorkorderInstanceDAO(db, zap.NewNop()), nil)
	gin.SetMode(gin.TestMode)
	engineFor := func(grants []*model.Api, uid int) *gin.Engine {
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: uid}); c.Next() })
		engine.Use(middleware.NewAuthMiddleware(eventRoles{grants: grants}).CheckAuth())
		NewNotificationHandler(svc).RegisterRouters(engine)
		return engine
	}
	listPath := "/api/workorder/notification/inbox"
	denied := httptest.NewRecorder()
	engineFor(nil, 7).ServeHTTP(denied, httptest.NewRequest("GET", listPath, nil))
	if denied.Code != 403 {
		t.Fatal("ungranted inbox was accessible")
	}
	grants := []*model.Api{{Method: 1, Path: listPath}, {Method: 1, Path: listPath + "/unread"}, {Method: 2, Path: listPath + "/:id/read"}, {Method: 2, Path: listPath + "/read-all"}, {Method: 4, Path: listPath + "/:id"}}
	engine := engineFor(grants, 7)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		engine.ServeHTTP(rec, req)
		return rec
	}
	list := request("GET", listPath+"?user_id=8&page=1&size=20", "")
	if list.Code != 200 || !strings.Contains(list.Body.String(), "own-subject") || strings.Contains(list.Body.String(), "other-secret") {
		t.Fatal("inbox list escaped authenticated user")
	}
	other := fmt.Sprintf("%s/%d", listPath, second.ID)
	for _, call := range []struct{ method, path string }{{"POST", other + "/read"}, {"DELETE", other}} {
		result := request(call.method, call.path, `{"user_id":8}`)
		var body struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(result.Body.Bytes(), &body); err != nil || body.Code == 0 {
			t.Fatal("cross-user mutation accepted")
		}
	}
	if result := request("POST", listPath+"/read-all", `{"user_id":8}`); result.Code != 200 {
		t.Fatal("read-all failed")
	}
	var saved model.WorkorderInboxMessage
	if err := db.First(&saved, second.ID).Error; err != nil || saved.ReadAt != nil {
		t.Fatal("read-all changed another user")
	}
	if result := request("GET", listPath+"/unread?user_id=8", ""); !strings.Contains(result.Body.String(), `"unread":0`) {
		t.Fatal("wrong user unread count")
	}
	if result := request("DELETE", fmt.Sprintf("%s/%d", listPath, first.ID), `{"user_id":8}`); result.Code != 200 {
		t.Fatal("owned deletion failed")
	}
	saved = model.WorkorderInboxMessage{}
	if err := db.First(&saved, first.ID).Error; err != nil || saved.DeletedAt == nil || saved.Content != "" {
		t.Fatal("owned deletion did not scrub message")
	}
}
