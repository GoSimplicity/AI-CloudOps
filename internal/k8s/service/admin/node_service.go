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

package admin

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeService interface {
	// ListNodeByClusterName 获取指定集群的节点列表
	ListNodeByClusterName(ctx context.Context, id int) ([]*model.K8sNode, error)
	// GetNodeDetail 获取指定节点详情
	GetNodeDetail(ctx context.Context, id int, name string) (*model.K8sNode, error)
	// AddOrUpdateNodeLabel 添加或删除指定节点的 Label
	AddOrUpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesRequest) error
}

type nodeService struct {
	clusterDao admin.ClusterDAO
	client     client.K8sClient
	l          *zap.Logger
}

func NewNodeService(clusterDao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) NodeService {
	return &nodeService{
		clusterDao: clusterDao,
		client:     client,
		l:          l,
	}
}

// ListNodeByClusterName 获取集群的节点列表
func (n *nodeService) ListNodeByClusterName(ctx context.Context, id int) ([]*model.K8sNode, error) {
	kubeClient, metricsClient, err := pkg.GetKubeAndMetricsClient(id, n.l, n.client)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	nodes, err := pkg.GetNodesByName(ctx, kubeClient, "")
	if err != nil {
		n.l.Error("获取节点列表失败", zap.Error(err))
		return nil, err
	}

	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	g, ctx := errgroup.WithContext(ctx)
	k8sNodes := make([]*model.K8sNode, len(nodes.Items))

	// 使用 Worker Pool 模式优化并发性能
	for i := range nodes.Items {
		node := nodes.Items[i] // 避免闭包变量问题
		g.Go(func() error {
			semaphore <- struct{}{}
			defer func() {
				<-semaphore
			}()

			k8sNode, err := pkg.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
			if err != nil {
				n.l.Error("构建 K8sNode 失败", zap.Error(err), zap.String("node", node.Name))
				return nil
			}
			k8sNodes[i] = k8sNode
			return nil
		})
	}

	// 等待所有协程完成
	if err := g.Wait(); err != nil {
		n.l.Error("并发处理节点信息失败", zap.Error(err))
		return nil, err
	}

	return k8sNodes, nil
}

// GetNodeDetail 获取指定节点详情
func (n *nodeService) GetNodeDetail(ctx context.Context, id int, name string) (*model.K8sNode, error) {
	kubeClient, metricsClient, err := pkg.GetKubeAndMetricsClient(id, n.l, n.client)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	nodes, err := pkg.GetNodesByName(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}

	return pkg.BuildK8sNode(ctx, id, nodes.Items[0], kubeClient, metricsClient)
}

// AddOrUpdateNodeLabel 更新节点标签（添加、删除或更新）
func (n *nodeService) AddOrUpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterId, n.client, n.l)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 校验传入的 Labels 数组长度是否为偶数
	if len(req.Labels)%2 != 0 {
		n.l.Error("传入的 Labels 数组不合法", zap.Int("labelsLength", len(req.Labels)))
		return fmt.Errorf("传入的 Labels 数组必须是偶数个元素")
	}

	// 将传入的 labels 转换为 map[string]string
	labelsMap := make(map[string]string)
	for i := 0; i < len(req.Labels); i += 2 {
		labelsMap[req.Labels[i]] = req.Labels[i+1]
	}

	// 获取指定节点
	node, err := kubeClient.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		n.l.Error("获取节点信息失败", zap.Error(err))
		return fmt.Errorf("获取节点 %s 信息失败: %w", req.NodeName, err)
	}

	// 根据操作类型进行标签处理
	switch req.ModType {
	case "add":
		node.Labels = labelsMap
	case "del":
		// 删除标签
		for key := range labelsMap {
			delete(node.Labels, key)
		}
	case "update":
		// 更新标签
		for key, value := range labelsMap {
			if _, exists := node.Labels[key]; exists {
				node.Labels[key] = value
			} else {
				n.l.Warn("标签键不存在，无法更新", zap.String("key", key))
				return fmt.Errorf("节点 %s 不存在标签键 %s，无法更新", req.NodeName, key)
			}
		}
	default:
		errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
		n.l.Error(errMsg)
		return errors.New(errMsg)
	}

	// 更新节点信息
	if _, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{}); err != nil {
		n.l.Error("更新节点信息失败", zap.Error(err))
		return fmt.Errorf("更新节点 %s 信息失败: %w", req.NodeName, err)
	}

	n.l.Info("更新节点Label成功", zap.String("nodeName", req.NodeName))

	// 刷新客户端
	if err := n.client.RefreshClients(ctx); err != nil {
		return fmt.Errorf("刷新客户端失败: %w", err)
	}

	return nil
}
