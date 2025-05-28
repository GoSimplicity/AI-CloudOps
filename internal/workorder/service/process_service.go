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
	"sync"
	"time"

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
	dao             dao.ProcessDAO
	userDao         userDao.UserDAO
	formDesignDao   dao.FormDesignDAO
	l               *zap.Logger
	mu              sync.RWMutex           // 读写锁，用于保护缓存
	processCache    map[int]*model.Process // 流程缓存
	cacheExpiration time.Duration          // 缓存过期时间
	validators      []ProcessValidator     // 流程验证器列表
}

type ProcessValidator interface {
	Validate(ctx context.Context, definition *model.ProcessDefinition) error
}

func NewProcessService(processDao dao.ProcessDAO, formDesignDao dao.FormDesignDAO, userDao userDao.UserDAO, l *zap.Logger) ProcessService {
	ps := &processService{
		dao:             processDao,
		userDao:         userDao,
		formDesignDao:   formDesignDao,
		l:               l,
		processCache:    make(map[int]*model.Process),
		cacheExpiration: 5 * time.Minute,
		validators:      make([]ProcessValidator, 0),
	}

	// 注册默认验证器
	ps.RegisterValidator(&defaultProcessValidator{})

	return ps
}

// RegisterValidator 注册流程验证器
func (p *processService) RegisterValidator(validator ProcessValidator) {
	p.validators = append(p.validators, validator)
}

// CreateProcess 创建流程
func (p *processService) CreateProcess(ctx context.Context, req *model.CreateProcessReq, creatorID int, creatorName string) error {
	// 参数验证
	if req == nil {
		return fmt.Errorf("创建流程请求不能为空")
	}

	if creatorID <= 0 {
		return fmt.Errorf("创建者ID无效")
	}

	if creatorName == "" {
		return fmt.Errorf("创建者名称不能为空")
	}

	// 检查流程名称是否已存在
	exists, err := p.checkProcessNameExists(ctx, req.Name)
	if err != nil {
		p.l.Error("检查流程名称失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("creatorID", creatorID))
		return fmt.Errorf("检查流程名称失败: %w", err)
	}

	if exists {
		return fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	// 验证表单设计是否存在
	if err := p.validateFormDesignExists(ctx, req.FormDesignID); err != nil {
		return err
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := p.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
		}
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

	// 处理流程定义
	if err := p.processDefinition(ctx, &req.Definition, process); err != nil {
		return err
	}

	// 创建流程
	if err := p.dao.CreateProcess(ctx, process); err != nil {
		p.l.Error("创建流程失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("creatorID", creatorID))
		return fmt.Errorf("创建流程失败: %w", err)
	}

	p.l.Info("创建流程成功",
		zap.Int("id", process.ID),
		zap.String("name", req.Name),
		zap.Int("creatorID", creatorID))

	return nil
}

// UpdateProcess 更新流程
func (p *processService) UpdateProcess(ctx context.Context, req *model.UpdateProcessReq) error {
	if req == nil {
		return fmt.Errorf("更新流程请求不能为空")
	}

	if req.ID <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取现有流程
	existingProcess, err := p.getProcessFromCacheOrDB(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程状态，已发布的流程需要特殊处理
	if existingProcess.Status == 1 {
		p.l.Warn("尝试更新已发布的流程",
			zap.Int("id", req.ID),
			zap.String("name", existingProcess.Name))
		// 可以选择创建新版本或者拒绝更新
		// 这里我们选择允许更新但记录警告
	}

	// 检查流程名称是否与其他流程重复
	if req.Name != existingProcess.Name {
		exists, err := p.dao.CheckProcessNameExists(ctx, req.Name, req.ID)
		if err != nil {
			p.l.Error("检查流程名称失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("检查流程名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("流程名称已存在: %s", req.Name)
		}
	}

	// 验证表单设计是否存在
	if req.FormDesignID != existingProcess.FormDesignID {
		if err := p.validateFormDesignExists(ctx, req.FormDesignID); err != nil {
			return err
		}
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := p.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
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

	// 处理流程定义
	if err := p.processDefinition(ctx, &req.Definition, process); err != nil {
		return err
	}

	// 更新流程
	if err := p.dao.UpdateProcess(ctx, process); err != nil {
		p.l.Error("更新流程失败",
			zap.Error(err),
			zap.Int("id", req.ID))
		return fmt.Errorf("更新流程失败: %w", err)
	}

	// 清除缓存
	p.invalidateCache(req.ID)

	p.l.Info("更新流程成功",
		zap.Int("id", req.ID),
		zap.String("name", req.Name),
		zap.Int("version", process.Version))

	return nil
}

// DeleteProcess 删除流程
func (p *processService) DeleteProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取流程信息用于日志记录
	process, err := p.getProcessFromCacheOrDB(ctx, id)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程是否可以删除
	if process.Status == 1 {
		return fmt.Errorf("已发布的流程不能删除")
	}

	// 检查是否有正在运行的实例
	// TODO: 添加检查逻辑

	// 执行删除
	if err := p.dao.DeleteProcess(ctx, id); err != nil {
		p.l.Error("删除流程失败",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("name", process.Name))
		return fmt.Errorf("删除流程失败: %w", err)
	}

	// 清除缓存
	p.invalidateCache(id)

	p.l.Info("删除流程成功",
		zap.Int("id", id),
		zap.String("name", process.Name))

	return nil
}

// ListProcess 获取流程列表
func (p *processService) ListProcess(ctx context.Context, req *model.ListProcessReq) (*model.ListResp[model.Process], error) {
	if req == nil {
		req = &model.ListProcessReq{}
	}

	// 设置默认分页参数
	p.setDefaultPagination(req)

	// 记录查询参数
	p.l.Debug("查询流程列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.Any("filters", map[string]interface{}{
			"name":         req.Search,
			"categoryID":   req.CategoryID,
			"formDesignID": req.FormDesignID,
			"status":       req.Status,
		}))

	result, err := p.dao.ListProcess(ctx, req)
	if err != nil {
		p.l.Error("获取流程列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取流程列表失败: %w", err)
	}

	// 批量获取创建者信息
	if err := p.enrichProcessListWithCreators(ctx, result.Items); err != nil {
		p.l.Warn("获取创建者信息失败", zap.Error(err))
		// 不影响主流程，继续返回结果
	}

	p.l.Info("获取流程列表成功",
		zap.Int("count", len(result.Items)),
		zap.Int64("total", result.Total))

	return result, nil
}

// DetailProcess 获取流程详情
func (p *processService) DetailProcess(ctx context.Context, id int, userID int) (*model.Process, error) {
	if id <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	// 从缓存或数据库获取
	process, err := p.getProcessFromCacheOrDB(ctx, id)
	if err != nil {
		p.l.Error("获取流程详情失败",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID))
		return nil, fmt.Errorf("获取流程详情失败: %w", err)
	}

	// 获取创建者信息
	if process.CreatorID > 0 && process.CreatorName == "" {
		if user, err := p.userDao.GetUserByID(ctx, process.CreatorID); err != nil {
			p.l.Warn("获取创建者信息失败",
				zap.Error(err),
				zap.Int("creatorID", process.CreatorID))
		} else {
			process.CreatorName = user.Username
		}
	}

	// 解析流程定义以便前端使用
	if process.Definition != "" {
		var def model.ProcessDefinition
		if err := json.Unmarshal([]byte(process.Definition), &def); err != nil {
			p.l.Warn("解析流程定义失败",
				zap.Error(err),
				zap.Int("id", id))
		}
	}

	p.l.Debug("获取流程详情成功",
		zap.Int("id", id),
		zap.String("name", process.Name))

	return process, nil
}

// GetProcessWithRelations 获取流程及其关联数据
func (p *processService) GetProcessWithRelations(ctx context.Context, id int) (*model.Process, error) {
	if id <= 0 {
		return nil, fmt.Errorf("流程ID无效")
	}

	process, err := p.dao.GetProcessWithRelations(ctx, id)
	if err != nil {
		p.l.Error("获取流程关联数据失败",
			zap.Error(err),
			zap.Int("id", id))
		return nil, fmt.Errorf("获取流程关联数据失败: %w", err)
	}

	// 获取创建者信息
	if process.CreatorID > 0 && process.CreatorName == "" {
		if user, err := p.userDao.GetUserByID(ctx, process.CreatorID); err != nil {
			p.l.Warn("获取创建者信息失败",
				zap.Error(err),
				zap.Int("creatorID", process.CreatorID))
		} else {
			process.CreatorName = user.Username
		}
	}

	p.l.Debug("获取流程关联数据成功",
		zap.Int("id", id),
		zap.String("name", process.Name))

	return process, nil
}

// PublishProcess 发布流程
func (p *processService) PublishProcess(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("流程ID无效")
	}

	// 获取流程详情
	process, err := p.getProcessFromCacheOrDB(ctx, id)
	if err != nil {
		return fmt.Errorf("获取流程失败: %w", err)
	}

	// 检查流程状态
	if process.Status == 1 {
		return fmt.Errorf("流程已经发布")
	}

	if process.Status == 2 {
		return fmt.Errorf("流程已被禁用，请先启用后再发布")
	}

	// 验证流程定义
	if process.Definition == "" {
		return fmt.Errorf("流程定义为空，无法发布")
	}

	var definition model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &definition); err != nil {
		p.l.Error("解析流程定义失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	// 执行所有验证器
	for _, validator := range p.validators {
		if err := validator.Validate(ctx, &definition); err != nil {
			p.l.Error("流程定义验证失败",
				zap.Error(err),
				zap.Int("id", id),
				zap.String("validator", fmt.Sprintf("%T", validator)))
			return fmt.Errorf("流程定义验证失败: %w", err)
		}
	}

	// 发布流程
	if err := p.dao.PublishProcess(ctx, id); err != nil {
		p.l.Error("发布流程失败",
			zap.Error(err),
			zap.Int("id", id))
		return fmt.Errorf("发布流程失败: %w", err)
	}

	// 清除缓存
	p.invalidateCache(id)

	p.l.Info("发布流程成功",
		zap.Int("id", id),
		zap.String("name", process.Name))

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

	if creatorID <= 0 {
		return nil, fmt.Errorf("创建者ID无效")
	}

	// 检查新名称是否已存在
	exists, err := p.checkProcessNameExists(ctx, req.Name)
	if err != nil {
		p.l.Error("检查流程名称失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("检查流程名称失败: %w", err)
	}

	if exists {
		return nil, fmt.Errorf("流程名称已存在: %s", req.Name)
	}

	// 获取原流程
	originalProcess, err := p.getProcessFromCacheOrDB(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("获取原流程失败: %w", err)
	}

	// 执行克隆
	clonedProcess, err := p.dao.CloneProcess(ctx, req.ID, req.Name, creatorID)
	if err != nil {
		p.l.Error("克隆流程失败",
			zap.Error(err),
			zap.Int("originalID", req.ID),
			zap.String("newName", req.Name))
		return nil, fmt.Errorf("克隆流程失败: %w", err)
	}

	p.l.Info("克隆流程成功",
		zap.Int("originalID", req.ID),
		zap.Int("newID", clonedProcess.ID),
		zap.String("originalName", originalProcess.Name),
		zap.String("newName", req.Name))

	return clonedProcess, nil
}

// ValidateProcess 校验流程定义
func (p *processService) ValidateProcess(ctx context.Context, id int, userID int) (*model.ValidateProcessResp, error) {
	p.l.Info("开始校验流程",
		zap.Int("processID", id),
		zap.Int("userID", userID))

	resp := &model.ValidateProcessResp{
		IsValid: true,
		Errors:  make([]string, 0),
	}

	if id <= 0 {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程ID无效")
		return resp, nil
	}

	process, err := p.getProcessFromCacheOrDB(ctx, id)
	if err != nil {
		p.l.Error("校验流程失败：获取流程失败",
			zap.Error(err),
			zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("获取流程 (ID: %d) 失败: %s", id, err.Error()))
		return resp, nil
	}

	if process.Definition == "" {
		p.l.Warn("流程定义为空", zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, "流程定义为空")
		return resp, nil
	}

	var def model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &def); err != nil {
		p.l.Error("校验流程失败：解析流程定义JSON失败",
			zap.Error(err),
			zap.Int("processID", id))
		resp.IsValid = false
		resp.Errors = append(resp.Errors, fmt.Sprintf("解析流程定义JSON失败: %s", err.Error()))
		return resp, nil
	}

	// 使用DAO层的验证方法
	if err := p.dao.ValidateProcessDefinition(ctx, &def); err != nil {
		resp.IsValid = false
		resp.Errors = append(resp.Errors, err.Error())
	}

	// 执行所有注册的验证器
	for _, validator := range p.validators {
		if err := validator.Validate(ctx, &def); err != nil {
			resp.IsValid = false
			resp.Errors = append(resp.Errors, err.Error())
		}
	}

	p.l.Info("流程校验完成",
		zap.Int("processID", id),
		zap.Bool("isValid", resp.IsValid),
		zap.Strings("errors", resp.Errors))

	return resp, nil
}

// 辅助方法

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
		req.Size = 100 // 限制最大分页大小
	}
}

// processDefinition 处理流程定义
func (p *processService) processDefinition(ctx context.Context, definition *model.ProcessDefinition, process *model.Process) error {
	// 如果有流程定义，进行验证和序列化
	if len(definition.Steps) > 0 || len(definition.Connections) > 0 {
		// 执行所有验证器
		for _, validator := range p.validators {
			if err := validator.Validate(ctx, definition); err != nil {
				p.l.Error("流程定义验证失败",
					zap.Error(err),
					zap.String("validator", fmt.Sprintf("%T", validator)))
				return fmt.Errorf("流程定义验证失败: %w", err)
			}
		}

		definitionJSON, err := json.Marshal(definition)
		if err != nil {
			p.l.Error("序列化流程定义失败", zap.Error(err))
			return fmt.Errorf("序列化流程定义失败: %w", err)
		}
		process.Definition = string(definitionJSON)
	} else {
		// 如果没有定义，设置为空的JSON对象
		process.Definition = "{}"
	}

	return nil
}

// getProcessFromCacheOrDB 从缓存或数据库获取流程
func (p *processService) getProcessFromCacheOrDB(ctx context.Context, id int) (*model.Process, error) {
	// 先从缓存获取
	p.mu.RLock()
	if cachedProcess, ok := p.processCache[id]; ok {
		p.mu.RUnlock()
		return cachedProcess, nil
	}
	p.mu.RUnlock()

	// 从数据库获取
	process, err := p.dao.GetProcess(ctx, id)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	p.mu.Lock()
	p.processCache[id] = process
	p.mu.Unlock()

	// 设置缓存过期清理
	go func() {
		time.Sleep(p.cacheExpiration)
		p.invalidateCache(id)
	}()

	return process, nil
}

// invalidateCache 清除缓存
func (p *processService) invalidateCache(id int) {
	p.mu.Lock()
	delete(p.processCache, id)
	p.mu.Unlock()
}

// enrichProcessListWithCreators 批量获取创建者信息
func (p *processService) enrichProcessListWithCreators(ctx context.Context, processes []model.Process) error {
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

// validateFormDesignExists 验证表单设计是否存在
func (p *processService) validateFormDesignExists(ctx context.Context, formDesignID int) error {
	_, err := p.formDesignDao.GetFormDesign(ctx, formDesignID)
	if err != nil {
		p.l.Error("验证表单发生错误", zap.Error(err))
		return err
	}
	return nil
}

// validateCategoryExists 验证分类是否存在
func (p *processService) validateCategoryExists(ctx context.Context, categoryID int) error {
	// TODO: 实现分类存在性验证
	// 这里应该调用分类的DAO或Service来验证

	return nil
}

// defaultProcessValidator 默认流程验证器
type defaultProcessValidator struct{}

// Validate 实现验证逻辑
func (v *defaultProcessValidator) Validate(ctx context.Context, definition *model.ProcessDefinition) error {
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
		if step.Type == "" {
			return fmt.Errorf("步骤类型不能为空")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("步骤ID重复: %s", step.ID)
		}
		stepIDs[step.ID] = true

		switch step.Type {
		case "start":
			if hasStartStep {
				return fmt.Errorf("流程只能有一个开始步骤")
			}
			hasStartStep = true
		case "end":
			hasEndStep = true
		case "approval", "process":
			// 验证审批和处理步骤必须有执行人
			if len(step.Roles) == 0 && len(step.Users) == 0 {
				return fmt.Errorf("步骤 %s 必须指定执行角色或用户", step.Name)
			}
		}

		// 验证时间限制
		if step.TimeLimit != nil && *step.TimeLimit <= 0 {
			return fmt.Errorf("步骤 %s 的时间限制必须大于0", step.Name)
		}
	}

	if !hasStartStep {
		return fmt.Errorf("流程必须包含一个开始步骤")
	}
	if !hasEndStep {
		return fmt.Errorf("流程必须包含至少一个结束步骤")
	}

	// 验证连接
	connectionMap := make(map[string][]string)
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

		// 记录连接关系
		connectionMap[conn.From] = append(connectionMap[conn.From], conn.To)
	}

	// 验证流程的连通性
	if err := v.validateConnectivity(definition.Steps, connectionMap); err != nil {
		return err
	}

	// 验证变量
	variableNames := make(map[string]bool)
	for _, variable := range definition.Variables {
		if variable.Name == "" {
			return fmt.Errorf("变量名不能为空")
		}
		if variable.Type == "" {
			return fmt.Errorf("变量 %s 的类型不能为空", variable.Name)
		}
		if variableNames[variable.Name] {
			return fmt.Errorf("变量名重复: %s", variable.Name)
		}
		variableNames[variable.Name] = true
	}

	return nil
}

// validateConnectivity 验证流程的连通性
func (v *defaultProcessValidator) validateConnectivity(steps []model.ProcessStep, connectionMap map[string][]string) error {
	// 找到开始节点
	var startNode string
	for _, step := range steps {
		if step.Type == "start" {
			startNode = step.ID
			break
		}
	}

	// 从开始节点进行深度优先搜索
	visited := make(map[string]bool)
	stack := []string{startNode}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}
		visited[current] = true

		if nexts, ok := connectionMap[current]; ok {
			for _, next := range nexts {
				if !visited[next] {
					stack = append(stack, next)
				}
			}
		}
	}

	// 检查是否所有节点都可达
	for _, step := range steps {
		if !visited[step.ID] && step.Type != "end" {
			return fmt.Errorf("步骤 %s 不可达", step.Name)
		}
	}

	// 检查是否至少有一条路径到达结束节点
	hasPathToEnd := false
	for _, step := range steps {
		if step.Type == "end" && visited[step.ID] {
			hasPathToEnd = true
			break
		}
	}

	if !hasPathToEnd {
		return fmt.Errorf("没有路径可以到达结束步骤")
	}

	return nil
}
