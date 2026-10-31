package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type contractStore struct {
	dao.CronJobDAO
	writes int
}

func (s *contractStore) CreateCronJob(context.Context, *model.CronJob) error { s.writes++; return nil }
func (s *contractStore) UpdateCronJob(context.Context, *model.CronJob, *int) error {
	s.writes++
	return nil
}

// 前端契约：开关关闭时创建/更新命令行(2)、脚本(4)任务返回 HTTP 200 + code=1 + 固定 message
func TestLocalExecDisabledResponseContract(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		for _, tt := range []struct{ method, path, body string }{
			{"POST", "/api/cron/job/create", `{"name":"job","job_type":2,"schedule":"@every 1h","command":"true"}`},
			{"POST", "/api/cron/job/create", `{"name":"job","job_type":4,"schedule":"@every 1h","script_type":"bash","script_content":"true"}`},
			{"PUT", "/api/cron/job/1/update", `{"name":"job","job_type":2,"schedule":"@every 1h","command":"true"}`},
		} {
			store := &contractStore{}
			handler := NewCronJobHandler(zap.NewNop(), service.NewCronService(zap.NewNop(), store, nil, nil, nil, func() bool { return allowed }))
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "operator"}) })
			handler.RegisterRouters(engine)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)
			var resp struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil || recorder.Code != 200 {
				t.Fatalf("unexpected response %d %v", recorder.Code, err)
			}
			if allowed {
				if resp.Code != 0 || store.writes != 1 {
					t.Fatalf("%s %s: switch on must keep behaviour, code=%d msg=%s", tt.method, tt.path, resp.Code, resp.Message)
				}
				continue
			}
			if resp.Code != 1 || resp.Message != task.ErrLocalExecDisabled.Error() || store.writes != 0 {
				t.Fatalf("%s %s: code=%d msg=%s writes=%d", tt.method, tt.path, resp.Code, resp.Message, store.writes)
			}
		}
	}
}
