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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type ProcessService interface {
	CreateProcess(ctx context.Context, req *model.CreateProcessReq) error
	UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req *model.ListProcessReq) (model.ListResp[*model.Process], error)
	DetailProcess(ctx context.Context, id int) (*model.Process, error)
	GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error)
	PublishProcess(ctx context.Context, id int) error
	CloneProcess(ctx context.Context, req *model.CloneProcessReq, creatorID int) (*model.Process, error)
}

type processService struct {
	dao           dao.ProcessDAO
	userDao       userDao.UserDAO
	formDesignDao dao.FormDesignDAO
	categoryDao   dao.CategoryDAO
	instanceDao   dao.InstanceDAO
	logger        *zap.Logger
}

func NewProcessService(
	processDao dao.ProcessDAO,
	formDesignDao dao.FormDesignDAO,
	userDao userDao.UserDAO,
	categoryDao dao.CategoryDAO,
	instanceDao dao.InstanceDAO,
	logger *zap.Logger,
) ProcessService {
	return &processService{
		dao:           processDao,
		userDao:       userDao,
		formDesignDao: formDesignDao,
		categoryDao:   categoryDao,
		instanceDao:   instanceDao,
		logger:        logger,
	}
}

// CreateProcess 创建流程
func (p *processService) CreateProcess(ctx context.Context, req *model.CreateProcessReq) error {
	if req.Name == "" {
		return fmt.Errorf("流程名称不能为空")
	}

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

	// 验证表单设计是否存在
	if req.FormDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	if _, err := p.formDesignDao.GetFormDesign(ctx, req.FormDesignID); err != nil {
		p.logger.Error("验证表单发生错误", zap.Error(err))
		return fmt.Errorf("验证表单设计失败: %w", err)
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if _, err := p.categoryDao.GetCategory(ctx, *req.CategoryID); err != nil {
			p.logger.Error("验证分类发生错误", zap.Error(err))
			return fmt.Errorf("验证分类失败: %w", err)
		}
	}

	// 转换请求到模型
	process := &model.Process{
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		CategoryID:   req.CategoryID,
		CreatorID:    req.CreatorID,
		CreatorName:  req.CreatorName,
		Status:       model.ProcessStatusDraft,
		Version:      req.Version,
	}

	// 处理流程定义
	if len(req.Definition.Steps) > 0 || len(req.Definition.Connections) > 0 {
		// 执行验证
		if err := p.dao.ValidateProcessDefinition(ctx, &req.Definition); err != nil {
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.logger.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		process.Definition = definitionJSON
	}

	// 创建流程
	if err := p.dao.CreateProcess(ctx, process); err != nil {
		p.logger.Error("创建流程失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("creatorID", req.CreatorID))
		return fmt.Errorf("创建流程失败: %w", err)
	}

	return nil
}

// UpdateProcess 更新流程
func (p *processService) UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error {
	if req.ID <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	if req.Name == "" {
		return fmt.Errorf("流程名称不能为空")
	}

	// 获取现有流程
	existingProcess, err := p.dao.GetProcess(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程状态，已发布的流程需要特殊处理
	if existingProcess.Status == 2 && req.Status != 1 {
		// 只允许从已发布(2)状态变更为草稿(1)状态，即取消发布
		p.logger.Warn("尝试更新已发布的流程",
			zap.Int("id", req.ID),
			zap.String("name", existingProcess.Name))
		// 拒绝更新
		return fmt.Errorf("已发布的流程只能取消发布，不能进行其他更新")
	}

	// 检查流程名称是否与其他流程重复
	if req.Name != existingProcess.Name {
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

	// 验证表单设计是否存在
	if req.FormDesignID <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	if req.FormDesignID != existingProcess.FormDesignID {
		if _, err := p.formDesignDao.GetFormDesign(ctx, req.FormDesignID); err != nil {
			p.logger.Error("验证表单发生错误", zap.Error(err))
			return fmt.Errorf("验证表单设计失败: %w", err)
		}
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if _, err := p.categoryDao.GetCategory(ctx, *req.CategoryID); err != nil {
			p.logger.Error("验证分类发生错误", zap.Error(err))
			return fmt.Errorf("验证分类失败: %w", err)
		}
	}

	// 构建更新的流程对象
	process := &model.Process{
		Model:        model.Model{ID: req.ID},
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		CategoryID:   req.CategoryID,
		Status:       req.Status,
		Version:      req.Version,
		CreatorID:    existingProcess.CreatorID,
		CreatorName:  existingProcess.CreatorName,
	}

	// 处理流程定义
	if len(req.Definition.Steps) > 0 || len(req.Definition.Connections) > 0 {
		// 执行验证
		if err := p.dao.ValidateProcessDefinition(ctx, &req.Definition); err != nil {
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.logger.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		process.Definition = definitionJSON
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

// DeleteProcess 删除流程
func (p *processService) DeleteProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取流程信息用于日志记录
	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程是否可以删除
	if process.Status == 2 {
		return fmt.Errorf("已发布的流程不能删除")
	}

	// 检查是否有正在运行的实例
	status := model.InstanceStatusProcessing
	instances, _, err := p.instanceDao.ListInstance(ctx, &model.ListInstanceReq{
		ProcessID: &id,
		Status:    &status,
	})
	if err != nil {
		return fmt.Errorf("获取实例失败: %w", err)
	}

	if len(instances) > 0 {
		return fmt.Errorf("流程有正在运行的实例，不能删除")
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

// ListProcess 获取流程列表
func (p *processService) ListProcess(ctx context.Context, req *model.ListProcessReq) (model.ListResp[*model.Process], error) {
	processes, total, err := p.dao.ListProcess(ctx, req)
	if err != nil {
		p.logger.Error("获取流程列表失败", zap.Error(err))
		return model.ListResp[*model.Process]{}, fmt.Errorf("获取流程列表失败: %w", err)
	}

	// 批量获取创建者信息
	if err := p.enrichProcessListWithCreators(ctx, processes); err != nil {
		p.logger.Warn("获取创建者信息失败", zap.Error(err))
	}

	result := model.ListResp[*model.Process]{
		Items: processes,
		Total: total,
	}

	return result, nil
}

// DetailProcess 获取流程详情
func (p *processService) DetailProcess(ctx context.Context, id int) (*model.Process, error) {
	if id <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	// 从数据库获取
	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		p.logger.Error("获取流程详情失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取流程详情失败: %w", err)
	}

	// 获取创建者信息
	if process.CreatorID > 0 && process.CreatorName == "" {
		if user, err := p.userDao.GetUserByID(ctx, process.CreatorID); err != nil {
			p.logger.Warn("获取创建者信息失败",
				zap.Error(err),
				zap.Int("creatorID", process.CreatorID))
		} else if user != nil {
			process.CreatorName = user.Username
		}
	}

	// 解析流程定义以便前端使用
	if process.Definition != nil {
		var def model.ProcessDefinition
		if err := json.Unmarshal([]byte(process.Definition), &def); err != nil {
			p.logger.Warn("解析流程定义失败",
				zap.Error(err),
				zap.Int("id", id))
		}
	}

	return process, nil
}

// GetProcessWithRelations 获取流程及其关联数据
func (p *processService) GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error) {
	if id <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	process, err := p.dao.GetProcessWithRelations(ctx, id)
	if err != nil {
		p.logger.Error("获取流程关联数据失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取流程关联数据失败: %w", err)
	}

	// 获取创建者信息
	if process.CreatorID > 0 && process.CreatorName == "" {
		if user, err := p.userDao.GetUserByID(ctx, process.CreatorID); err != nil {
			p.logger.Warn("获取创建者信息失败",
				zap.Error(err),
				zap.Int("creatorID", process.CreatorID))
		} else if user != nil {
			process.CreatorName = user.Username
		}
	}

	return process, nil
}

// PublishProcess 发布流程
func (p *processService) PublishProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取流程详情
	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程状态
	if process.Status == 2 {
		return fmt.Errorf("流程已经发布")
	}

	if process.Status == 3 {
		return fmt.Errorf("流程已被禁用，请先启用后再发布")
	}

	// 验证流程定义
	if process.Definition == nil {
		return fmt.Errorf("流程定义为空，无法发布")
	}

	var definition model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &definition); err != nil {
		p.logger.Error("解析流程定义失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	// 验证流程定义
	if err := p.dao.ValidateProcessDefinition(ctx, &definition); err != nil {
		p.logger.Error("流程定义验证失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("流程定义验证失败: %w", err)
	}

	// 发布流程
	if err := p.dao.PublishProcess(ctx, id); err != nil {
		p.logger.Error("发布流程失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("发布流程失败: %w", err)
	}

	return nil
}

// CloneProcess 克隆流程
func (p *processService) CloneProcess(ctx context.Context, req *model.CloneProcessReq, creatorID int) (*model.Process, error) {
	if req.ID <= 0 {
		return nil, fmt.Errorf("克隆流程请求无效")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("新流程名称不能为空")
	}

	if creatorID <= 0 {
		return nil, fmt.Errorf("创建者ID无效")
	}

	// 检查新名称是否已存在
	exists, err := p.dao.CheckProcessNameExists(ctx, req.Name)
	if err != nil {
		p.logger.Error("检查流程名称失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("检查流程名称失败: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	// 执行克隆
	clonedProcess, err := p.dao.CloneProcess(ctx, req.ID, req.Name, creatorID)
	if err != nil {
		p.logger.Error("克隆流程失败",
			zap.Error(err),
			zap.Int("originalID", req.ID),
			zap.String("newName", req.Name))
		return nil, fmt.Errorf("克隆流程失败: %w", err)
	}

	return clonedProcess, nil
}

// enrichProcessListWithCreators 批量获取创建者信息
func (p *processService) enrichProcessListWithCreators(ctx context.Context, processes []*model.Process) error {
	// 收集所有创建者ID
	creatorIDs := make([]int, 0)
	creatorIDMap := make(map[int]bool)

	for _, process := range processes {
		if process.CreatorID > 0 && !creatorIDMap[process.CreatorID] {
			creatorIDs = append(creatorIDs, process.CreatorID)
			creatorIDMap[process.CreatorID] = true
		}
	}

	if len(creatorIDs) == 0 {
		return nil
	}

	// 批量获取用户信息
	users, err := p.userDao.GetUserByIDs(ctx, creatorIDs)
	if err != nil {
		return err
	}

	// 构建用户ID到名称的映射
	userMap := make(map[int]string)
	for _, user := range users {
		userMap[user.ID] = user.Username
	}

	// 填充创建者名称
	for i := range processes {
		if name, ok := userMap[processes[i].CreatorID]; ok {
			processes[i].CreatorName = name
		}
	}

	return nil
}
