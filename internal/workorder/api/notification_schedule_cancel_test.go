package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	workservice "github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type scheduleCancelStub struct {
	workservice.WorkorderNotificationService
	actor, calls int
	eventID      uint64
	reason       string
}

func (s *scheduleCancelStub) CancelScheduledNotification(_ context.Context, eventID uint64, reason string, actor int) error {
	s.actor, s.eventID, s.reason = actor, eventID, reason
	s.calls++
	return nil
}

func TestScheduledCancelRequiresGrantAndUsesTrustedOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name          string
		grants        []*model.Api
		uid           int
		body          string
		status, calls int
	}{
		{"no grant", nil, 7, `{"reason":"verified cancellation"}`, 403, 0},
		{"invalid identity", []*model.Api{{Method: 2, Path: "/api/workorder/notification/events/:id/cancel"}}, 0, `{"reason":"verified cancellation"}`, 403, 0},
		{"granted ignores spoof", []*model.Api{{Method: 2, Path: "/api/workorder/notification/events/:id/cancel"}}, 7, `{"reason":"verified cancellation","operator_id":999}`, 200, 1},
		{"missing reason", []*model.Api{{Method: 2, Path: "/api/workorder/notification/events/:id/cancel"}}, 7, `{}`, 400, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := &scheduleCancelStub{}
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: tc.uid}); c.Next() })
			engine.Use(middleware.NewAuthMiddleware(eventRoles{grants: tc.grants}).CheckAuth())
			handler := NewNotificationHandler(stub)
			engine.POST("/api/workorder/notification/events/:id/cancel", handler.CancelScheduledNotification)
			req := httptest.NewRequest("POST", "/api/workorder/notification/events/12/cancel", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			engine.ServeHTTP(res, req)
			if res.Code != tc.status || stub.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", res.Code, stub.calls, res.Body.String())
			}
			if tc.calls == 1 && (stub.actor != 7 || stub.eventID != 12 || stub.reason != "verified cancellation") {
				t.Fatal("untrusted operator or wrong event")
			}
		})
	}
}
