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

// 错误定义
var (
	ErrInstanceNotFound = errors.New("工单实例不存在")
	ErrInvalidAction    = errors.New("无效的操作类型")
)

type InstanceFlowService interface {
	ListInstanceFlows(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) (*model.ListResp[*model.WorkorderInstanceFlow], error)
	DetailInstanceFlow(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error)
}

type instanceFlowService struct {
	dao    dao.WorkorderInstanceFlowDAO
	logger *zap.Logger
}

func NewInstanceFlowService(dao dao.WorkorderInstanceFlowDAO, logger *zap.Logger) InstanceFlowService {
	return &instanceFlowService{
		dao:    dao,
		logger: logger,
	}
}

// ListInstanceFlows 分页获取工单流程记录列表
func (s *instanceFlowService) ListInstanceFlows(ctx context.Context, req *model.ListWorkorderInstanceFlowReq) (*model.ListResp[*model.WorkorderInstanceFlow], error) {
	if req == nil {
		return nil, dao.ErrInstanceInvalidID
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	flows, total, err := s.dao.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("获取工单流程记录列表失败: %w", err)
	}

	// 转换为指针切片
	flowPtrs := make([]*model.WorkorderInstanceFlow, len(flows))
	for i := range flows {
		flowPtrs[i] = &flows[i]
	}

	return &model.ListResp[*model.WorkorderInstanceFlow]{
		Items: flowPtrs,
		Total: total,
	}, nil
}

// DetailInstanceFlow 获取工单流转记录详情
func (s *instanceFlowService) DetailInstanceFlow(ctx context.Context, id int) (*model.WorkorderInstanceFlow, error) {
	if id <= 0 {
		return nil, dao.ErrInstanceInvalidID
	}

	flow, err := s.dao.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, dao.ErrInstanceFlowNotFound) {
			return nil, ErrInstanceNotFound
		}
		return nil, fmt.Errorf("获取工单流转记录失败: %w", err)
	}

	return flow, nil
}

// 辅助方法

// validateCreateRequest 验证创建请求
func (s *instanceFlowService) validateCreateRequest(req *model.CreateWorkorderInstanceFlowReq) error {
	if req == nil {
		return dao.ErrFlowNilPointer
	}

	if req.InstanceID <= 0 {
		return fmt.Errorf("工单ID无效")
	}

	if strings.TrimSpace(req.StepID) == "" {
		return fmt.Errorf("步骤ID不能为空")
	}

	if strings.TrimSpace(req.StepName) == "" {
		return fmt.Errorf("步骤名称不能为空")
	}

	if strings.TrimSpace(req.Action) == "" {
		return fmt.Errorf("操作动作不能为空")
	}

	if req.OperatorID <= 0 {
		return fmt.Errorf("操作人ID无效")
	}

	if strings.TrimSpace(req.OperatorName) == "" {
		return fmt.Errorf("操作人名称不能为空")
	}

	// 验证操作动作是否有效
	validActions := []string{
		model.FlowActionCreate, model.FlowActionSubmit, model.FlowActionApprove,
		model.FlowActionReject, model.FlowActionTransfer, model.FlowActionAssign,
		model.FlowActionRevoke, model.FlowActionCancel, model.FlowActionReturn,
		model.FlowActionComplete,
	}

	actionValid := false
	for _, action := range validActions {
		if req.Action == action {
			actionValid = true
			break
		}
	}

	if !actionValid {
		return ErrInvalidAction
	}

	// 如果指定了结果，验证结果是否有效
	if req.Result != "" {
		validResults := []string{model.FlowResultSuccess, model.FlowResultFailed, model.FlowResultPending}
		resultValid := false
		for _, result := range validResults {
			if req.Result == result {
				resultValid = true
				break
			}
		}
		if !resultValid {
			return fmt.Errorf("无效的处理结果")
		}
	}

	return nil
}
