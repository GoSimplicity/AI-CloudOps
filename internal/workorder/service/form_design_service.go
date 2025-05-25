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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesignReq *model.CreateFormDesignReq, creatorID int, creatorName string) error
	UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateFormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int) error
	PublishFormDesign(ctx context.Context, id int) error
	CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesign, error)
	DetailFormDesign(ctx context.Context, id int) (*model.FormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (*model.ListResp[model.FormDesign], error)
	PreviewFormDesign(ctx context.Context, id int, schema model.FormSchema, userID int) error
	CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error)
}

type formDesignService struct {
	dao     dao.FormDesignDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewFormDesignService(dao dao.FormDesignDAO, userDao userDao.UserDAO, l *zap.Logger) FormDesignService {
	return &formDesignService{
		dao:     dao,
		userDao: userDao,
		l:       l,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignService) CreateFormDesign(ctx context.Context, formDesignReq *model.CreateFormDesignReq, creatorID int, creatorName string) error {
	// 检查表单设计名称是否已存在
	exists, err := f.dao.CheckFormDesignNameExists(ctx, formDesignReq.Name)
	if err != nil {
		f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", formDesignReq.Name))
		return fmt.Errorf("检查表单设计名称失败: %w", err)
	}
	if exists {
		f.l.Warn("表单设计名称已存在", zap.String("name", formDesignReq.Name))
		return dao.ErrFormDesignNameExists
	}

	// 转换请求模型为数据模型
	formDesign, err := utils.ConvertCreateFormDesignReqToModel(formDesignReq)
	if err != nil {
		f.l.Error("转换创建表单设计请求失败", zap.Error(err))
		return fmt.Errorf("转换请求数据失败: %w", err)
	}

	// 设置创建者信息
	formDesign.CreatorID = creatorID
	formDesign.Status = 0  // 默认为草稿状态
	formDesign.Version = 1 // 默认版本为1

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
	// 检查表单设计是否存在
	existingFormDesign, err := f.dao.GetFormDesign(ctx, formDesignReq.ID)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	// 检查名称是否与其他记录重复（排除当前记录）
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

	// 转换请求模型为数据模型
	formDesign, err := utils.ConvertUpdateFormDesignReqToModel(formDesignReq)
	if err != nil {
		f.l.Error("转换更新表单设计请求失败", zap.Error(err))
		return fmt.Errorf("转换请求数据失败: %w", err)
	}

	// 更新表单设计
	if err := f.dao.UpdateFormDesign(ctx, formDesign); err != nil {
		f.l.Error("更新表单设计失败", zap.Error(err), zap.Int("id", formDesignReq.ID))
		return err
	}

	f.l.Info("更新表单设计成功", zap.Int("id", formDesignReq.ID), zap.String("name", formDesignReq.Name))
	return nil
}

// DeleteFormDesign 删除表单设计
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int) error {
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

	f.l.Info("删除表单设计成功", zap.Int("id", id))
	return nil
}

// PublishFormDesign 发布表单设计
func (f *formDesignService) PublishFormDesign(ctx context.Context, id int) error {
	// 检查表单设计是否存在
	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 检查状态是否为草稿
	if formDesign.Status != 0 {
		f.l.Warn("表单设计状态不是草稿，无法发布", zap.Int("id", id), zap.Int8("status", formDesign.Status))
		return dao.ErrFormDesignCannotPublish
	}

	// 发布表单设计
	if err := f.dao.PublishFormDesign(ctx, id); err != nil {
		f.l.Error("发布表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	f.l.Info("发布表单设计成功", zap.Int("id", id))
	return nil
}

// CloneFormDesign 克隆表单设计
func (f *formDesignService) CloneFormDesign(ctx context.Context, id int, name string, creatorID int) (*model.FormDesign, error) {
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

	// 克隆表单设计
	clonedFormDesign, err := f.dao.CloneFormDesign(ctx, id, name, creatorID)
	if err != nil {
		f.l.Error("克隆表单设计失败", zap.Error(err), zap.Int("originalID", id), zap.String("newName", name))
		return nil, err
	}

	// 获取创建者名称
	user, err := f.userDao.GetUserByID(ctx, creatorID)
	if err != nil {
		f.l.Warn("获取创建者用户信息失败", zap.Error(err), zap.Int("creatorID", creatorID))
		// 不阻断流程，仅记录警告
	} else {
		clonedFormDesign.CreatorName = user.Username
	}

	f.l.Info("克隆表单设计成功",
		zap.Int("originalID", id),
		zap.Int("newID", clonedFormDesign.ID),
		zap.String("newName", name))

	return clonedFormDesign, nil
}

// DetailFormDesign 获取表单设计详情
func (f *formDesignService) DetailFormDesign(ctx context.Context, id int) (*model.FormDesign, error) {
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
		// 不阻断流程，使用默认值
		formDesign.CreatorName = "未知用户"
	} else {
		formDesign.CreatorName = user.Username
	}

	f.l.Debug("获取表单设计详情成功", zap.Int("id", id), zap.String("name", formDesign.Name))
	return formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) (*model.ListResp[model.FormDesign], error) {
	// 参数验证
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}
	if req.Size > 100 {
		req.Size = 100 // 限制最大页面大小
	}

	// 获取表单设计列表
	result, err := f.dao.ListFormDesign(ctx, req)
	if err != nil {
		f.l.Error("获取表单设计列表失败", zap.Error(err))
		return nil, err
	}

	// 批量获取创建者信息
	creatorIDs := make([]int, 0, len(result.Items))
	creatorIDMap := make(map[int]bool)

	for _, formDesign := range result.Items {
		if !creatorIDMap[formDesign.CreatorID] {
			creatorIDs = append(creatorIDs, formDesign.CreatorID)
			creatorIDMap[formDesign.CreatorID] = true
		}
	}

	// 获取用户信息映射
	userMap := make(map[int]string)
	if len(creatorIDs) > 0 {
		users, err := f.userDao.GetUserByIDs(ctx, creatorIDs)
		if err != nil {
			f.l.Warn("批量获取用户信息失败", zap.Error(err))
		} else {
			for _, user := range users {
				userMap[user.ID] = user.Username
			}
		}
	}

	// 设置创建者名称
	for i := range result.Items {
		if username, exists := userMap[result.Items[i].CreatorID]; exists {
			result.Items[i].CreatorName = username
		} else {
			result.Items[i].CreatorName = "未知用户"
		}
	}

	f.l.Debug("获取表单设计列表成功",
		zap.Int("count", len(result.Items)),
		zap.Int64("total", result.Total))

	return result, nil
}

// PreviewFormDesign 预览表单设计
func (f *formDesignService) PreviewFormDesign(ctx context.Context, id int, schema model.FormSchema, userID int) error {
	// 检查表单设计是否存在
	_, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 检查用户是否存在
	_, err = f.userDao.GetUserByID(ctx, userID)
	if err != nil {
		f.l.Error("获取用户失败", zap.Error(err), zap.Int("userID", userID))
		return fmt.Errorf("用户不存在: %w", err)
	}

	f.l.Info("预览表单设计",
		zap.Int("formDesignID", id),
		zap.Int("userID", userID))

	// TODO: 实现预览逻辑
	// 1. 验证 schema 格式
	// 2. 生成预览数据
	// 3. 可能需要临时存储预览状态
	// 4. 记录预览操作日志

	return nil
}

// CheckFormDesignNameExists 检查表单设计名称是否存在
func (f *formDesignService) CheckFormDesignNameExists(ctx context.Context, name string, excludeID ...int) (bool, error) {
	exists, err := f.dao.CheckFormDesignNameExists(ctx, name, excludeID...)
	if err != nil {
		f.l.Error("检查表单设计名称是否存在失败", zap.Error(err), zap.String("name", name))
		return false, err
	}

	return exists, nil
}
