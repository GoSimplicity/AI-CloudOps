package app

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func TestWorkerMuxRegistersCronAndAllNotificationTasks(t *testing.T) {
	cronCalls := 0
	mux := NewTaskMux(asynq.HandlerFunc(func(context.Context, *asynq.Task) error { cronCalls++; return nil }), notification.NewProcessor(nil, zap.NewNop()))
	for _, kind := range []string{"cron:task", model.TaskTypeSendNotification, model.TaskTypeBatchSendNotification, model.TaskTypeScheduledNotification, model.TaskTypeRetryFailedNotification} {
		t.Run(kind, func(t *testing.T) {
			task := asynq.NewTask(kind, nil)
			_, pattern := mux.Handler(task)
			if pattern != kind {
				t.Fatalf("task %s registered as %q", kind, pattern)
			}
			if kind == "cron:task" {
				if err := mux.ProcessTask(context.Background(), task); err != nil || cronCalls != 1 {
					t.Fatal("cron route changed")
				}
			}
		})
	}
	_, pattern := mux.Handler(asynq.NewTask("unknown", nil))
	if pattern != "" {
		t.Fatal("unknown task silently captured")
	}
}
