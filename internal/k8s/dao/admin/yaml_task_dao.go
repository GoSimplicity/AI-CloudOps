package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type YamlTaskDAO interface {
	ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error)
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	DeleteYamlTask(ctx context.Context, id int) error
	GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error)
	GetYamlTaskByTemplateID(ctx context.Context, templateID int) ([]*model.K8sYamlTask, error)
}

type yamlTaskDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewYamlTaskDAO(db *gorm.DB, l *zap.Logger) YamlTaskDAO {
	return &yamlTaskDAO{
		db: db,
		l:  l,
	}
}

// ListAllYamlTasks 查询所有 YAML 任务
func (y *yamlTaskDAO) ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask

	if err := y.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		y.l.Error("ListAllYamlTasks 查询所有Yaml任务失败", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}

// CreateYamlTask 创建 YAML 任务
func (y *yamlTaskDAO) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := y.db.WithContext(ctx).Create(&task).Error; err != nil {
		y.l.Error("CreateYamlTask 创建Yaml任务失败", zap.Error(err), zap.Any("task", task))
		return err
	}

	return nil
}

// UpdateYamlTask 更新 YAML 任务
func (y *yamlTaskDAO) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := y.db.WithContext(ctx).Where("id = ?", task.ID).Updates(task).Error; err != nil {
		y.l.Error("UpdateYamlTask 更新Yaml任务失败", zap.Int("taskID", task.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteYamlTask 删除 YAML 任务
func (y *yamlTaskDAO) DeleteYamlTask(ctx context.Context, id int) error {
	if err := y.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sYamlTask{}).Error; err != nil {
		y.l.Error("DeleteYamlTask 删除Yaml任务失败", zap.Int("taskID", id), zap.Error(err))
		return err
	}

	return nil
}

// GetYamlTaskByID 根据 ID 查询 YAML 任务
func (y *yamlTaskDAO) GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error) {
	var task *model.K8sYamlTask

	if err := y.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		y.l.Error("GetYamlTaskByID 查询Yaml任务失败", zap.Int("taskID", id), zap.Error(err))
		return nil, fmt.Errorf("YamlTask with ID %d not found: %w", id, err)
	}

	return task, nil
}

// GetYamlTaskByTemplateID 根据模板 ID 查询 YAML 任务
func (y *yamlTaskDAO) GetYamlTaskByTemplateID(ctx context.Context, templateID int) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask

	if err := y.db.WithContext(ctx).Where("template_id = ?", templateID).Find(&tasks).Error; err != nil {
		y.l.Error("GetYamlTaskByTemplateID 查询Yaml任务失败", zap.Int("templateID", templateID), zap.Error(err))
		return nil, err
	}

	if len(tasks) == 0 {
		y.l.Info("GetYamlTaskByTemplateID 未找到相关Yaml任务", zap.Int("templateID", templateID))
	}

	return tasks, nil
}
