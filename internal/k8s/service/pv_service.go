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

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVService interface {
	// 获取PV列表
	GetPVList(ctx context.Context, req *model.K8sPVListReq) ([]*model.K8sPVEntity, error)
	GetPVsByCluster(ctx context.Context, clusterID int) ([]*model.K8sPVEntity, error)

	// 获取PV详情
	GetPV(ctx context.Context, clusterID int, name string) (*model.K8sPVEntity, error)
	GetPVYaml(ctx context.Context, clusterID int, name string) (string, error)

	// PV操作
	CreatePV(ctx context.Context, req *model.K8sPVCreateReq) error
	UpdatePV(ctx context.Context, req *model.K8sPVUpdateReq) error
	DeletePV(ctx context.Context, req *model.K8sPVDeleteReq) error

	// 批量操作
	BatchDeletePVs(ctx context.Context, req *model.K8sPVBatchDeleteReq) error

	// 高级功能（TODO实现）
	GetPVEvents(ctx context.Context, req *model.K8sPVEventReq) ([]*model.K8sEvent, error)
	GetPVUsage(ctx context.Context, req *model.K8sPVUsageReq) (*model.K8sPVUsageInfo, error)
	ReclaimPV(ctx context.Context, clusterID int, name string) error
}

type pvService struct {
	dao    dao.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewPVService 创建新的 PVService 实例
func NewPVService(dao dao.ClusterDAO, client client.K8sClient, logger *zap.Logger) PVService {
	return &pvService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetPVList 获取PV列表
func (p *pvService) GetPVList(ctx context.Context, req *model.K8sPVListReq) ([]*model.K8sPVEntity, error) {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := metav1.ListOptions{}
	if req.LabelSelector != "" {
		listOptions.LabelSelector = req.LabelSelector
	}
	if req.FieldSelector != "" {
		listOptions.FieldSelector = req.FieldSelector
	}

	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取PV列表失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PV列表失败")
	}

	entities := make([]*model.K8sPVEntity, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		entity := p.convertPVToEntity(&pv, req.ClusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetPVsByCluster 根据集群获取PV列表
func (p *pvService) GetPVsByCluster(ctx context.Context, clusterID int) ([]*model.K8sPVEntity, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取PV列表失败", zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sPVEntity, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		entity := p.convertPVToEntity(&pv, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetPV 获取单个PV详情
func (p *pvService) GetPV(ctx context.Context, clusterID int, name string) (*model.K8sPVEntity, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取PV详情失败",
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PV详情失败")
	}

	return p.convertPVToEntity(pv, clusterID), nil
}

// GetPVYaml 获取PV的YAML
func (p *pvService) GetPVYaml(ctx context.Context, clusterID int, name string) (string, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取PV失败",
			zap.String("name", name),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PV失败")
	}

	yamlData, err := yaml.Marshal(pv)
	if err != nil {
		p.logger.Error("序列化PV为YAML失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化PV为YAML失败")
	}

	return string(yamlData), nil
}

// CreatePV 创建PV
func (p *pvService) CreatePV(ctx context.Context, req *model.K8sPVCreateReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.PVYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV YAML不能为空")
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().Create(ctx, req.PVYaml, metav1.CreateOptions{})
	if err != nil {
		p.logger.Error("创建PV失败",
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PV失败")
	}

	p.logger.Info("成功创建PV", zap.String("name", req.Name))
	return nil
}

// UpdatePV 更新PV
func (p *pvService) UpdatePV(ctx context.Context, req *model.K8sPVUpdateReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.PVYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV YAML不能为空")
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().Update(ctx, req.PVYaml, metav1.UpdateOptions{})
	if err != nil {
		p.logger.Error("更新PV失败",
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PV失败")
	}

	p.logger.Info("成功更新PV", zap.String("name", req.Name))
	return nil
}

// DeletePV 删除PV
func (p *pvService) DeletePV(ctx context.Context, req *model.K8sPVDeleteReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.CoreV1().PersistentVolumes().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		p.logger.Error("删除PV失败",
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除PV失败")
	}

	p.logger.Info("成功删除PV", zap.String("name", req.Name))
	return nil
}

// BatchDeletePVs 批量删除PV
func (p *pvService) BatchDeletePVs(ctx context.Context, req *model.K8sPVBatchDeleteReq) error {
	// TODO: 实现批量删除功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "批量删除PV功能尚未实现")
}

// GetPVEvents 获取PV事件
func (p *pvService) GetPVEvents(ctx context.Context, req *model.K8sPVEventReq) ([]*model.K8sEvent, error) {
	// TODO: 实现获取事件功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取PV事件功能尚未实现")
}

// GetPVUsage 获取PV使用情况
func (p *pvService) GetPVUsage(ctx context.Context, req *model.K8sPVUsageReq) (*model.K8sPVUsageInfo, error) {
	// TODO: 实现获取使用情况功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取PV使用情况功能尚未实现")
}

// ReclaimPV 回收PV
func (p *pvService) ReclaimPV(ctx context.Context, clusterID int, name string) error {
	// TODO: 实现PV回收功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "PV回收功能尚未实现")
}

// convertPVToEntity 将Kubernetes PV对象转换为实体模型
func (p *pvService) convertPVToEntity(pv *corev1.PersistentVolume, clusterID int) *model.K8sPVEntity {
	// 获取容量
	capacity := ""
	if pv.Spec.Capacity != nil {
		if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = storage.String()
		}
	}

	// 获取访问模式
	accessModes := make([]string, 0, len(pv.Spec.AccessModes))
	for _, mode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取回收策略
	reclaimPolicy := string(pv.Spec.PersistentVolumeReclaimPolicy)

	// 获取存储类
	storageClass := pv.Spec.StorageClassName

	// 计算年龄
	age := pkg.GetAge(pv.CreationTimestamp.Time)

	// 获取绑定信息
	claimRef := make(map[string]string)
	if pv.Spec.ClaimRef != nil {
		claimRef["namespace"] = pv.Spec.ClaimRef.Namespace
		claimRef["name"] = pv.Spec.ClaimRef.Name
	}

	return &model.K8sPVEntity{
		Name:              pv.Name,
		ClusterID:         clusterID,
		UID:               string(pv.UID),
		Capacity:          capacity,
		AccessModes:       accessModes,
		ReclaimPolicy:     reclaimPolicy,
		Status:            string(pv.Status.Phase),
		StorageClass:      storageClass,
		VolumeMode:        string(*pv.Spec.VolumeMode),
		ClaimRef:          claimRef,
		Labels:            pv.Labels,
		Annotations:       pv.Annotations,
		CreationTimestamp: pv.CreationTimestamp.Time,
		Age:               age,
	}
}
