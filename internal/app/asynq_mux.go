package app

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
)

// NewTaskMux 在唯一实际 worker 上同时注册定时任务与现有通知任务
func NewTaskMux(cron asynq.Handler, processor *notification.Processor) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.Handle("cron:task", cron)
	processor.RegisterTasks(mux)
	return mux
}
