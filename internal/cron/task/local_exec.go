package task

import (
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ErrLocalExecDisabled 本地命令/脚本执行开关关闭时返回的固定错误，消息对前端稳定可读
var ErrLocalExecDisabled = errors.New("本地命令/脚本任务已停用（cron.allow_local_exec=false），请改用 HTTP 或 SSH 任务")

// IsLocalExecJobType 判断任务类型是否会在后端主机本地执行（命令行、脚本）
func IsLocalExecJobType(jobType model.CronJobType) bool {
	return jobType == model.CronJobTypeCommand || jobType == model.CronJobTypeScript
}
