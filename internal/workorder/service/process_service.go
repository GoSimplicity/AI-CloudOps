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
	"reflect"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type ProcessService interface {
	CreateProcess(ctx context.Context, req *model.CreateProcessReq, creatorID int, creatorName string) error
	UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error
	DeleteProcess(ctx context.Context, id int) error
	ListProcess(ctx context.Context, req *model.ListProcessReq) (*model.ListResp[model.Process], error)
	DetailProcess(ctx context.Context, id int, userID int) (*model.Process, error)
	GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error)
	PublishProcess(ctx context.Context, id int) error
	CloneProcess(ctx context.Context, req *model.CloneProcessReq, creatorID int) (*model.Process, error)
	ValidateProcess(ctx context.Context, id int, userID int) (*model.ValidateProcessResp, error)
}

type processService struct {
	dao     dao.ProcessDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewProcessService(dao dao.ProcessDAO, userDao userDao.UserDAO, l *zap.Logger) ProcessService {
	return &processService{
		dao:     dao,
		userDao: userDao,
		l:       l,
	}
}

// CreateProcess 创建流程
func (p *processService) CreateProcess(ctx context.Context, req *model.CreateProcessReq, creatorID int, creatorName string) error {
	// 检查流程名称是否已存在
	exists, err := p.checkProcessNameExists(ctx, req.Name)
	if err != nil {
		p.l.Error("检查流程名称失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("检查流程名称失败: %w", err)
	}
	if exists {
		return fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	// 转换请求到模型
	process := &model.Process{
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		CategoryID:   req.CategoryID,
		CreatorID:    creatorID,
		CreatorName:  creatorName,
		Status:       0, // 草稿状态
		Version:      1,
	}

	// 如果有流程定义，进行验证和序列化
	if len(req.Definition.Steps) > 0 || len(req.Definition.Connections) > 0 {
		if err := p.validateProcessDefinition(&req.Definition); err != nil {
			p.l.Error("流程定义验证失败", zap.Error(err))
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.l.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		process.Definition = string(definitionJSON)
	}

	if err := p.dao.CreateProcess(ctx, process); err != nil {
		p.l.Error("创建流程失败", zap.Error(err), zap.String("name", req.Name))
		return err
	}

	p.l.Info("创建流程成功", zap.Int("id", process.ID), zap.String("name", req.Name))
	return nil
}

// UpdateProcess 更新流程
func (p *processService) UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error {
	if req == nil {
		return fmt.Errorf("更新流程请求不能为空")
	}

	// 检查流程是否存在
	existingProcess, err := p.dao.GetProcess(ctx, req.ID)
	if err != nil {
		return err
	}

	// 检查流程名称是否与其他流程重复
	if req.Name != existingProcess.Name {
		exists, err := p.dao.CheckProcessNameExists(ctx, req.Name, req.ID)
		if err != nil {
			p.l.Error("检查流程名称失败", zap.Error(err), zap.String("name", req.Name))
			return fmt.Errorf("检查流程名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("流程名称已存在: %s", req.Name)
		}
	}

	// 构建更新的流程对象
	process := &model.Process{
		Model:        model.Model{ID: req.ID},
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		CategoryID:   req.CategoryID,
		Status:       existingProcess.Status,
		Version:      existingProcess.Version + 1, // 版本号递增
		CreatorID:    existingProcess.CreatorID,
		CreatorName:  existingProcess.CreatorName,
	}

	// 如果有流程定义，进行验证和序列化
	if !reflect.DeepEqual(req.Definition, model.ProcessDefinition{}) {
		if err := p.validateProcessDefinition(&req.Definition); err != nil {
			p.l.Error("流程定义验证失败", zap.Error(err))
			return fmt.Errorf("流程定义验证失败: %w", err)
		}

		definitionJSON, err := json.Marshal(req.Definition)
		if err != nil {
			p.l.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		process.Definition = string(definitionJSON)
	} else {
		process.Definition = existingProcess.Definition
	}

	if err := p.dao.UpdateProcess(ctx, process); err != nil {
		p.l.Error("更新流程失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	p.l.Info("更新流程成功", zap.Int("id", req.ID), zap.String("name", req.Name))
	return nil
}

// DeleteProcess 删除流程
func (p *processService) DeleteProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	if err := p.dao.DeleteProcess(ctx, id); err != nil {
		p.l.Error("删除流程失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	p.l.Info("删除流程成功", zap.Int("id", id))
	return nil
}

// ListProcess 获取流程列表
func (p *processService) ListProcess(ctx context.Context, req *model.ListProcessReq) (*model.ListResp[model.Process], error) {
	if req == nil {
		return nil, fmt.Errorf("列表请求不能为空")
	}

	// 设置默认分页参数
	p.setDefaultPagination(req)

	result, err := p.dao.ListProcess(ctx, req)
	if err != nil {
		p.l.Error("获取流程列表失败", zap.Error(err))
		return nil, err
	}

	p.l.Info("获取流程列表成功", zap.Int("count", len(result.Items)), zap.Int64("total", result.Total))
	return result, nil
}

// DetailProcess 获取流程详情
func (p *processService) DetailProcess(ctx context.Context, id int, userID int) (*model.Process, error) {
	if id <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		p.l.Error("获取流程详情失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	// 获取创建者信息
	if process.CreatorID > 0 {
		if user, err := p.userDao.GetUserByID(ctx, process.CreatorID); err != nil {
			p.l.Warn("获取创建者信息失败", zap.Error(err), zap.Int("creatorID", process.CreatorID))
		} else {
			process.CreatorName = user.Username
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
		p.l.Error("获取流程关联数据失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return process, nil
}

// PublishProcess 发布流程
func (p *processService) PublishProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取流程详情并验证
	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		return err
	}

	// 验证流程定义
	if process.Definition == "" {
		return fmt.Errorf("流程定义为空，无法发布")
	}

	var definition model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &definition); err != nil {
		p.l.Error("解析流程定义失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	if err := p.validateProcessDefinition(&definition); err != nil {
		p.l.Error("流程定义验证失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("流程定义验证失败: %w", err)
	}

	if err := p.dao.PublishProcess(ctx, id); err != nil {
		p.l.Error("发布流程失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	p.l.Info("发布流程成功", zap.Int("id", id))
	return nil
}

// CloneProcess 克隆流程
func (p *processService) CloneProcess(ctx context.Context, req *model.CloneProcessReq, creatorID int) (*model.Process, error) {
	if req == nil || req.ID <= 0 {
		return nil, fmt.Errorf("克隆流程请求无效")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("新流程名称不能为空")
	}

	// 检查新名称是否已存在
	exists, err := p.checkProcessNameExists(ctx, req.Name)
	if err != nil {
		p.l.Error("检查流程名称失败", zap.Error(err), zap.String("name", req.Name))
		return nil, fmt.Errorf("检查流程名称失败: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	clonedProcess, err := p.dao.CloneProcess(ctx, req.ID, req.Name, creatorID)
	if err != nil {
		p.l.Error("克隆流程失败", zap.Error(err), zap.Int("originalID", req.ID), zap.String("newName", req.Name))
		return nil, err
	}

	p.l.Info("克隆流程成功", zap.Int("originalID", req.ID), zap.Int("newID", clonedProcess.ID), zap.String("newName", req.Name))
	return clonedProcess, nil
}

// ValidateProcess 校验流程定义
func (p *processService) ValidateProcess(ctx context.Context, id int, userID int) (*model.ValidateProcessResp, error) {
	p.l.Info("开始校验流程", zap.Int("processID", id), zap.Int("userID", userID))

	resp := &model.ValidateProcessResp{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		p.l.Error("校验流程失败：获取流程失败", zap.Error(err), zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("获取流程 (ID: %d) 失败: %s", id, err.Error()))
		return resp, err
	}

	if process.Definition == "" {
		p.l.Warn("流程定义为空", zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程定义为空")
		return resp, nil
	}

	var def model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &def); err != nil {
		p.l.Error("校验流程失败：解析流程定义JSON失败", zap.Error(err), zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("解析流程定义JSON失败: %s", err.Error()))
		return resp, nil
	}

	// 使用DAO层的验证方法
	if err := p.dao.ValidateProcessDefinition(ctx, &def); err != nil {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, err.Error())
	}

	// 额外的业务层验证
	if err := p.validateProcessDefinition(&def); err != nil {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, err.Error())
	}

	p.l.Info("流程校验完成", zap.Int("processID", id), zap.Bool("isValid", resp.IsValid), zap.Strings("errors", resp.Errors))
	return resp, nil
}

// checkProcessNameExists 检查流程名称是否存在
func (p *processService) checkProcessNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("流程名称不能为空")
	}
	return p.dao.CheckProcessNameExists(ctx, name, excludeID...)
}

// setDefaultPagination 设置默认分页参数
func (p *processService) setDefaultPagination(req *model.ListProcessReq) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100
	}
}

// validateProcessDefinition 验证流程定义的业务逻辑
func (p *processService) validateProcessDefinition(definition *model.ProcessDefinition) error {
	if definition == nil {
		return fmt.Errorf("流程定义不能为空")
	}

	if len(definition.Steps) == 0 {
		return fmt.Errorf("流程必须包含至少一个步骤")
	}

	// 验证步骤
	stepIDs := make(map[string]bool)
	hasStartStep := false
	hasEndStep := false

	for _, step := range definition.Steps {
		if step.ID == "" {
			return fmt.Errorf("步骤ID不能为空")
		}
		if step.Name == "" {
			return fmt.Errorf("步骤名称不能为空")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}
		stepIDs[step.ID] = true

		if step.Type == "start" {
			hasStartStep = true
		}
		if step.Type == "end" {
			hasEndStep = true
		}
	}

	if !hasStartStep {
		return fmt.Errorf("流程必须包含一个开始步骤")
	}
	if !hasEndStep {
		return fmt.Errorf("流程必须包含一个结束步骤")
	}

	// 验证连接
	for _, conn := range definition.Connections {
		if conn.From == "" || conn.To == "" {
			return fmt.Errorf("连接的起始和目标步骤ID不能为空")
		}
		if !stepIDs[conn.From] {
			return fmt.Errorf("连接的起始步骤不存在: %s", conn.From)
		}
		if !stepIDs[conn.To] {
			return fmt.Errorf("连接的目标步骤不存在: %s", conn.To)
		}
	}

	return nil
}
