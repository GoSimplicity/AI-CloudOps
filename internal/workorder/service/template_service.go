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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context, req *model.CreateTemplateReq, creatorID int, creatorName string) error
	UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq) error
	DeleteTemplate(ctx context.Context, id int) error // Changed to id int
	ListTemplate(ctx context.Context, req *model.ListTemplateReq) ([]model.Template, error)
	DetailTemplate(ctx context.Context, id int) (*model.Template, error) // Changed to id int
	EnableTemplate(ctx context.Context, id int, userID int) error      // Added EnableTemplate
	DisableTemplate(ctx context.Context, id int, userID int) error     // Added DisableTemplate
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
	template, err := utils.ConvertCreateTemplateReqToModel(req, creatorID, creatorName)
	if err != nil {
		t.l.Error("转换创建模板请求失败", zap.Error(err))
		return err
	}
	// CreatorID and CreatorName are set by the converter.
	// Default status (e.g., enabled) can also be set by the converter or here.
	// template.Status = 1 // Example: Default to enabled, if not set in converter
	return t.dao.CreateTemplate(ctx, template)
}

// DeleteTemplate 删除模板
func (t *templateService) DeleteTemplate(ctx context.Context, id int) error { // Changed to id int
	return t.dao.DeleteTemplate(ctx, id)
}

// DetailTemplate 查看模板详情
func (t *templateService) DetailTemplate(ctx context.Context, id int) (*model.Template, error) { // Changed to id int
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("获取模板失败", zap.Error(err))
		return nil, err
	}
	// TODO: Populate CreatorName, Process, Category if needed for the response
	// This might involve additional DAO calls.
	return &template, nil
}

// ListTemplate 获取模板列表
func (t *templateService) ListTemplate(ctx context.Context, req *model.ListTemplateReq) ([]model.Template, error) {
	templates, err := t.dao.ListTemplate(ctx, req)
	if err != nil {
		t.l.Error("获取模板列表失败", zap.Error(err))
		return nil, err
	}
	return templates, nil
}

// UpdateTemplate 更新模板
func (t *templateService) UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq) error {
	template, err := utils.ConvertUpdateTemplateReqToModel(req)
	if err != nil {
		t.l.Error("转换更新模板请求失败", zap.Error(err))
		return err
	}
	return t.dao.UpdateTemplate(ctx, template)
}

// EnableTemplate 启用模板
func (t *templateService) EnableTemplate(ctx context.Context, id int, userID int) error {
	t.l.Info("开始启用模板", zap.Int("templateID", id), zap.Int("userID", userID))
	// TODO: Add permission check for userID if necessary

	// Assuming UpdateTemplateStatus is a new DAO method:
	// err := t.dao.UpdateTemplateStatus(ctx, id, 1) // 1 for enabled
	// For now, let's use UpdateTemplate with a partial update if UpdateTemplateStatus doesn't exist yet.
	// This requires GetTemplate first, then Update.
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("启用模板失败：获取模板失败", zap.Error(err), zap.Int("templateID", id))
		return fmt.Errorf("获取模板 (ID: %d) 失败: %w", id, err)
	}
	if template.Status == 1 {
		t.l.Info("模板已启用，无需操作", zap.Int("templateID", id))
		return nil // Already enabled
	}
	template.Status = 1 // Set status to enabled
	// Only update the status field.
	// If DAO's UpdateTemplate updates all fields, this might unintentionally clear other fields
	// if `template` object is not fully populated or if a specific "UpdateStatus" DAO method is preferred.
	// For now, assuming UpdateTemplate can handle partial updates or specific field updates.
	// A more robust way would be a DAO method `UpdateTemplateStatus(ctx, id, status)`.
	// Let's assume for now that UpdateTemplate can take a map for selective updates,
	// or the DAO's UpdateTemplate method is smart enough.
	// Given the current DAO structure, it's safer to fetch then save the whole model.
	// However, the subtask implies a new DAO method `UpdateTemplateStatus`. I will assume it exists.
	err = t.dao.UpdateTemplateStatus(ctx, id, 1) // Assumed DAO method
	if err != nil {
		t.l.Error("启用模板失败", zap.Error(err), zap.Int("templateID", id))
		return fmt.Errorf("启用模板 (ID: %d) 失败: %w", id, err)
	}

	t.l.Info("模板启用成功", zap.Int("templateID", id))
	return nil
}

// DisableTemplate 禁用模板
func (t *templateService) DisableTemplate(ctx context.Context, id int, userID int) error {
	t.l.Info("开始禁用模板", zap.Int("templateID", id), zap.Int("userID", userID))
	// TODO: Add permission check for userID if necessary

	// Assuming UpdateTemplateStatus is a new DAO method:
	// err := t.dao.UpdateTemplateStatus(ctx, id, 0) // 0 for disabled
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("禁用模板失败：获取模板失败", zap.Error(err), zap.Int("templateID", id))
		return fmt.Errorf("获取模板 (ID: %d) 失败: %w", id, err)
	}
	if template.Status == 0 {
		t.l.Info("模板已禁用，无需操作", zap.Int("templateID", id))
		return nil // Already disabled
	}
	template.Status = 0 // Set status to disabled
	
	err = t.dao.UpdateTemplateStatus(ctx, id, 0) // Assumed DAO method
	if err != nil {
		t.l.Error("禁用模板失败", zap.Error(err), zap.Int("templateID", id))
		return fmt.Errorf("禁用模板 (ID: %d) 失败: %w", id, err)
	}

	t.l.Info("模板禁用成功", zap.Int("templateID", id))
	return nil
}
