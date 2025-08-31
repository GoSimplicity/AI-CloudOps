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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

type NodeService interface {
	GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error)
	GetNodeDetail(ctx context.Context, req *model.GetNodeDetailReq) (*model.K8sNode, error)
	AddOrUpdateNodeLabel(ctx context.Context, req *model.AddLabelNodesReq) error
	GetNodeResource(ctx context.Context, req *model.GetNodeResourceReq) (*model.NodeResource, error)
	GetNodeEvents(ctx context.Context, req *model.GetNodeEventsReq) (model.ListResp[*model.NodeEvent], error)
	GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaintEntity], error)
	DrainNode(ctx context.Context, req *model.DrainNodeReq) error
	CordonNode(ctx context.Context, req *model.NodeCordonReq) error
	UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error
	DeleteNodeLabel(ctx context.Context, req *model.DeleteLabelNodesReq) error
	GetNodeMetrics(ctx context.Context, req *model.GetNodeMetricsReq) (model.ListResp[*model.NodeMetrics], error)
}

type nodeService struct {
	clusterDao  dao.ClusterDAO
	client      client.K8sClient
	nodeManager manager.NodeManager
	l           *zap.Logger
}

func NewNodeService(clusterDao dao.ClusterDAO, client client.K8sClient, nodeManager manager.NodeManager, l *zap.Logger) NodeService {
	return &nodeService{
		clusterDao:  clusterDao,
		client:      client,
		nodeManager: nodeManager,
		l:           l,
	}
}

// GetNodeList 获取节点列表
func (n *nodeService) GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error) {
	if req == nil {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("集群 ID 不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildNodeListOptions(req)

	// 使用 NodeManager 获取节点列表
	nodeList, total, err := n.nodeManager.GetNodeList(ctx, req.ClusterID, listOptions)
	if err != nil {
		n.l.Error("GetNodeList: 获取节点列表失败", zap.Error(err), zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表失败: %w", err)
	}

	nodes := nodeList.Items

	// 根据条件过滤节点
	if len(req.NodeNames) > 0 {
		nodes = utils.FilterNodesByNames(nodes, req.NodeNames)
	}
	if len(req.Status) > 0 {
		nodes = utils.FilterNodesByStatus(nodes, req.Status)
	}
	if len(req.Roles) > 0 {
		nodes = utils.FilterNodesByRoles(nodes, req.Roles)
	}

	// 分页处理
	start := int64(req.Page-1) * int64(req.Size)
	end := start + int64(req.Size)

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 获取当前页数据
	var pagedNodes []corev1.Node
	if start < total {
		pagedNodes = nodes[start:end]
	}

	// 转换为响应格式
	var items []*model.K8sNode
	for _, node := range pagedNodes {
		k8sNode, err := n.nodeManager.BuildK8sNode(ctx, req.ClusterID, node)
		if err != nil {
			n.l.Warn("GetNodeList: 构建节点信息失败", zap.Error(err), zap.String("nodeName", node.Name))
			continue
		}
		items = append(items, k8sNode)
	}

	return model.ListResp[*model.K8sNode]{
		Total: total,
		Items: items,
	}, nil
}

// GetNodeDetail 获取节点详情
func (n *nodeService) GetNodeDetail(ctx context.Context, req *model.GetNodeDetailReq) (*model.K8sNode, error) {
	if req == nil {
		return nil, fmt.Errorf("获取节点详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return nil, fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 获取节点
	node, err := n.nodeManager.GetNode(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		n.l.Error("GetNodeDetail: 获取节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 使用 NodeManager 构建详细信息
	k8sNode, err := n.nodeManager.BuildK8sNode(ctx, req.ClusterID, *node)
	if err != nil {
		n.l.Error("GetNodeDetail: 构建节点详细信息失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("构建节点详细信息失败: %w", err)
	}

	return k8sNode, nil
}

// AddOrUpdateNodeLabel 添加或更新节点标签
func (n *nodeService) AddOrUpdateNodeLabel(ctx context.Context, req *model.AddLabelNodesReq) error {
	if req == nil {
		return fmt.Errorf("添加节点标签请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	if len(req.Labels) == 0 {
		return fmt.Errorf("要添加的标签不能为空")
	}

	// 验证标签
	if err := utils.ValidateNodeLabels(req.Labels); err != nil {
		n.l.Error("AddOrUpdateNodeLabel: 标签验证失败", zap.Error(err))
		return fmt.Errorf("标签验证失败: %w", err)
	}

	// 使用 NodeManager 添加或更新节点标签
	err := n.nodeManager.AddOrUpdateNodeLabels(ctx, req.ClusterID, req.NodeName, req.Labels, req.Overwrite)
	if err != nil {
		n.l.Error("AddOrUpdateNodeLabel: 添加节点标签失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels))
		return fmt.Errorf("添加节点标签失败: %w", err)
	}

	n.l.Info("AddOrUpdateNodeLabel: 成功添加节点标签", zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels), zap.Bool("overwrite", req.Overwrite))
	return nil
}

// DeleteNodeLabel 删除节点标签
func (n *nodeService) DeleteNodeLabel(ctx context.Context, req *model.DeleteLabelNodesReq) error {
	if req == nil {
		return fmt.Errorf("删除节点标签请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	if len(req.LabelKeys) == 0 {
		return fmt.Errorf("要删除的标签键不能为空")
	}

	// 使用 NodeManager 删除节点标签
	err := n.nodeManager.DeleteNodeLabels(ctx, req.ClusterID, req.NodeName, req.LabelKeys)
	if err != nil {
		n.l.Error("DeleteNodeLabel: 删除节点标签失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Strings("labelKeys", req.LabelKeys))
		return fmt.Errorf("删除节点标签失败: %w", err)
	}

	n.l.Info("DeleteNodeLabel: 成功删除节点标签", zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Strings("labelKeys", req.LabelKeys))

	return nil
}

// GetNodeResource 获取节点资源
func (n *nodeService) GetNodeResource(ctx context.Context, req *model.GetNodeResourceReq) (*model.NodeResource, error) {
	if req == nil {
		return nil, fmt.Errorf("获取节点资源请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return nil, fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 获取节点资源
	resources, err := n.nodeManager.GetNodeResource(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		n.l.Error("GetNodeResource: 获取节点资源失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("获取节点资源失败: %w", err)
	}

	if resources != nil {
		return resources, nil
	}

	return &model.NodeResource{}, nil
}

// GetNodeEvents 获取节点事件
func (n *nodeService) GetNodeEvents(ctx context.Context, req *model.GetNodeEventsReq) (model.ListResp[*model.NodeEvent], error) {
	if req == nil {
		return model.ListResp[*model.NodeEvent]{}, fmt.Errorf("获取节点事件请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.NodeEvent]{}, fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return model.ListResp[*model.NodeEvent]{}, fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 获取节点事件
	events, total, err := n.nodeManager.GetNodeEvents(ctx, req.ClusterID, req.NodeName, req.Limit)
	if err != nil {
		n.l.Error("GetNodeEvents: 获取节点事件失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return model.ListResp[*model.NodeEvent]{}, fmt.Errorf("获取节点事件失败: %w", err)
	}

	return model.ListResp[*model.NodeEvent]{
		Total: total,
		Items: events,
	}, nil
}

// GetNodeTaints 获取节点污点
func (n *nodeService) GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaintEntity], error) {
	if req == nil {
		return model.ListResp[*model.NodeTaintEntity]{}, fmt.Errorf("获取节点污点请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.NodeTaintEntity]{}, fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return model.ListResp[*model.NodeTaintEntity]{}, fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 获取节点污点
	taints, total, err := n.nodeManager.GetNodeTaints(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		n.l.Error("GetNodeTaints: 获取节点污点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return model.ListResp[*model.NodeTaintEntity]{}, fmt.Errorf("获取节点污点失败: %w", err)
	}

	return model.ListResp[*model.NodeTaintEntity]{
		Total: total,
		Items: taints,
	}, nil
}

// DrainNode 驱逐节点
func (n *nodeService) DrainNode(ctx context.Context, req *model.DrainNodeReq) error {
	if req == nil {
		return fmt.Errorf("驱逐节点请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 驱逐节点
	err := n.nodeManager.DrainNode(ctx, req.ClusterID, req.NodeName, &utils.DrainOptions{
		Force:              req.Force,
		IgnoreDaemonSets:   req.IgnoreDaemonSets,
		DeleteLocalData:    req.DeleteLocalData,
		GracePeriodSeconds: req.GracePeriodSeconds,
		TimeoutSeconds:     req.TimeoutSeconds,
	})
	if err != nil {
		n.l.Error("DrainNode: 驱逐节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("驱逐节点失败: %w", err)
	}

	return nil
}

// CordonNode 禁止节点调度
func (n *nodeService) CordonNode(ctx context.Context, req *model.NodeCordonReq) error {
	if req == nil {
		return fmt.Errorf("禁止节点调度请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 禁止节点调度
	if err := n.nodeManager.CordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		n.l.Error("CordonNode: 禁止节点调度失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("禁止节点 %s 调度失败: %w", req.NodeName, err)
	}

	return nil
}

// UncordonNode 解除节点调度限制
func (n *nodeService) UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error {
	if req == nil {
		return fmt.Errorf("解除节点调度限制请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	// 使用 NodeManager 解除节点调度限制
	if err := n.nodeManager.UncordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		n.l.Error("UncordonNode: 解除节点调度限制失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("解除节点 %s 调度限制失败: %w", req.NodeName, err)
	}

	return nil
}

// GetNodeMetrics 获取节点指标
func (n *nodeService) GetNodeMetrics(ctx context.Context, req *model.GetNodeMetricsReq) (model.ListResp[*model.NodeMetrics], error) {
	if req == nil {
		return model.ListResp[*model.NodeMetrics]{}, fmt.Errorf("获取节点指标请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.NodeMetrics]{}, fmt.Errorf("集群 ID 不能为空")
	}

	// 使用 NodeManager 获取节点指标
	metrics, total, err := n.nodeManager.GetNodeMetrics(ctx, req.ClusterID, req.NodeNames)
	if err != nil {
		n.l.Error("GetNodeMetrics: 获取节点指标失败", zap.Error(err), zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.NodeMetrics]{}, fmt.Errorf("获取节点指标失败: %w", err)
	}

	return model.ListResp[*model.NodeMetrics]{
		Total: total,
		Items: metrics,
	}, nil
}
