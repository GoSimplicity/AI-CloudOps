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
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type InstanceCommentService interface {
	// 评论功能
	CommentInstance(ctx context.Context, req *model.InstanceCommentReq, creatorID int, creatorName string) error
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error)
}

type instanceCommentService struct {
	dao         dao.InstanceCommentDAO
	instanceDao dao.InstanceDAO
	logger      *zap.Logger
}

func NewInstanceCommentService(dao dao.InstanceCommentDAO, instanceDao dao.InstanceDAO, logger *zap.Logger) InstanceCommentService {
	return &instanceCommentService{
		dao:         dao,
		instanceDao: instanceDao,
		logger:      logger,
	}
}

// CommentInstance 添加工单评论
func (s *instanceCommentService) CommentInstance(ctx context.Context, req *model.InstanceCommentReq, creatorID int, creatorName string) error {
	if err := s.validateCommentRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 验证工单是否存在
	if _, err := s.instanceDao.GetInstance(ctx, req.InstanceID); err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			return ErrInstanceNotFound
		}
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	comment := &model.InstanceComment{
		InstanceID:  req.InstanceID,
		Content:     strings.TrimSpace(req.Content),
		UserID:      creatorID,
		CreatorName: creatorName,
		ParentID:    req.ParentID,
		IsSystem:    false,
	}

	if err := s.dao.CreateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("创建工单评论失败", zap.Error(err))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	s.logger.Info("创建工单评论成功",
		zap.Int("instanceID", req.InstanceID),
		zap.Int("creatorID", creatorID))

	return nil
}

// GetInstanceComments 获取工单评论
func (s *instanceCommentService) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error) {
	if instanceID <= 0 {
		return nil, ErrInvalidRequest
	}

	comments, err := s.dao.GetInstanceComments(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	// 构建评论树
	return s.buildCommentTree(comments, nil), nil
}

// buildCommentTree 构建评论树
func (s *instanceCommentService) buildCommentTree(comments []model.InstanceComment, parentID *int) []model.InstanceCommentResp {
	tree := make([]model.InstanceCommentResp, 0)

	for _, comment := range comments {
		// 检查是否为当前层级的评论
		if (parentID == nil && comment.ParentID == nil) ||
			(parentID != nil && comment.ParentID != nil && *parentID == *comment.ParentID) {

			children := s.buildCommentTree(comments, &comment.ID)

			respComment := model.InstanceCommentResp{
				ID:          comment.ID,
				InstanceID:  comment.InstanceID,
				Content:     comment.Content,
				UserID:      comment.UserID,
				CreatorName: comment.CreatorName,
				ParentID:    comment.ParentID,
				IsSystem:    comment.IsSystem,
				CreatedAt:   comment.CreatedAt,
				Children:    children,
			}

			tree = append(tree, respComment)
		}
	}

	return tree
}

// validateCommentRequest 验证评论请求
func (s *instanceCommentService) validateCommentRequest(req *model.InstanceCommentReq) error {
	if req == nil {
		return ErrInvalidRequest
	}

	if req.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}

	if strings.TrimSpace(req.Content) == "" {
		return fmt.Errorf("评论内容不能为空")
	}

	if len(strings.TrimSpace(req.Content)) > MaxCommentLength {
		return fmt.Errorf("评论内容超过最大长度限制(%d)", MaxCommentLength)
	}

	return nil
}
