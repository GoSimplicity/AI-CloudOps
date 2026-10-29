package api

import (
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestDeliveryMaskHTTPRequiresReadGrantAndDoesNotLeakPlan(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "mask-http.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.AutoMigrate(&model.WorkorderNotificationEvent{}, &model.WorkorderNotificationDelivery{}); err != nil {
		t.Fatal(err)
	}
	address := "alice.private@example.test"
	event := model.WorkorderNotificationEvent{Status: "uncertain", Snapshot: []byte("snapshot-canary"), Plan: []byte(`[{"config":{"id":5},"channel":"email","recipients":[{"address":"alice.private@example.test"}],"content":"body-canary"}]`), NextAttemptAt: time.Now().UTC()}
	if err := db.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.WorkorderNotificationDelivery{EventID: event.ID, NotificationID: 5, Channel: "email", RecipientHash: fmt.Sprintf("%x", sha256.Sum256([]byte(address))), Status: "uncertain"}).Error; err != nil {
		t.Fatal(err)
	}
	instanceDAO := dao.NewWorkorderInstanceDAO(db, zap.NewNop())
	svc := service.NewWorkorderNotificationService(nil, nil, zap.NewNop(), instanceDAO, nil)
	gin.SetMode(gin.TestMode)
	for _, allowed := range []bool{false, true} {
		grants := []*model.Api{}
		if allowed {
			grants = append(grants, &model.Api{Method: 1, Path: "/api/workorder/notification/events/:id/deliveries"})
		}
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
		engine.Use(middleware.NewAuthMiddleware(eventRoles{grants: grants}).CheckAuth())
		NewNotificationHandler(svc).RegisterRouters(engine)
		response := httptest.NewRecorder()
		engine.ServeHTTP(response, httptest.NewRequest("GET", fmt.Sprintf("/api/workorder/notification/events/%d/deliveries", event.ID), nil))
		if !allowed && response.Code != 403 {
			t.Fatalf("unauthorized status %d", response.Code)
		}
		if allowed && (response.Code != 200 || !strings.Contains(response.Body.String(), `"recipient_masked":"a***@e***"`)) {
			t.Fatalf("authorized response: %d %s", response.Code, response.Body.String())
		}
		for _, secret := range []string{address, "body-canary", "snapshot-canary"} {
			if strings.Contains(response.Body.String(), secret) {
				t.Fatal("HTTP response leaked plan")
			}
		}
	}
}
