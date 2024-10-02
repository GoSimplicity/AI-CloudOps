package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type K8sDAO interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error
	DeleteCluster(ctx context.Context, id int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error)
	BatchEnableSwitchClusters(ctx context.Context, ids []int) error
	BatchDeleteClusters(ctx context.Context, ids []int) error

	ListAllNodes(ctx context.Context) ([]*model.K8sNode, error)
	GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error)
	GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error)
	CheckTaintYaml(ctx context.Context, taintYaml string) error
	BatchEnableSwitchNodes(ctx context.Context, ids []int) error
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	AddNodeTaint(ctx context.Context, nodeID int, taintKey, taintValue string) error
	DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error
	DeleteNodeTaint(ctx context.Context, nodeID int, taintKey string) error
	DrainPods(ctx context.Context, nodeID int) error
}

type k8sDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewK8sDAO(db *gorm.DB, l *zap.Logger) K8sDAO {
	return &k8sDAO{
		db: db,
		l:  l,
	}
}

func (k *k8sDAO) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	var clusters []*model.K8sCluster

	if err := k.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		k.l.Error("ListAllClusters 查询所有集群失败", zap.Error(err))
		return nil, err
	}

	return clusters, nil
}

func (k *k8sDAO) ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) CreateCluster(ctx context.Context, cluster *model.K8sCluster) error {
	if err := k.db.WithContext(ctx).Create(&cluster).Error; err != nil {
		k.l.Error("CreateCluster 创建集群失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) DeleteCluster(ctx context.Context, id int) error {
	if err := k.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sCluster{}).Error; err != nil {
		k.l.Error("DeleteCluster 删除集群失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error {
	tx := k.db.WithContext(ctx).Begin()

	defer func() {
		if err := recover(); err != nil {
			k.l.Error("UpdateCluster 更新集群失败,触发回滚", zap.Int("id", id))
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	if err := tx.Where("id = ?", id).Updates(&cluster).Error; err != nil {
		panic(err)
	}
	return nil
}

func (k *k8sDAO) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	if err := k.db.WithContext(ctx).Where("id = ?", id).First(&model.K8sCluster{}).Error; err != nil {
		k.l.Error("GetClusterByID 查询集群失败", zap.Error(err))
		return nil, err
	}
	return &model.K8sCluster{}, nil
}

func (k *k8sDAO) GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) BatchEnableSwitchClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) BatchDeleteClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) ListAllNodes(ctx context.Context) ([]*model.K8sNode, error) {
	//
	var nodes []*model.K8sNode

	if err := k.db.WithContext(ctx).Find(&nodes).Error; err != nil {
		k.l.Error("ListAllNodes 查询所有节点失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (k *k8sDAO) GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) CheckTaintYaml(ctx context.Context, taintYaml string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) BatchEnableSwitchNodes(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) AddNodeTaint(ctx context.Context, nodeID int, taintKey, taintValue string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) DeleteNodeTaint(ctx context.Context, nodeID int, taintKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sDAO) DrainPods(ctx context.Context, nodeID int) error {
	//TODO implement me
	panic("implement me")
}
