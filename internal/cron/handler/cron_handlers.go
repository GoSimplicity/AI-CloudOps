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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	treeDAO "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type CronHandlers struct {
	logger *zap.Logger

	// DAO层依赖
	cronDAO      dao.CronJobDAO
	treeLocalDAO treeDAO.TreeLocalDAO
	onDutyDAO    alert.AlertManagerOnDutyDAO
	k8sDAO       k8sDao.ClusterDAO

	// 系统任务依赖
	k8sClient       client.K8sClient
	clusterMgr      manager.ClusterManager
	promConfigCache cache.MonitorCache

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
	onDutyDAO alert.AlertManagerOnDutyDAO,
	k8sDAO k8sDao.ClusterDAO,
	k8sClient client.K8sClient,
	clusterMgr manager.ClusterManager,
	promConfigCache cache.MonitorCache,
) *CronHandlers {
	return &CronHandlers{
		logger:          logger,
		cronDAO:         cronDAO,
		treeLocalDAO:    treeLocalDAO,
		onDutyDAO:       onDutyDAO,
		k8sDAO:          k8sDAO,
		k8sClient:       k8sClient,
		clusterMgr:      clusterMgr,
		promConfigCache: promConfigCache,
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
	case model.CronJobTypeSystem:
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

	// 更新任务运行信息
	var status int8 = 2 // 失败
	if result.Success {
		status = 1 // 成功
	}

	output := result.Output
	if output == "" && result.Success {
		output = "执行成功"
	}

	if err := h.cronDAO.UpdateCronJobRunInfo(ctx, payload.JobID, &startTime, status, duration, output, result.ErrorMsg); err != nil {
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

// executeSystemTask 执行系统内置任务
func (h *CronHandlers) executeSystemTask(ctx context.Context, job *model.CronJob) (string, error) {
	taskType := job.Command // 任务类型存储在Command字段中

	h.logger.Info("手动执行系统内置任务",
		zap.Int("jobID", job.ID),
		zap.String("jobName", job.Name),
		zap.String("taskType", taskType))

	switch taskType {
	case "on_duty_history":
		h.logger.Info("执行值班历史记录管理任务")
		return h.executeOnDutyHistoryTask(ctx)

	case "k8s_status_check":
		h.logger.Info("执行K8s集群状态检查任务")
		return h.executeK8sStatusCheckTask(ctx)

	case "prometheus_config_refresh":
		h.logger.Info("执行Prometheus配置刷新任务")
		return h.executePrometheusConfigRefreshTask(ctx)

	default:
		h.logger.Warn("未知的系统任务类型", zap.String("taskType", taskType))
		return fmt.Sprintf("未知的系统任务类型: %s", taskType), fmt.Errorf("未知的系统任务类型: %s", taskType)
	}
}

// executeOnDutyHistoryTask 执行值班历史记录管理任务
func (h *CronHandlers) executeOnDutyHistoryTask(ctx context.Context) (string, error) {
	h.logger.Info("开始执行值班历史记录管理任务")
	// 这里可以实现值班历史记录的核心逻辑
	// 为了简化，目前只返回成功消息
	return "值班历史记录管理任务执行成功", nil
}

// executeK8sStatusCheckTask 执行K8s集群状态检查任务
func (h *CronHandlers) executeK8sStatusCheckTask(ctx context.Context) (string, error) {
	h.logger.Info("开始执行K8s集群状态检查任务")

	// 获取所有集群
	clusters, _, err := h.k8sDAO.GetClusterList(ctx, &model.ListClustersReq{
		ListReq: model.ListReq{
			Page: 1,
			Size: 100, // 获取前100个集群
		},
	})
	if err != nil {
		h.logger.Error("获取集群列表失败", zap.Error(err))
		return "", fmt.Errorf("获取集群列表失败: %w", err)
	}

	if len(clusters) == 0 {
		return "没有找到K8s集群", nil
	}

	checkedCount := 0
	errorCount := 0

	// 检查每个集群的状态
	for _, cluster := range clusters {
		if err := h.clusterMgr.CheckClusterStatus(ctx, cluster.ID); err != nil {
			h.logger.Warn("集群状态检查失败",
				zap.String("cluster", cluster.Name),
				zap.Error(err))
			errorCount++
		} else {
			checkedCount++
		}
	}

	return fmt.Sprintf("K8s集群状态检查完成: 总计%d个集群，成功检查%d个，失败%d个",
		len(clusters), checkedCount, errorCount), nil
}

// executePrometheusConfigRefreshTask 执行Prometheus配置刷新任务
func (h *CronHandlers) executePrometheusConfigRefreshTask(ctx context.Context) (string, error) {
	h.logger.Info("开始执行Prometheus配置刷新任务")

	if err := h.promConfigCache.MonitorCacheManager(ctx); err != nil {
		h.logger.Error("Prometheus配置刷新失败", zap.Error(err))
		return "", fmt.Errorf("Prometheus配置刷新失败: %w", err)
	}

	return "Prometheus配置刷新成功", nil
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success  bool   `json:"success"`
	Output   string `json:"output"`
	ErrorMsg string `json:"error_msg"`
}
