package service

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type retryStore struct {
	dao.CronJobDAO
	created *model.CronJob
	job     *model.CronJob
}

func (s *retryStore) CreateCronJob(_ context.Context, job *model.CronJob) error {
	copy := *job
	s.created = &copy
	return nil
}
func (s *retryStore) GetCronJob(context.Context, int) (*model.CronJob, error) { return s.job, nil }

type retryQueue struct{ options []asynq.Option }

func (s *retryQueue) EnqueueContext(_ context.Context, _ *asynq.Task, options ...asynq.Option) (*asynq.TaskInfo, error) {
	s.options = options
	return &asynq.TaskInfo{ID: "fixture"}, nil
}

func TestCreateRetryDTOOmittedAndExplicitZero(t *testing.T) {
	for _, tt := range []struct {
		name, extra string
		want        int
		invalid     bool
	}{
		{name: "omitted", want: 3}, {name: "zero", extra: `,"max_retry":0`, want: 0}, {name: "nonzero", extra: `,"max_retry":5`, want: 5},
		{name: "negative", extra: `,"max_retry":-1`, invalid: true}, {name: "over limit", extra: `,"max_retry":11`, invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			context, _ := gin.CreateTestContext(httptest.NewRecorder())
			context.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"job","job_type":2,"schedule":"@every 1h","command":"true"`+tt.extra+`}`))
			context.Request.Header.Set("Content-Type", "application/json")
			var request model.CreateCronJobReq
			err := context.ShouldBindJSON(&request)
			if (err != nil) != tt.invalid {
				t.Fatalf("unexpected DTO validation: %v", err)
			}
			if tt.invalid {
				return
			}
			store := &retryStore{}
			service := &cronService{logger: zap.NewNop(), cronDAO: store}
			if err := service.CreateCronJob(context.Request.Context(), &request); err != nil {
				t.Fatal(err)
			}
			if store.created.MaxRetry != tt.want {
				t.Fatal("request retry semantics changed")
			}
		})
	}
}

func TestManualEnqueuePassesConfiguredRetryToSDK(t *testing.T) {
	for _, count := range []int{0, 3, 10} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			queue := &retryQueue{}
			store := &retryStore{job: &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusEnabled, MaxRetry: count}}
			service := &cronService{logger: zap.NewNop(), cronDAO: store, client: queue}
			if err := service.TriggerCronJob(context.Background(), 1); err != nil {
				t.Fatal(err)
			}
			found := false
			for _, option := range queue.options {
				if option.Type() == asynq.MaxRetryOpt {
					found = true
					if option.Value().(int) != count {
						t.Fatal("wrong Asynq retry budget")
					}
				}
			}
			if !found {
				t.Fatal("missing explicit retry option would fall back to SDK 25")
			}
		})
	}
}
