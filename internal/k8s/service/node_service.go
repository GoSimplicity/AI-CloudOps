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
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"go.uber.org/zap"
)

type NodeService interface {
	GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error)
	GetNodeDetail(ctx context.Context, req *model.GetNodeDetailReq) (*model.K8sNode, error)
	UpdateNodeLabels(ctx context.Context, req *model.UpdateNodeLabelsReq) error
	GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaint], error)
	DrainNode(ctx context.Context, req *model.DrainNodeReq) error
	CordonNode(ctx context.Context, req *model.NodeCordonReq) error
	UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error
}

type nodeService struct {
	clusterDao  dao.ClusterDAO
	client      client.K8sClient
	nodeManager manager.NodeManager
	logger      *zap.Logger
}

func NewNodeService(clusterDao dao.ClusterDAO, client client.K8sClient, nodeManager manager.NodeManager, logger *zap.Logger) NodeService {
	return &nodeService{
		clusterDao:  clusterDao,
		client:      client,
		nodeManager: nodeManager,
		logger:      logger,
	}
}

func (s *nodeService) GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error) {
	if req == nil {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表请求参数不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("集群 ID 不能为空")
	}

	listOptions := utils.BuildNodeListOptions(req)

	nodeList, total, err := s.nodeManager.GetNodeList(ctx, req.ClusterID, listOptions)
	if err != nil {
		s.logger.Error("获取节点列表失败", zap.Error(err), zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表失败: %w", err)
	}

	nodes := nodeList.Items

	// 应用过滤条件
	// 状态过滤
	if len(req.Status) > 0 {
		nodes = utils.FilterNodesByStatus(nodes, req.Status)
	}

	// 名称过滤（使用通用的Search字段，支持不区分大小写）
	var filteredNodes []corev1.Node
	for _, node := range nodes {
		if utils.FilterByName(node.Name, req.Search) {
			filteredNodes = append(filteredNodes, node)
		}
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredNodes, func(node corev1.Node) time.Time {
		return node.CreationTimestamp.Time
	})

	// 分页处理
	pagedNodes, totalAfterFilter := utils.BuildNodeListPagination(filteredNodes, req.Page, req.Size)
	total = totalAfterFilter

	var items []*model.K8sNode
	for _, node := range pagedNodes {
		k8sNode, err := s.nodeManager.BuildK8sNode(ctx, req.ClusterID, node)
		if err != nil {
			s.logger.Warn("构建节点信息失败", zap.Error(err), zap.String("nodeName", node.Name))
			continue
		}
		items = append(items, k8sNode)
	}

	return model.ListResp[*model.K8sNode]{
		Total: total,
		Items: items,
	}, nil
}

func (s *nodeService) GetNodeDetail(ctx context.Context, req *model.GetNodeDetailReq) (*model.K8sNode, error) {
	if req == nil {
		return nil, fmt.Errorf("获取节点详情请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return nil, err
	}

	node, err := s.nodeManager.GetNode(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		s.logger.Error("获取节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	k8sNode, err := s.nodeManager.BuildK8sNode(ctx, req.ClusterID, *node)
	if err != nil {
		s.logger.Error("构建节点详细信息失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("构建节点详细信息失败: %w", err)
	}

	return k8sNode, nil
}

func (s *nodeService) UpdateNodeLabels(ctx context.Context, req *model.UpdateNodeLabelsReq) error {
	if req == nil {
		return fmt.Errorf("更新节点标签请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	// 允许传入空标签，表示清空所有标签
	if req.Labels != nil {

		if err := utils.ValidateNodeLabelsMap(req.Labels); err != nil {
			s.logger.Error("标签验证失败", zap.Error(err))
			return fmt.Errorf("标签验证失败: %w", err)
		}
	}

	err := s.nodeManager.UpdateNodeLabels(ctx, req.ClusterID, req.NodeName, req.Labels)
	if err != nil {
		s.logger.Error("更新节点标签失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels))
		return fmt.Errorf("更新节点标签失败: %w", err)
	}

	s.logger.Info("成功更新节点标签", zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels))

	return nil
}

func (s *nodeService) GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaint], error) {
	if req == nil {
		return model.ListResp[*model.NodeTaint]{}, fmt.Errorf("获取节点污点请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return model.ListResp[*model.NodeTaint]{}, err
	}

	taints, total, err := s.nodeManager.GetNodeTaints(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		s.logger.Error("获取节点污点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return model.ListResp[*model.NodeTaint]{}, fmt.Errorf("获取节点污点失败: %w", err)
	}

	return model.ListResp[*model.NodeTaint]{
		Total: total,
		Items: taints,
	}, nil
}

func (s *nodeService) DrainNode(ctx context.Context, req *model.DrainNodeReq) error {
	if req == nil {
		return fmt.Errorf("驱逐节点请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	err := s.nodeManager.DrainNode(ctx, req.ClusterID, req.NodeName, &utils.DrainOptions{
		Force:              req.Force,
		IgnoreDaemonSets:   req.IgnoreDaemonSets,
		DeleteLocalData:    req.DeleteLocalData,
		GracePeriodSeconds: req.GracePeriodSeconds,
		TimeoutSeconds:     req.TimeoutSeconds,
	})
	if err != nil {
		s.logger.Error("驱逐节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("驱逐节点失败: %w", err)
	}

	return nil
}

func (s *nodeService) CordonNode(ctx context.Context, req *model.NodeCordonReq) error {
	if req == nil {
		return fmt.Errorf("禁止节点调度请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	if err := s.nodeManager.CordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		s.logger.Error("禁止节点调度失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("禁止节点 %s 调度失败: %w", req.NodeName, err)
	}

	return nil
}

func (s *nodeService) UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error {
	if req == nil {
		return fmt.Errorf("解除节点调度限制请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	if err := s.nodeManager.UncordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		s.logger.Error("解除节点调度限制失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("解除节点 %s 调度限制失败: %w", req.NodeName, err)
	}

	return nil
}
