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

func (d *workorderInstanceCommentDAO) CreateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error {
	if comment == nil {
		return ErrCommentNilPointer
	}

	if err := d.validateComment(comment); err != nil {
		return fmt.Errorf("评论验证失败: %w", err)
	}

	if err := DBWithContext(ctx, d.db).Create(comment).Error; err != nil {
		d.logger.Error("创建工单评论失败", zap.Error(err), zap.Int("instanceID", comment.InstanceID))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	d.logger.Info("创建工单评论成功", zap.Int("id", comment.ID), zap.Int("instanceID", comment.InstanceID))
	return nil
}

func (d *workorderInstanceCommentDAO) GetInstanceComments(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
	if instanceID <= 0 {
		return nil, ErrInstanceInvalidID
	}

	var comments []*model.WorkorderInstanceComment
	err := DBWithContext(ctx, d.db).
		Where("instance_id = ? AND status = ?", instanceID, model.CommentStatusNormal).
		Order("created_at ASC, id ASC").
		Find(&comments).Error

	if err != nil {
		d.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comments, nil
}

func (d *workorderInstanceCommentDAO) UpdateInstanceComment(ctx context.Context, comment *model.WorkorderInstanceComment) error {
	if comment == nil || comment.ID <= 0 {
		return fmt.Errorf("评论ID无效")
	}

	result := DBWithContext(ctx, d.db).
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

func (d *workorderInstanceCommentDAO) DeleteInstanceComment(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("评论ID无效")
	}

	result := DBWithContext(ctx, d.db).Delete(&model.WorkorderInstanceComment{}, id)
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
	err := DBWithContext(ctx, d.db).
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

func (d *workorderInstanceCommentDAO) ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) ([]*model.WorkorderInstanceComment, int64, error) {
	var comments []*model.WorkorderInstanceComment
	var total int64

	req.Page, req.Size = ValidatePagination(req.Page, req.Size)

	db := DBWithContext(ctx, d.db).Model(&model.WorkorderInstanceComment{})

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

	if err := db.Count(&total).Error; err != nil {
		d.logger.Error("获取评论总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取评论总数失败: %w", err)
	}

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

func (d *workorderInstanceCommentDAO) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
	comments, err := d.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	return d.buildCommentTree(comments)
}

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

// buildCommentTree finishes descendants before copying a child value into its
// parent. Children deliberately remains a value slice in the public JSON model.
// Missing-parent components are not promoted to roots, preserving the existing
// filtered/orphan contract. Invalid cycles fail the read rather than hide part
// of the available history or recurse without a bound.
func (d *workorderInstanceCommentDAO) buildCommentTree(comments []*model.WorkorderInstanceComment) ([]*model.WorkorderInstanceComment, error) {
	if len(comments) == 0 {
		return comments, nil
	}
	type node struct {
		source    *model.WorkorderInstanceComment
		parent    *node
		children  []*node
		remaining int
		built     model.WorkorderInstanceComment
	}
	byID := make(map[int]*node, len(comments))
	ordered := make([]*node, 0, len(comments))
	roots := make([]*node, 0)
	for _, comment := range comments {
		if comment == nil || comment.ID <= 0 || byID[comment.ID] != nil {
			return nil, errors.New("评论关系无效")
		}
		current := &node{source: comment}
		byID[comment.ID] = current
		ordered = append(ordered, current)
	}
	for _, current := range ordered {
		parentID := current.source.ParentID
		if parentID == nil || *parentID == 0 {
			roots = append(roots, current)
			continue
		}
		if parent := byID[*parentID]; parent != nil {
			current.parent = parent
			parent.children = append(parent.children, current)
			parent.remaining++
		}
	}
	ready := make([]*node, 0, len(ordered))
	for _, current := range ordered {
		if current.remaining == 0 {
			ready = append(ready, current)
		}
	}
	for head := 0; head < len(ready); head++ {
		current := ready[head]
		current.built = *current.source
		current.built.Children = make([]model.WorkorderInstanceComment, 0, len(current.children))
		for _, child := range current.children {
			current.built.Children = append(current.built.Children, child.built)
		}
		if current.parent != nil {
			current.parent.remaining--
			if current.parent.remaining == 0 {
				ready = append(ready, current.parent)
			}
		}
	}
	if len(ready) != len(ordered) {
		return nil, errors.New("评论关系无效")
	}
	if len(roots) == 0 {
		return nil, nil
	}
	result := make([]*model.WorkorderInstanceComment, 0, len(roots))
	for _, root := range roots {
		result = append(result, &root.built)
	}
	return result, nil
}
