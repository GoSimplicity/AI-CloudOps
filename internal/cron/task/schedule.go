package task

import (
	"errors"
	"strings"

	"github.com/robfig/cron/v3"
)

// ParseSchedule 与 Asynq 调度器使用的五字段标准解析器保持一致
func ParseSchedule(expression string) (cron.Schedule, error) {
	// robfig/cron v3.0.1 对没有空格的时区前缀会切片越界
	if (strings.HasPrefix(expression, "TZ=") || strings.HasPrefix(expression, "CRON_TZ=")) && !strings.Contains(expression, " ") {
		return nil, errors.New("时区前缀后必须提供调度表达式")
	}
	return cron.ParseStandard(expression)
}
