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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context, req *model.CreateTemplateReq, creatorID int, creatorName string) error
	UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq) error
	DeleteTemplate(ctx context.Context, id int) error
	ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error)
	DetailTemplate(ctx context.Context, id int) (*model.Template, error)
	EnableTemplate(ctx context.Context, id int, userID int) error
	DisableTemplate(ctx context.Context, id int, userID int) error
	GetTemplatesByProcessID(ctx context.Context, processID int) ([]*model.Template, error)
	GetTemplatesByCategory(ctx context.Context, categoryID int) ([]*model.Template, error)
	BatchUpdateStatus(ctx context.Context, ids []int, status int8) error
	GetTemplateCount(ctx context.Context) (int64, error)
	IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error)
}

type templateService struct {
	dao dao.TemplateDAO
	l   *zap.Logger
}

func NewTemplateService(dao dao.TemplateDAO, l *zap.Logger) TemplateService {
	return &templateService{
		dao: dao,
		l:   l,
	}
}

// CreateTemplate 创建模板
func (t *templateService) CreateTemplate(ctx context.Context, req *model.CreateTemplateReq, creatorID int, creatorName string) error {
	if req == nil {
		return fmt.Errorf("创建模板请求不能为空")
	}

	t.l.Info("开始创建模板",
		zap.String("name", req.Name),
		zap.Int("creator_id", creatorID),
		zap.String("creator_name", creatorName))

	// 检查模板名称是否已存在
	exists, err := t.dao.IsTemplateNameExists(ctx, req.Name, 0)
	if err != nil {
		t.l.Error("检查模板名称是否存在失败", zap.Error(err))
		return fmt.Errorf("检查模板名称失败: %w", err)
	}
	if exists {
		t.l.Warn("模板名称已存在", zap.String("name", req.Name))
		return dao.ErrTemplateNameExists
	}

	// 转换请求为模型
	template, err := utils.ConvertCreateTemplateReqToModel(req, creatorID, creatorName)
	if err != nil {
		t.l.Error("转换创建模板请求失败", zap.Error(err))
		return fmt.Errorf("转换创建模板请求失败: %w", err)
	}

	// 设置默认状态为启用
	template.Status = 1

	err = t.dao.CreateTemplate(ctx, template)
	if err != nil {
		t.l.Error("创建模板失败", zap.Error(err))
		return err
	}

	t.l.Info("模板创建成功", zap.Int("id", template.ID), zap.String("name", template.Name))
	return nil
}

// UpdateTemplate 更新模板
func (t *templateService) UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq) error {
	if req == nil || req.ID == 0 {
		return fmt.Errorf("更新模板请求无效")
	}

	t.l.Info("开始更新模板", zap.Int("id", req.ID), zap.String("name", req.Name))

	// 检查模板是否存在
	existingTemplate, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		t.l.Error("获取模板失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	// 检查名称是否与其他模板冲突
	if req.Name != "" && req.Name != existingTemplate.Name {
		exists, err := t.dao.IsTemplateNameExists(ctx, req.Name, req.ID)
		if err != nil {
			t.l.Error("检查模板名称是否存在失败", zap.Error(err))
			return fmt.Errorf("检查模板名称失败: %w", err)
		}
		if exists {
			t.l.Warn("模板名称已存在", zap.String("name", req.Name))
			return dao.ErrTemplateNameExists
		}
	}

	// 转换请求为模型
	template, err := utils.ConvertUpdateTemplateReqToModel(req)
	if err != nil {
		t.l.Error("转换更新模板请求失败", zap.Error(err))
		return fmt.Errorf("转换更新模板请求失败: %w", err)
	}

	err = t.dao.UpdateTemplate(ctx, template)
	if err != nil {
		t.l.Error("更新模板失败", zap.Error(err))
		return err
	}

	t.l.Info("模板更新成功", zap.Int("id", req.ID))
	return nil
}

// DeleteTemplate 删除模板
func (t *templateService) DeleteTemplate(ctx context.Context, id int) error {
	if id <= 0 {
		return dao.ErrInvalidID
	}

	t.l.Info("开始删除模板", zap.Int("id", id))

	// 检查模板是否存在
	_, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("获取模板失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	err = t.dao.DeleteTemplate(ctx, id)
	if err != nil {
		t.l.Error("删除模板失败", zap.Error(err))
		return err
	}

	t.l.Info("模板删除成功", zap.Int("id", id))
	return nil
}

// ListTemplate 获取模板列表
func (t *templateService) ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error) {
	if req == nil {
		return nil, fmt.Errorf("查询请求不能为空")
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100 // 限制最大页面大小
	}

	t.l.Debug("获取模板列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.String("search", req.Search))

	result, err := t.dao.ListTemplate(ctx, req)
	if err != nil {
		t.l.Error("获取模板列表失败", zap.Error(err))
		return nil, err
	}

	t.l.Info("模板列表获取成功", zap.Int64("total", result.Total), zap.Int("count", len(result.Items)))
	return result, nil
}

// DetailTemplate 查看模板详情
func (t *templateService) DetailTemplate(ctx context.Context, id int) (*model.Template, error) {
	if id <= 0 {
		return nil, dao.ErrInvalidID
	}

	t.l.Debug("获取模板详情", zap.Int("id", id))

	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("获取模板详情失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	t.l.Debug("模板详情获取成功", zap.Int("id", id), zap.String("name", template.Name))
	return template, nil
}

// EnableTemplate 启用模板
func (t *templateService) EnableTemplate(ctx context.Context, id int, userID int) error {
	if id <= 0 {
		return dao.ErrInvalidID
	}

	t.l.Info("开始启用模板", zap.Int("template_id", id), zap.Int("user_id", userID))

	// 检查模板是否存在
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("启用模板失败：获取模板失败", zap.Error(err), zap.Int("template_id", id))
		return err
	}

	// 检查当前状态
	if template.Status == 1 {
		t.l.Info("模板已启用，无需操作", zap.Int("template_id", id))
		return nil
	}

	// 更新状态
	err = t.dao.UpdateTemplateStatus(ctx, id, 1)
	if err != nil {
		t.l.Error("启用模板失败", zap.Error(err), zap.Int("template_id", id))
		return fmt.Errorf("启用模板失败: %w", err)
	}

	t.l.Info("模板启用成功", zap.Int("template_id", id), zap.Int("user_id", userID))
	return nil
}

// DisableTemplate 禁用模板
func (t *templateService) DisableTemplate(ctx context.Context, id int, userID int) error {
	if id <= 0 {
		return dao.ErrInvalidID
	}

	t.l.Info("开始禁用模板", zap.Int("template_id", id), zap.Int("user_id", userID))

	// 检查模板是否存在
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("禁用模板失败：获取模板失败", zap.Error(err), zap.Int("template_id", id))
		return err
	}

	// 检查当前状态
	if template.Status == 0 {
		t.l.Info("模板已禁用，无需操作", zap.Int("template_id", id))
		return nil
	}

	// 更新状态
	err = t.dao.UpdateTemplateStatus(ctx, id, 0)
	if err != nil {
		t.l.Error("禁用模板失败", zap.Error(err), zap.Int("template_id", id))
		return fmt.Errorf("禁用模板失败: %w", err)
	}

	t.l.Info("模板禁用成功", zap.Int("template_id", id), zap.Int("user_id", userID))
	return nil
}

// GetTemplatesByProcessID 根据流程ID获取模板列表
func (t *templateService) GetTemplatesByProcessID(ctx context.Context, processID int) ([]*model.Template, error) {
	if processID <= 0 {
		return nil, dao.ErrInvalidID
	}

	t.l.Debug("根据流程ID获取模板", zap.Int("process_id", processID))

	templates, err := t.dao.GetTemplatesByProcessID(ctx, processID)
	if err != nil {
		t.l.Error("根据流程ID获取模板失败", zap.Error(err), zap.Int("process_id", processID))
		return nil, err
	}

	t.l.Info("根据流程ID获取模板成功", zap.Int("process_id", processID), zap.Int("count", len(templates)))
	return templates, nil
}

// GetTemplatesByCategory 根据分类ID获取模板列表
func (t *templateService) GetTemplatesByCategory(ctx context.Context, categoryID int) ([]*model.Template, error) {
	if categoryID <= 0 {
		return nil, dao.ErrInvalidID
	}

	t.l.Debug("根据分类ID获取模板", zap.Int("category_id", categoryID))

	templates, err := t.dao.GetTemplatesByCategory(ctx, categoryID)
	if err != nil {
		t.l.Error("根据分类ID获取模板失败", zap.Error(err), zap.Int("category_id", categoryID))
		return nil, err
	}

	t.l.Info("根据分类ID获取模板成功", zap.Int("category_id", categoryID), zap.Int("count", len(templates)))
	return templates, nil
}

// BatchUpdateStatus 批量更新状态
func (t *templateService) BatchUpdateStatus(ctx context.Context, ids []int, status int8) error {
	if len(ids) == 0 {
		return fmt.Errorf("模板ID列表不能为空")
	}

	if status != 0 && status != 1 {
		return dao.ErrInvalidStatus
	}

	t.l.Info("开始批量更新模板状态", zap.Ints("ids", ids), zap.Int8("status", status))

	err := t.dao.BatchUpdateStatus(ctx, ids, status)
	if err != nil {
		t.l.Error("批量更新模板状态失败", zap.Error(err))
		return err
	}

	statusText := "禁用"
	if status == 1 {
		statusText = "启用"
	}

	t.l.Info("批量更新模板状态成功",
		zap.Ints("ids", ids),
		zap.String("status", statusText),
		zap.Int("count", len(ids)))

	return nil
}

// GetTemplateCount 获取模板总数
func (t *templateService) GetTemplateCount(ctx context.Context) (int64, error) {
	t.l.Debug("获取模板总数")

	count, err := t.dao.GetTemplateCount(ctx)
	if err != nil {
		t.l.Error("获取模板总数失败", zap.Error(err))
		return 0, err
	}

	t.l.Debug("模板总数获取成功", zap.Int64("count", count))
	return count, nil
}

// IsTemplateNameExists 检查模板名称是否存在
func (t *templateService) IsTemplateNameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("模板名称不能为空")
	}

	t.l.Debug("检查模板名称是否存在", zap.String("name", name), zap.Int("exclude_id", excludeID))

	exists, err := t.dao.IsTemplateNameExists(ctx, name, excludeID)
	if err != nil {
		t.l.Error("检查模板名称是否存在失败", zap.Error(err))
		return false, err
	}

	t.l.Debug("模板名称检查完成", zap.String("name", name), zap.Bool("exists", exists))
	return exists, nil
}
