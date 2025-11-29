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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/cron/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/handler"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/scheduler"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type CronService interface {
	CreateCronJob(ctx context.Context, req *model.CreateCronJobReq) error
	UpdateCronJob(ctx context.Context, req *model.UpdateCronJobReq) error
	DeleteCronJob(ctx context.Context, id int) error
	GetCronJob(ctx context.Context, id int) (*model.CronJob, error)
	GetCronJobList(ctx context.Context, req *model.GetCronJobListReq) (model.ListResp[*model.CronJob], error)
	EnableCronJob(ctx context.Context, id int) error
	DisableCronJob(ctx context.Context, id int) error
	TriggerCronJob(ctx context.Context, id int) error
	ValidateSchedule(ctx context.Context, req *model.ValidateScheduleReq) (*model.ValidateScheduleResp, error)
}

type cronService struct {
	logger        *zap.Logger
	cronDAO       dao.CronJobDAO
	userDAO       userDao.UserDAO
	client        *asynq.Client
	cronScheduler *scheduler.CronScheduler
}

func NewCronService(
	logger *zap.Logger,
	cronDAO dao.CronJobDAO,
	userDAO userDao.UserDAO,
	client *asynq.Client,
	cronScheduler *scheduler.CronScheduler,
) CronService {
	return &cronService{
		logger:        logger,
		cronDAO:       cronDAO,
		userDAO:       userDAO,
		client:        client,
		cronScheduler: cronScheduler,
	}
}

// CreateCronJob 创建任务
func (s *cronService) CreateCronJob(ctx context.Context, req *model.CreateCronJobReq) error {
	s.logger.Info("创建任务", zap.String("name", req.Name))

	// 基本验证
	if err := s.validateBasicJobConfig(req.JobType, req.Command, req.HTTPUrl, req.ScriptContent, req.SSHResourceID, req.SSHCommand); err != nil {
		s.logger.Error("任务配置验证失败", zap.String("name", req.Name), zap.Error(err))
		return err
	}

	job := &model.CronJob{
		Name:           req.Name,
		Description:    req.Description,
		JobType:        req.JobType,
		Schedule:       req.Schedule,
		Command:        req.Command,
		Args:           req.Args,
		WorkDir:        req.WorkDir,
		Environment:    req.Environment,
		HTTPMethod:     req.HTTPMethod,
		HTTPUrl:        req.HTTPUrl,
		HTTPHeaders:    req.HTTPHeaders,
		HTTPBody:       req.HTTPBody,
		ScriptType:     req.ScriptType,
		ScriptContent:  req.ScriptContent,
		SSHResourceID:  req.SSHResourceID,
		SSHCommand:     req.SSHCommand,
		SSHWorkDir:     req.SSHWorkDir,
		SSHEnvironment: req.SSHEnvironment,
		Timeout:        req.Timeout,
		MaxRetry:       req.MaxRetry,
		CreatedBy:      req.CreatedBy,
		CreatedByName:  req.CreatedByName,
	}

	if err := s.cronDAO.CreateCronJob(ctx, job); err != nil {
		s.logger.Error("创建任务失败", zap.String("name", req.Name), zap.Error(err))
		return err
	}

	s.logger.Info("创建任务成功", zap.String("name", job.Name), zap.Int("id", job.ID))
	return nil
}

// UpdateCronJob 更新任务
func (s *cronService) UpdateCronJob(ctx context.Context, req *model.UpdateCronJobReq) error {
	s.logger.Info("更新任务", zap.Int("id", req.ID), zap.String("name", req.Name))

	// 基本验证
	if err := s.validateBasicJobConfig(req.JobType, req.Command, req.HTTPUrl, req.ScriptContent, req.SSHResourceID, req.SSHCommand); err != nil {
		s.logger.Error("任务配置验证失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	job := &model.CronJob{
		Model:          model.Model{ID: req.ID},
		Name:           req.Name,
		Description:    req.Description,
		JobType:        req.JobType,
		Schedule:       req.Schedule,
		Command:        req.Command,
		Args:           req.Args,
		WorkDir:        req.WorkDir,
		Environment:    req.Environment,
		HTTPMethod:     req.HTTPMethod,
		HTTPUrl:        req.HTTPUrl,
		HTTPHeaders:    req.HTTPHeaders,
		HTTPBody:       req.HTTPBody,
		ScriptType:     req.ScriptType,
		ScriptContent:  req.ScriptContent,
		SSHResourceID:  req.SSHResourceID,
		SSHCommand:     req.SSHCommand,
		SSHWorkDir:     req.SSHWorkDir,
		SSHEnvironment: req.SSHEnvironment,
		Timeout:        req.Timeout,
		MaxRetry:       req.MaxRetry,
	}

	if err := s.cronDAO.UpdateCronJob(ctx, job); err != nil {
		s.logger.Error("更新任务失败", zap.Int("id", req.ID), zap.Error(err))
		return err
	}

	s.logger.Info("更新任务成功", zap.String("name", job.Name), zap.Int("id", job.ID))
	return nil
}

// DeleteCronJob 删除任务
func (s *cronService) DeleteCronJob(ctx context.Context, id int) error {
	s.logger.Info("删除任务", zap.Int("id", id))

	// 先检查任务是否存在并且是否为内置任务
	job, err := s.cronDAO.GetCronJob(ctx, id)
	if err != nil {
		s.logger.Error("获取任务信息失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	// 检查是否为内置任务
	if job.IsBuiltIn == 1 {
		s.logger.Warn("尝试删除内置任务被拒绝", zap.Int("id", id), zap.String("name", job.Name))
		return fmt.Errorf("内置系统任务不能被删除")
	}

	if err := s.cronDAO.DeleteCronJob(ctx, id); err != nil {
		s.logger.Error("删除任务失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	// 立即从调度器中移除任务，避免已删除的任务继续执行
	if s.cronScheduler != nil && job.JobType != model.CronJobTypeSystem {
		if err := s.cronScheduler.RemoveScheduledJob(id); err != nil {
			// 记录警告但不影响删除操作的成功
			s.logger.Warn("从调度器移除任务失败，但任务已成功删除",
				zap.Int("id", id),
				zap.String("name", job.Name),
				zap.Error(err))
		} else {
			s.logger.Info("成功从调度器移除任务", zap.Int("id", id), zap.String("name", job.Name))
		}
	}

	s.logger.Info("删除任务成功", zap.Int("id", id), zap.String("name", job.Name))
	return nil
}

// GetCronJob 获取任务详情
func (s *cronService) GetCronJob(ctx context.Context, id int) (*model.CronJob, error) {
	job, err := s.cronDAO.GetCronJob(ctx, id)
	if err != nil {
		s.logger.Error("获取任务详情失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return job, nil
}

// GetCronJobList 获取任务列表
func (s *cronService) GetCronJobList(ctx context.Context, req *model.GetCronJobListReq) (model.ListResp[*model.CronJob], error) {
	jobs, total, err := s.cronDAO.GetCronJobList(ctx, req)
	if err != nil {
		s.logger.Error("获取任务列表失败", zap.Error(err))
		return model.ListResp[*model.CronJob]{}, err
	}

	for _, job := range jobs {
		// 计算NextRunTime
		s.fillNextRunTime(job)

		// 填充CreatedByName
		s.fillCreatedByName(ctx, job)
	}

	return model.ListResp[*model.CronJob]{
		Items: jobs,
		Total: total,
	}, nil
}

// EnableCronJob 启用任务
func (s *cronService) EnableCronJob(ctx context.Context, id int) error {
	s.logger.Info("启用任务", zap.Int("id", id))

	if err := s.cronDAO.UpdateCronJobStatus(ctx, id, model.CronJobStatusEnabled); err != nil {
		s.logger.Error("启用任务失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("启用任务成功", zap.Int("id", id))
	return nil
}

// DisableCronJob 禁用任务
func (s *cronService) DisableCronJob(ctx context.Context, id int) error {
	s.logger.Info("禁用任务", zap.Int("id", id))

	if err := s.cronDAO.UpdateCronJobStatus(ctx, id, model.CronJobStatusDisabled); err != nil {
		s.logger.Error("禁用任务失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	s.logger.Info("禁用任务成功", zap.Int("id", id))
	return nil
}

// TriggerCronJob 手动触发任务
func (s *cronService) TriggerCronJob(ctx context.Context, id int) error {
	s.logger.Info("手动触发任务", zap.Int("id", id))

	// 获取任务详情
	job, err := s.cronDAO.GetCronJob(ctx, id)
	if err != nil {
		s.logger.Error("获取任务详情失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("获取任务详情失败: %w", err)
	}

	// 检查任务状态
	if job.Status != model.CronJobStatusEnabled {
		return fmt.Errorf("任务未启用，无法手动触发")
	}

	// 记录系统内置任务的手动触发
	if job.JobType == model.CronJobTypeSystem {
		s.logger.Info("手动触发系统内置任务",
			zap.Int("id", id),
			zap.String("name", job.Name),
			zap.String("taskType", job.Command))
	}

	// 创建任务载荷
	payload := handler.CronTaskPayload{
		JobID:     job.ID,
		JobName:   job.Name,
		TaskType:  job.JobType,
		TriggerBy: "manual",
		Data: map[string]interface{}{
			"triggered_at": time.Now().Format(time.RFC3339),
		},
	}

	// 序列化载荷
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("序列化任务载荷失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("序列化任务载荷失败: %w", err)
	}

	// 创建Asynq任务
	task := asynq.NewTask("cron:task", payloadBytes)

	// 立即执行任务
	taskInfo, err := s.client.Enqueue(task, asynq.ProcessIn(0))
	if err != nil {
		s.logger.Error("入队任务失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("入队任务失败: %w", err)
	}

	s.logger.Info("手动触发任务成功",
		zap.Int("id", id),
		zap.String("taskID", taskInfo.ID))
	return nil
}

// ValidateSchedule 验证调度表达式
func (s *cronService) ValidateSchedule(ctx context.Context, req *model.ValidateScheduleReq) (*model.ValidateScheduleResp, error) {
	s.logger.Info("验证调度表达式", zap.String("schedule", req.Schedule))

	// 使用robfig/cron库验证表达式
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(req.Schedule)
	if err != nil {
		s.logger.Warn("调度表达式验证失败", zap.String("schedule", req.Schedule), zap.Error(err))
		return &model.ValidateScheduleResp{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("调度表达式无效: %v", err),
		}, nil
	}

	// 生成下次运行时间预览（接下来5次）
	nextRunTimes := s.generateNextRunTimes(schedule, 5)

	resp := &model.ValidateScheduleResp{
		Valid:        true,
		NextRunTimes: nextRunTimes,
	}

	s.logger.Info("调度表达式验证成功", zap.String("schedule", req.Schedule))
	return resp, nil
}

// fillNextRunTime 填充下次运行时间
func (s *cronService) fillNextRunTime(job *model.CronJob) {
	if job == nil || job.Schedule == "" {
		return
	}

	// 如果任务已禁用，不计算下次运行时间
	if job.Status != model.CronJobStatusEnabled {
		return
	}

	// 解析cron表达式
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(job.Schedule)
	if err != nil {
		s.logger.Warn("解析调度表达式失败",
			zap.Int("jobID", job.ID),
			zap.String("schedule", job.Schedule),
			zap.Error(err))
		return
	}

	// 计算下次运行时间
	now := time.Now()
	nextTime := schedule.Next(now)
	job.NextRunTime = &nextTime
}

// fillCreatedByName 填充创建者姓名
func (s *cronService) fillCreatedByName(ctx context.Context, job *model.CronJob) {
	if job == nil || job.CreatedBy <= 0 {
		return
	}

	// 如果已经有创建者姓名，跳过
	if job.CreatedByName != "" {
		return
	}

	// 查询用户信息
	user, err := s.userDAO.GetByID(ctx, job.CreatedBy)
	if err != nil {
		s.logger.Warn("查询创建者信息失败",
			zap.Int("jobID", job.ID),
			zap.Int("createdBy", job.CreatedBy),
			zap.Error(err))
		return
	}

	if user != nil {
		job.CreatedByName = user.Username
	}
}

// validateBasicJobConfig 验证任务配置的基本要求
func (s *cronService) validateBasicJobConfig(jobType model.CronJobType, command, httpUrl, scriptContent string, sshResourceID *int, sshCommand string) error {
	switch jobType {
	case model.CronJobTypeCommand:
		if command == "" {
			return fmt.Errorf("命令行任务必须指定执行命令")
		}
	case model.CronJobTypeHTTP:
		if httpUrl == "" {
			return fmt.Errorf("HTTP任务必须指定请求URL")
		}
	case model.CronJobTypeScript:
		if scriptContent == "" {
			return fmt.Errorf("脚本任务必须提供脚本内容")
		}
	case model.CronJobTypeSSH:
		if sshResourceID == nil || *sshResourceID == 0 {
			return fmt.Errorf("SSH任务必须指定SSH资源ID")
		}
		if sshCommand == "" {
			return fmt.Errorf("SSH任务必须指定执行命令")
		}
	case model.CronJobTypeSystem:
		// 系统任务由内置任务管理器处理
	default:
		return fmt.Errorf("不支持的任务类型: %d", int8(jobType))
	}
	return nil
}

// generateNextRunTimes 生成接下来N次的运行时间
func (s *cronService) generateNextRunTimes(schedule cron.Schedule, count int) model.StringList {
	var nextRunTimes []string
	now := time.Now()

	// 确保时间格式化使用本地时区
	for i := 0; i < count; i++ {
		next := schedule.Next(now)
		// 使用中国标准时间格式，更友好的显示
		nextRunTimes = append(nextRunTimes, next.Format("2006-01-02 15:04:05 MST"))
		now = next
	}

	return model.StringList(nextRunTimes)
}
