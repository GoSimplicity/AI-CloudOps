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

type YamlTemplateDAO interface {
	ListAllYamlTemplates(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error)
	CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error
	UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error
	DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error
	GetYamlTemplateByID(ctx context.Context, id int, clusterId int) (*model.K8sYamlTemplate, error)
}

type yamlTemplateDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewYamlTemplateDAO(db *gorm.DB, l *zap.Logger) YamlTemplateDAO {
	return &yamlTemplateDAO{
		db: db,
		l:  l,
	}
}

// ListAllYamlTemplates 查询所有 YAML 模板
func (y *yamlTemplateDAO) ListAllYamlTemplates(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error) {
	var yamls []*model.K8sYamlTemplate

	if err := y.db.WithContext(ctx).Where("cluster_id = ?", clusterId).Find(&yamls).Error; err != nil {
		y.l.Error("ListAllYamlTemplates 查询所有Yaml模板失败", zap.Error(err))
		return nil, err
	}

	return yamls, nil
}

// CreateYamlTemplate 创建 YAML 模板
func (y *yamlTemplateDAO) CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error {
	if err := y.db.WithContext(ctx).Create(&yaml).Error; err != nil {
		y.l.Error("CreateYamlTemplate 创建Yaml模板失败", zap.Error(err), zap.Any("yaml", yaml))
		return err
	}

	return nil
}

// UpdateYamlTemplate 更新 YAML 模板
func (y *yamlTemplateDAO) UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error {
	if yaml.ID == 0 {
		y.l.Error("UpdateYamlTemplate ID 不能为空", zap.Any("yaml", yaml))
		return fmt.Errorf("invalid yaml ID")
	}

	if err := y.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", yaml.ID, yaml.ClusterId).Updates(yaml).Error; err != nil {
		y.l.Error("UpdateYamlTemplate 更新Yaml模板失败", zap.Int("yamlID", yaml.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteYamlTemplate 删除 YAML 模板
func (y *yamlTemplateDAO) DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error {
	if id == 0 {
		y.l.Error("DeleteYamlTemplate ID 不能为空", zap.Int("id", id))
		return fmt.Errorf("invalid yaml template ID")
	}

	if err := y.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterId).Delete(&model.K8sYamlTemplate{}).Error; err != nil {
		y.l.Error("DeleteYamlTemplate 删除Yaml模板失败", zap.Int("yamlID", id), zap.Error(err))
		return err
	}

	return nil
}

// GetYamlTemplateByID 根据 ID 查询 YAML 模板
func (y *yamlTemplateDAO) GetYamlTemplateByID(ctx context.Context, id int, clusterId int) (*model.K8sYamlTemplate, error) {
	if id == 0 {
		y.l.Error("GetYamlTemplateByID ID 不能为空", zap.Int("id", id))
		return nil, fmt.Errorf("invalid yaml template ID")
	}

	var yaml *model.K8sYamlTemplate

	if err := y.db.WithContext(ctx).Where("id = ? AND cluster_id = ?", id, clusterId).First(&yaml).Error; err != nil {
		y.l.Error("GetYamlTemplateByID 查询Yaml模板失败", zap.Int("yamlID", id), zap.Error(err))
		return nil, err
	}

	return yaml, nil
}
