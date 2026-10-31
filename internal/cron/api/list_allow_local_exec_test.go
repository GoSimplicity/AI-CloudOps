package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type listStore struct {
	dao.CronJobDAO
	jobs []*model.CronJob
}

func (s *listStore) GetCronJobList(context.Context, *model.GetCronJobListReq) ([]*model.CronJob, int64, error) {
	return s.jobs, int64(len(s.jobs)), nil
}

// 方案 A 契约：GET /api/cron/job/list 的 data 在 items/total 之外附带只读字段 allow_local_exec
func TestCronJobListExposesAllowLocalExec(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		store := &listStore{jobs: []*model.CronJob{{
			Model:         model.Model{ID: 1},
			Name:          "http-job",
			JobType:       model.CronJobTypeHTTP,
			Status:        model.CronJobStatusDisabled,
			Schedule:      "@every 1h",
			CreatedBy:     7,
			CreatedByName: "operator",
		}}}
		handler := NewCronJobHandler(zap.NewNop(), service.NewCronService(zap.NewNop(), store, nil, nil, nil, func() bool { return allowed }))
		gin.SetMode(gin.TestMode)
		engine := gin.New()
		handler.RegisterRouters(engine)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/cron/job/list?page=1&size=10", nil))
		if recorder.Code != 200 {
			t.Fatalf("allowed=%v: unexpected status %d", allowed, recorder.Code)
		}
		t.Logf("allowed=%v response: %s", allowed, recorder.Body.String())

		var resp struct {
			Code int                        `json:"code"`
			Data map[string]json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil || resp.Code != 0 {
			t.Fatalf("allowed=%v: code=%d err=%v body=%s", allowed, resp.Code, err, recorder.Body.String())
		}
		for _, key := range []string{"items", "total", "allow_local_exec"} {
			if _, ok := resp.Data[key]; !ok {
				t.Fatalf("allowed=%v: data 缺少字段 %s: %s", allowed, key, recorder.Body.String())
			}
		}
		var got bool
		if err := json.Unmarshal(resp.Data["allow_local_exec"], &got); err != nil || got != allowed {
			t.Fatalf("allow_local_exec=%s, want %v", resp.Data["allow_local_exec"], allowed)
		}
		var total int64
		var items []map[string]any
		if json.Unmarshal(resp.Data["total"], &total) != nil || total != 1 ||
			json.Unmarshal(resp.Data["items"], &items) != nil || len(items) != 1 || items[0]["name"] != "http-job" {
			t.Fatalf("allowed=%v: items/total 被破坏: %s", allowed, recorder.Body.String())
		}
	}
}
