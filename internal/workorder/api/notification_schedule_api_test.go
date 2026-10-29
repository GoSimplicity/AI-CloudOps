package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type scheduleCreateStub struct {
	service.WorkorderNotificationService
	captured *model.CreateWorkorderNotificationReq
}

func (s *scheduleCreateStub) CreateNotification(_ context.Context, req *model.CreateWorkorderNotificationReq) error {
	s.captured = req
	return nil
}

func TestScheduleCreateHTTPBindsAdvancedFieldsAndTrustedActor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := &scheduleCreateStub{}
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7}); c.Next() })
	r.POST("/create", NewNotificationHandler(store).CreateNotification)
	body := `{"name":"fixture","event_types":["instance_created"],"channels":["email"],"recipient_types":["creator"],"message_template":"fixture","trigger_type":"delayed","delay_minutes":5,"repeat_interval":10,"repeat_count":3,"user_id":999}`
	request := httptest.NewRequest("POST", "/create", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != 200 || store.captured == nil {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	got := store.captured
	if got.UserID != 7 || got.DelayMinutes == nil || *got.DelayMinutes != 5 || got.RepeatInterval == nil || *got.RepeatInterval != 10 || got.RepeatCount == nil || *got.RepeatCount != 3 {
		t.Fatalf("incorrect request binding: %+v", got)
	}
}
