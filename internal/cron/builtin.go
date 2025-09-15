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

package cron

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// BuiltinTaskDefinition 内置任务定义
type BuiltinTaskDefinition struct {
	Name        string
	Description string
	Schedule    string // cron表达式，留空表示使用默认间隔
	TaskType    string // 任务标识符，用于启动对应的处理函数
	Enabled     bool   // 默认是否启用
}

// GetBuiltinTasks 获取所有内置任务定义
func GetBuiltinTasks() []BuiltinTaskDefinition {
	return []BuiltinTaskDefinition{
		{
			Name:        "值班历史记录管理",
			Description: "自动管理值班历史记录，处理换班和轮替逻辑",
			Schedule:    "*/10 * * * *", // 每10分钟执行一次
			TaskType:    "on_duty_history",
			Enabled:     true,
		},
		{
			Name:        "K8s集群状态检查",
			Description: "定期检查Kubernetes集群状态并更新数据库",
			Schedule:    "0 * * * *", // 每小时执行一次
			TaskType:    "k8s_status_check",
			Enabled:     true,
		},
		{
			Name:        "Prometheus配置刷新",
			Description: "定期刷新Prometheus监控配置",
			Schedule:    "*/15 * * * *", // 每15分钟执行一次
			TaskType:    "prometheus_config_refresh",
			Enabled:     true,
		},
	}
}

// BuiltinTaskManager 内置任务管理器
type BuiltinTaskManager struct {
	logger  *zap.Logger
	cronDAO dao.CronJobDAO
}

// NewBuiltinTaskManager 创建内置任务管理器
func NewBuiltinTaskManager(logger *zap.Logger, cronDAO dao.CronJobDAO) *BuiltinTaskManager {
	return &BuiltinTaskManager{
		logger:  logger,
		cronDAO: cronDAO,
	}
}

// InitializeBuiltinTasks 初始化内置任务到数据库
func (btm *BuiltinTaskManager) InitializeBuiltinTasks(ctx context.Context) error {
	btm.logger.Info("开始初始化内置任务")

	builtinTasks := GetBuiltinTasks()

	for _, taskDef := range builtinTasks {
		// 检查任务是否已存在
		existingJob, err := btm.cronDAO.GetCronJobByName(ctx, taskDef.Name)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			btm.logger.Error("检查内置任务是否存在失败",
				zap.String("taskName", taskDef.Name),
				zap.Error(err))
			continue
		}

		if existingJob != nil {
			// 如果任务存在，检查是否需要更新配置
			needsUpdate := false
			if existingJob.Schedule != taskDef.Schedule {
				btm.logger.Info("内置任务调度表达式需要更新",
					zap.String("taskName", taskDef.Name),
					zap.String("oldSchedule", existingJob.Schedule),
					zap.String("newSchedule", taskDef.Schedule))
				existingJob.Schedule = taskDef.Schedule
				needsUpdate = true
			}
			if existingJob.Description != taskDef.Description {
				existingJob.Description = taskDef.Description
				needsUpdate = true
			}
			if existingJob.Command != taskDef.TaskType {
				existingJob.Command = taskDef.TaskType
				needsUpdate = true
			}

			if needsUpdate {
				if err := btm.cronDAO.UpdateCronJob(ctx, existingJob); err != nil {
					btm.logger.Error("更新内置任务失败",
						zap.String("taskName", taskDef.Name),
						zap.Error(err))
				} else {
					btm.logger.Info("内置任务更新成功",
						zap.String("taskName", taskDef.Name),
						zap.Int("taskID", existingJob.ID))
				}
			} else {
				btm.logger.Info("内置任务已存在且配置正确，跳过",
					zap.String("taskName", taskDef.Name),
					zap.Int("taskID", existingJob.ID))
			}
			continue
		}

		// 创建内置任务
		job := &model.CronJob{
			Name:        taskDef.Name,
			Description: taskDef.Description,
			JobType:     model.CronJobTypeSystem,
			Status:      model.CronJobStatusEnabled,
			IsBuiltIn:   true,
			Schedule:    taskDef.Schedule,
			Timeout:     300,              // 5分钟超时
			MaxRetry:    3,                // 最大重试3次
			Command:     taskDef.TaskType, // 将任务类型存储在Command字段中
		}

		if !taskDef.Enabled {
			job.Status = model.CronJobStatusDisabled
		}

		if err := btm.cronDAO.CreateCronJob(ctx, job); err != nil {
			btm.logger.Error("创建内置任务失败",
				zap.String("taskName", taskDef.Name),
				zap.Error(err))
			continue
		}

		btm.logger.Info("成功创建内置任务",
			zap.String("taskName", taskDef.Name),
			zap.Int("taskID", job.ID),
			zap.String("schedule", job.Schedule))
	}

	btm.logger.Info("内置任务初始化完成")
	return nil
}

// GetEnabledBuiltinTasks 获取启用的内置任务
func (btm *BuiltinTaskManager) GetEnabledBuiltinTasks(ctx context.Context) ([]*model.CronJob, error) {
	// 获取所有启用的内置系统任务
	enabledStatus := model.CronJobStatusEnabled
	systemJobType := model.CronJobTypeSystem

	jobs, _, err := btm.cronDAO.GetCronJobList(ctx, &model.GetCronJobListReq{
		ListReq: model.ListReq{Page: 1, Size: 100},
		Status:  &enabledStatus,
		JobType: &systemJobType,
	})
	if err != nil {
		return nil, err
	}

	// 过滤出内置任务
	var builtinJobs []*model.CronJob
	for _, job := range jobs {
		if job.IsBuiltIn {
			builtinJobs = append(builtinJobs, job)
		}
	}

	return builtinJobs, nil
}

// ForceInitializeBuiltinTasks 强制重新初始化内置任务（用于修复）
func (btm *BuiltinTaskManager) ForceInitializeBuiltinTasks(ctx context.Context) error {
	btm.logger.Info("开始强制重新初始化内置任务")

	builtinTasks := GetBuiltinTasks()
	successCount := 0
	errorCount := 0

	for _, taskDef := range builtinTasks {
		// 首先尝试删除现有的任务（如果存在）
		existingJob, err := btm.cronDAO.GetCronJobByName(ctx, taskDef.Name)
		if err == nil && existingJob != nil {
			btm.logger.Info("发现现有内置任务，将删除并重建",
				zap.String("taskName", taskDef.Name),
				zap.Int("existingID", existingJob.ID))

			// 注意：这里我们直接删除，不通过常规的DeleteCronJob方法（它会检查IsBuiltIn）
			if err := btm.cronDAO.DeleteCronJob(ctx, existingJob.ID); err != nil {
				btm.logger.Error("删除现有内置任务失败",
					zap.String("taskName", taskDef.Name),
					zap.Error(err))
			}
		}

		// 创建新的内置任务
		job := &model.CronJob{
			Name:        taskDef.Name,
			Description: taskDef.Description,
			JobType:     model.CronJobTypeSystem,
			Status:      model.CronJobStatusEnabled,
			IsBuiltIn:   true,
			Schedule:    taskDef.Schedule,
			Timeout:     300,              // 5分钟超时
			MaxRetry:    3,                // 最大重试3次
			Command:     taskDef.TaskType, // 将任务类型存储在Command字段中
		}

		if !taskDef.Enabled {
			job.Status = model.CronJobStatusDisabled
		}

		if err := btm.cronDAO.CreateCronJob(ctx, job); err != nil {
			btm.logger.Error("强制创建内置任务失败",
				zap.String("taskName", taskDef.Name),
				zap.Error(err))
			errorCount++
			continue
		}

		btm.logger.Info("强制创建内置任务成功",
			zap.String("taskName", taskDef.Name),
			zap.Int("taskID", job.ID),
			zap.String("schedule", job.Schedule))
		successCount++
	}

	btm.logger.Info("强制初始化内置任务完成",
		zap.Int("success", successCount),
		zap.Int("errors", errorCount))

	if errorCount > 0 {
		return fmt.Errorf("强制初始化过程中有 %d 个任务失败", errorCount)
	}
	return nil
}

// ValidateBuiltinTasks 验证内置任务完整性
func (btm *BuiltinTaskManager) ValidateBuiltinTasks(ctx context.Context) error {
	btm.logger.Info("开始验证内置任务完整性")

	builtinTaskDefs := GetBuiltinTasks()
	missingTasks := []string{}

	for _, taskDef := range builtinTaskDefs {
		_, err := btm.cronDAO.GetCronJobByName(ctx, taskDef.Name)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				missingTasks = append(missingTasks, taskDef.Name)
			} else {
				btm.logger.Error("检查内置任务时发生错误",
					zap.String("taskName", taskDef.Name),
					zap.Error(err))
			}
		}
	}

	if len(missingTasks) > 0 {
		btm.logger.Error("发现缺失的内置任务",
			zap.Strings("missingTasks", missingTasks))
		return fmt.Errorf("发现 %d 个缺失的内置任务: %v", len(missingTasks), missingTasks)
	}

	btm.logger.Info("内置任务完整性验证通过")
	return nil
}
