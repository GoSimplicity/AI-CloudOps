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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrCommentNilPointer = errors.New("评论对象为空")
)

type WorkorderInstanceCommentDAO interface {
	CreateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error
	UpdateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error
	DeleteInstanceComment(ctx context.Context, id int) error
	GetInstanceCommentByID(ctx context.Context, id int) (*model.WorkorderInstanceComment, error)
	GetInstanceComments(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error)
	GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error)
	ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) ([]*model.WorkorderInstanceComment, int64, error)
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
func (d *workorderInstanceCommentDAO) GetInstanceComments(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
	if instanceID <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var comments []*model.WorkorderInstanceComment
	err := d.db.WithContext(ctx).
		Where("instance_id = ? AND status = ?", instanceID, model.CommentStatusNormal).
		Order("created_at ASC").
		Find(&comments).Error

	if err != nil {
		d.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comments, nil
}

// UpdateInstanceComment 更新工单评论
func (d *workorderInstanceCommentDAO) UpdateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error {
	if comment == nil || comment.ID <= 0 {
		return fmt.Errorf("评论ID无效")
	}

	result := d.db.WithContext(ctx).
		Model(&model.WorkorderInstanceComment{}).
		Where("id = ?", comment.ID).
		Updates(map[string]any{
			"content":   comment.Content,
			"status":    comment.Status,
			"is_system": comment.IsSystem,
		})

	if result.Error != nil {
		d.logger.Error("更新工单评论失败", zap.Error(result.Error), zap.Int("id", comment.ID))
		return fmt.Errorf("更新工单评论失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("评论不存在")
	}

	return nil
}

// DeleteInstanceComment 删除工单评论
func (d *workorderInstanceCommentDAO) DeleteInstanceComment(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("评论ID无效")
	}

	result := d.db.WithContext(ctx).Delete(&model.WorkorderInstanceComment{}, id)
	if result.Error != nil {
		d.logger.Error("删除工单评论失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除工单评论失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("评论不存在")
	}

	return nil
}

// GetInstanceCommentByID 根据ID获取工单评论
func (d *workorderInstanceCommentDAO) GetInstanceCommentByID(ctx context.Context, id int) (*model.WorkorderInstanceComment, error) {
	if id <= 0 {
		return nil, fmt.Errorf("评论ID无效")
	}

	var comment model.WorkorderInstanceComment
	err := d.db.WithContext(ctx).
		Where("id = ?", id).
		First(&comment).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("评论不存在")
		}
		d.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return &comment, nil
}

// ListInstanceComments 分页获取工单评论列表
func (d *workorderInstanceCommentDAO) ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) ([]*model.WorkorderInstanceComment, int64, error) {
	var comments []*model.WorkorderInstanceComment
	var total int64

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := d.db.WithContext(ctx).Model(&model.WorkorderInstanceComment{})

	// 构建查询条件
	if req.InstanceID != nil {
		db = db.Where("instance_id = ?", *req.InstanceID)
	}
	if req.Type != nil {
		db = db.Where("type = ?", *req.Type)
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if req.Search != "" {
		searchTerm := sanitizeSearchInput(req.Search)
		db = db.Where("content LIKE ?", "%"+searchTerm+"%")
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取评论总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取评论总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	err := db.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&comments).Error

	if err != nil {
		d.logger.Error("获取评论列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取评论列表失败: %w", err)
	}

	return comments, total, nil
}

// GetInstanceCommentsTree 获取工单评论树结构
func (d *workorderInstanceCommentDAO) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
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
	if comment.OperatorID <= 0 {
		return fmt.Errorf("用户ID无效")
	}
	if comment.Content == "" {
		return fmt.Errorf("评论内容不能为空")
	}
	return nil
}

// buildCommentTree 构建评论树结构
func (d *workorderInstanceCommentDAO) buildCommentTree(comments []*model.WorkorderInstanceComment) []*model.WorkorderInstanceComment {
	if len(comments) == 0 {
		return comments
	}

	// 创建ID到评论的映射
	commentMap := make(map[int]*model.WorkorderInstanceComment)
	var rootComments []*model.WorkorderInstanceComment

	// 初始化映射表和根评论列表
	for _, comment := range comments {
		commentMap[comment.ID] = comment
		comment.Children = make([]model.WorkorderInstanceComment, 0)
	}

	// 构建树形结构
	for _, comment := range comments {
		if comment.ParentID == nil || *comment.ParentID == 0 {
			// 根评论
			rootComments = append(rootComments, comment)
		} else {
			// 子评论
			if parent, exists := commentMap[*comment.ParentID]; exists {
				parent.Children = append(parent.Children, *comment)
			}
		}
	}

	return rootComments
}
