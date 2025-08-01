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

package dao

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrCommentNilPointer = errors.New("评论对象为空")
)

type WorkorderInstanceCommentDAO interface {
	// 评论方法
	CreateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error)
	GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error)
}

type workorderInstanceCommentDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewWorkorderInstanceCommentDAO(db *gorm.DB, logger *zap.Logger) WorkorderInstanceCommentDAO {
	return &workorderInstanceCommentDAO{
		db:     db,
		logger: logger,
	}
}

// CreateInstanceComment 创建工单评论
func (d *workorderInstanceCommentDAO) CreateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error {
	if comment == nil {
		return ErrCommentNilPointer
	}

	if err := d.validateComment(comment); err != nil {
		return fmt.Errorf("评论验证失败: %w", err)
	}

	if err := d.db.WithContext(ctx).Create(comment).Error; err != nil {
		d.logger.Error("创建工单评论失败", zap.Error(err), zap.Int("instanceID", comment.InstanceID))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	d.logger.Info("创建工单评论成功", zap.Int("id", comment.ID), zap.Int("instanceID", comment.InstanceID))
	return nil
}

// GetInstanceComments 获取工单评论
func (d *workorderInstanceCommentDAO) GetInstanceComments(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error) {
	if instanceID <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var comments []model.WorkorderInstanceComment
	err := d.db.WithContext(ctx).
		Where("instance_id = ?", instanceID).
		Order("created_at ASC").
		Find(&comments).Error

	if err != nil {
		d.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comments, nil
}

// GetInstanceCommentsTree 获取工单评论树结构
func (d *workorderInstanceCommentDAO) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error) {
	comments, err := d.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	// 构建评论树结构
	return d.buildCommentTree(comments), nil
}

// validateComment 验证评论数据
func (d *workorderInstanceCommentDAO) validateComment(comment *model.WorkorderInstanceComment) error {
	if comment.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	if comment.CreateUserID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if strings.TrimSpace(comment.Content) == "" {
		return fmt.Errorf("评论内容不能为空")
	}
	return nil
}

// buildCommentTree 构建评论树结构
func (d *workorderInstanceCommentDAO) buildCommentTree(comments []model.WorkorderInstanceComment) []model.WorkorderInstanceComment {
	// 简化实现，实际应根据parent_id构建树结构
	return comments
}
