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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesignReq *model.CreateWorkorderFormDesignReq) error
	UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateWorkorderFormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int) error
	GetFormDesign(ctx context.Context, id int) (*model.WorkorderFormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListWorkorderFormDesignReq) (model.ListResp[*model.WorkorderFormDesign], error)
}

type formDesignService struct {
	dao         dao.WorkorderFormDesignDAO
	categoryDao dao.WorkorderCategoryDAO
	logger      *zap.Logger
}

func NewFormDesignService(dao dao.WorkorderFormDesignDAO, categoryDao dao.WorkorderCategoryDAO, logger *zap.Logger) FormDesignService {
	return &formDesignService{
		dao:         dao,
		categoryDao: categoryDao,
		logger:      logger,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignService) CreateFormDesign(ctx context.Context, formDesignReq *model.CreateWorkorderFormDesignReq) error {
	// 检查名称唯一性
	exists, err := f.dao.CheckFormDesignNameExists(ctx, formDesignReq.Name)
	if err != nil {
		f.logger.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", formDesignReq.Name))
		return fmt.Errorf("检查表单设计名称失败: %w", err)
	}
	if exists {
		f.logger.Warn("表单设计名称已存在", zap.String("name", formDesignReq.Name))
		return dao.ErrFormDesignNameExists
	}

	// 验证表单结构
	if formDesignReq.Schema.Fields == nil || len(formDesignReq.Schema.Fields) == 0 {
		f.logger.Error("表单结构不能为空")
		return errors.New("表单结构不能为空")
	}

	// 校验标签
	if len(formDesignReq.Tags) > 0 {
		for _, tag := range formDesignReq.Tags {
			if strings.TrimSpace(tag) == "" {
				return errors.New("标签不能为空字符串")
			}
		}
	}

	schemaJSON, err := json.Marshal(formDesignReq.Schema)
	if err != nil {
		f.logger.Error("表单结构序列化失败", zap.Error(err))
		return fmt.Errorf("表单结构序列化失败: %w", err)
	}

	// 构建表单设计实体
	formDesign := &model.WorkorderFormDesign{
		Name:           formDesignReq.Name,
		Description:    formDesignReq.Description,
		Schema:         schemaJSON,
		Status:         formDesignReq.Status,
		CategoryID:     formDesignReq.CategoryID,
		CreateUserID:   formDesignReq.CreateUserID,
		CreateUserName: formDesignReq.CreateUserName,
		Tags:           formDesignReq.Tags,
		IsTemplate:     formDesignReq.IsTemplate,
	}

	// 创建表单设计
	if err := f.dao.CreateFormDesign(ctx, formDesign); err != nil {
		f.logger.Error("创建表单设计失败", zap.Error(err), zap.String("name", formDesignReq.Name))
		return err
	}

	return nil
}

// UpdateFormDesign 更新表单设计
func (f *formDesignService) UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateWorkorderFormDesignReq) error {
	// 检查表单设计是否存在
	existingFormDesign, err := f.dao.GetFormDesign(ctx, formDesignReq.ID)
	if err != nil {
		f.logger.Error("获取表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	// 检查名称唯一性
	if formDesignReq.Name != existingFormDesign.Name {
		exists, err := f.dao.CheckFormDesignNameExists(ctx, formDesignReq.Name, formDesignReq.ID)
		if err != nil {
			f.logger.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", formDesignReq.Name))
			return fmt.Errorf("检查表单设计名称失败: %w", err)
		}
		if exists {
			f.logger.Warn("表单设计名称已存在", zap.String("name", formDesignReq.Name))
			return dao.ErrFormDesignNameExists
		}
	}

	// 验证表单结构
	if formDesignReq.Schema.Fields == nil || len(formDesignReq.Schema.Fields) == 0 {
		f.logger.Error("表单结构不能为空")
		return errors.New("表单结构不能为空")
	}

	// 校验标签
	if len(formDesignReq.Tags) > 0 {
		for _, tag := range formDesignReq.Tags {
			if strings.TrimSpace(tag) == "" {
				return errors.New("标签不能为空字符串")
			}
		}
	}

	schemaJSON, err := json.Marshal(formDesignReq.Schema)
	if err != nil {
		f.logger.Error("表单结构序列化失败", zap.Error(err))
		return fmt.Errorf("表单结构序列化失败: %w", err)
	}

	// 构建更新数据
	formDesign := &model.WorkorderFormDesign{
		Model:       model.Model{ID: formDesignReq.ID},
		Name:        formDesignReq.Name,
		Description: formDesignReq.Description,
		Schema:      schemaJSON,
		CategoryID:  formDesignReq.CategoryID,
		Status:      formDesignReq.Status,
		Tags:        formDesignReq.Tags,
		IsTemplate:  formDesignReq.IsTemplate,
	}

	// 更新表单设计
	if err := f.dao.UpdateFormDesign(ctx, formDesign); err != nil {
		f.logger.Error("更新表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	return nil
}

// DeleteFormDesign 删除表单设计
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("表单设计ID无效")
	}

	// 检查表单设计是否存在
	_, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.logger.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 删除表单设计
	if err := f.dao.DeleteFormDesign(ctx, id); err != nil {
		f.logger.Error("删除表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// GetFormDesign 获取表单设计详情
func (f *formDesignService) GetFormDesign(ctx context.Context, id int) (*model.WorkorderFormDesign, error) {
	if id <= 0 {
		return nil, errors.New("表单设计ID无效")
	}

	// 获取表单设计
	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.logger.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListWorkorderFormDesignReq) (model.ListResp[*model.WorkorderFormDesign], error) {
	// 获取表单设计列表
	formDesigns, total, err := f.dao.ListFormDesign(ctx, req)
	if err != nil {
		f.logger.Error("获取表单设计列表失败", zap.Error(err))
		return model.ListResp[*model.WorkorderFormDesign]{}, err
	}

	return model.ListResp[*model.WorkorderFormDesign]{
		Items: formDesigns,
		Total: total,
	}, nil
}
