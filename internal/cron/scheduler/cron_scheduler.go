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

package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/handler"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// CronScheduler 定时任务调度器
type CronScheduler struct {
	logger    *zap.Logger
	cronDAO   dao.CronJobDAO
	scheduler *asynq.Scheduler
	client    *asynq.Client
}

// NewCronScheduler 创建调度器
func NewCronScheduler(
	logger *zap.Logger,
	cronDAO dao.CronJobDAO,
	scheduler *asynq.Scheduler,
	client *asynq.Client,
) *CronScheduler {
	return &CronScheduler{
		logger:    logger,
		cronDAO:   cronDAO,
		scheduler: scheduler,
		client:    client,
	}
}

// StartScheduler 启动调度器 - 从数据库加载任务并调度
func (cs *CronScheduler) StartScheduler(ctx context.Context) error {
	cs.logger.Info("启动Cron任务调度器")

	// 加载并注册所有启用的任务
	if err := cs.loadAndScheduleJobs(ctx); err != nil {
		cs.logger.Error("加载任务失败", zap.Error(err))
		return err
	}

	// 定期重新加载任务配置（每分钟检查一次）
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			cs.logger.Info("Cron任务调度器已停止")
			return nil
		case <-ticker.C:
			if err := cs.loadAndScheduleJobs(ctx); err != nil {
				cs.logger.Error("重新加载任务失败", zap.Error(err))
			}
		}
	}
}

// loadAndScheduleJobs 加载并调度任务
func (cs *CronScheduler) loadAndScheduleJobs(ctx context.Context) error {
	// 获取所有启用的任务
	enabledStatus := model.CronJobStatusEnabled
	jobs, _, err := cs.cronDAO.GetCronJobList(ctx, &model.GetCronJobListReq{
		ListReq: model.ListReq{Page: 1, Size: 1000}, // 获取所有任务
		Status:  &enabledStatus,                     // 只获取启用的任务
	})
	if err != nil {
		return err
	}

	// 清除现有的调度任务
	cs.scheduler.Unregister("*")

	// 为每个任务注册调度
	for _, job := range jobs {
		if err := cs.scheduleJob(job); err != nil {
			cs.logger.Error("调度任务失败",
				zap.Int("jobID", job.ID),
				zap.String("jobName", job.Name),
				zap.Error(err))
			continue
		}
		cs.logger.Info("任务调度成功",
			zap.Int("jobID", job.ID),
			zap.String("jobName", job.Name),
			zap.String("schedule", job.Schedule))
	}

	return nil
}

// scheduleJob 调度单个任务
func (cs *CronScheduler) scheduleJob(job *model.CronJob) error {
	// 创建任务载荷
	payload := handler.CronTaskPayload{
		JobID:    job.ID,
		JobName:  job.Name,
		TaskType: job.JobType,
		Data: map[string]interface{}{
			"schedule": job.Schedule,
		},
	}

	// 序列化载荷
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 创建Asynq任务
	task := asynq.NewTask("cron:task", payloadBytes)

	// 使用任务ID作为调度ID
	entryID := generateEntryID(job.ID)

	// 注册到调度器
	_, err = cs.scheduler.Register(job.Schedule, task, asynq.TaskID(entryID))
	return err
}

// generateEntryID 生成调度条目ID
func generateEntryID(jobID int) string {
	return "cron_job_" + fmt.Sprintf("%d", jobID)
}
