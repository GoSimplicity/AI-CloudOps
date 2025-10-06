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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type YamlTaskDAO interface {
	ListAllYamlTasks(ctx context.Context, req *model.YamlTaskListReq) ([]*model.K8sYamlTask, error)
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	DeleteYamlTask(ctx context.Context, id int, clusterID int) error
	GetYamlTaskByID(ctx context.Context, id int, clusterID int) (*model.K8sYamlTask, error)
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
func (d *yamlTaskDAO) ListAllYamlTasks(ctx context.Context, req *model.YamlTaskListReq) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask
	query := d.db.WithContext(ctx)

	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.TemplateID > 0 {
		query = query.Where("template_id = ?", req.TemplateID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Size > 0 {
		offset := (req.Page - 1) * req.Size
		query = query.Offset(offset).Limit(req.Size)
	}

	query = query.Order("created_at DESC")

	if err := query.Find(&tasks).Error; err != nil {
		d.logger.Error("ListAllYamlTasks 查询所有Yaml任务失败", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}

func (d *yamlTaskDAO) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := d.db.WithContext(ctx).Create(task).Error; err != nil {
		d.logger.Error("CreateYamlTask 创建Yaml任务失败", zap.Error(err), zap.Any("task", task))
		return err
	}
	return nil
}

func (d *yamlTaskDAO) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if task.ID == 0 {
		d.logger.Error("UpdateYamlTask ID 不能为空", zap.Any("task", task))
		return fmt.Errorf("invalid task ID")
	}

	if err := d.db.WithContext(ctx).Model(&model.K8sYamlTask{}).Where("id = ? AND cluster_id = ?", task.ID, task.ClusterID).Updates(task).Error; err != nil {
		d.logger.Error("UpdateYamlTask 更新Yaml任务失败", zap.Int("taskID", task.ID), zap.Int("clusterID", task.ClusterID), zap.Error(err))
		return err
	}
	return nil
}

func (d *yamlTaskDAO) DeleteYamlTask(ctx context.Context, id int, clusterID int) error {
	if id == 0 {
		d.logger.Error("DeleteYamlTask ID 不能为空", zap.Int("id", id))
		return fmt.Errorf("invalid task ID")
	}

	if err := d.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterID).Delete(&model.K8sYamlTask{}).Error; err != nil {
		d.logger.Error("DeleteYamlTask 删除Yaml任务失败", zap.Int("taskID", id), zap.Int("clusterID", clusterID), zap.Error(err))
		return err
	}
	return nil
}

func (d *yamlTaskDAO) GetYamlTaskByID(ctx context.Context, id int, clusterID int) (*model.K8sYamlTask, error) {
	if id == 0 {
		d.logger.Error("GetYamlTaskByID ID 不能为空", zap.Int("id", id))
		return nil, fmt.Errorf("invalid task ID")
	}

	var task model.K8sYamlTask
	if err := d.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterID).First(&task).Error; err != nil {
		d.logger.Error("GetYamlTaskByID 查询Yaml任务失败", zap.Int("taskID", id), zap.Int("clusterID", clusterID), zap.Error(err))
		return nil, fmt.Errorf("YAML 任务 ID %d (集群 %d) 未找到: %w", id, clusterID, err)
	}
	return &task, nil
}

func (d *yamlTaskDAO) GetYamlTaskByTemplateID(ctx context.Context, templateID int) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask
	if err := d.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&tasks).Error; err != nil {
		d.logger.Error("GetYamlTaskByTemplateID 查询Yaml任务失败", zap.Int("templateID", templateID), zap.Error(err))
		return nil, err
	}

	if len(tasks) == 0 {
		d.logger.Info("GetYamlTaskByTemplateID 未找到相关Yaml任务", zap.Int("templateID", templateID))
	}

	return tasks, nil
}
