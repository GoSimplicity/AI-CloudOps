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

package uesr

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CornJobDAO interface {
	CreateCornJobOne(ctx context.Context, job *model.K8sCronjob) error
	GetCronjobList(ctx context.Context) ([]*model.K8sCronjob, error)
	GetCronjobOne(ctx context.Context, id int64) (*model.K8sCronjob, error)
	UpdateCronjobOne(ctx context.Context, id int64, job model.K8sCronjob) error
}
type cornJobDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewCornJobDAO(db *gorm.DB, l *zap.Logger) CornJobDAO {
	return &cornJobDAO{
		db: db,
		l:  l,
	}
}
func (c *cornJobDAO) CreateCornJobOne(ctx context.Context, job *model.K8sCronjob) error {
	// 使用 gorm 插入新的 CronJob 记录到数据库
	result := c.db.WithContext(ctx).Create(job)
	if result.Error != nil {
		// 记录错误日志
		c.l.Error("Failed to create CronJob in database",
			zap.String("name", job.Name),
			zap.Error(result.Error))
		return result.Error
	}
	c.l.Info("CronJob created successfully in database",
		zap.String("name", job.Name))
	return nil
}
func (c *cornJobDAO) GetCronjobList(ctx context.Context) ([]*model.K8sCronjob, error) {
	var jobs []*model.K8sCronjob
	result := c.db.WithContext(ctx).Find(&jobs)
	if result.Error != nil {
		return nil, result.Error
	}
	return jobs, nil
}
func (c *cornJobDAO) GetCronjobOne(ctx context.Context, id int64) (*model.K8sCronjob, error) {
	var job model.K8sCronjob
	result := c.db.WithContext(ctx).Where("id = ?", id).First(&job)
	if result.Error != nil {
		return nil, result.Error
	}
	return &job, nil
}
func (c *cornJobDAO) UpdateCronjobOne(ctx context.Context, id int64, job model.K8sCronjob) error {
	var existingJob model.K8sCronjob
	if err := c.db.First(&existingJob, id).Error; err != nil {
		// 如果没有找到记录，返回错误
		return fmt.Errorf("failed to find CronJob with id %d: %w", id, err)
	}

	// 将 job 直接赋值给 existingJob
	existingJob = job

	// 保存更新后的数据回数据库，传递 context
	if err := c.db.WithContext(ctx).Save(&existingJob).Error; err != nil {
		return fmt.Errorf("failed to update CronJob: %w", err)
	}

	// 返回成功
	return nil
}
