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
	CreateTemplate(ctx context.Context, req *model.CreateTemplateReq) error
	UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq) error
	DeleteTemplate(ctx context.Context, req *model.DeleteTemplateReq) error
	ListTemplate(ctx context.Context, req *model.ListTemplateReq) ([]model.Template, error)
	DetailTemplate(ctx context.Context, req *model.DetailTemplateReq) (*model.Template, error)
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
func (t *templateService) CreateTemplate(ctx context.Context, req *model.CreateTemplateReq) error {
	template, err := utils.ConvertTemplateReq(&req)
	if err != nil {
		return err
	}
	return t.dao.CreateTemplate(ctx, template)
}

// DeleteTemplate 删除模板
func (t *templateService) DeleteTemplate(ctx context.Context, req *model.DeleteTemplateReq) error {
	return t.dao.DeleteTemplate(ctx, req.ID)
}

// DetailTemplate 查看模板详情
func (t *templateService) DetailTemplate(ctx context.Context, req *model.DetailTemplateReq) (*model.Template, error) {
	template, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		t.l.Error("获取模板失败", zap.Error(err))
		return nil, err
	}
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
	template, err := utils.ConvertTemplateReq(&req)
	if err != nil {
		return err
	}
	return t.dao.UpdateTemplate(ctx, template)
}
