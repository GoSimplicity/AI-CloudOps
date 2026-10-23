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

package dao

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CronJobDAO interface {
	AcquireCronJobLease(context.Context, int, string) (bool, error)
	RenewCronJobLease(context.Context, int, string) (bool, error)
	CompleteCronJobLease(context.Context, int, string, *time.Time, int8, int, string, string) (bool, error)
	CreateCronJob(ctx context.Context, job *model.CronJob) error
	GetCronJob(ctx context.Context, id int) (*model.CronJob, error)
	GetCronJobByName(ctx context.Context, name string) (*model.CronJob, error)
	GetCronJobList(ctx context.Context, req *model.GetCronJobListReq) ([]*model.CronJob, int64, error)
	UpdateCronJob(ctx context.Context, job *model.CronJob) error
	DeleteCronJob(ctx context.Context, id int) error
	UpdateCronJobStatus(ctx context.Context, id int, status model.CronJobStatus) error
	CompareAndSwapCronJobStatus(ctx context.Context, id int, from, to model.CronJobStatus) (bool, error)
	GetEnabledCronJobs(ctx context.Context) ([]*model.CronJob, error)
	UpdateNextRunTime(ctx context.Context, id int, nextRunTime time.Time) error
}

type cronJobDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewCronJobDAO(logger *zap.Logger, db *gorm.DB) CronJobDAO {
	return &cronJobDAO{
		logger: logger,
		db:     db,
	}
}

func (d *cronJobDAO) CreateCronJob(ctx context.Context, job *model.CronJob) error {
	if job == nil {
		return errors.New("任务信息不能为空")
	}
	if strings.TrimSpace(job.Name) == "" {
		return errors.New("任务名称不能为空")
	}
	if strings.TrimSpace(job.Schedule) == "" {
		return errors.New("调度表达式不能为空")
	}

	var count int64
	if err := d.db.WithContext(ctx).Model(&model.CronJob{}).
		Where("name = ?", job.Name).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("任务名称已存在")
	}

	if job.Status == 0 {
		job.Status = model.CronJobStatusEnabled
	}
	if job.Timeout == 0 {
		job.Timeout = 300 // 默认5分钟超时
	}
	if job.MaxRetry == 0 {
		job.MaxRetry = 3 // 默认重试3次
	}

	if err := d.db.WithContext(ctx).Create(job).Error; err != nil {
		d.logger.Error("创建任务失败", zap.String("name", job.Name), zap.Error(err))
		return err
	}

	d.logger.Info("成功创建任务",
		zap.String("name", job.Name),
		zap.Int("id", job.ID),
		zap.String("schedule", job.Schedule))
	return nil
}

func (d *cronJobDAO) GetCronJob(ctx context.Context, id int) (*model.CronJob, error) {
	var job model.CronJob
	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("任务不存在")
		}
		d.logger.Error("获取任务失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	return &job, nil
}

func (d *cronJobDAO) GetCronJobByName(ctx context.Context, name string) (*model.CronJob, error) {
	var job model.CronJob
	if err := d.db.WithContext(ctx).Where("name = ?", name).First(&job).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		d.logger.Error("根据名称获取任务失败", zap.String("name", name), zap.Error(err))
		return nil, err
	}
	return &job, nil
}

func (d *cronJobDAO) GetCronJobList(ctx context.Context, req *model.GetCronJobListReq) ([]*model.CronJob, int64, error) {
	var jobs []*model.CronJob
	var count int64

	query := d.db.WithContext(ctx).Model(&model.CronJob{})

	// 添加过滤条件
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.JobType != nil {
		query = query.Where("job_type = ?", *req.JobType)
	}
	if strings.TrimSpace(req.Search) != "" {
		like := "%" + strings.TrimSpace(req.Search) + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	if err := query.Count(&count).Error; err != nil {
		d.logger.Error("获取任务总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.Size
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&jobs).Error; err != nil {
		d.logger.Error("获取任务列表失败", zap.Error(err))
		return nil, 0, err
	}

	return jobs, count, nil
}

func (d *cronJobDAO) UpdateCronJob(ctx context.Context, job *model.CronJob) error {
	if job == nil {
		return errors.New("任务信息不能为空")
	}
	if strings.TrimSpace(job.Name) == "" {
		return errors.New("任务名称不能为空")
	}
	if strings.TrimSpace(job.Schedule) == "" {
		return errors.New("调度表达式不能为空")
	}

	existingJob, err := d.GetCronJob(ctx, job.ID)
	if err != nil {
		return err
	}

	// 检查名称唯一性（排除自己）
	var count int64
	if err := d.db.WithContext(ctx).Model(&model.CronJob{}).
		Where("name = ? AND id != ?", job.Name, job.ID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("任务名称已存在")
	}

	job.NextRunTime = existingJob.NextRunTime
	job.LastRunTime = existingJob.LastRunTime
	job.LastRunStatus = existingJob.LastRunStatus
	job.LastRunDuration = existingJob.LastRunDuration
	job.LastRunError = existingJob.LastRunError
	job.RunCount = existingJob.RunCount
	job.SuccessCount = existingJob.SuccessCount
	job.FailureCount = existingJob.FailureCount

	if err := d.db.WithContext(ctx).Save(job).Error; err != nil {
		d.logger.Error("更新任务失败", zap.Int("id", job.ID), zap.Error(err))
		return err
	}

	d.logger.Info("成功更新任务",
		zap.String("name", job.Name),
		zap.Int("id", job.ID))
	return nil
}

func (d *cronJobDAO) DeleteCronJob(ctx context.Context, id int) error {
	job, err := d.GetCronJob(ctx, id)
	if err != nil {
		return err
	}

	if job.IsBuiltIn == 1 {
		return errors.New("内置系统任务不能被删除")
	}

	if job.Status == model.CronJobStatusRunning {
		return errors.New("无法删除正在运行的任务")
	}

	if err := d.db.WithContext(ctx).Delete(&model.CronJob{}, id).Error; err != nil {
		d.logger.Error("删除任务失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	d.logger.Info("成功删除任务",
		zap.String("name", job.Name),
		zap.Int("id", id))
	return nil
}

func (d *cronJobDAO) UpdateCronJobStatus(ctx context.Context, id int, status model.CronJobStatus) error {
	if err := d.db.WithContext(ctx).Model(&model.CronJob{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		d.logger.Error("更新任务状态失败", zap.Int("id", id), zap.Int8("status", int8(status)), zap.Error(err))
		return err
	}
	return nil
}

func (d *cronJobDAO) GetEnabledCronJobs(ctx context.Context) ([]*model.CronJob, error) {
	var jobs []*model.CronJob
	if err := d.db.WithContext(ctx).
		Where("status IN ?", []model.CronJobStatus{model.CronJobStatusEnabled, model.CronJobStatusRunning}).
		Find(&jobs).Error; err != nil {
		d.logger.Error("获取启用的任务失败", zap.Error(err))
		return nil, err
	}
	return jobs, nil
}

func (d *cronJobDAO) UpdateNextRunTime(ctx context.Context, id int, nextRunTime time.Time) error {
	if err := d.db.WithContext(ctx).Model(&model.CronJob{}).
		Where("id = ?", id).
		Update("next_run_time", nextRunTime).Error; err != nil {
		d.logger.Error("更新下次运行时间失败", zap.Int("id", id), zap.Time("nextRunTime", nextRunTime), zap.Error(err))
		return err
	}
	return nil
}

// CompareAndSwapCronJobStatus 仅修改预期状态，避免执行器覆盖人工停用
func (d *cronJobDAO) CompareAndSwapCronJobStatus(ctx context.Context, id int, from, to model.CronJobStatus) (bool, error) {
	result := d.db.WithContext(ctx).Model(&model.CronJob{}).Where("id = ? AND status = ?", id, from).Update("status", to)
	return result.RowsAffected == 1, result.Error
}
