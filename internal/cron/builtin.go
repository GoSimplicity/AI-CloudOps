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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type BuiltinTaskDefinition struct {
	Name        string
	Description string
	Schedule    string // cron表达式，留空表示使用默认间隔
	TaskType    string // 任务标识符，用于启动对应的处理函数
	Enabled     bool   // 默认是否启用
}

func GetBuiltinTasks() []BuiltinTaskDefinition {
	return []BuiltinTaskDefinition{
		{
			Name:        "值班历史记录管理",
			Description: "自动管理值班历史记录，处理换班和轮替逻辑",
			Schedule:    "@every 10s", // 与旧运行间隔一致
			TaskType:    "on_duty_history",
			Enabled:     true,
		},
		{
			Name:        "K8s集群状态检查",
			Description: "定期检查Kubernetes集群状态并更新数据库",
			Schedule:    "@every 60s", // 与旧运行间隔一致
			TaskType:    "k8s_status_check",
			Enabled:     true,
		},
		{
			Name:        "Prometheus配置刷新",
			Description: "定期刷新Prometheus监控配置",
			Schedule:    "@every 15s", // 与旧默认运行间隔一致
			TaskType:    "prometheus_config_refresh",
			Enabled:     true,
		},
	}
}

type BuiltinTaskManager struct {
	logger  *zap.Logger
	cronDAO dao.CronJobDAO
}

func NewBuiltinTaskManager(logger *zap.Logger, cronDAO dao.CronJobDAO) *BuiltinTaskManager {
	return &BuiltinTaskManager{
		logger:  logger,
		cronDAO: cronDAO,
	}
}

func (btm *BuiltinTaskManager) InitializeBuiltinTasks(ctx context.Context) error {
	// command 是内置任务稳定身份，名称、表达式和状态均允许由管理员修改
	byCommand := make(map[string]*model.CronJob)
	jobType := model.CronJobTypeSystem
	for page := 1; ; page++ {
		jobs, total, err := btm.cronDAO.GetCronJobList(ctx, &model.GetCronJobListReq{ListReq: model.ListReq{Page: page, Size: 100}, JobType: &jobType})
		if err != nil {
			return fmt.Errorf("读取内置任务失败: %w", err)
		}
		for _, job := range jobs {
			if job == nil || job.IsBuiltIn != 1 {
				continue
			}
			if !supportedBuiltin(job.Command) {
				return fmt.Errorf("内置任务 %d 类型无效", job.ID)
			}
			if previous := byCommand[job.Command]; previous != nil && previous.ID != job.ID {
				return fmt.Errorf("内置任务类型 %s 存在重复记录", job.Command)
			}
			byCommand[job.Command] = job
		}
		if len(jobs) < 100 || int64(page*100) >= total {
			break
		}
	}
	for _, def := range GetBuiltinTasks() {
		if byCommand[def.TaskType] != nil {
			continue
		}
		status := model.CronJobStatusEnabled
		if !def.Enabled {
			status = model.CronJobStatusDisabled
		}
		job := &model.CronJob{Name: def.Name, Description: def.Description, JobType: model.CronJobTypeSystem, Status: status, IsBuiltIn: 1, Schedule: def.Schedule, Timeout: 300, MaxRetry: 3, Command: def.TaskType}
		if err := btm.cronDAO.CreateCronJob(ctx, job); err != nil {
			// 同时启动的副本可能已先创建同名任务；仅核对真实身份后接受
			existing, lookupErr := btm.cronDAO.GetCronJobByName(ctx, def.Name)
			if lookupErr != nil || existing == nil || existing.IsBuiltIn != 1 || existing.JobType != model.CronJobTypeSystem || existing.Command != def.TaskType {
				return fmt.Errorf("创建内置任务失败: %w", err)
			}
		}
	}
	return nil
}

func (btm *BuiltinTaskManager) GetEnabledBuiltinTasks(ctx context.Context) ([]*model.CronJob, error) {
	jobs, err := btm.cronDAO.GetEnabledCronJobs(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*model.CronJob, 0, len(jobs))
	for _, job := range jobs {
		if job != nil && job.IsBuiltIn == 1 && job.JobType == model.CronJobTypeSystem {
			result = append(result, job)
		}
	}
	return result, nil
}

// ForceInitializeBuiltinTasks 强制重新初始化内置任务（用于修复）
func (btm *BuiltinTaskManager) ForceInitializeBuiltinTasks(ctx context.Context) error {
	// 旧版强制重建会删除用户状态及调度配置；沿用非破坏式初始化
	return btm.InitializeBuiltinTasks(ctx)
}
