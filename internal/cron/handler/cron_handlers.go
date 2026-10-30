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
	"errors"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	croncore "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/executor"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/task"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type CronHandlers struct {
	leaseTicks <-chan time.Time
	logger     *zap.Logger

	// DAO层依赖
	cronDAO   dao.CronJobDAO
	onDutyDAO alert.AlertManagerOnDutyDAO
	k8sDAO    k8sDao.ClusterDAO

	k8sClient       client.K8sClient
	clusterMgr      manager.ClusterManager
	promConfigCache cache.MonitorCache

	commandExecutor *CommandExecutor
	httpExecutor    *HTTPExecutor
	scriptExecutor  *ScriptExecutor
	sshExecutor     *executor.SSHExecutor

	// localExecAllowed 读取 cron.allow_local_exec；为 nil 时读取全局配置（测试可替换）
	localExecAllowed func() bool
}

// allowLocalExec 当前是否允许执行命令行/脚本任务
func (h *CronHandlers) allowLocalExec() bool {
	if h.localExecAllowed != nil {
		return h.localExecAllowed()
	}
	return config.CronLocalExecAllowed()
}

func NewCronHandlers(
	logger *zap.Logger,
	cronDAO dao.CronJobDAO,
	sshExecutor *executor.SSHExecutor,
	onDutyDAO alert.AlertManagerOnDutyDAO,
	k8sDAO k8sDao.ClusterDAO,
	k8sClient client.K8sClient,
	clusterMgr manager.ClusterManager,
	promConfigCache cache.MonitorCache,
) *CronHandlers {
	return &CronHandlers{
		logger:          logger,
		cronDAO:         cronDAO,
		onDutyDAO:       onDutyDAO,
		k8sDAO:          k8sDAO,
		k8sClient:       k8sClient,
		clusterMgr:      clusterMgr,
		promConfigCache: promConfigCache,
		commandExecutor: NewCommandExecutor(logger),
		httpExecutor:    NewHTTPExecutor(logger),
		scriptExecutor:  NewScriptExecutor(logger),
		sshExecutor:     sshExecutor,
	}
}

// IsTaskFailure 供内置调度器识别租约协调失败，避免已有执行完成后重复执行
func (h *CronHandlers) IsTaskFailure(err error) bool { return IsTaskFailure(err) }

func (h *CronHandlers) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload task.CronTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		h.logger.Error("解析任务载荷失败", zap.Error(err), zap.String("payload", string(t.Payload())))
		return fmt.Errorf("解析任务载荷失败: %w", err)
	}

	h.logger.Info("开始处理任务",
		zap.Int("jobID", payload.JobID),
		zap.String("jobName", payload.JobName),
		zap.Int8("taskType", int8(payload.TaskType)))

	job, err := h.cronDAO.GetCronJob(ctx, payload.JobID)
	if err != nil {
		// 如果任务不存在，记录信息级别日志并跳过执行，这样可以防止调度器重复尝试
		if err.Error() == "任务不存在" || err.Error() == "record not found" {
			h.logger.Info("任务已被删除，跳过执行（正常情况，调度器将在下次重新加载时清理）",
				zap.Int("jobID", payload.JobID),
				zap.String("jobName", payload.JobName))
			return nil // 返回nil表示任务"成功"完成（跳过执行）
		}
		h.logger.Error("获取任务详情失败", zap.Int("jobID", payload.JobID), zap.Error(err))
		return fmt.Errorf("获取任务详情失败: %w", err)
	}

	if payload.TriggerBy == "builtin_scheduled" {
		if job.IsBuiltIn != 1 || job.JobType != model.CronJobTypeSystem || payload.Data["schedule"] != job.Schedule || payload.Data["command"] != job.Command {
			return nil
		}
	}
	if job.Status != model.CronJobStatusEnabled && job.Status != model.CronJobStatusRunning {
		h.logger.Warn("任务已禁用，跳过执行", zap.Int("jobID", payload.JobID))
		return nil
	}

	if job.Status == model.CronJobStatusRunning && job.LeaseOwner == "" {
		return fmt.Errorf("旧版Running任务缺少执行归属，须在停止旧worker并核对结果后重新启用")
	}
	timeout := time.Duration(job.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancelExecution := context.WithTimeout(ctx, timeout)
	defer cancelExecution()
	startTime := time.Now()
	owner := uuid.NewString()
	claimed, err := h.cronDAO.AcquireCronJobLease(ctx, payload.JobID, owner)
	if err != nil {
		return fmt.Errorf("获取任务执行租约失败: %w", err)
	}
	if !claimed {
		return errCronLeaseBusy
	}
	stopLease := h.maintainLease(ctx, cancelExecution, payload.JobID, owner)
	defer stopLease()

	var result *ExecutionResult
	localExecBlocked := task.IsLocalExecJobType(job.JobType) && !h.allowLocalExec()
	if localExecBlocked {
		// 只记录任务ID与类型，不记录命令/脚本内容；按失败记录本次运行，不伪装成功
		h.logger.Warn("本地命令/脚本执行已停用，跳过任务",
			zap.Int("jobID", job.ID),
			zap.Int8("jobType", int8(job.JobType)))
		result = &ExecutionResult{
			Success:  false,
			ErrorMsg: task.ErrLocalExecDisabled.Error(),
		}
	}
	switch {
	case localExecBlocked:
	case job.JobType == model.CronJobTypeCommand:
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
	case job.JobType == model.CronJobTypeHTTP:
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
	case job.JobType == model.CronJobTypeScript:
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
	case job.JobType == model.CronJobTypeSSH:
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
	case job.JobType == model.CronJobTypeSystem:
		output, err := h.executeSystemTask(ctx, job)
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

	var status int8 = 2 // 失败
	if result.Success {
		status = 1 // 成功
	}

	output := result.Output
	if output == "" && result.Success {
		output = "执行成功"
	}

	if err := stopLease(); err != nil {
		return err
	}
	// 任务取消后仍有界保存结果，但只能完成自己仍有效的租约
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	completed, err := h.cronDAO.CompleteCronJobLease(cleanupCtx, payload.JobID, owner, &startTime, status, duration, output, result.ErrorMsg)
	if err != nil {
		return fmt.Errorf("完成任务租约失败: %w", err)
	}
	if !completed {
		return errCronLeaseLost
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
		if localExecBlocked {
			// 配置未开启前重试没有意义，不消耗重试次数
			return fmt.Errorf("任务执行失败: %w: %w", task.ErrLocalExecDisabled, asynq.SkipRetry)
		}
		return fmt.Errorf("任务执行失败: %s", result.ErrorMsg)
	}
}

func (h *CronHandlers) executeSystemTask(ctx context.Context, job *model.CronJob) (string, error) {
	if job == nil || job.IsBuiltIn != 1 {
		return "", errors.New("拒绝执行非内置系统任务")
	}
	return croncore.ExecuteBuiltinOnce(ctx, job.Command, h.logger, h.onDutyDAO, h.k8sDAO, h.clusterMgr, h.promConfigCache)
}

type ExecutionResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error_msg"`
}
