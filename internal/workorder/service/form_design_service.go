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
	"regexp"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"

	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

// 状态常量
const (
	FormDesignStatusDraft     int8 = 1 // 草稿状态
	FormDesignStatusPublished int8 = 2 // 已发布状态
	FormDesignStatusDisabled  int8 = 3 // 已禁用状态
)

// 支持的字段类型
var supportedFieldTypes = map[string]bool{
	"text":        true, // 文本
	"textarea":    true, // 文本域
	"number":      true, // 数字
	"email":       true, // 邮箱
	"password":    true, // 密码
	"date":        true, // 日期
	"datetime":    true, // 日期时间
	"time":        true, // 时间
	"url":         true, // 链接
	"tel":         true, // 电话
	"select":      true, // 选择
	"radio":       true, // 单选
	"checkbox":    true, // 多选
	"multiselect": true, // 多选
	"file":        true, // 文件
	"image":       true, // 图片
	"switch":      true, // 开关
	"slider":      true, // 滑块
	"rate":        true, // 评分
	"color":       true, // 颜色
}

// 需要选项的字段类型
var optionFieldTypes = map[string]bool{
	"select":      true, // 下拉框
	"radio":       true, // 单选
	"checkbox":    true, // 复选框
	"multiselect": true, // 多选
}

// 字段名称验证正则
var fieldNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesignReq *model.CreateFormDesignReq) error
	UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateFormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int) error
	PublishFormDesign(ctx context.Context, id int) error
	CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesign, error)
	DetailFormDesign(ctx context.Context, id int) (*model.FormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (model.ListResp[*model.FormDesign], error)
	PreviewFormDesign(ctx context.Context, id int, userID int) (*model.FormDesign, error)
	CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error)
	GetFormStatistics(ctx context.Context) (*model.FormStatistics, error)
}

type formDesignService struct {
	dao         dao.FormDesignDAO
	userDao     userDao.UserDAO
	categoryDao dao.CategoryDAO
	l           *zap.Logger
}

func NewFormDesignService(dao dao.FormDesignDAO, userDao userDao.UserDAO, categoryDao dao.CategoryDAO, l *zap.Logger) FormDesignService {
	return &formDesignService{
		dao:         dao,
		userDao:     userDao,
		categoryDao: categoryDao,
		l:           l,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignService) CreateFormDesign(ctx context.Context, formDesignReq *model.CreateFormDesignReq) error {
	if formDesignReq == nil {
		return fmt.Errorf("表单设计请求不能为空")
	}

	// 验证名称
	if err := f.validateName(formDesignReq.Name); err != nil {
		return err
	}

	// 检查名称唯一性
	exists, err := f.dao.CheckFormDesignNameExists(ctx, formDesignReq.Name)
	if err != nil {
		f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", formDesignReq.Name))
		return fmt.Errorf("检查表单设计名称失败: %w", err)
	}
	if exists {
		f.l.Warn("表单设计名称已存在", zap.String("name", formDesignReq.Name))
		return dao.ErrFormDesignNameExists
	}

	// 验证表单结构
	if err := f.validateFormSchema(&formDesignReq.Schema); err != nil {
		f.l.Error("表单结构验证失败", zap.Error(err))
		return fmt.Errorf("表单结构验证失败: %w", err)
	}

	schemaJSON, err := json.Marshal(formDesignReq.Schema)
	if err != nil {
		f.l.Error("表单结构序列化失败", zap.Error(err))
		return fmt.Errorf("表单结构序列化失败: %w", err)
	}

	// 构建表单设计实体
	formDesign := &model.FormDesign{
		Name:        strings.TrimSpace(formDesignReq.Name),
		Description: strings.TrimSpace(formDesignReq.Description),
		Schema:      schemaJSON,
		Version:     1,
		Status:      FormDesignStatusDraft,
		CategoryID:  formDesignReq.CategoryID,
		CreatorID:   formDesignReq.UserID,
		CreatorName: formDesignReq.UserName,
	}

	// 创建表单设计
	if err := f.dao.CreateFormDesign(ctx, formDesign); err != nil {
		f.l.Error("创建表单设计失败", zap.Error(err), zap.String("name", formDesignReq.Name))
		return err
	}

	f.l.Info("创建表单设计成功", zap.Int("id", formDesign.ID), zap.String("name", formDesignReq.Name))
	return nil
}

// UpdateFormDesign 更新表单设计
func (f *formDesignService) UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateFormDesignReq) error {
	// 验证名称
	if err := f.validateName(formDesignReq.Name); err != nil {
		return err
	}

	// 检查表单设计是否存在
	existingFormDesign, err := f.dao.GetFormDesign(ctx, formDesignReq.ID)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	// 检查名称唯一性（排除当前记录）
	if formDesignReq.Name != existingFormDesign.Name {
		exists, err := f.dao.CheckFormDesignNameExists(ctx, formDesignReq.Name, formDesignReq.ID)
		if err != nil {
			f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", formDesignReq.Name))
			return fmt.Errorf("检查表单设计名称失败: %w", err)
		}
		if exists {
			f.l.Warn("表单设计名称已存在", zap.String("name", formDesignReq.Name))
			return dao.ErrFormDesignNameExists
		}
	}

	// 验证表单结构
	if err := f.validateFormSchema(&formDesignReq.Schema); err != nil {
		f.l.Error("表单结构验证失败", zap.Error(err))
		return fmt.Errorf("表单结构验证失败: %w", err)
	}

	schemaJSON, err := json.Marshal(formDesignReq.Schema)
	if err != nil {
		f.l.Error("表单结构序列化失败", zap.Error(err))
		return fmt.Errorf("表单结构序列化失败: %w", err)
	}

	// 构建更新数据
	formDesign := &model.FormDesign{
		Model:       model.Model{ID: formDesignReq.ID},
		Name:        strings.TrimSpace(formDesignReq.Name),
		Description: strings.TrimSpace(formDesignReq.Description),
		Schema:      schemaJSON,
		CategoryID:  formDesignReq.CategoryID,
		Status:      formDesignReq.Status,
		Version:     formDesignReq.Version,
	}

	// 更新表单设计
	if err := f.dao.UpdateFormDesign(ctx, formDesign); err != nil {
		f.l.Error("更新表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	return nil
}

// DeleteFormDesign 删除表单设计
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	// 检查表单设计是否存在
	_, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 删除表单设计
	if err := f.dao.DeleteFormDesign(ctx, id); err != nil {
		f.l.Error("删除表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// PublishFormDesign 发布表单设计
func (f *formDesignService) PublishFormDesign(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("表单设计ID无效")
	}

	// 检查表单设计是否存在
	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 检查状态是否为草稿
	if formDesign.Status != FormDesignStatusDraft {
		f.l.Warn("表单设计状态不是草稿，无法发布",
			zap.Int("id", id),
			zap.Int8("status", formDesign.Status),
			zap.Int8("expectedStatus", FormDesignStatusDraft))
		return dao.ErrFormDesignCannotPublish
	}

	schemaObj := &model.FormSchema{}
	if err := json.Unmarshal(formDesign.Schema, schemaObj); err != nil {
		f.l.Error("表单结构反序列化失败", zap.Error(err))
		return fmt.Errorf("表单结构反序列化失败: %w", err)
	}

	// 验证表单结构完整性
	if err := f.validateFormSchema(schemaObj); err != nil {
		f.l.Error("表单结构验证失败，无法发布", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("表单结构验证失败，无法发布: %w", err)
	}

	// 发布表单设计
	if err := f.dao.PublishFormDesign(ctx, id); err != nil {
		f.l.Error("发布表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	return nil
}

// CloneFormDesign 克隆表单设计
func (f *formDesignService) CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesign, error) {
	if id <= 0 {
		return nil, fmt.Errorf("原始表单设计ID无效")
	}

	if creatorID <= 0 {
		return nil, fmt.Errorf("创建者ID无效")
	}

	// 验证名称
	if err := f.validateName(name); err != nil {
		return nil, err
	}

	// 检查新名称是否已存在
	exists, err := f.dao.CheckFormDesignNameExists(ctx, name)
	if err != nil {
		f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", name))
		return nil, fmt.Errorf("检查表单设计名称失败: %w", err)
	}
	if exists {
		f.l.Warn("表单设计名称已存在", zap.String("name", name))
		return nil, dao.ErrFormDesignNameExists
	}

	// 验证创建者是否存在
	_, err = f.userDao.GetUserByID(ctx, creatorID)
	if err != nil {
		f.l.Error("获取创建者用户信息失败", zap.Error(err), zap.Int("creatorID", creatorID))
		return nil, fmt.Errorf("创建者用户不存在: %w", err)
	}

	// 克隆表单设计
	clonedFormDesign, err := f.dao.CloneFormDesign(ctx, id, name, creatorID)
	if err != nil {
		f.l.Error("克隆表单设计失败", zap.Error(err), zap.Int("originalID", id), zap.String("newName", name))
		return nil, err
	}

	return clonedFormDesign, nil
}

// DetailFormDesign 获取表单设计详情
func (f *formDesignService) DetailFormDesign(ctx context.Context, id int) (*model.FormDesign, error) {
	if id <= 0 {
		return nil, fmt.Errorf("表单设计ID无效")
	}

	// 获取表单设计
	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	// 获取创建者名称
	user, err := f.userDao.GetUserByID(ctx, formDesign.CreatorID)
	if err != nil {
		f.l.Warn("获取创建者用户信息失败", zap.Error(err), zap.Int("creatorID", formDesign.CreatorID))
		formDesign.CreatorName = "未知用户"
	} else {
		formDesign.CreatorName = user.Username
	}

	return formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (model.ListResp[*model.FormDesign], error) {
	// 验证状态参数
	if req.Status != nil && !f.isValidStatus(*req.Status) {
		return model.ListResp[*model.FormDesign]{}, fmt.Errorf("无效的状态值: %d，有效范围：%d-%d",
			*req.Status, FormDesignStatusDraft, FormDesignStatusDisabled)
	}

	// 获取表单设计列表
	formDesigns, total, err := f.dao.ListFormDesign(ctx, req)
	if err != nil {
		f.l.Error("获取表单设计列表失败", zap.Error(err))
		return model.ListResp[*model.FormDesign]{}, err
	}

	if len(formDesigns) == 0 {
		return model.ListResp[*model.FormDesign]{
			Items: formDesigns,
			Total: total,
		}, nil
	}

	// 使用map去重收集创建者ID
	creatorIDSet := make(map[int]bool, len(formDesigns))
	categoryIDs := make([]int, 0, len(formDesigns))

	// 一次遍历同时收集创建者ID和分类ID
	for _, formDesign := range formDesigns {
		if formDesign != nil {
			if !creatorIDSet[formDesign.CreatorID] {
				creatorIDSet[formDesign.CreatorID] = true
			}

			if formDesign.CategoryID != nil {
				categoryIDs = append(categoryIDs, *formDesign.CategoryID)
			}
		}
	}

	// 转换创建者ID为切片
	creatorIDs := make([]int, 0, len(creatorIDSet))
	for id := range creatorIDSet {
		creatorIDs = append(creatorIDs, id)
	}

	// 批量获取用户信息
	userMap := make(map[int]string)
	if len(creatorIDs) > 0 {
		users, err := f.userDao.GetUserByIDs(ctx, creatorIDs)
		if err != nil {
			f.l.Error("获取用户信息失败", zap.Error(err))
			return model.ListResp[*model.FormDesign]{}, err
		}
		for _, user := range users {
			userMap[user.ID] = user.Username
		}
	}

	// 批量获取分类信息
	categoryMap := make(map[int]string)
	if len(categoryIDs) > 0 {
		categories, err := f.categoryDao.GetCategoryByIDs(ctx, categoryIDs)
		if err != nil {
			f.l.Error("获取分类信息失败", zap.Error(err))
			return model.ListResp[*model.FormDesign]{}, err
		}
		for _, category := range categories {
			categoryMap[category.ID] = category.Name
		}
	}

	// 设置创建者名称和分类名称
	for _, formDesign := range formDesigns {
		if formDesign != nil {
			if username, exists := userMap[formDesign.CreatorID]; exists {
				formDesign.CreatorName = username
			} else {
				formDesign.CreatorName = "未知用户"
			}

			if formDesign.CategoryID != nil {
				if categoryName, exists := categoryMap[*formDesign.CategoryID]; exists {
					formDesign.CategoryName = categoryName
				} else {
					formDesign.CategoryName = "未知分类"
				}
			}
		}
	}

	return model.ListResp[*model.FormDesign]{
		Items: formDesigns,
		Total: total,
	}, nil
}

// PreviewFormDesign 预览表单设计
func (f *formDesignService) PreviewFormDesign(ctx context.Context, id int, userID int) (*model.FormDesign, error) {
	if id <= 0 {
		return nil, fmt.Errorf("表单设计ID无效")
	}

	if userID <= 0 {
		return nil, fmt.Errorf("用户ID无效")
	}

	// 检查表单设计是否存在
	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	// 验证表单结构
	schemaObj := &model.FormSchema{}
	if err := json.Unmarshal(formDesign.Schema, schemaObj); err != nil {
		f.l.Error("表单结构反序列化失败", zap.Error(err))
		return nil, fmt.Errorf("表单结构反序列化失败: %w", err)
	}

	if err := f.validateFormSchema(schemaObj); err != nil {
		return nil, fmt.Errorf("表单结构验证失败: %w", err)
	}

	return formDesign, nil
}

// CheckFormDesignNameExists 检查表单设计名称是否存在
func (f *formDesignService) CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	if err := f.validateName(name); err != nil {
		return false, err
	}

	exists, err := f.dao.CheckFormDesignNameExists(ctx, name, excludeID...)
	if err != nil {
		f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", name))
		return false, err
	}

	return exists, nil
}

func (f *formDesignService) GetFormStatistics(ctx context.Context) (*model.FormStatistics, error) {
	statistics, err := f.dao.GetFormStatistics(ctx)
	if err != nil {
		f.l.Error("获取表单设计统计信息失败", zap.Error(err))
		return nil, err
	}

	return statistics, nil
}

// validateName 验证名称
func (f *formDesignService) validateName(name string) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return fmt.Errorf("表单设计名称不能为空")
	}
	if len(trimmedName) > 100 {
		return fmt.Errorf("表单设计名称长度不能超过100个字符")
	}
	return nil
}

// validateFormSchema 验证表单结构
func (f *formDesignService) validateFormSchema(schema *model.FormSchema) error {
	if schema == nil {
		return fmt.Errorf("表单结构不能为空")
	}

	if len(schema.Fields) == 0 {
		return fmt.Errorf("表单必须至少包含一个字段")
	}

	// 检查字段名称唯一性
	fieldNames := make(map[string]bool)

	// 验证每个字段
	for i, field := range schema.Fields {
		// 验证必填字段
		if field.Type == "" {
			return fmt.Errorf("第%d个字段类型不能为空", i+1)
		}
		if field.Label == "" {
			return fmt.Errorf("第%d个字段标签不能为空", i+1)
		}
		if field.Name == "" {
			return fmt.Errorf("第%d个字段名称不能为空", i+1)
		}

		// 验证字段类型
		if !supportedFieldTypes[field.Type] {
			return fmt.Errorf("不支持的字段类型: %s", field.Type)
		}

		// 验证字段名称格式
		if !fieldNameRegex.MatchString(field.Name) {
			return fmt.Errorf("字段名称格式无效: %s（只允许字母、数字、下划线，且不能以数字开头）", field.Name)
		}

		// 检查字段名称唯一性
		if fieldNames[field.Name] {
			return fmt.Errorf("字段名称重复: %s", field.Name)
		}
		fieldNames[field.Name] = true

		// 验证选项类型字段
		if optionFieldTypes[field.Type] {
			if len(field.Options) == 0 {
				return fmt.Errorf("选项类型字段必须包含选项: %s", field.Name)
			}

			// 验证选项内容
			optionValues := make(map[interface{}]bool)
			for j, option := range field.Options {
				if option.Label == "" {
					return fmt.Errorf("字段 %s 的第%d个选项标签不能为空", field.Name, j+1)
				}
				if option.Value == nil {
					return fmt.Errorf("字段 %s 的第%d个选项值不能为空", field.Name, j+1)
				}
				// 检查选项值唯一性
				if optionValues[option.Value] {
					return fmt.Errorf("字段 %s 的选项值重复: %v", field.Name, option.Value)
				}
				optionValues[option.Value] = true
			}
		}

		// 验证数值范围
		if field.Validation.Min != nil && field.Validation.Max != nil {
			if *field.Validation.Min > *field.Validation.Max {
				return fmt.Errorf("字段 %s 的最小值不能大于最大值", field.Name)
			}
		}

		// 验证长度范围
		if field.Validation.MinLength != nil && field.Validation.MaxLength != nil {
			if *field.Validation.MinLength > *field.Validation.MaxLength {
				return fmt.Errorf("字段 %s 的最小长度不能大于最大长度", field.Name)
			}
		}

		// 验证正则表达式
		if field.Validation.Pattern != "" {
			if _, err := regexp.Compile(field.Validation.Pattern); err != nil {
				return fmt.Errorf("字段 %s 的正则表达式无效: %w", field.Name, err)
			}
		}
	}

	return nil
}

// isValidStatus 验证状态值是否有效
func (f *formDesignService) isValidStatus(status int8) bool {
	return status >= FormDesignStatusDraft && status <= FormDesignStatusDisabled
}

// GetFormDesignStatusText 获取状态文本描述
func GetFormDesignStatusText(status int8) string {
	switch status {
	case FormDesignStatusDraft:
		return "草稿"
	case FormDesignStatusPublished:
		return "已发布"
	case FormDesignStatusDisabled:
		return "已禁用	"
	default:
		return "未知状态"
	}
}

// IsValidFormDesignStatus 验证状态值是否有效
func IsValidFormDesignStatus(status int8) bool {
	return status >= FormDesignStatusDraft && status <= FormDesignStatusDisabled
}
