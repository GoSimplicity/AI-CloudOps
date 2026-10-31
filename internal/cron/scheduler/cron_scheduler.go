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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type scheduledEntry struct {
	id        string
	signature string
	spec      string
	task      *asynq.Task
	options   []asynq.Option
}

// StatusSnapshot 反映本进程最近一次调度配置加载结果，不包含数据库错误正文
type StatusSnapshot struct {
	Running             bool
	Ready               bool
	ConsecutiveFailures int
	LastSuccess         time.Time
}

type taskScheduler interface {
	Register(string, *asynq.Task, ...asynq.Option) (string, error)
	Unregister(string) error
}

type CronScheduler struct {
	mu        sync.Mutex
	entries   map[int]scheduledEntry
	logger    *zap.Logger
	cronDAO   dao.CronJobDAO
	scheduler taskScheduler
	client    *asynq.Client
	statusMu  sync.RWMutex
	status    StatusSnapshot
	retryWait func(context.Context, time.Duration) error
	// localExecAllowed 显式注入动态执行门禁；nil 时拒绝本地执行。
	localExecAllowed config.CronLocalExecPolicy
}

// Status 返回线程安全的本地调度状态快照
func (cs *CronScheduler) Status() StatusSnapshot {
	cs.statusMu.RLock()
	defer cs.statusMu.RUnlock()
	return cs.status
}

func (cs *CronScheduler) setRunning(running bool) {
	cs.statusMu.Lock()
	defer cs.statusMu.Unlock()
	cs.status.Running = running
	if !running {
		cs.status.Ready = false
	}
}

func (cs *CronScheduler) recordLoadResult(err error) {
	cs.statusMu.Lock()
	defer cs.statusMu.Unlock()
	if err != nil {
		cs.status.Ready = false
		cs.status.ConsecutiveFailures++
		return
	}
	cs.status.Ready = true
	cs.status.ConsecutiveFailures = 0
	cs.status.LastSuccess = time.Now().UTC()
}

func (cs *CronScheduler) wait(ctx context.Context, delay time.Duration) error {
	if cs.retryWait != nil {
		return cs.retryWait(ctx, delay)
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// allowLocalExec 当前是否允许调度命令行(2)/脚本(4)任务
func (cs *CronScheduler) allowLocalExec() bool {
	return cs.localExecAllowed.IsAllowed()
}

// shouldSchedule 是否由本调度器登记该任务：系统任务由统一 Cron 管理器负责；
// 开关关闭时命令行/脚本任务不入队（而不是入队后再在执行阶段报失败）
func (cs *CronScheduler) shouldSchedule(job *model.CronJob, allowLocalExec bool) bool {
	if job.JobType == model.CronJobTypeSystem {
		return false
	}
	return allowLocalExec || !task.IsLocalExecJobType(job.JobType)
}

func NewCronScheduler(
	logger *zap.Logger,
	cronDAO dao.CronJobDAO,
	scheduler *asynq.Scheduler,
	client *asynq.Client,
	localExecAllowed config.CronLocalExecPolicy,
) *CronScheduler {
	return &CronScheduler{
		logger:           logger,
		cronDAO:          cronDAO,
		scheduler:        scheduler,
		client:           client,
		localExecAllowed: localExecAllowed,
	}
}

// StartScheduler 启动调度器 - 从数据库加载任务并调度
func (cs *CronScheduler) StartScheduler(ctx context.Context) error {
	cs.logger.Info("启动Cron任务调度器")
	cs.setRunning(true)
	defer cs.setRunning(false)
	const maxDelay = 30 * time.Second
	retryDelay := time.Second
	for {
		if ctx.Err() != nil {
			return nil
		}
		err := cs.loadAndScheduleJobs(ctx)
		cs.recordLoadResult(err)
		delay := maxDelay
		if err != nil {
			cs.logger.Error("加载调度配置失败，等待重试", zap.String("error_type", fmt.Sprintf("%T", err)), zap.Duration("retry_delay", retryDelay))
			delay = retryDelay
			retryDelay *= 2
			if retryDelay > maxDelay {
				retryDelay = maxDelay
			}
		} else {
			retryDelay = time.Second
			cs.logger.Debug("任务配置重新加载完成")
		}
		if cs.wait(ctx, delay) != nil {
			cs.logger.Info("Cron任务调度器已停止")
			return nil
		}
	}
}

func (cs *CronScheduler) loadAndScheduleJobs(ctx context.Context) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	jobs, err := cs.cronDAO.GetEnabledCronJobs(ctx)
	if err != nil {
		cs.logger.Error("获取启用任务列表失败", zap.String("error_type", fmt.Sprintf("%T", err)))
		return err
	}

	// 仅移除本管理器登记且已停用/删除的任务，不影响共享调度器其他条目
	// 开关关闭后，已登记的命令行/脚本任务也会在本轮被移除
	allowLocalExec := cs.allowLocalExec()
	desired := make(map[int]bool, len(jobs))
	for _, job := range jobs {
		if cs.shouldSchedule(job, allowLocalExec) {
			desired[job.ID] = true
		}
	}
	for jobID, entry := range cs.entries {
		if !desired[jobID] {
			if err := cs.scheduler.Unregister(entry.id); err != nil {
				return fmt.Errorf("移除失效调度任务 %d: %w", jobID, err)
			}
			delete(cs.entries, jobID)
		}
	}

	scheduledCount := 0
	skippedCount := 0
	failedCount := 0

	for _, job := range jobs {
		// 跳过系统内置任务，这些任务由统一Cron管理器直接管理
		if job.JobType == model.CronJobTypeSystem {
			cs.logger.Debug("跳过系统内置任务",
				zap.Int("jobID", job.ID),
				zap.String("jobName", job.Name))
			skippedCount++
			continue
		}
		if !cs.shouldSchedule(job, allowLocalExec) {
			cs.logger.Debug("本地执行开关关闭，跳过命令行/脚本任务",
				zap.Int("jobID", job.ID),
				zap.String("jobName", job.Name),
				zap.Int8("jobType", int8(job.JobType)))
			skippedCount++
			continue
		}

		if err := cs.scheduleJob(job); err != nil {
			cs.logger.Error("调度任务失败",
				zap.Int("jobID", job.ID),
				zap.String("jobName", job.Name),
				zap.String("error_type", fmt.Sprintf("%T", err)))
			failedCount++
			continue
		}

		cs.logger.Debug("任务调度成功",
			zap.Int("jobID", job.ID),
			zap.String("jobName", job.Name),
			zap.String("schedule", job.Schedule))
		scheduledCount++
	}

	cs.logger.Info("任务调度完成",
		zap.Int("total", len(jobs)),
		zap.Int("scheduled", scheduledCount),
		zap.Int("skipped", skippedCount),
		zap.Int("failed", failedCount))

	if failedCount > 0 {
		return fmt.Errorf("%d 个任务未能登记", failedCount)
	}
	return nil
}

func (cs *CronScheduler) scheduleJob(job *model.CronJob) error {
	payload := task.CronTaskPayload{
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

	// 与实际 Asynq 使用同一解析合同，失败时不能先注销旧配置
	if _, err := task.ParseSchedule(job.Schedule); err != nil {
		return fmt.Errorf("无效的调度表达式: %w", err)
	}

	// 创建Asynq任务
	queueTask := asynq.NewTask("cron:task", payloadBytes)
	options := []asynq.Option{asynq.MaxRetry(job.MaxRetry)}

	signature := fmt.Sprintf("%d:%s:%s", job.MaxRetry, job.Schedule, string(payloadBytes))
	previous, replacing := cs.entries[job.ID]
	if replacing {
		if previous.signature == signature {
			return nil
		}
		if err := cs.scheduler.Unregister(previous.id); err != nil {
			return fmt.Errorf("注销旧调度失败: %w", err)
		}
		delete(cs.entries, job.ID)
	}
	// 每次入队由 Asynq 生成独立任务 ID，避免已归档任务阻塞后续周期
	// 调度条目的注销仍使用 Register 返回的条目 ID
	entryID, err := cs.scheduler.Register(job.Schedule, queueTask, options...)
	if err != nil {
		if replacing && previous.task != nil {
			restored, restoreErr := cs.scheduler.Register(previous.spec, previous.task, previous.options...)
			if restoreErr != nil {
				return fmt.Errorf("新调度登记失败且旧调度恢复失败: %w", errors.Join(err, restoreErr))
			}
			previous.id = restored
			cs.entries[job.ID] = previous
			return fmt.Errorf("新调度登记失败，旧配置已恢复: %w", err)
		}
		return fmt.Errorf("调度登记失败: %w", err)
	}
	if cs.entries == nil {
		cs.entries = make(map[int]scheduledEntry)
	}
	cs.entries[job.ID] = scheduledEntry{id: entryID, signature: signature, spec: job.Schedule, task: queueTask, options: options}
	return nil
}

// RemoveScheduledJob 立即从调度器中移除指定任务
func (cs *CronScheduler) RemoveScheduledJob(jobID int) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	entry, exists := cs.entries[jobID]
	if !exists {
		return nil
	}
	entryID := entry.id
	err := cs.scheduler.Unregister(entryID)
	if err != nil {
		cs.logger.Warn("从调度器移除任务失败，可能任务不存在",
			zap.Int("jobID", jobID),
			zap.String("entryID", entryID),
			zap.Error(err))
		return err
	}

	delete(cs.entries, jobID)
	cs.logger.Info("成功从调度器移除任务",
		zap.Int("jobID", jobID),
		zap.String("entryID", entryID))
	return nil
}
