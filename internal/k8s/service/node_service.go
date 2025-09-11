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
)

type NodeService interface {
	GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error)
	GetNodeDetail(ctx context.Context, req *model.GetNodeDetailReq) (*model.K8sNode, error)
	AddOrUpdateNodeLabel(ctx context.Context, req *model.AddLabelNodesReq) error
	GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaint], error)
	DrainNode(ctx context.Context, req *model.DrainNodeReq) error
	CordonNode(ctx context.Context, req *model.NodeCordonReq) error
	UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error
	DeleteNodeLabel(ctx context.Context, req *model.DeleteLabelNodesReq) error
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

// GetNodeList 获取节点列表
func (n *nodeService) GetNodeList(ctx context.Context, req *model.GetNodeListReq) (model.ListResp[*model.K8sNode], error) {
	if req == nil {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表请求参数不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("集群 ID 不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildNodeListOptions(req)

	// 使用 NodeManager 获取节点列表
	nodeList, total, err := n.nodeManager.GetNodeList(ctx, req.ClusterID, listOptions)
	if err != nil {
		n.logger.Error("GetNodeList: 获取节点列表失败", zap.Error(err), zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sNode]{}, fmt.Errorf("获取节点列表失败: %w", err)
	}

	nodes := nodeList.Items

	// 根据条件过滤节点
	if len(req.Status) > 0 {
		nodes = utils.FilterNodesByStatus(nodes, req.Status)
	}

	// 使用工具函数进行分页处理
	pagedNodes, totalAfterFilter := utils.BuildNodeListPagination(nodes, req.Page, req.Size)
	total = totalAfterFilter

	// 转换为响应格式
	var items []*model.K8sNode
	for _, node := range pagedNodes {
		k8sNode, err := n.nodeManager.BuildK8sNode(ctx, req.ClusterID, node)
		if err != nil {
			n.logger.Warn("GetNodeList: 构建节点信息失败", zap.Error(err), zap.String("nodeName", node.Name))
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
		return nil, fmt.Errorf("获取节点详情请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return nil, err
	}

	// 使用 NodeManager 获取节点
	node, err := n.nodeManager.GetNode(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		n.logger.Error("GetNodeDetail: 获取节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 使用 NodeManager 构建详细信息
	k8sNode, err := n.nodeManager.BuildK8sNode(ctx, req.ClusterID, *node)
	if err != nil {
		n.logger.Error("GetNodeDetail: 构建节点详细信息失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return nil, fmt.Errorf("构建节点详细信息失败: %w", err)
	}

	return k8sNode, nil
}

// AddOrUpdateNodeLabel 添加或更新节点标签
func (n *nodeService) AddOrUpdateNodeLabel(ctx context.Context, req *model.AddLabelNodesReq) error {
	if req == nil {
		return fmt.Errorf("添加节点标签请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	if len(req.Labels) == 0 {
		return fmt.Errorf("要添加的标签不能为空")
	}

	// 验证标签
	if err := utils.ValidateNodeLabelsMap(req.Labels); err != nil {
		n.logger.Error("AddOrUpdateNodeLabel: 标签验证失败", zap.Error(err))
		return fmt.Errorf("标签验证失败: %w", err)
	}

	// 使用 NodeManager 添加或更新节点标签
	err := n.nodeManager.AddOrUpdateNodeLabels(ctx, req.ClusterID, req.NodeName, req.Labels, req.Overwrite)
	if err != nil {
		n.logger.Error("AddOrUpdateNodeLabel: 添加节点标签失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels))
		return fmt.Errorf("添加节点标签失败: %w", err)
	}

	n.logger.Info("AddOrUpdateNodeLabel: 成功添加节点标签", zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Any("labels", req.Labels), zap.Bool("overwrite", req.Overwrite == 1))
	return nil
}

// DeleteNodeLabel 删除节点标签
func (n *nodeService) DeleteNodeLabel(ctx context.Context, req *model.DeleteLabelNodesReq) error {
	if req == nil {
		return fmt.Errorf("删除节点标签请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	if len(req.LabelKeys) == 0 {
		return fmt.Errorf("要删除的标签键不能为空")
	}

	// 使用 NodeManager 删除节点标签
	err := n.nodeManager.DeleteNodeLabels(ctx, req.ClusterID, req.NodeName, req.LabelKeys)
	if err != nil {
		n.logger.Error("DeleteNodeLabel: 删除节点标签失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Strings("labelKeys", req.LabelKeys))
		return fmt.Errorf("删除节点标签失败: %w", err)
	}

	n.logger.Info("DeleteNodeLabel: 成功删除节点标签", zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName), zap.Strings("labelKeys", req.LabelKeys))

	return nil
}

// GetNodeTaints 获取节点污点
func (n *nodeService) GetNodeTaints(ctx context.Context, req *model.GetNodeTaintsReq) (model.ListResp[*model.NodeTaint], error) {
	if req == nil {
		return model.ListResp[*model.NodeTaint]{}, fmt.Errorf("获取节点污点请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return model.ListResp[*model.NodeTaint]{}, err
	}

	// 使用 NodeManager 获取节点污点
	taints, total, err := n.nodeManager.GetNodeTaints(ctx, req.ClusterID, req.NodeName)
	if err != nil {
		n.logger.Error("GetNodeTaints: 获取节点污点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return model.ListResp[*model.NodeTaint]{}, fmt.Errorf("获取节点污点失败: %w", err)
	}

	return model.ListResp[*model.NodeTaint]{
		Total: total,
		Items: taints,
	}, nil
}

// DrainNode 驱逐节点
func (n *nodeService) DrainNode(ctx context.Context, req *model.DrainNodeReq) error {
	if req == nil {
		return fmt.Errorf("驱逐节点请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
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
		n.logger.Error("DrainNode: 驱逐节点失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("驱逐节点失败: %w", err)
	}

	return nil
}

// CordonNode 禁止节点调度
func (n *nodeService) CordonNode(ctx context.Context, req *model.NodeCordonReq) error {
	if req == nil {
		return fmt.Errorf("禁止节点调度请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	// 使用 NodeManager 禁止节点调度
	if err := n.nodeManager.CordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		n.logger.Error("CordonNode: 禁止节点调度失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("禁止节点 %s 调度失败: %w", req.NodeName, err)
	}

	return nil
}

// UncordonNode 解除节点调度限制
func (n *nodeService) UncordonNode(ctx context.Context, req *model.NodeUncordonReq) error {
	if req == nil {
		return fmt.Errorf("解除节点调度限制请求参数不能为空")
	}

	if err := utils.ValidateBasicParams(req.ClusterID, req.NodeName); err != nil {
		return err
	}

	// 使用 NodeManager 解除节点调度限制
	if err := n.nodeManager.UncordonNode(ctx, req.ClusterID, req.NodeName); err != nil {
		n.logger.Error("UncordonNode: 解除节点调度限制失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("nodeName", req.NodeName))
		return fmt.Errorf("解除节点 %s 调度限制失败: %w", req.NodeName, err)
	}

	return nil
}
