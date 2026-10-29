package api

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	systemservice "github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	workservice "github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type eventRoles struct {
	systemservice.RoleService
	grants []*model.Api
}

func (r eventRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: r.grants}}}, nil
}

type eventAPIStub struct {
	workservice.WorkorderNotificationService
	actor int
	calls int
}

func (s *eventAPIStub) ListNotificationEvents(context.Context, *model.ListWorkorderNotificationEventsReq) (*model.ListResp[*model.WorkorderNotificationEventStatus], error) {
	s.calls++
	return &model.ListResp[*model.WorkorderNotificationEventStatus]{Items: []*model.WorkorderNotificationEventStatus{}}, nil
}
func (s *eventAPIStub) ResolveNotificationDelivery(_ context.Context, _ uint64, _ *model.ResolveWorkorderNotificationDeliveryReq, actor int) error {
	s.calls++
	s.actor = actor
	return nil
}
func (s *eventAPIStub) RetryNotificationEvent(_ context.Context, _ uint64, _ string, actor int) error {
	s.calls++
	s.actor = actor
	return nil
}
func TestNotificationEventRoutesRequireSeparateGrantsAndTrustedActor(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, body string
		grants                   []*model.Api
		status                   int
		actor                    int
	}{
		{"events-denied", "GET", "/api/workorder/notification/events", "", nil, 403, 0},
		{"events-granted", "GET", "/api/workorder/notification/events", "", []*model.Api{{Method: 1, Path: "/api/workorder/notification/events"}}, 200, 0},
		{"resolve-denied-with-read-grant", "POST", "/api/workorder/notification/events/1/resolve", `{"delivery_id":2,"action":"retry","reason":"核对后重试"}`, []*model.Api{{Method: 1, Path: "/api/workorder/notification/events"}}, 403, 0},
		{"resolve-denied-with-wildcard", "POST", "/api/workorder/notification/events/1/resolve", `{"delivery_id":2,"action":"retry","reason":"verified recipient"}`, []*model.Api{{Method: 2, Path: "/api/*"}}, 403, 0},
		{"retry-denied-with-wildcard", "POST", "/api/workorder/notification/events/1/retry", `{"reason":"verified recipient"}`, []*model.Api{{Method: 2, Path: "/api/*"}}, 403, 0},
		{"retry-granted-trusted-actor", "POST", "/api/workorder/notification/events/1/retry", `{"reason":"verified recipient","operator_id":999}`, []*model.Api{{Method: 2, Path: "/api/workorder/notification/events/:id/retry"}}, 200, 7},
		{"resolve-granted-trusted-actor", "POST", "/api/workorder/notification/events/1/resolve", `{"delivery_id":2,"action":"retry","reason":"verified recipient","operator_id":999}`, []*model.Api{{Method: 2, Path: "/api/workorder/notification/events/:id/resolve"}}, 200, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			stub := &eventAPIStub{}
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
			engine.Use(middleware.NewAuthMiddleware(eventRoles{grants: tc.grants}).CheckAuth())
			NewNotificationHandler(stub).RegisterRouters(engine)
			request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if stub.actor != tc.actor {
				t.Fatalf("actor=%d", stub.actor)
			}
			if tc.status == http.StatusForbidden && stub.calls != 0 {
				t.Fatal("unauthorized request reached handler")
			}
		})
	}
}
