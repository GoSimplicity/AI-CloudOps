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

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/executor"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeDAO "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type CronHandlers struct {
	logger *zap.Logger

	// DAO层依赖
	cronDAO      dao.CronJobDAO
	treeLocalDAO treeDAO.TreeLocalDAO

	// 任务执行器
	commandExecutor *CommandExecutor
	httpExecutor    *HTTPExecutor
	scriptExecutor  *ScriptExecutor
	sshExecutor     *executor.SSHExecutor
}

func NewCronHandlers(
	logger *zap.Logger,
	cronDAO dao.CronJobDAO,
	treeLocalDAO treeDAO.TreeLocalDAO,
) *CronHandlers {
	return &CronHandlers{
		logger:          logger,
		cronDAO:         cronDAO,
		treeLocalDAO:    treeLocalDAO,
		commandExecutor: NewCommandExecutor(logger),
		httpExecutor:    NewHTTPExecutor(logger),
		scriptExecutor:  NewScriptExecutor(logger),
		sshExecutor:     executor.NewSSHExecutor(logger, treeLocalDAO),
	}
}

// CronTaskPayload 任务载荷
type CronTaskPayload struct {
	JobID     int                    `json:"job_id"`
	JobName   string                 `json:"job_name"`
	TaskType  model.CronJobType      `json:"task_type"`
	TriggerBy string                 `json:"trigger_by,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ProcessTask 处理任务的主入口
func (h *CronHandlers) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload CronTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.logger.Error("解析任务载荷失败", zap.Error(err), zap.String("payload", string(t.Payload())))
		return fmt.Errorf("解析任务载荷失败: %w", err)
	}

	h.logger.Info("开始处理任务",
		zap.Int("jobID", payload.JobID),
		zap.String("jobName", payload.JobName),
		zap.Int8("taskType", int8(payload.TaskType)))

	// 获取任务详情
	job, err := h.cronDAO.GetCronJob(ctx, payload.JobID)
	if err != nil {
		// 如果任务不存在，记录警告但不返回错误，这样可以防止调度器重复尝试
		if err.Error() == "任务不存在" || err.Error() == "record not found" {
			h.logger.Warn("任务已被删除或不存在，跳过执行",
				zap.Int("jobID", payload.JobID),
				zap.String("jobName", payload.JobName),
				zap.Error(err))
			return nil // 返回nil表示任务"成功"完成（虽然是跳过）
		}
		h.logger.Error("获取任务详情失败", zap.Int("jobID", payload.JobID), zap.Error(err))
		return fmt.Errorf("获取任务详情失败: %w", err)
	}

	// 检查任务状态
	if job.Status != model.CronJobStatusEnabled {
		h.logger.Warn("任务已禁用，跳过执行", zap.Int("jobID", payload.JobID))
		return nil
	}

	// 更新任务状态为运行中
	startTime := time.Now()
	if err := h.cronDAO.UpdateCronJobStatus(ctx, payload.JobID, model.CronJobStatusRunning); err != nil {
		h.logger.Error("更新任务状态失败", zap.Int("jobID", payload.JobID), zap.Error(err))
	}

	// 执行任务
	var result *ExecutionResult
	switch job.JobType {
	case model.CronJobTypeCommand:
		task := &CommandTask{
			Command:     job.Command,
			Args:        job.Args,
			WorkDir:     job.WorkDir,
			Environment: job.Environment,
			Timeout:     job.Timeout,
		}
		output, err := h.commandExecutor.Execute(ctx, task)
		if err != nil {
			result = &ExecutionResult{
				Success:  false,
				Output:   output,
				ErrorMsg: err.Error(),
			}
		} else {
			result = &ExecutionResult{
				Success:  true,
				Output:   output,
				ErrorMsg: "",
			}
		}
	case model.CronJobTypeHTTP:
		task := &HTTPTask{
			Method:  job.HTTPMethod,
			URL:     job.HTTPUrl,
			Headers: job.HTTPHeaders,
			Body:    job.HTTPBody,
			Timeout: job.Timeout,
		}
		output, err := h.httpExecutor.Execute(ctx, task)
		if err != nil {
			result = &ExecutionResult{
				Success:  false,
				Output:   output,
				ErrorMsg: err.Error(),
			}
		} else {
			result = &ExecutionResult{
				Success:  true,
				Output:   output,
				ErrorMsg: "",
			}
		}
	case model.CronJobTypeScript:
		task := &ScriptTask{
			Type:    job.ScriptType,
			Content: job.ScriptContent,
			Timeout: job.Timeout,
		}
		output, err := h.scriptExecutor.Execute(ctx, task)
		if err != nil {
			result = &ExecutionResult{
				Success:  false,
				Output:   output,
				ErrorMsg: err.Error(),
			}
		} else {
			result = &ExecutionResult{
				Success:  true,
				Output:   output,
				ErrorMsg: "",
			}
		}
	case model.CronJobTypeSSH:
		output, err := h.sshExecutor.ExecuteSSHJob(ctx, job)
		if err != nil {
			result = &ExecutionResult{
				Success:  false,
				Output:   output,
				ErrorMsg: err.Error(),
			}
		} else {
			result = &ExecutionResult{
				Success:  true,
				Output:   output,
				ErrorMsg: "",
			}
		}
	default:
		result = &ExecutionResult{
			Success:  false,
			Output:   "",
			ErrorMsg: fmt.Sprintf("不支持的任务类型: %d", int8(job.JobType)),
		}
	}

	// 计算执行时长
	duration := int(time.Since(startTime).Milliseconds())

	// 更新任务运行信息
	var status int8 = 2 // 失败
	if result.Success {
		status = 1 // 成功
	}

	if err := h.cronDAO.UpdateCronJobRunInfo(ctx, payload.JobID, &startTime, status, duration, result.ErrorMsg); err != nil {
		h.logger.Error("更新任务运行信息失败", zap.Int("jobID", payload.JobID), zap.Error(err))
	}

	// 恢复任务状态为启用
	if err := h.cronDAO.UpdateCronJobStatus(ctx, payload.JobID, model.CronJobStatusEnabled); err != nil {
		h.logger.Error("恢复任务状态失败", zap.Int("jobID", payload.JobID), zap.Error(err))
	}

	if result.Success {
		h.logger.Info("任务执行成功",
			zap.Int("jobID", payload.JobID),
			zap.String("jobName", payload.JobName),
			zap.Int("duration", duration))
		return nil
	} else {
		h.logger.Error("任务执行失败",
			zap.Int("jobID", payload.JobID),
			zap.String("jobName", payload.JobName),
			zap.String("error", result.ErrorMsg),
			zap.Int("duration", duration))
		return fmt.Errorf("任务执行失败: %s", result.ErrorMsg)
	}
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error_msg"`
}
