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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type InstanceCommentService interface {
	CreateInstanceComment(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq) error
	UpdateInstanceComment(ctx context.Context, req *model.UpdateWorkorderInstanceCommentReq, userID int) error
	DeleteInstanceComment(ctx context.Context, id int, userID int) error
	GetInstanceComment(ctx context.Context, id int) (*model.WorkorderInstanceComment, error)
	ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) (*model.ListResp[*model.WorkorderInstanceComment], error)
	GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error)
}

type instanceCommentService struct {
	dao                 dao.WorkorderInstanceCommentDAO
	instanceDao         dao.WorkorderInstanceDAO
	notificationService WorkorderNotificationService
	logger              *zap.Logger
}

func NewInstanceCommentService(
	dao dao.WorkorderInstanceCommentDAO,
	instanceDao dao.WorkorderInstanceDAO,
	notificationService WorkorderNotificationService,
	logger *zap.Logger,
) InstanceCommentService {
	return &instanceCommentService{
		dao:                 dao,
		instanceDao:         instanceDao,
		notificationService: notificationService,
		logger:              logger,
	}
}

func (s *instanceCommentService) CreateInstanceComment(ctx context.Context, req *model.CreateWorkorderInstanceCommentReq) error {
	if req == nil {
		return fmt.Errorf("评论请求不能为空")
	}
	if s.notificationService != nil && req.IsSystem != 1 && !dao.InTransaction(ctx) {
		store, ok := s.instanceDao.(instanceEventStore)
		if !ok {
			return fmt.Errorf("工单通知事务存储不可用")
		}
		return store.InTransaction(ctx, func(txCtx context.Context) error { return s.CreateInstanceComment(txCtx, req) })
	}
	instance, err := s.instanceDao.GetInstanceByID(ctx, req.InstanceID)
	if err != nil {
		s.logger.Error("工单不存在", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("工单不存在: %w", err)
	}

	// 验证父评论是否存在（如果有父评论）
	if req.ParentID != nil && *req.ParentID > 0 {
		parentComment, err := s.dao.GetInstanceCommentByID(ctx, *req.ParentID)
		if err != nil {
			s.logger.Error("父评论不存在", zap.Error(err), zap.Int("parentID", *req.ParentID))
			return fmt.Errorf("父评论不存在: %w", err)
		}
		// 确保父评论属于同一个工单
		if parentComment.InstanceID != req.InstanceID {
			return fmt.Errorf("父评论不属于当前工单")
		}
	}

	comment := &model.WorkorderInstanceComment{
		InstanceID:   req.InstanceID,
		OperatorID:   req.OperatorID,
		OperatorName: req.OperatorName,
		Content:      strings.TrimSpace(req.Content),
		ParentID:     req.ParentID,
		Type:         req.Type,
		Status:       model.CommentStatusNormal,
		IsSystem:     req.IsSystem,
	}

	if comment.Type == "" {
		comment.Type = model.CommentTypeNormal
	}

	if err := s.dao.CreateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("创建工单评论失败", zap.Error(err))
		return fmt.Errorf("创建工单评论失败: %w", err)
	}

	// 普通评论与通知事件同事务提交；系统内部评论不生成重复通知。
	if s.notificationService != nil && comment.IsSystem != 1 {
		outbox := &instanceService{dao: s.instanceDao, logger: s.logger}
		if err := outbox.recordNotificationEvent(ctx, comment.InstanceID, model.EventTypeInstanceCommented, comment.OperatorID, comment.Content, instance); err != nil {
			return err
		}
	}

	return nil
}

// 只允许创建者修改自己的评论
func (s *instanceCommentService) UpdateInstanceComment(ctx context.Context, req *model.UpdateWorkorderInstanceCommentReq, userID int) error {
	existingComment, err := s.dao.GetInstanceCommentByID(ctx, req.ID)
	if err != nil {
		s.logger.Error("获取评论失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("获取评论失败: %w", err)
	}

	// 只允许创建者修改自己的评论（系统评论除外）
	if existingComment.IsSystem != 1 && existingComment.OperatorID != userID {
		return fmt.Errorf("只能修改自己的评论")
	}

	comment := &model.WorkorderInstanceComment{
		Model:    model.Model{ID: req.ID},
		Content:  strings.TrimSpace(req.Content),
		Status:   req.Status,
		IsSystem: req.IsSystem,
	}

	if err := s.dao.UpdateInstanceComment(ctx, comment); err != nil {
		s.logger.Error("更新工单评论失败", zap.Error(err), zap.Int("id", req.ID))
		return fmt.Errorf("更新工单评论失败: %w", err)
	}

	return nil
}

func (s *instanceCommentService) DeleteInstanceComment(ctx context.Context, id int, userID int) error {
	comment, err := s.dao.GetInstanceCommentByID(ctx, id)
	if err != nil {
		s.logger.Error("获取评论失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("获取评论失败: %w", err)
	}

	// 只允许创建者删除自己的评论（系统评论除外）
	if comment.IsSystem != 1 && comment.OperatorID != userID {
		return fmt.Errorf("只能删除自己的评论")
	}

	if err := s.dao.DeleteInstanceComment(ctx, id); err != nil {
		s.logger.Error("删除工单评论失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除工单评论失败: %w", err)
	}

	return nil
}

func (s *instanceCommentService) GetInstanceComment(ctx context.Context, id int) (*model.WorkorderInstanceComment, error) {
	comment, err := s.dao.GetInstanceCommentByID(ctx, id)
	if err != nil {
		s.logger.Error("获取工单评论失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取工单评论失败: %w", err)
	}

	return comment, nil
}

func (s *instanceCommentService) ListInstanceComments(ctx context.Context, req *model.ListWorkorderInstanceCommentReq) (*model.ListResp[*model.WorkorderInstanceComment], error) {
	comments, total, err := s.dao.ListInstanceComments(ctx, req)
	if err != nil {
		s.logger.Error("获取工单评论列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取工单评论列表失败: %w", err)
	}

	return &model.ListResp[*model.WorkorderInstanceComment]{
		Items: comments,
		Total: total,
	}, nil
}

func (s *instanceCommentService) GetInstanceCommentsTree(ctx context.Context, instanceID int) ([]*model.WorkorderInstanceComment, error) {
	_, err := s.instanceDao.GetInstanceByID(ctx, instanceID)
	if err != nil {
		s.logger.Error("工单不存在", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("工单不存在: %w", err)
	}

	comments, err := s.dao.GetInstanceCommentsTree(ctx, instanceID)
	if err != nil {
		s.logger.Error("获取工单评论树失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取工单评论树失败: %w", err)
	}

	return comments, nil
}
