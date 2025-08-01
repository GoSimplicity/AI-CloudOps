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
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context, req *model.CreateWorkorderTemplateReq, creatorID int, creatorName string) error
	UpdateTemplate(ctx context.Context, req *model.UpdateWorkorderTemplateReq, userID int) error
	DeleteTemplate(ctx context.Context, id int, userID int) error
	ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) (*model.ListResp[*model.WorkorderTemplate], error)
	DetailTemplate(ctx context.Context, id int, userID int) (*model.WorkorderTemplate, error)
}

type templateService struct {
	dao         dao.TemplateDAO
	userDao     userDao.UserDAO
	processDao  dao.ProcessDAO
	categoryDao dao.WorkorderCategoryDAO
	instanceDao dao.WorkorderInstanceDAO
	l           *zap.Logger
	validators  []TemplateValidator // 模板验证器列表
}

type TemplateValidator interface {
	Validate(ctx context.Context, template *model.WorkorderTemplate) error
}

func NewTemplateService(
	dao dao.TemplateDAO,
	userDao userDao.UserDAO,
	processDao dao.ProcessDAO,
	categoryDao dao.WorkorderCategoryDAO,
	instanceDao dao.WorkorderInstanceDAO,
	l *zap.Logger,
) TemplateService {
	ts := &templateService{
		dao:         dao,
		userDao:     userDao,
		processDao:  processDao,
		categoryDao: categoryDao,
		instanceDao: instanceDao,
		l:           l,
		validators:  make([]TemplateValidator, 0),
	}
	ts.RegisterValidator(&defaultTemplateValidator{})
	return ts
}

// RegisterValidator 注册模板验证器
func (t *templateService) RegisterValidator(validator TemplateValidator) {
	t.validators = append(t.validators, validator)
}

// CreateTemplate 创建模板
func (t *templateService) CreateTemplate(ctx context.Context, req *model.CreateWorkorderTemplateReq, creatorID int, creatorName string) error {
	if req == nil {
		return errors.New("创建模板请求不能为空")
	}
	if creatorID <= 0 {
		return errors.New("创建者ID无效")
	}
	if creatorName == "" {
		return errors.New("创建者名称不能为空")
	}
	// 检查模板名称是否已存在
	exists, err := t.checkTemplateNameExists(ctx, req.Name)
	if err != nil {
		t.l.Error("检查模板名称失败", zap.Error(err), zap.String("name", req.Name), zap.Int("creatorID", creatorID))
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
	// 构建模板对象
	template := &model.WorkorderTemplate{
		Name:           req.Name,
		Description:    req.Description,
		ProcessID:      req.ProcessID,
		DefaultValues:  req.DefaultValues,
		Status:         1, // 默认启用
		CategoryID:     req.CategoryID,
		CreateUserID:   creatorID,
		CreateUserName: creatorName,
	}
	// 执行验证器
	for _, validator := range t.validators {
		if err := validator.Validate(ctx, template); err != nil {
			t.l.Error("模板验证失败", zap.Error(err), zap.String("validator", fmt.Sprintf("%T", validator)))
			return fmt.Errorf("模板验证失败: %w", err)
		}
	}
	// 创建模板
	if err := t.dao.CreateTemplate(ctx, template); err != nil {
		t.l.Error("创建模板失败", zap.Error(err), zap.String("name", req.Name), zap.Int("creatorID", creatorID))
		return fmt.Errorf("创建模板失败: %w", err)
	}
	t.l.Info("创建模板成功", zap.Int("id", template.ID), zap.String("name", req.Name), zap.Int("creatorID", creatorID))
	return nil
}

// UpdateTemplate 更新模板
func (t *templateService) UpdateTemplate(ctx context.Context, req *model.UpdateWorkorderTemplateReq, userID int) error {
	if req == nil {
		return errors.New("更新模板请求不能为空")
	}
	if req.ID <= 0 {
		return errors.New("模板ID无效")
	}
	// 获取现有模板
	existingTemplate, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}
	// 检查操作权限
	if !t.hasPermissionToModify(existingTemplate, userID) {
		t.l.Warn("用户权限不足，无法修改模板", zap.Int("templateID", req.ID), zap.Int("userID", userID))
		return errors.New("没有权限修改此模板")
	}
	// 检查模板名称是否与其他模板重复
	if req.Name != existingTemplate.Name {
		exists, err := t.dao.IsTemplateNameExists(ctx, req.Name, req.ID)
		if err != nil {
			t.l.Error("检查模板名称失败", zap.Error(err), zap.String("name", req.Name))
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
	// 构建更新的模板对象
	template := &model.WorkorderTemplate{
		Model:          model.Model{ID: req.ID},
		Name:           req.Name,
		Description:    req.Description,
		ProcessID:      req.ProcessID,
		DefaultValues:  req.DefaultValues,
		Status:         req.Status,
		CategoryID:     req.CategoryID,
		CreateUserID:   existingTemplate.CreateUserID,
		CreateUserName: existingTemplate.CreateUserName,
	}
	// 执行验证器
	for _, validator := range t.validators {
		if err := validator.Validate(ctx, template); err != nil {
			t.l.Error("模板验证失败", zap.Error(err), zap.String("validator", fmt.Sprintf("%T", validator)))
			return fmt.Errorf("模板验证失败: %w", err)
		}
	}
	// 更新模板
	if err := t.dao.UpdateTemplate(ctx, template); err != nil {
		t.l.Error("更新模板失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新模板失败: %w", err)
	}
	t.l.Info("更新模板成功", zap.Int("id", req.ID), zap.String("name", req.Name))
	return nil
}

// DeleteTemplate 删除模板
func (t *templateService) DeleteTemplate(ctx context.Context, id int, userID int) error {
	if id <= 0 {
		return errors.New("模板ID无效")
	}
	// 获取模板信息用于日志记录
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		return fmt.Errorf("获取模板失败: %w", err)
	}
	// 检查操作权限
	if !t.hasPermissionToModify(template, userID) {
		t.l.Warn("用户权限不足，无法删除模板", zap.Int("templateID", id), zap.Int("userID", userID))
		return errors.New("没有权限删除此模板")
	}
	// 检查是否有关联的工单
	instances, _, err := t.instanceDao.ListInstance(ctx, &model.ListWorkorderInstanceReq{
		ProcessID: &template.ProcessID,
	})
	if err != nil {
		t.l.Error("获取关联工单失败", zap.Error(err), zap.Int("templateID", id))
		return fmt.Errorf("获取关联工单失败: %w", err)
	}
	if len(instances) > 0 {
		t.l.Warn("模板有关联的工单，无法删除", zap.Int("templateID", id), zap.Int("instanceCount", len(instances)))
		return errors.New("模板有关联的工单，无法删除")
	}
	// 执行删除
	if err := t.dao.DeleteTemplate(ctx, id); err != nil {
		t.l.Error("删除模板失败", zap.Error(err), zap.Int("id", id), zap.String("name", template.Name))
		return fmt.Errorf("删除模板失败: %w", err)
	}
	t.l.Info("删除模板成功", zap.Int("id", id), zap.String("name", template.Name))
	return nil
}

// ListTemplate 获取模板列表
func (t *templateService) ListTemplate(ctx context.Context, req *model.ListWorkorderTemplateReq) (*model.ListResp[*model.WorkorderTemplate], error) {
	if req == nil {
		req = &model.ListWorkorderTemplateReq{}
	}
	t.setDefaultPagination(req)
	t.l.Debug("查询模板列表",
		zap.Int("page", req.Page),
		zap.Int("size", req.Size),
		zap.Any("filters", map[string]interface{}{
			"name":       req.Search,
			"categoryID": req.CategoryID,
			"processID":  req.ProcessID,
			"status":     req.Status,
		}),
	)
	result, err := t.dao.ListTemplate(ctx, req)
	if err != nil {
		t.l.Error("获取模板列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取模板列表失败: %w", err)
	}
	// 批量获取创建者信息
	if err := t.enrichTemplateListWithCreators(ctx, result.Items); err != nil {
		t.l.Warn("获取创建者信息失败", zap.Error(err))
	}
	t.l.Info("获取模板列表成功", zap.Int("count", len(result.Items)), zap.Int64("total", result.Total))
	return result, nil
}

// DetailTemplate 获取模板详情
func (t *templateService) DetailTemplate(ctx context.Context, id int, userID int) (*model.WorkorderTemplate, error) {
	if id <= 0 {
		return nil, errors.New("模板ID无效")
	}
	template, err := t.dao.GetTemplate(ctx, id)
	if err != nil {
		t.l.Error("获取模板详情失败", zap.Error(err), zap.Int("id", id), zap.Int("userID", userID))
		return nil, fmt.Errorf("获取模板详情失败: %w", err)
	}
	// 获取创建者信息
	if template.CreateUserID > 0 && template.CreateUserName == "" {
		if user, err := t.userDao.GetUserByID(ctx, template.CreateUserID); err != nil {
			t.l.Warn("获取创建者信息失败", zap.Error(err), zap.Int("creatorID", template.CreateUserID))
		} else {
			template.CreateUserName = user.Username
		}
	}
	// 解析默认值以便前端使用
	if template.DefaultValues != nil && len(template.DefaultValues) > 0 {
		var defaultValues model.JSONMap
		if err := json.Unmarshal([]byte(template.DefaultValues), &defaultValues); err != nil {
			t.l.Warn("解析模板默认值失败", zap.Error(err), zap.Int("id", id))
		}
	}
	t.l.Debug("获取模板详情成功", zap.Int("id", id), zap.String("name", template.Name))
	return template, nil
}

// checkTemplateNameExists 检查模板名称是否存在
func (t *templateService) checkTemplateNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if name == "" {
		return false, errors.New("模板名称不能为空")
	}
	var id int
	if len(excludeID) > 0 {
		id = excludeID[0]
	}
	return t.dao.IsTemplateNameExists(ctx, name, id)
}

// setDefaultPagination 设置默认分页参数
func (t *templateService) setDefaultPagination(req *model.ListWorkorderTemplateReq) {
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

// enrichTemplateListWithCreators 批量获取创建者信息
func (t *templateService) enrichTemplateListWithCreators(ctx context.Context, templates []*model.WorkorderTemplate) error {
	creatorIDs := make([]int, 0)
	creatorIDMap := make(map[int]struct{})
	for _, template := range templates {
		if template.CreateUserID > 0 {
			if _, ok := creatorIDMap[template.CreateUserID]; !ok {
				creatorIDs = append(creatorIDs, template.CreateUserID)
				creatorIDMap[template.CreateUserID] = struct{}{}
			}
		}
	}
	if len(creatorIDs) == 0 {
		return nil
	}
	users, err := t.userDao.GetUserByIDs(ctx, creatorIDs)
	if err != nil {
		return err
	}
	userMap := make(map[int]string)
	for _, user := range users {
		userMap[user.ID] = user.Username
	}
	for i := range templates {
		if name, ok := userMap[templates[i].CreateUserID]; ok {
			templates[i].CreateUserName = name
		}
	}
	return nil
}

// hasPermissionToModify 检查是否有权限修改模板
func (t *templateService) hasPermissionToModify(template *model.WorkorderTemplate, userID int) bool {
	// 优化：允许创建者本人或超级管理员（如ID=1）修改
	if template == nil {
		return false
	}
	return userID == 1 || userID == template.CreateUserID
}

// validateProcessExists 验证流程是否存在
func (t *templateService) validateProcessExists(ctx context.Context, processID int) error {
	if processID <= 0 {
		return errors.New("流程ID无效")
	}
	_, err := t.processDao.GetProcessByID(ctx, processID)
	if err != nil {
		t.l.Error("验证流程发生错误", zap.Error(err))
		return errors.New("关联的流程不存在或无效")
	}
	return nil
}

// validateCategoryExists 验证分类是否存在
func (t *templateService) validateCategoryExists(ctx context.Context, categoryID int) error {
	if categoryID <= 0 {
		return errors.New("分类ID无效")
	}
	_, err := t.categoryDao.GetCategory(ctx, categoryID)
	if err != nil {
		t.l.Error("验证分类发生错误", zap.Error(err))
		return errors.New("关联的分类不存在或无效")
	}
	return nil
}

// defaultTemplateValidator 默认模板验证器
type defaultTemplateValidator struct{}

// Validate 实现验证逻辑
func (v *defaultTemplateValidator) Validate(ctx context.Context, template *model.WorkorderTemplate) error {
	if template == nil {
		return errors.New("模板不能为空")
	}
	if template.Name == "" {
		return errors.New("模板名称不能为空")
	}
	if len(template.Name) > 100 {
		return errors.New("模板名称长度不能超过100个字符")
	}
	if template.ProcessID <= 0 {
		return errors.New("模板必须关联一个有效的流程")
	}
	if template.Status < 0 || template.Status > 1 {
		return errors.New("模板状态无效")
	}
	// 验证默认值格式
	if template.DefaultValues != nil && len(template.DefaultValues) > 0 {
		var defaultValues model.JSONMap
		if err := json.Unmarshal([]byte(template.DefaultValues), &defaultValues); err != nil {
			return fmt.Errorf("默认值格式错误: %w", err)
		}
		// 验证优先级
		if val, ok := defaultValues["priority"]; ok && val != nil {
			priority, ok := val.(float64)
			if !ok {
				return errors.New("默认优先级值类型无效")
			}
			if priority < 0 || priority > 3 {
				return errors.New("默认优先级值无效，必须在0-3之间")
			}
		}
		// 验证截止时间
		if val, ok := defaultValues["due_hours"]; ok && val != nil {
			dueHours, ok := val.(float64)
			if !ok {
				return errors.New("默认截止时间类型无效")
			}
			if dueHours <= 0 {
				return errors.New("默认截止时间必须大于0")
			}
		}
	}
	return nil
}
