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
	"sync"
)

type NodeService interface {
	// ListNodeByClusterId 获取指定集群的节点列表
	ListNodeByClusterId(ctx context.Context, id int) ([]*model.K8sNode, error)
	// GetNodeByName 根据 ID 获取指定节点
	GetNodeByName(ctx context.Context, id int, name string) (*model.K8sNode, error)
	// UpdateNodeLabel 添加或删除指定节点的 Label
	UpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesRequest) error
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

// ListNodeByClusterId 获取集群的节点列表
func (n *nodeService) ListNodeByClusterId(ctx context.Context, id int) ([]*model.K8sNode, error) {
	kubeClient, err := n.client.GetKubeClient(id)
	if err != nil {
		n.l.Error("ListAppointCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	metricsClient, err := n.client.GetMetricsClient(id)
	if err != nil {
		n.l.Error("ListAppointCluster: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, constants.ErrorMetricsClientNotReady
	}

	// 获取节点列表
	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, "")
	if err != nil {
		n.l.Error("ListAppointCluster: 获取节点列表失败", zap.Error(err))
		return nil, err
	}

	// 并发控制，限制最大并发数为 10
	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	g, ctx := errgroup.WithContext(ctx)

	// 使用互斥锁保护对共享切片的访问
	var mu sync.Mutex
	var k8sNodes []*model.K8sNode

	// 遍历节点并发处理
	for _, node := range nodes.Items {
		g.Go(func() error {
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 构建 K8sNode 对象
			k8sNode, err := pkg.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
			if err != nil {
				n.l.Error("ListAppointCluster: 构建 K8sNode 失败", zap.Error(err), zap.String("node", node.Name))
				return nil
			}

			// 使用互斥锁更新节点列表
			mu.Lock()
			k8sNodes = append(k8sNodes, k8sNode)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		n.l.Error("ListAppointCluster: 并发处理节点信息失败", zap.Error(err))
		return nil, err
	}

	return k8sNodes, nil
}

// GetNodeByName 根据 ID 获取指定节点
func (n *nodeService) GetNodeByName(ctx context.Context, id int, name string) (*model.K8sNode, error) {
	kubeClient, err := n.client.GetKubeClient(id)
	if err != nil {
		n.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	metricsClient, err := n.client.GetMetricsClient(id)
	if err != nil {
		n.l.Error("CreateCluster: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, err
	}

	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}

	return pkg.BuildK8sNode(ctx, id, nodes.Items[0], kubeClient, metricsClient)
}

// UpdateNodeLabel 更新节点标签（添加或删除）
func (n *nodeService) UpdateNodeLabel(ctx context.Context, nodeResource *model.LabelK8sNodesRequest) error {
	kubeClient, err := pkg.GetKubeClient(ctx, nodeResource.ClusterName, n.clusterDao, n.client, n.l)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	var errs []error

	for _, nodeName := range nodeResource.NodeNames {
		// 获取节点
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			n.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		switch nodeResource.ModType {
		case "add":
			for key, value := range nodeResource.Labels {
				node.Labels[key] = value
			}

		case "del":
			for key := range nodeResource.Labels {
				delete(node.Labels, key)
			}

		default:
			errMsg := fmt.Sprintf("未知的修改类型: %s", nodeResource.ModType)
			n.l.Error(errMsg)
			errs = append(errs, errors.New(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			n.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		n.l.Info("更新节点Label成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Labels 时遇到以下错误: %v", errs)
	}

	return nil
}
