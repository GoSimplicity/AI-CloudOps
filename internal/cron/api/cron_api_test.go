package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type createJobRecorder struct {
	service.CronService
	request *model.CreateCronJobReq
}

func (s *createJobRecorder) CreateCronJob(_ context.Context, req *model.CreateCronJobReq) error {
	snapshot := *req
	s.request = &snapshot
	return nil
}

func TestCreateCronJobUsesAuthenticatedCreator(t *testing.T) {
	for _, tt := range []struct {
		name, body string
		called     bool
	}{
		{"normal", `{"name":"job","job_type":2,"schedule":"* * * * *","command":"true"}`, true},
		{"forged creator", `{"name":"job","job_type":2,"schedule":"* * * * *","command":"true","created_by":999,"created_by_name":"admin"}`, true},
		{"invalid request", `{"created_by":999,"created_by_name":"admin"}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := &createJobRecorder{}
			handler := NewCronJobHandler(zap.NewNop(), recorder)
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"}) })
			engine.POST("/job", handler.CreateCronJob)
			req := httptest.NewRequest("POST", "/job", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			engine.ServeHTTP(httptest.NewRecorder(), req)
			if (recorder.request != nil) != tt.called {
				t.Fatal("unexpected service invocation")
			}
			if recorder.request != nil && (recorder.request.CreatedBy != 7 || recorder.request.CreatedByName != "operator") {
				t.Fatal("creator did not match authenticated user")
			}
		})
	}
}
