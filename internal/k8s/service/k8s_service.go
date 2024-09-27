package service

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type K8sService interface {
	// ListAllClusters 获取所有 Kubernetes 集群
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	// ListClustersForSelect 获取用于选择的 Kubernetes 集群列表
	ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error)
	// CreateCluster 创建一个新的 Kubernetes 集群
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	// UpdateCluster 更新指定 ID 的 Kubernetes 集群
	UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error
	// BatchEnableSwitchClusters 批量启用或切换 Kubernetes 集群调度
	BatchEnableSwitchClusters(ctx context.Context, ids []int) error
	// BatchDeleteClusters 批量删除 Kubernetes 集群
	BatchDeleteClusters(ctx context.Context, ids []int) error
	// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)

	// ListAllNodes 获取所有 Kubernetes 节点
	ListAllNodes(ctx context.Context) ([]*model.K8sNode, error)
	// GetNodeByID 根据 ID 获取单个 Kubernetes 节点
	GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error)
	// GetPodsByNodeID 根据 Node ID 获取 Pod 列表
	GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error)
	// CheckTaintYaml 检查 Taint Yaml 是否合法
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// BatchEnableSwitchNodes 批量启用或切换 Kubernetes 节点调度
	BatchEnableSwitchNodes(ctx context.Context, ids []int) error
	// AddNodeLabel 添加标签到指定 Node
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	// AddNodeTaint 添加 Taint 到指定 Node
	AddNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// DeleteNodeLabel 删除指定 Node 的标签
	DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error
	// DrainPods 删除指定 Node 上的 Pod
	DrainPods(ctx context.Context, nodeID int) error
}

type k8sService struct {
	dao    dao.K8sDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewK8sService(dao dao.K8sDAO, client client.K8sClient, l *zap.Logger) K8sService {
	return &k8sService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

func (k *k8sService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	return k.dao.ListAllClusters(ctx)
}

func (k *k8sService) ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) CreateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) BatchEnableSwitchClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) ListAllNodes(ctx context.Context) ([]*model.K8sNode, error) {
	return k.dao.ListAllNodes(ctx)
}

func (k *k8sService) GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error {
	panic("implement me")

}

func (k *k8sService) BatchEnableSwitchNodes(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error {
	//TODO implement me
	panic("implement me")
}

// AddNodeTaint 添加或删除 Kubernetes 节点的 Taint
func (k *k8sService) AddNodeTaint(ctx context.Context, req *model.TaintK8sNodesRequest) error {
	// 获取集群信息
	cluster, err := k.dao.GetClusterByName(ctx, req.ClusterName)
	if err != nil {
		k.l.Error("获取集群信息失败", zap.Error(err))
		return err
	}

	// 获取 Kubernetes 客户端
	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 解析 YAML 配置中的 Taints
	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(req.TaintYaml), &taintsToProcess); err != nil {
		k.l.Error("解析 YAML 失败", zap.Error(err))
		return err
	}

	var errs []error

	// 遍历每个节点名称
	for _, nodeName := range req.NodeNames {
		// 获取节点信息
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			k.l.Error("获取节点信息失败", zap.String("node", nodeName), zap.Error(err))
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		switch req.ModType {
		case "add":
			// 添加新的 Taints，避免重复
			existingTaints := node.Spec.Taints
			taintsMap := make(map[string]corev1.Taint)

			// 记录现有 taints，键为 "Key:Value:Effect" 形式
			for _, taint := range existingTaints {
				key := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, taint.Effect)
				taintsMap[key] = taint
			}

			// 添加新的 Taints，避免重复
			for _, newTaint := range taintsToProcess {
				key := fmt.Sprintf("%s:%s:%s", newTaint.Key, newTaint.Value, newTaint.Effect)
				if _, exists := taintsMap[key]; !exists {
					existingTaints = append(existingTaints, newTaint)
				}
			}

			node.Spec.Taints = existingTaints

		case "delete":
			// 删除指定的 Taints
			taintsToDelete := make(map[string]struct{})
			for _, delTaint := range taintsToProcess {
				key := fmt.Sprintf("%s:%s:%s", delTaint.Key, delTaint.Value, delTaint.Effect)
				taintsToDelete[key] = struct{}{}
			}

			var updatedTaints []corev1.Taint

			for _, existingTaint := range node.Spec.Taints {
				key := fmt.Sprintf("%s:%s:%s", existingTaint.Key, existingTaint.Value, existingTaint.Effect)
				if _, shouldDelete := taintsToDelete[key]; !shouldDelete {
					updatedTaints = append(updatedTaints, existingTaint)
				}
			}
			node.Spec.Taints = updatedTaints

		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			k.l.Error(errMsg, zap.String("ModType", req.ModType))
			errs = append(errs, fmt.Errorf(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.String("node", nodeName), zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("成功更新节点 Taints", zap.String("node", nodeName), zap.String("ModType", req.ModType))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

func (k *k8sService) DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DrainPods(ctx context.Context, nodeID int) error {
	//TODO implement me
	panic("implement me")
}
