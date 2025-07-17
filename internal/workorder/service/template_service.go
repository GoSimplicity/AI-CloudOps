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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context, req *model.CreateTemplateReq, creatorID int, creatorName string) error
	UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq, userID int) error
	DeleteTemplate(ctx context.Context, id int, userID int) error
	ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error)
	DetailTemplate(ctx context.Context, id int, userID int) (*model.Template, error)
	CloneTemplate(ctx context.Context, req *model.CloneTemplateReq, creatorID int) (*model.Template, error)
}

type templateService struct {
	dao         dao.TemplateDAO
	userDao     userDao.UserDAO
	processDao  dao.ProcessDAO
	categoryDao dao.CategoryDAO
	instanceDao dao.InstanceDAO
	l           *zap.Logger
	validators  []TemplateValidator // 模板验证器列表
}

type TemplateValidator interface {
	Validate(ctx context.Context, template *model.Template) error
}

func NewTemplateService(dao dao.TemplateDAO, userDao userDao.UserDAO, processDao dao.ProcessDAO, categoryDao dao.CategoryDAO, instanceDao dao.InstanceDAO, l *zap.Logger) TemplateService {
	ts := &templateService{
		dao:         dao,
		userDao:     userDao,
		processDao:  processDao,
		categoryDao: categoryDao,
		instanceDao: instanceDao,
		l:           l,
		validators:  make([]TemplateValidator, 0),
	}

	// 注册默认验证器
	ts.RegisterValidator(&defaultTemplateValidator{})

	return ts
}

// RegisterValidator 注册模板验证器
func (t *templateService) RegisterValidator(validator TemplateValidator) {
	t.validators = append(t.validators, validator)
}

// CreateTemplate 创建模板
func (t *templateService) CreateTemplate(ctx context.Context, req *model.CreateTemplateReq, creatorID int, creatorName string) error {
	// 参数验证
	if req == nil {
		return fmt.Errorf("创建模板请求不能为空")
	}

	if creatorID <= 0 {
		return fmt.Errorf("创建者ID无效")
	}

	if creatorName == "" {
		return fmt.Errorf("创建者名称不能为空")
	}

	// 检查模板名称是否已存在
	exists, err := t.checkTemplateNameExists(ctx, req.Name)
	if err != nil {
		t.l.Error("检查模板名称失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("creatorID", creatorID))
		return fmt.Errorf("检查模板名称失败: %w", err)
	}

	if exists {
		return fmt.Errorf("模板名称已存在: %s", req.Name)
	}

	// 验证流程是否存在
	if err := t.validateProcessExists(ctx, req.ProcessID); err != nil {
		return err
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := t.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
		}
	}

	// 序列化默认值
	defaultValuesJSON, err := t.serializeDefaultValues(req.DefaultValues)
	if err != nil {
		t.l.Error("序列化默认值失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("默认值格式错误: %w", err)
	}

	// 构建模板对象
	template := &model.Template{
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		DefaultValues: defaultValuesJSON,
		Icon:          req.Icon,
		Status:        1, // 默认启用
		SortOrder:     req.SortOrder,
		CategoryID:    req.CategoryID,
		CreatorID:     creatorID,
		CreatorName:   creatorName,
	}

	// 执行验证器
	for _, validator := range t.validators {
		if err := validator.Validate(ctx, template); err != nil {
			t.l.Error("模板验证失败",
				zap.Error(err),
				zap.String("validator", fmt.Sprintf("%T", validator)))
			return fmt.Errorf("模板验证失败: %w", err)
		}
	}

	// 创建模板
	if err := t.dao.CreateTemplate(ctx, template); err != nil {
		t.l.Error("创建模板失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.Int("creatorID", creatorID))
		return fmt.Errorf("创建模板失败: %w", err)
	}

	t.l.Info("创建模板成功",
		zap.Int("id", template.ID),
		zap.String("name", req.Name),
		zap.Int("creatorID", creatorID))

	return nil
}

// UpdateTemplate 更新模板
func (t *templateService) UpdateTemplate(ctx context.Context, req *model.UpdateTemplateReq, userID int) error {
	if req == nil {
		return fmt.Errorf("更新模板请求不能为空")
	}

	if req.ID <= 0 {
		return fmt.Errorf("模板ID无效")
	}

	// 获取现有模板
	existingTemplate, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}

	// 检查操作权限
	if !t.hasPermissionToModify(existingTemplate, userID) {
		t.l.Warn("用户权限不足，无法修改模板",
			zap.Int("templateID", req.ID),
			zap.Int("userID", userID))
		return fmt.Errorf("没有权限修改此模板")
	}

	// 检查模板名称是否与其他模板重复
	if req.Name != existingTemplate.Name {
		exists, err := t.dao.IsTemplateNameExists(ctx, req.Name, req.ID)
		if err != nil {
			t.l.Error("检查模板名称失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("检查模板名称失败: %w", err)
		}
		if exists {
			return fmt.Errorf("模板名称已存在: %s", req.Name)
		}
	}

	// 验证流程是否存在
	if req.ProcessID != existingTemplate.ProcessID {
		if err := t.validateProcessExists(ctx, req.ProcessID); err != nil {
			return err
		}
	}

	// 验证分类是否存在
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if err := t.validateCategoryExists(ctx, *req.CategoryID); err != nil {
			return err
		}
	}

	// 序列化默认值
	defaultValuesJSON, err := t.serializeDefaultValues(req.DefaultValues)
	if err != nil {
		t.l.Error("序列化默认值失败",
			zap.Error(err),
			zap.Int("id", req.ID))
		return fmt.Errorf("默认值格式错误: %w", err)
	}

	// 构建更新的模板对象
	template := &model.Template{
		Model:         model.Model{ID: req.ID},
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		DefaultValues: defaultValuesJSON,
		Icon:          req.Icon,
		Status:        req.Status,
		SortOrder:     req.SortOrder,
		CategoryID:    req.CategoryID,
		CreatorID:     existingTemplate.CreatorID,
		CreatorName:   existingTemplate.CreatorName,
	}

	// 执行验证器
	for _, validator := range t.validators {
		if err := validator.Validate(ctx, template); err != nil {
			t.l.Error("模板验证失败",
				zap.Error(err),
				zap.String("validator", fmt.Sprintf("%T", validator)))
			return fmt.Errorf("模板验证失败: %w", err)
		}
	}

	// 更新模板
	if err := t.dao.UpdateTemplate(ctx, template); err != nil {
		t.l.Error("更新模板失败",
			zap.Error(err),
			zap.Int("id", req.ID))
		return fmt.Errorf("更新模板失败: %w", err)
	}

	t.l.Info("更新模板成功",
		zap.Int("id", req.ID),
		zap.String("name", req.Name))

	return nil
}

// DeleteTemplate 删除模板
func (t *templateService) DeleteTemplate(ctx context.Context, id int, userID int) error {
	if id <= 0 {
		return fmt.Errorf("模板ID无效")
	}

	// 获取模板信息用于日志记录
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}

	// 检查操作权限
	if !t.hasPermissionToModify(template, userID) {
		t.l.Warn("用户权限不足，无法删除模板",
			zap.Int("templateID", id),
			zap.Int("userID", userID))
		return fmt.Errorf("没有权限删除此模板")
	}

	// 检查是否有关联的工单
	instances, err := t.instanceDao.GetInstanceByTemplateID(ctx, id)
	if err != nil {
		t.l.Error("获取关联工单失败",
			zap.Error(err),
			zap.Int("templateID", id))
		return fmt.Errorf("获取关联工单失败: %w", err)
	}

	if len(instances) > 0 {
		t.l.Warn("模板有关联的工单，无法删除",
			zap.Int("templateID", id),
			zap.Int("instanceCount", len(instances)))
		return fmt.Errorf("模板有关联的工单，无法删除")
	}

	// 执行删除
	if err := t.dao.DeleteTemplate(ctx, id); err != nil {
		t.l.Error("删除模板失败",
			zap.Error(err),
			zap.Int("id", id),
			zap.String("name", template.Name))
		return fmt.Errorf("删除模板失败: %w", err)
	}

	t.l.Info("删除模板成功",
		zap.Int("id", id),
		zap.String("name", template.Name))

	return nil
}

// ListTemplate 获取模板列表
func (t *templateService) ListTemplate(ctx context.Context, req *model.ListTemplateReq) (*model.ListResp[*model.Template], error) {
	if req == nil {
		req = &model.ListTemplateReq{}
	}

	// 设置默认分页参数
	t.setDefaultPagination(req)

	// 记录查询参数
	t.l.Debug("查询模板列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.Any("filters", map[string]interface{}{
			"name":       req.Search,
			"categoryID": req.CategoryID,
			"processID":  req.ProcessID,
			"status":     req.Status,
		}))

	result, err := t.dao.ListTemplate(ctx, req)
	if err != nil {
		t.l.Error("获取模板列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取模板列表失败: %w", err)
	}

	// 批量获取创建者信息
	if err := t.enrichTemplateListWithCreators(ctx, result.Items); err != nil {
		t.l.Warn("获取创建者信息失败", zap.Error(err))
		// 不影响主流程，继续返回结果
	}

	t.l.Info("获取模板列表成功",
		zap.Int("count", len(result.Items)),
		zap.Int64("total", result.Total))

	return result, nil
}

// DetailTemplate 获取模板详情
func (t *templateService) DetailTemplate(ctx context.Context, id int, userID int) (*model.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("模板ID无效")
	}

	// 从数据库获取
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("获取模板详情失败",
			zap.Error(err),
			zap.Int("id", id),
			zap.Int("userID", userID))
		return nil, fmt.Errorf("获取模板详情失败: %w", err)
	}

	// 获取创建者信息
	if template.CreatorID > 0 && template.CreatorName == "" {
		if user, err := t.userDao.GetUserByID(ctx, template.CreatorID); err != nil {
			t.l.Warn("获取创建者信息失败",
				zap.Error(err),
				zap.Int("creatorID", template.CreatorID))
		} else {
			template.CreatorName = user.Username
		}
	}

	// 解析默认值以便前端使用
	if template.DefaultValues != "" && template.DefaultValues != "{}" {
		var defaultValues model.TemplateDefaultValues
		if err := json.Unmarshal([]byte(template.DefaultValues), &defaultValues); err != nil {
			t.l.Warn("解析模板默认值失败",
				zap.Error(err),
				zap.Int("id", id))
		}
	}

	t.l.Debug("获取模板详情成功",
		zap.Int("id", id),
		zap.String("name", template.Name))

	return template, nil
}

// CloneTemplate 克隆模板
func (t *templateService) CloneTemplate(ctx context.Context, req *model.CloneTemplateReq, creatorID int) (*model.Template, error) {
	// 参数验证
	if req == nil || req.ID <= 0 {
		return nil, fmt.Errorf("克隆模板请求无效")
	}

	if creatorID <= 0 {
		return nil, fmt.Errorf("创建者ID无效")
	}

	// 获取原模板
	originalTemplate, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		t.l.Error("获取原模板失败",
			zap.Error(err),
			zap.Int("templateID", req.ID))
		return nil, fmt.Errorf("获取原模板失败: %w", err)
	}

	// 生成新名称
	newName := req.Name

	// 检查新名称是否已存在
	exists, err := t.checkTemplateNameExists(ctx, newName, req.ID)
	if err != nil {
		t.l.Error("检查模板名称失败",
			zap.Error(err),
			zap.String("name", newName))
		return nil, fmt.Errorf("检查模板名称失败: %w", err)
	}

	if exists {
		// 如果名称已存在，添加时间戳
		newName = fmt.Sprintf("%s - 副本 (%s)", newName,
			time.Now().Format("20060102150405"))
	}

	// 创建新模板（复制原模板的数据）
	newTemplate := &model.Template{
		Name:          newName,
		Description:   originalTemplate.Description,
		ProcessID:     originalTemplate.ProcessID,
		DefaultValues: originalTemplate.DefaultValues,
		Icon:          originalTemplate.Icon,
		Status:        originalTemplate.Status,
		SortOrder:     originalTemplate.SortOrder,
		CategoryID:    originalTemplate.CategoryID,
		CreatorID:     creatorID,
	}

	// 保存新模板到数据库
	if err := t.dao.CreateTemplate(ctx, newTemplate); err != nil {
		t.l.Error("保存克隆模板失败",
			zap.Error(err),
			zap.Int("originalID", req.ID),
			zap.String("newName", newName))
		return nil, fmt.Errorf("保存克隆模板失败: %w", err)
	}

	t.l.Info("克隆模板成功",
		zap.Int("originalID", req.ID),
		zap.Int("newID", newTemplate.ID),
		zap.String("originalName", originalTemplate.Name),
		zap.String("newName", newName))

	return newTemplate, nil
}

// checkTemplateNameExists 检查模板名称是否存在
func (t *templateService) checkTemplateNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, fmt.Errorf("模板名称不能为空")
	}

	var id int
	if len(excludeID) > 0 {
		id = excludeID[0]
	}

	return t.dao.IsTemplateNameExists(ctx, name, id)
}

// setDefaultPagination 设置默认分页参数
func (t *templateService) setDefaultPagination(req *model.ListTemplateReq) {
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

// serializeDefaultValues 序列化默认值
func (t *templateService) serializeDefaultValues(defaultValues model.TemplateDefaultValues) (string, error) {
	if defaultValues.Fields == nil && len(defaultValues.Approvers) == 0 && defaultValues.Priority == 0 && defaultValues.DueHours == nil {
		return "{}", nil
	}

	data, err := json.Marshal(defaultValues)
	if err != nil {
		return "", fmt.Errorf("序列化默认值失败: %w", err)
	}

	return string(data), nil
}

// enrichTemplateListWithCreators 批量获取创建者信息
func (t *templateService) enrichTemplateListWithCreators(ctx context.Context, templates []*model.Template) error {
	// 收集所有创建者ID
	creatorIDs := make([]int, 0)
	creatorIDMap := make(map[int]bool)

	for _, template := range templates {
		if template.CreatorID > 0 && !creatorIDMap[template.CreatorID] {
			creatorIDs = append(creatorIDs, template.CreatorID)
			creatorIDMap[template.CreatorID] = true
		}
	}

	if len(creatorIDs) == 0 {
		return nil
	}

	// 批量获取用户信息
	users, err := t.userDao.GetUserByIDs(ctx, creatorIDs)
	if err != nil {
		return err
	}

	// 构建用户ID到名称的映射
	userMap := make(map[int]string)
	for _, user := range users {
		userMap[user.ID] = user.Username
	}

	// 填充创建者名称
	for i := range templates {
		if name, ok := userMap[templates[i].CreatorID]; ok {
			templates[i].CreatorName = name
		}
	}

	return nil
}

// hasPermissionToModify 检查是否有权限修改模板
func (t *templateService) hasPermissionToModify(_ *model.Template, userID int) bool {
	return userID == 1
}

// validateProcessExists 验证流程是否存在
func (t *templateService) validateProcessExists(ctx context.Context, processID int) error {
	_, err := t.processDao.GetProcess(ctx, processID)
	if err != nil {
		t.l.Error("验证流程发生错误", zap.Error(err))
		return fmt.Errorf("关联的流程不存在或无效")
	}
	return nil
}

// validateCategoryExists 验证分类是否存在
func (t *templateService) validateCategoryExists(ctx context.Context, categoryID int) error {
	_, err := t.categoryDao.GetCategory(ctx, categoryID)
	if err != nil {
		t.l.Error("验证分类发生错误", zap.Error(err))
		return fmt.Errorf("关联的分类不存在或无效")
	}
	return nil
}

// defaultTemplateValidator 默认模板验证器
type defaultTemplateValidator struct{}

// Validate 实现验证逻辑
func (v *defaultTemplateValidator) Validate(ctx context.Context, template *model.Template) error {
	if template == nil {
		return fmt.Errorf("模板不能为空")
	}

	if template.Name == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	if len(template.Name) > 100 {
		return fmt.Errorf("模板名称长度不能超过100个字符")
	}

	if template.ProcessID <= 0 {
		return fmt.Errorf("模板必须关联一个有效的流程")
	}

	if template.Status < 0 || template.Status > 1 {
		return fmt.Errorf("模板状态无效")
	}

	// 验证默认值格式
	if template.DefaultValues != "" && template.DefaultValues != "{}" {
		var defaultValues model.TemplateDefaultValues
		if err := json.Unmarshal([]byte(template.DefaultValues), &defaultValues); err != nil {
			return fmt.Errorf("默认值格式错误: %w", err)
		}

		// 验证优先级
		if defaultValues.Priority < 0 || defaultValues.Priority > 3 {
			return fmt.Errorf("默认优先级值无效，必须在0-3之间")
		}

		// 验证截止时间
		if defaultValues.DueHours != nil && *defaultValues.DueHours <= 0 {
			return fmt.Errorf("默认截止时间必须大于0")
		}
	}

	return nil
}
