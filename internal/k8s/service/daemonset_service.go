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
	"time"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSetService interface {
	// 获取DaemonSet列表
	GetDaemonSetList(ctx context.Context, req *model.K8sDaemonSetListReq) ([]*model.K8sDaemonSetEntity, error)
	GetDaemonSetsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sDaemonSetEntity, error)

	// 获取DaemonSet详情
	GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*model.K8sDaemonSetEntity, error)
	GetDaemonSetYaml(ctx context.Context, clusterID int, namespace, name string) (string, error)

	// DaemonSet操作
	CreateDaemonSet(ctx context.Context, req *model.K8sDaemonSetCreateReq) error
	UpdateDaemonSet(ctx context.Context, req *model.K8sDaemonSetUpdateReq) error
	DeleteDaemonSet(ctx context.Context, req *model.K8sDaemonSetDeleteReq) error
	RestartDaemonSet(ctx context.Context, req *model.K8sDaemonSetRestartReq) error

	// 批量操作
	BatchDeleteDaemonSets(ctx context.Context, req *model.K8sDaemonSetBatchDeleteReq) error
	BatchRestartDaemonSets(ctx context.Context, req *model.K8sDaemonSetBatchRestartReq) error

	// 高级功能（TODO实现）
	GetDaemonSetHistory(ctx context.Context, req *model.K8sDaemonSetHistoryReq) (interface{}, error)
	GetDaemonSetEvents(ctx context.Context, req *model.K8sDaemonSetEventReq) ([]*model.K8sEvent, error)
	GetDaemonSetNodePods(ctx context.Context, req *model.K8sDaemonSetNodePodsReq) ([]*model.K8sPod, error)
}

type daemonSetService struct {
	dao    dao.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewDaemonSetService 创建新的 DaemonSetService 实例
func NewDaemonSetService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) DaemonSetService {
	return &daemonSetService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetDaemonSetList 获取DaemonSet列表
func (d *daemonSetService) GetDaemonSetList(ctx context.Context, req *model.K8sDaemonSetListReq) ([]*model.K8sDaemonSetEntity, error) {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := metav1.ListOptions{}
	if req.LabelSelector != "" {
		listOptions.LabelSelector = req.LabelSelector
	}
	if req.FieldSelector != "" {
		listOptions.FieldSelector = req.FieldSelector
	}

	daemonSets, err := kubeClient.AppsV1().DaemonSets(req.Namespace).List(ctx, listOptions)
	if err != nil {
		d.logger.Error("获取DaemonSet列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取DaemonSet列表失败")
	}

	entities := make([]*model.K8sDaemonSetEntity, 0, len(daemonSets.Items))
	for _, ds := range daemonSets.Items {
		entity := d.convertDaemonSetToEntity(&ds, req.ClusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetDaemonSetsByNamespace 根据命名空间获取DaemonSet列表
func (d *daemonSetService) GetDaemonSetsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sDaemonSetEntity, error) {
	kubeClient, err := d.client.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	daemonSets, err := kubeClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		d.logger.Error("获取DaemonSet列表失败",
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sDaemonSetEntity, 0, len(daemonSets.Items))
	for _, ds := range daemonSets.Items {
		entity := d.convertDaemonSetToEntity(&ds, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetDaemonSet 获取单个DaemonSet详情
func (d *daemonSetService) GetDaemonSet(ctx context.Context, clusterID int, namespace, name string) (*model.K8sDaemonSetEntity, error) {
	kubeClient, err := d.client.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取DaemonSet详情失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取DaemonSet详情失败")
	}

	return d.convertDaemonSetToEntity(daemonSet, clusterID), nil
}

// GetDaemonSetYaml 获取DaemonSet的YAML
func (d *daemonSetService) GetDaemonSetYaml(ctx context.Context, clusterID int, namespace, name string) (string, error) {
	kubeClient, err := d.client.GetKubeClient(clusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取DaemonSet失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取DaemonSet失败")
	}

	yamlData, err := yaml.Marshal(daemonSet)
	if err != nil {
		d.logger.Error("序列化DaemonSet为YAML失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化DaemonSet为YAML失败")
	}

	return string(yamlData), nil
}

// CreateDaemonSet 创建DaemonSet
func (d *daemonSetService) CreateDaemonSet(ctx context.Context, req *model.K8sDaemonSetCreateReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.DaemonSetYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "DaemonSet YAML不能为空")
	}

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Create(ctx, req.DaemonSetYaml, metav1.CreateOptions{})
	if err != nil {
		d.logger.Error("创建DaemonSet失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建DaemonSet失败")
	}

	d.logger.Info("成功创建DaemonSet",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateDaemonSet 更新DaemonSet
func (d *daemonSetService) UpdateDaemonSet(ctx context.Context, req *model.K8sDaemonSetUpdateReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.DaemonSetYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "DaemonSet YAML不能为空")
	}

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, req.DaemonSetYaml, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("更新DaemonSet失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新DaemonSet失败")
	}

	d.logger.Info("成功更新DaemonSet",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteDaemonSet 删除DaemonSet
func (d *daemonSetService) DeleteDaemonSet(ctx context.Context, req *model.K8sDaemonSetDeleteReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.AppsV1().DaemonSets(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		d.logger.Error("删除DaemonSet失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除DaemonSet失败")
	}

	d.logger.Info("成功删除DaemonSet",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// RestartDaemonSet 重启DaemonSet
func (d *daemonSetService) RestartDaemonSet(ctx context.Context, req *model.K8sDaemonSetRestartReq) error {
	kubeClient, err := d.client.GetKubeClient(req.ClusterID)
	if err != nil {
		d.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 获取现有的DaemonSet
	daemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取DaemonSet失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取DaemonSet失败")
	}

	// 添加重启注解
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		d.logger.Error("重启DaemonSet失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "重启DaemonSet失败")
	}

	d.logger.Info("成功重启DaemonSet",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// BatchDeleteDaemonSets 批量删除DaemonSet
func (d *daemonSetService) BatchDeleteDaemonSets(ctx context.Context, req *model.K8sDaemonSetBatchDeleteReq) error {
	// TODO: 实现批量删除功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "批量删除DaemonSet功能尚未实现")
}

// BatchRestartDaemonSets 批量重启DaemonSet
func (d *daemonSetService) BatchRestartDaemonSets(ctx context.Context, req *model.K8sDaemonSetBatchRestartReq) error {
	// TODO: 实现批量重启功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "批量重启DaemonSet功能尚未实现")
}

// GetDaemonSetHistory 获取DaemonSet历史版本
func (d *daemonSetService) GetDaemonSetHistory(ctx context.Context, req *model.K8sDaemonSetHistoryReq) (interface{}, error) {
	// TODO: 实现获取历史版本功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取DaemonSet历史版本功能尚未实现")
}

// GetDaemonSetEvents 获取DaemonSet事件
func (d *daemonSetService) GetDaemonSetEvents(ctx context.Context, req *model.K8sDaemonSetEventReq) ([]*model.K8sEvent, error) {
	// TODO: 实现获取事件功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取DaemonSet事件功能尚未实现")
}

// GetDaemonSetNodePods 获取DaemonSet在指定节点上的Pod
func (d *daemonSetService) GetDaemonSetNodePods(ctx context.Context, req *model.K8sDaemonSetNodePodsReq) ([]*model.K8sPod, error) {
	// TODO: 实现获取节点Pod功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取DaemonSet节点Pod功能尚未实现")
}

// convertDaemonSetToEntity 将Kubernetes DaemonSet转换为实体模型
func (d *daemonSetService) convertDaemonSetToEntity(daemonSet *appsv1.DaemonSet, clusterID int) *model.K8sDaemonSetEntity {
	// 提取镜像列表
	images := make([]string, 0)
	for _, container := range daemonSet.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	// 计算年龄
	age := pkg.GetAge(daemonSet.CreationTimestamp.Time)

	// 确定状态
	status := "Running"
	if daemonSet.Status.NumberReady == 0 {
		status = "Pending"
	} else if daemonSet.Status.NumberReady < daemonSet.Status.DesiredNumberScheduled {
		status = "Partial"
	}

	return &model.K8sDaemonSetEntity{
		Name:                   daemonSet.Name,
		Namespace:              daemonSet.Namespace,
		ClusterID:              clusterID,
		UID:                    string(daemonSet.UID),
		DesiredNumberScheduled: daemonSet.Status.DesiredNumberScheduled,
		CurrentNumberScheduled: daemonSet.Status.CurrentNumberScheduled,
		NumberReady:            daemonSet.Status.NumberReady,
		NumberAvailable:        daemonSet.Status.NumberAvailable,
		NumberUnavailable:      daemonSet.Status.NumberUnavailable,
		UpdatedNumberScheduled: daemonSet.Status.UpdatedNumberScheduled,
		NumberMisscheduled:     daemonSet.Status.NumberMisscheduled,
		UpdateStrategy:         string(daemonSet.Spec.UpdateStrategy.Type),
		Selector:               daemonSet.Spec.Selector.MatchLabels,
		Labels:                 daemonSet.Labels,
		Annotations:            daemonSet.Annotations,
		CreationTimestamp:      daemonSet.CreationTimestamp.Time,
		Age:                    age,
		Status:                 status,
		Images:                 images,
	}
}
