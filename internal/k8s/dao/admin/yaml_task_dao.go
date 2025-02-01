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

package admin

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type YamlTaskDAO interface {
	// ListAllYamlTasks 查询所有 YAML 任务
	ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error)
	// CreateYamlTask 创建 YAML 任务
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// UpdateYamlTask 更新 YAML 任务
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	// DeleteYamlTask 删除 YAML 任务
	DeleteYamlTask(ctx context.Context, id int) error
	// GetYamlTaskByID 根据 ID 查询 YAML 任务
	GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error)
	// GetYamlTaskByTemplateID 根据模板 ID 查询 YAML 任务
	GetYamlTaskByTemplateID(ctx context.Context, templateID int) ([]*model.K8sYamlTask, error)
}

type yamlTaskDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewYamlTaskDAO(db *gorm.DB, logger *zap.Logger) YamlTaskDAO {
	return &yamlTaskDAO{
		db:     db,
		logger: logger,
	}
}

// ListAllYamlTasks 查询所有 YAML 任务
func (y *yamlTaskDAO) ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask

	if err := y.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		y.logger.Error("ListAllYamlTasks 查询所有Yaml任务失败", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}

// CreateYamlTask 创建 YAML 任务
func (y *yamlTaskDAO) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := y.db.WithContext(ctx).Create(task).Error; err != nil {
		y.logger.Error("CreateYamlTask 创建Yaml任务失败", zap.Error(err), zap.Any("task", task))
		return err
	}

	return nil
}

// UpdateYamlTask 更新 YAML 任务
func (y *yamlTaskDAO) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := y.db.WithContext(ctx).Model(&model.K8sYamlTask{}).Where("id = ?", task.ID).Updates(task).Error; err != nil {
		y.logger.Error("UpdateYamlTask 更新Yaml任务失败", zap.Int("taskID", task.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteYamlTask 删除 YAML 任务
func (y *yamlTaskDAO) DeleteYamlTask(ctx context.Context, id int) error {
	if err := y.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sYamlTask{}).Error; err != nil {
		y.logger.Error("DeleteYamlTask 删除Yaml任务失败", zap.Int("taskID", id), zap.Error(err))
		return err
	}

	return nil
}

// GetYamlTaskByID 根据 ID 查询 YAML 任务
func (y *yamlTaskDAO) GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error) {
	var task model.K8sYamlTask

	if err := y.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		y.logger.Error("GetYamlTaskByID 查询Yaml任务失败", zap.Int("taskID", id), zap.Error(err))
		return nil, fmt.Errorf("YAML 任务 ID %d 未找到: %w", id, err)
	}

	return &task, nil
}

// GetYamlTaskByTemplateID 根据模板 ID 查询 YAML 任务
func (y *yamlTaskDAO) GetYamlTaskByTemplateID(ctx context.Context, templateID int) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask

	if err := y.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&tasks).Error; err != nil {
		y.logger.Error("GetYamlTaskByTemplateID 查询Yaml任务失败", zap.Int("templateID", templateID), zap.Error(err))
		return nil, err
	}

	// 若未找到任务，记录信息日志
	if len(tasks) == 0 {
		y.logger.Info("GetYamlTaskByTemplateID 未找到相关Yaml任务", zap.Int("templateID", templateID))
	}

	return tasks, nil
}
