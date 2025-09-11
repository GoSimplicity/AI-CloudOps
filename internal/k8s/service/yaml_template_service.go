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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type YamlTemplateService interface {
	// GetYamlTemplateList 获取 YAML 模板列表
	GetYamlTemplateList(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error)
	// CreateYamlTemplate 创建 YAML 模板
	CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// CheckYamlTemplate 检查 YAML 模板是否正确
	CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// UpdateYamlTemplate 更新 YAML 模板
	UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error
	// DeleteYamlTemplate 删除 YAML 模板
	DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error
	// GetYamlTemplateDetail 获取 YAML 模板详情
	GetYamlTemplateDetail(ctx context.Context, id int, clusterId int) (string, error)
}

type yamlTemplateService struct {
	manager manager.YamlManager
	logger  *zap.Logger
}

func NewYamlTemplateService(manager manager.YamlManager, logger *zap.Logger) YamlTemplateService {
	return &yamlTemplateService{
		manager: manager,
		logger:  logger,
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (y *yamlTemplateService) GetYamlTemplateList(ctx context.Context, clusterId int) ([]*model.K8sYamlTemplate, error) {
	return y.manager.GetYamlTemplateList(ctx, clusterId)
}

// CreateYamlTemplate 创建 YAML 模板
func (y *yamlTemplateService) CreateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	return y.manager.CreateYamlTemplate(ctx, template)
}

// CheckYamlTemplate 检查 YAML 模板是否正确
func (y *yamlTemplateService) CheckYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	return y.manager.CheckYamlTemplate(ctx, template)
}

// UpdateYamlTemplate 更新 YAML 模板
func (y *yamlTemplateService) UpdateYamlTemplate(ctx context.Context, template *model.K8sYamlTemplate) error {
	return y.manager.UpdateYamlTemplate(ctx, template)
}

// DeleteYamlTemplate 删除 YAML 模板
func (y *yamlTemplateService) DeleteYamlTemplate(ctx context.Context, id int, clusterId int) error {
	return y.manager.DeleteYamlTemplate(ctx, id, clusterId)
}

func (y *yamlTemplateService) GetYamlTemplateDetail(ctx context.Context, id int, clusterId int) (string, error) {
	return y.manager.GetYamlTemplateDetail(ctx, id, clusterId)
}
