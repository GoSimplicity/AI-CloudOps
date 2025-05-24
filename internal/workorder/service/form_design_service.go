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

	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesignReq *model.CreateFormDesignReq, creatorID int, creatorName string) error
	UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateFormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int) error
	PublishFormDesign(ctx context.Context, id int) error // Corrected typo PublishFormDescrollern
	CloneFormDesign(ctx context.Context, id int, name string) error
	DetailFormDesign(ctx context.Context, id int, userId int) (*model.FormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error)
	PreviewFormDesign(ctx context.Context, id int, schema model.FormSchema, userID int) error // Signature updated as per subtask
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
	formDesign, err := utils.ConvertCreateFormDesignReqToModel(formDesignReq)
	if err != nil {
		f.l.Error("转换创建表单设计请求失败", zap.Error(err))
		return err
	}
	formDesign.CreatorID = creatorID
	formDesign.CreatorName = creatorName // This is gorm:"-" so it's for response, not DB storage.
	                                     // Actual CreatorName might be fetched during Get/List operations.
	formDesign.Status = 0 // Default to draft
	formDesign.Version = 1 // Default to version 1

	return f.dao.CreateFormDesign(ctx, formDesign)
}

// UpdateFormDesign 更新表单设计
func (f *formDesignService) UpdateFormDesign(ctx context.Context, formDesignReq *model.UpdateFormDesignReq) error {
	formDesign, err := utils.ConvertUpdateFormDesignReqToModel(formDesignReq)
	if err != nil {
		f.l.Error("转换更新表单设计请求失败", zap.Error(err))
		return err
	}
	// Note: Version increment logic might be needed here or in DAO if status changes from non-published to published.
	// Status changes (e.g., to published) are handled by PublishFormDesign.
	// Here, we assume the request might carry a status or it's an update to a draft.
	// The ConvertUpdateFormDesignReqToModel doesn't set status/version, so they'd be zero-valued if not in req.
	// The DAO update only updates specific fields, so if formDesign.Status is 0, it won't update it unless explicitly set.
	// If formDesignReq had Status and Version fields, they would be mapped in the converter.
	// Current UpdateFormDesignReq does not have Status or Version.
	// So, this update will primarily update Name, Description, Schema, CategoryID.
	// Version and Status are managed by other operations like Publish or Clone.

	return f.dao.UpdateFormDesign(ctx, formDesign)
}

// DeleteFormDesign 删除表单设计
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int) error {
	return f.dao.DeleteFormDesign(ctx, id)
}

// PublishFormDesign 发布表单设计
func (f *formDesignService) PublishFormDesign(ctx context.Context, id int) error {
	return f.dao.PublishFormDesign(ctx, id)
}

// CloneFormDesign 克隆表单设计
func (f *formDesignService) CloneFormDesign(ctx context.Context, id int, name string) error {
	return f.dao.CloneFormDesign(ctx, id, name)
}

// DetailFormDesign 获取表单设计详情
func (f *formDesignService) DetailFormDesign(ctx context.Context, id int, userId int) (*model.FormDesign, error) {
	// 根据userid查询中文名称
	user, err := f.userDao.GetUserByID(ctx, userId)
	if err != nil {
		f.l.Error("获取用户失败", zap.Error(err))
		return nil, err
	}

	formDesign, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		f.l.Error("获取表单设计失败", zap.Error(err))
		return nil, err
	}

	formDesign.CreatorName = user.Username

	return formDesign, nil
}

// ListFormDesign 获取表单设计列表
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error) {
	return f.dao.ListFormDesign(ctx, req)
}

// PreviewFormDesign 预览表单设计
// For now, this is a placeholder. Actual preview might involve more complex logic.
func (f *formDesignService) PreviewFormDesign(ctx context.Context, id int, schema model.FormSchema, userID int) error {
	f.l.Info("开始预览表单设计",
		zap.Int("formDesignID", id),
		zap.Any("schema", schema),
		zap.Int("userID", userID))

	// Placeholder implementation:
	// In a real scenario, this might involve:
	// 1. Validating the schema.
	// 2. Potentially storing a temporary preview version or rendering it.
	// 3. Checking user permissions (hence userID).
	// For this subtask, we just log and return nil.
	f.l.Info("表单设计预览功能占位实现完成。")
	return nil
}
