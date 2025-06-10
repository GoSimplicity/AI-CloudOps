/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type TimedTask struct {
	l         *zap.Logger
	k8sClient client.K8sClient
	promCache cache.MonitorCache
	cronMgr   cron.CronManager
}

type TimedPayload struct {
	TaskName    string    `json:"task_name"`
	LastRunTime time.Time `json:"last_run_time"`
}

func NewTimedTask(l *zap.Logger, k8sClient client.K8sClient, promCache cache.MonitorCache, cronMgr cron.CronManager) *TimedTask {
	return &TimedTask{
		l:         l,
		k8sClient: k8sClient,
		promCache: promCache,
		cronMgr:   cronMgr,
	}
}

func (t *TimedTask) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload TimedPayload

	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("解析任务载荷失败: %v: %w", err, asynq.SkipRetry)
	}

	t.l.Info("开始处理定时任务",
		zap.String("task_name", payload.TaskName),
		zap.Time("last_run_time", payload.LastRunTime))

	taskCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 定义任务处理映射
	taskHandlers := map[string]func(context.Context) error{
		RefreshPrometheusCacheTask: t.promCache.MonitorCacheManager,
		CheckK8sStatusTask:         t.cronMgr.StartCheckK8sStatusManager,
		CheckHostStatusTask:        t.cronMgr.StartCheckHostStatusManager,
	}

	// 获取对应的处理函数
	handler, exists := taskHandlers[payload.TaskName]
	if !exists {
		return fmt.Errorf("未知的任务类型: %s", payload.TaskName)
	}

	// 执行任务处理
	if err := handler(taskCtx); err != nil {
		t.l.Error("任务执行失败",
			zap.String("task_name", payload.TaskName),
			zap.Error(err))
		return fmt.Errorf("%s: %w", payload.TaskName, err)
	}

	t.l.Info("成功完成任务", zap.String("task_name", payload.TaskName))
	return nil
}
