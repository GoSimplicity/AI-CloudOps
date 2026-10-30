package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"testing"
)

func TestCreateCronJobInitialStatus(t *testing.T) {
	for _, tc := range []struct {
		name     string
		supplied bool
		value    int
		valid    bool
		expected model.CronJobStatus
	}{
		{"omitted remains enabled", false, 0, true, model.CronJobStatusEnabled},
		{"enabled", true, 1, true, model.CronJobStatusEnabled},
		{"disabled", true, 2, true, model.CronJobStatusDisabled},
		{"zero", true, 0, false, 0}, {"running", true, 3, false, 0}, {"error", true, 4, false, 0}, {"negative", true, -1, false, 0}, {"overflow", true, 99999, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &localExecStore{}
			req := createReq(model.CronJobTypeHTTP)
			req.HTTPUrl = "http://127.0.0.1:9/fixture"
			if tc.supplied {
				req.Status = &tc.value
			}
			// No queue exists: creation must only persist, never execute or enqueue.
			err := newLocalExecService(false, store, nil).CreateCronJob(context.Background(), req)
			if (err == nil) != tc.valid || (store.created != nil) != tc.valid {
				t.Fatalf("valid=%v error=%v persisted=%v", tc.valid, err, store.created != nil)
			}
			if tc.valid && store.created.Status != tc.expected {
				t.Fatalf("saved status=%d want=%d", store.created.Status, tc.expected)
			}
		})
	}
}

func TestDisabledCronDoesNotBypassExecutionConfigurationGates(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind model.CronJobType
	}{
		{"local command remains closed", model.CronJobTypeCommand},
		{"local script remains closed", model.CronJobTypeScript},
		{"SSH still requires its target", model.CronJobTypeSSH},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &localExecStore{}
			req := createReq(tc.kind)
			status := 2
			req.Status = &status
			if tc.kind == model.CronJobTypeSSH {
				req.SSHResourceID = nil
			}
			if err := newLocalExecService(false, store, nil).CreateCronJob(context.Background(), req); err == nil || store.created != nil {
				t.Fatal("disabled state bypassed an execution configuration gate")
			}
		})
	}
}
