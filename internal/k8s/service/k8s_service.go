package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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
	// DeleteCluster 删除指定 ID 的 Kubernetes 集群
	DeleteCluster(ctx context.Context, id int) error
	// EnableSwitchCluster 启用或切换指定 ID 的 Kubernetes 集群调度
	EnableSwitchCluster(ctx context.Context, id int) error
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
	CheckTaintYaml(ctx context.Context, taintYaml string) error
	// BatchEnableSwitchNodes 批量启用或切换 Kubernetes 节点调度
	BatchEnableSwitchNodes(ctx context.Context, ids []int) error
	// AddNodeLabel 添加标签到指定 Node
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	// AddNodeTaint 添加 Taint 到指定 Node
	AddNodeTaint(ctx context.Context, nodeID int, taintKey, taintValue string) error
	// DeleteNodeLabel 删除指定 Node 的标签
	DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error
	// DeleteNodeTaint 删除指定 Node 的 Taint
	DeleteNodeTaint(ctx context.Context, nodeID int, taintKey string) error
	// DrainPods 删除指定 Node 上的 Pod
	DrainPods(ctx context.Context, nodeID int) error
}

type k8sService struct {
	dao dao.K8sDAO
}

func NewK8sService(dao dao.K8sDAO) K8sService {
	return &k8sService{
		dao: dao,
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

func (k *k8sService) DeleteCluster(ctx context.Context, id int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) EnableSwitchCluster(ctx context.Context, id int) error {
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

func (k *k8sService) CheckTaintYaml(ctx context.Context, taintYaml string) error {
	//TODO implement me
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

func (k *k8sService) AddNodeTaint(ctx context.Context, nodeID int, taintKey, taintValue string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DeleteNodeTaint(ctx context.Context, nodeID int, taintKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DrainPods(ctx context.Context, nodeID int) error {
	//TODO implement me
	panic("implement me")
}
