package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type executionStore struct {
	dao.CronJobDAO
	status          model.CronJobStatus
	claimErr        error
	beforeClaim     func()
	stored          bool
	cleanupCanceled bool
}

func (s *executionStore) GetCronJob(context.Context, int) (*model.CronJob, error) {
	return &model.CronJob{Model: model.Model{ID: 1}, Status: model.CronJobStatusEnabled, JobType: model.CronJobTypeHTTP, HTTPMethod: "GET", HTTPUrl: "http://fake.invalid"}, nil
}
func (s *executionStore) CompareAndSwapCronJobStatus(ctx context.Context, _ int, from, to model.CronJobStatus) (bool, error) {
	if s.beforeClaim != nil {
		s.beforeClaim()
		s.beforeClaim = nil
	}
	if s.claimErr != nil {
		return false, s.claimErr
	}
	if ctx.Err() != nil {
		s.cleanupCanceled = true
		return false, ctx.Err()
	}
	if s.status != from {
		return false, nil
	}
	s.status = to
	return true, nil
}
func (s *executionStore) UpdateCronJobRunInfo(ctx context.Context, _ int, _ *time.Time, _ int8, _ int, _, _ string) error {
	s.cleanupCanceled = ctx.Err() != nil
	s.stored = true
	return ctx.Err()
}

type memoryTransport func(*http.Request) (*http.Response, error)

func (f memoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProcessTaskPreservesAdministrativeState(t *testing.T) {
	for _, tt := range []struct {
		name                                                                                   string
		disableDuringRun, cancelDuringRun, disableBeforeClaim, otherWorkerClaims, claimFailure bool
		wantStatus                                                                             model.CronJobStatus
		wantExecuted, wantError                                                                bool
	}{
		{name: "successful release", wantStatus: model.CronJobStatusEnabled, wantExecuted: true},
		{name: "manual disable survives completion", disableDuringRun: true, wantStatus: model.CronJobStatusDisabled, wantExecuted: true},
		{name: "cancelled execution releases state", cancelDuringRun: true, wantStatus: model.CronJobStatusEnabled, wantExecuted: true, wantError: true},
		{name: "disable after initial read", disableBeforeClaim: true, wantStatus: model.CronJobStatusDisabled},
		{name: "another worker claimed", otherWorkerClaims: true, wantStatus: model.CronJobStatusRunning},
		{name: "claim persistence failure", claimFailure: true, wantStatus: model.CronJobStatusEnabled, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			store := &executionStore{status: model.CronJobStatusEnabled}
			if tt.disableBeforeClaim {
				store.beforeClaim = func() { store.status = model.CronJobStatusDisabled }
			}
			if tt.otherWorkerClaims {
				store.beforeClaim = func() { store.status = model.CronJobStatusRunning }
			}
			if tt.claimFailure {
				store.claimErr = errors.New("database unavailable")
			}
			executed := false
			executor := NewHTTPExecutor(zap.NewNop())
			executor.client.Transport = memoryTransport(func(r *http.Request) (*http.Response, error) {
				executed = true
				if store.status != model.CronJobStatusRunning {
					t.Error("execution was not claimed")
				}
				if tt.disableDuringRun {
					store.status = model.CronJobStatusDisabled
				}
				if tt.cancelDuringRun {
					cancel()
					return nil, context.Canceled
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok")), Header: make(http.Header)}, nil
			})
			handler := &CronHandlers{logger: zap.NewNop(), cronDAO: store, httpExecutor: executor}
			err := handler.ProcessTask(ctx, asynq.NewTask("cron:task", []byte(`{"job_id":1}`)))
			if (err != nil) != tt.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			if executed != tt.wantExecuted || store.stored != tt.wantExecuted {
				t.Fatalf("executed=%v stored=%v", executed, store.stored)
			}
			if store.status != tt.wantStatus {
				t.Fatalf("status=%d want=%d", store.status, tt.wantStatus)
			}
			if store.cleanupCanceled {
				t.Fatal("cleanup reused cancelled execution context")
			}
		})
	}
}
