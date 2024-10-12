package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type K8sDAO interface {
	// Cluster
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error)
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error
	DeleteCluster(ctx context.Context, id int) error
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)
	GetClusterByName(ctx context.Context, name string) (*model.K8sCluster, error)
	BatchEnableSwitchClusters(ctx context.Context, ids []int) error
	BatchDeleteClusters(ctx context.Context, ids []int) error

	// Node
	ListAllNodes(ctx context.Context) ([]*model.K8sNode, error)
	GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error)
	GetNodeByName(ctx context.Context, name string) (*model.K8sNode, error)
	GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error)
	CheckTaintYaml(ctx context.Context, taintYaml string) error
	BatchEnableSwitchNodes(ctx context.Context, ids []int) error
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	AddNodeTaint(ctx context.Context, nodeID int, taintKey, taintValue string) error
	DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error
	DeleteNodeTaint(ctx context.Context, nodeID int, taintKey string) error
	DrainPods(ctx context.Context, nodeID int) error

	// Yaml
	ListAllYamlTemplates(ctx context.Context) ([]*model.K8sYamlTemplate, error)
	CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error
	UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error
	DeleteYamlTemplate(ctx context.Context, id int) error
	GetYamlTemplateByID(ctx context.Context, id int) (*model.K8sYamlTemplate, error)

	// YamlTask
	ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error)
	CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error
	DeleteYamlTask(ctx context.Context, id int) error
	GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error)
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
	if err := k.db.WithContext(ctx).Where("id = ?", id).Unscoped().Delete(&model.K8sCluster{}).Error; err != nil {
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
	var cluster *model.K8sCluster

	if err := k.db.WithContext(ctx).Where("name = ?", name).First(&cluster).Error; err != nil {
		k.l.Error("GetClusterByName 查询集群失败", zap.Error(err))
		return nil, err
	}

	return cluster, nil
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

func (k *k8sDAO) GetNodeByName(ctx context.Context, name string) (*model.K8sNode, error) {
	var node *model.K8sNode

	if err := k.db.WithContext(ctx).Where("name = ?", name).First(&node).Error; err != nil {
		k.l.Error("GetNodeByName 查询节点失败", zap.Error(err))
		return nil, err
	}

	return node, nil
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

func (k *k8sDAO) ListAllYamlTemplates(ctx context.Context) ([]*model.K8sYamlTemplate, error) {
	var yamls []*model.K8sYamlTemplate

	if err := k.db.WithContext(ctx).Find(&yamls).Error; err != nil {
		k.l.Error("ListAllYamlTemplates 查询所有Yaml模板失败", zap.Error(err))
		return nil, err
	}

	return yamls, nil
}

func (k *k8sDAO) CreateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error {
	if err := k.db.WithContext(ctx).Create(&yaml).Error; err != nil {
		k.l.Error("CreateYamlTemplate 创建Yaml模板失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) UpdateYamlTemplate(ctx context.Context, yaml *model.K8sYamlTemplate) error {
	if err := k.db.WithContext(ctx).Where("id = ?", yaml.ID).Updates(&yaml).Error; err != nil {
		k.l.Error("UpdateYamlTemplate 更新Yaml模板失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) DeleteYamlTemplate(ctx context.Context, id int) error {
	if err := k.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sYamlTemplate{}).Error; err != nil {
		k.l.Error("DeleteYamlTemplate 删除Yaml模板失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) ListAllYamlTasks(ctx context.Context) ([]*model.K8sYamlTask, error) {
	var tasks []*model.K8sYamlTask

	if err := k.db.WithContext(ctx).Find(&tasks).Error; err != nil {
		k.l.Error("ListAllYamlTasks 查询所有Yaml任务失败", zap.Error(err))
		return nil, err
	}

	return tasks, nil
}

func (k *k8sDAO) CreateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := k.db.WithContext(ctx).Create(&task).Error; err != nil {
		k.l.Error("CreateYamlTask 创建Yaml任务失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) UpdateYamlTask(ctx context.Context, task *model.K8sYamlTask) error {
	if err := k.db.WithContext(ctx).Where("id = ?", task.ID).Updates(&task).Error; err != nil {
		k.l.Error("UpdateYamlTask 更新Yaml任务失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) DeleteYamlTask(ctx context.Context, id int) error {
	if err := k.db.WithContext(ctx).Where("id = ?", id).Delete(&model.K8sYamlTask{}).Error; err != nil {
		k.l.Error("DeleteYamlTask 删除Yaml任务失败", zap.Error(err))
		return err
	}

	return nil
}

func (k *k8sDAO) GetYamlTemplateByID(ctx context.Context, id int) (*model.K8sYamlTemplate, error) {
	var yaml *model.K8sYamlTemplate

	if err := k.db.WithContext(ctx).Where("id = ?", id).First(&yaml).Error; err != nil {
		k.l.Error("GetYamlTemplateByID 查询Yaml模板失败", zap.Error(err))
		return nil, err
	}

	return yaml, nil
}

func (k *k8sDAO) GetYamlTaskByID(ctx context.Context, id int) (*model.K8sYamlTask, error) {
	var task *model.K8sYamlTask

	if err := k.db.WithContext(ctx).Where("id = ?", id).First(&task).Error; err != nil {
		k.l.Error("GetYamlTaskByID 查询Yaml任务失败", zap.Error(err))
		return nil, err
	}

	return task, nil
}
