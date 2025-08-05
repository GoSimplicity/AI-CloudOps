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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type WorkorderProcessService interface {
	CreateWorkorderProcess(ctx context.Context, req *model.CreateWorkorderProcessReq) error
	UpdateWorkorderProcess(ctx context.Context, req *model.UpdateWorkorderProcessReq) error
	DeleteWorkorderProcess(ctx context.Context, id int) error
	ListWorkorderProcess(ctx context.Context, req *model.ListWorkorderProcessReq) (*model.ListResp[*model.WorkorderProcess], error)
	DetailWorkorderProcess(ctx context.Context, id int) (*model.WorkorderProcess, error)
}

type workorderProcessService struct {
	dao           dao.WorkorderProcessDAO
	formDesignDao dao.WorkorderFormDesignDAO
	categoryDao   dao.WorkorderCategoryDAO
	instanceDao   dao.WorkorderInstanceDAO
	logger        *zap.Logger
}

func NewWorkorderProcessService(
	processDao dao.WorkorderProcessDAO,
	formDesignDao dao.WorkorderFormDesignDAO,
	categoryDao dao.WorkorderCategoryDAO,
	instanceDao dao.WorkorderInstanceDAO,
	logger *zap.Logger,
) WorkorderProcessService {
	return &workorderProcessService{
		dao:           processDao,
		formDesignDao: formDesignDao,
		categoryDao:   categoryDao,
		instanceDao:   instanceDao,
		logger:        logger,
	}
}

// CreateWorkorderProcess 创建流程
func (p *workorderProcessService) CreateWorkorderProcess(ctx context.Context, req *model.CreateWorkorderProcessReq) error {
	// 检查流程名称是否已存在
	exists, err := p.dao.CheckProcessNameExists(ctx, req.Name)
	if err != nil {
		p.logger.Error("检查流程名称失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("检查流程名称失败: %w", err)
	}
	if exists {
		return fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	// 检查表单设计是否存在
	if req.FormDesignID > 0 {
		_, err := p.formDesignDao.GetFormDesign(ctx, req.FormDesignID)
		if err != nil {
			p.logger.Error("表单设计不存在", zap.Error(err), zap.Int("formDesignID", req.FormDesignID))
			return fmt.Errorf("表单设计不存在: %w", err)
		}
	}

	// 检查分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		_, err := p.categoryDao.GetCategory(ctx, *req.CategoryID)
		if err != nil {
			p.logger.Error("流程分类不存在", zap.Error(err), zap.Int("categoryID", *req.CategoryID))
			return fmt.Errorf("流程分类不存在: %w", err)
		}
	}

	process := &model.WorkorderProcess{
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		Status:       req.Status,
		CategoryID:   req.CategoryID,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
		Tags:         req.Tags,
		IsDefault:    req.IsDefault,
	}

	// 处理流程定义
	if len(req.Definition.Steps) > 0 || len(req.Definition.Connections) > 0 {
		// 执行验证
		if err := p.dao.ValidateProcessDefinition(ctx, &req.Definition); err != nil {
			p.logger.Error("流程定义验证失败", zap.Error(err))
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		// 序列化流程定义
		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.logger.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		// 转换为JSONMap
		var definitionMap model.JSONMap
		err = json.Unmarshal(definitionJSON, &definitionMap)
		if err != nil {
			p.logger.Error("转换流程定义为JSONMap失败", zap.Error(err))
			return fmt.Errorf("转换流程定义失败: %w", err)
		}
		process.Definition = definitionMap
	}

	// 创建流程
	if err := p.dao.CreateProcess(ctx, process); err != nil {
		p.logger.Error("创建流程失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("createUserID", req.OperatorID))
		return fmt.Errorf("创建流程失败: %w", err)
	}

	return nil
}

// UpdateWorkorderProcess 更新流程
func (p *workorderProcessService) UpdateWorkorderProcess(ctx context.Context, req *model.UpdateWorkorderProcessReq) error {
	if req.ID <= 0 {
		return errors.New("流程ID无效")
	}

	// 获取现有流程
	existingProcess, err := p.dao.GetProcessByID(ctx, req.ID)
	if err != nil {
		p.logger.Error("获取流程失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程名称是否已存在
	if req.Name != "" && req.Name != existingProcess.Name {
		exists, err := p.dao.CheckProcessNameExists(ctx, req.Name, req.ID)
		if err != nil {
			p.logger.Error("检查流程名称失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("检查流程名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("流程名称已存在: %s", req.Name)
		}
	}

	// 检查表单设计是否存在
	if req.FormDesignID > 0 && req.FormDesignID != existingProcess.FormDesignID {
		_, err := p.formDesignDao.GetFormDesign(ctx, req.FormDesignID)
		if err != nil {
			p.logger.Error("表单设计不存在", zap.Error(err), zap.Int("formDesignID", req.FormDesignID))
			return fmt.Errorf("表单设计不存在: %w", err)
		}
	}

	// 检查分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 && req.CategoryID != existingProcess.CategoryID {
		_, err := p.categoryDao.GetCategory(ctx, *req.CategoryID)
		if err != nil {
			p.logger.Error("流程分类不存在", zap.Error(err), zap.Int("categoryID", *req.CategoryID))
			return fmt.Errorf("流程分类不存在: %w", err)
		}
	}

	// 构建更新的流程对象
	process := &model.WorkorderProcess{
		Model:        model.Model{ID: req.ID},
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		Status:       req.Status,
		CategoryID:   req.CategoryID,
		Tags:         req.Tags,
		IsDefault:    req.IsDefault,
	}

	// 处理流程定义
	if len(req.Definition.Steps) > 0 || len(req.Definition.Connections) > 0 {
		// 执行验证
		if err := p.dao.ValidateProcessDefinition(ctx, &req.Definition); err != nil {
			p.logger.Error("流程定义验证失败", zap.Error(err))
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.logger.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		// 转换为JSONMap
		var definitionMap model.JSONMap
		err = json.Unmarshal(definitionJSON, &definitionMap)
		if err != nil {
			p.logger.Error("转换流程定义为JSONMap失败", zap.Error(err))
			return fmt.Errorf("转换流程定义失败: %w", err)
		}
		process.Definition = definitionMap
	}

	// 更新流程
	if err := p.dao.UpdateProcess(ctx, process); err != nil {
		p.logger.Error("更新流程失败",
			zap.Error(err),
			zap.Int("id", req.ID))
		return fmt.Errorf("更新流程失败: %w", err)
	}

	return nil
}

// DeleteWorkorderProcess 删除流程
func (p *workorderProcessService) DeleteWorkorderProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("流程ID无效")
	}

	process, err := p.dao.GetProcessByID(ctx, id)
	if err != nil {
		p.logger.Error("获取流程失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("获取流程失败: %w", err)
	}

	if process.Status == model.ProcessStatusPublished {
		return fmt.Errorf("已发布的流程不能删除")
	}

	page := 1
	size := 100
	for {
		instances, total, err := p.instanceDao.ListInstance(ctx, &model.ListWorkorderInstanceReq{
			ProcessID: &id,
			ListReq: model.ListReq{
				Page: page,
				Size: size,
			},
		})
		if err != nil {
			p.logger.Error("获取流程实例失败", zap.Error(err))
			return fmt.Errorf("获取流程实例失败: %w", err)
		}
		if len(instances) > 0 {
			return fmt.Errorf("流程有正在运行的实例，不能删除")
		}
		// 如果本次返回数量小于size，说明已经没有更多数据
		if total <= int64(page*size) {
			break
		}
		page++
	}

	// 执行删除
	if err := p.dao.DeleteProcess(ctx, id); err != nil {
		p.logger.Error("删除流程失败",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("name", process.Name))
		return fmt.Errorf("删除流程失败: %w", err)
	}

	return nil
}

// ListWorkorderProcess 获取流程列表
func (p *workorderProcessService) ListWorkorderProcess(ctx context.Context, req *model.ListWorkorderProcessReq) (*model.ListResp[*model.WorkorderProcess], error) {
	processes, total, err := p.dao.ListProcess(ctx, req)
	if err != nil {
		p.logger.Error("获取流程列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取流程列表失败: %w", err)
	}

	result := &model.ListResp[*model.WorkorderProcess]{
		Items: processes,
		Total: total,
	}

	return result, nil
}

// DetailWorkorderProcess 获取流程详情
func (p *workorderProcessService) DetailWorkorderProcess(ctx context.Context, id int) (*model.WorkorderProcess, error) {
	if id <= 0 {
		return nil, errors.New("流程ID无效")
	}

	// 从数据库获取
	process, err := p.dao.GetProcessByID(ctx, id)
	if err != nil {
		p.logger.Error("获取流程详情失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取流程详情失败: %w", err)
	}

	return process, nil
}
