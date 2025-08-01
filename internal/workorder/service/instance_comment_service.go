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
	// 添加评论
	CommentInstance(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq, creatorID int, creatorName string) error
	// 获取评论树
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error)
}

type instanceCommentService struct {
	dao         dao.WorkorderInstanceCommentDAO
	instanceDao dao.WorkorderInstanceDAO
	logger      *zap.Logger
}

func NewInstanceCommentService(commentDAO dao.WorkorderInstanceCommentDAO, instanceDao dao.WorkorderInstanceDAO, logger *zap.Logger) InstanceCommentService {
	return &instanceCommentService{
		dao:         commentDAO,
		instanceDao: instanceDao,
		logger:      logger,
	}
}

// CommentInstance 添加工单评论
func (s *instanceCommentService) CommentInstance(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq, creatorID int, creatorName string) error {
	// 参数校验
	if err := s.validateCommentRequest(req); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 校验工单实例是否存在
	_, err := s.instanceDao.GetInstanceByID(ctx, req.InstanceID)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceNotFound) {
			s.logger.Warn("工单实例不存在", zap.Int("instanceID", req.InstanceID))
			return ErrInstanceNotFound
		}
		s.logger.Error("获取工单实例失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("获取工单实例失败: %w", err)
	}

	comment := &model.WorkorderInstanceComment{
		InstanceID:     req.InstanceID,
		Content:        strings.TrimSpace(req.Content),
		CreateUserID:   creatorID,
		CreateUserName: creatorName,
		ParentID:       req.ParentID,
		IsSystem:       false,
	}

	if err := s.dao.CreateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("创建工单评论失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	s.logger.Info("创建工单评论成功",
		zap.Int("instanceID", req.InstanceID),
		zap.Int("creatorID", creatorID),
		zap.Intp("parentID", req.ParentID),
	)

	return nil
}

// GetInstanceComments 获取工单评论树
func (s *instanceCommentService) GetInstanceComments(ctx context.Context, instanceID int) ([]model.WorkorderInstanceComment, error) {
	if instanceID <= 0 {
		s.logger.Warn("获取工单评论失败，工单ID无效", zap.Int("instanceID", instanceID))
		return nil, ErrInvalidRequest
	}

	comments, err := s.dao.GetInstanceComments(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	// 构建评论树
	return buildCommentTree(comments, nil), nil
}

// buildCommentTree 构建评论树结构
func buildCommentTree(comments []model.WorkorderInstanceComment, parentID *int) []model.WorkorderInstanceComment {
	var tree []model.WorkorderInstanceComment
	for i := range comments {
		comment := &comments[i]
		if (parentID == nil && comment.ParentID == nil) ||
			(parentID != nil && comment.ParentID != nil && *parentID == *comment.ParentID) {

			children := buildCommentTree(comments, &comment.ID)
			c := *comment
			c.Children = children
			tree = append(tree, c)
		}
	}
	return tree
}

// validateCommentRequest 验证评论请求
func (s *instanceCommentService) validateCommentRequest(req *model.CreateWorkorderInstanceCommentReq) error {
	if req == nil {
		return ErrInvalidRequest
	}
	if req.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return fmt.Errorf("评论内容不能为空")
	}
	if len(content) > 2000 {
		return fmt.Errorf("评论内容超过最大长度限制(%d)", 2000)
	}
	return nil
}
