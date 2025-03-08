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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProjectDAO interface {
	CreateProjectOne(ctx context.Context, project *model.K8sProject) error
	GetAll(ctx context.Context) ([]model.K8sProject, error)
	GetByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error)
	DeleteProjectById(ctx context.Context, id int64) (model.K8sProject, error)
}

type projectDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewProjectDAO(db *gorm.DB, l *zap.Logger) ProjectDAO {
	return &projectDAO{
		db: db,
		l:  l,
	}
}
func (p *projectDAO) CreateProjectOne(ctx context.Context, project *model.K8sProject) error {
	if err := p.db.WithContext(ctx).Create(project).Error; err != nil {
		p.l.Error("CreateProjectOne 创建k8sProject失败", zap.Error(err))
		return err
	}
	return nil
}
func (p *projectDAO) GetAll(ctx context.Context) ([]model.K8sProject, error) {
	var projects []model.K8sProject
	// 执行查询操作，从数据库中获取所有 K8sProject 记录
	err := p.db.WithContext(ctx).Find(&projects).Error
	if err != nil {
		// 若查询出错，记录错误日志并返回错误信息
		p.l.Error("GetAll 获取所有 k8sProject 失败", zap.Error(err))
		return nil, err
	}
	return projects, nil
}
func (p *projectDAO) GetByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error) {
	var projects []model.K8sProject
	// 使用 IN 条件查询指定 ID 的项目
	err := p.db.WithContext(ctx).Where("id IN ?", ids).Find(&projects).Error
	if err != nil {
		// 若查询出错，记录错误日志并返回错误信息
		p.l.Error("GetByIds 根据 ID 获取 k8sProject 失败", zap.Int64s("ids", ids), zap.Error(err))
		return nil, err
	}
	return projects, nil
}

// 项目删除方法
func (p *projectDAO) DeleteProjectById(ctx context.Context, id int64) (model.K8sProject, error) {
	var project model.K8sProject
	result := p.db.WithContext(ctx).
		Model(&model.K8sProject{}).
		Where("id = ?", id).
		Update("deleted_at", 1)

	if result.Error == nil && result.RowsAffected > 0 {
		p.db.WithContext(ctx).First(&project, id)
	}
	return project, result.Error
}
