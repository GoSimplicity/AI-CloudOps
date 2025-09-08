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

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVCService interface {
	// 获取PVC列表
	GetPVCList(ctx context.Context, req *model.K8sPVCListReq) ([]*model.K8sPVCEntity, error)
	GetPVCsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPVCEntity, error)

	// 获取PVC详情
	GetPVC(ctx context.Context, clusterID int, namespace, name string) (*model.K8sPVCEntity, error)
	GetPVCYaml(ctx context.Context, clusterID int, namespace, name string) (string, error)

	// PVC操作
	CreatePVC(ctx context.Context, req *model.K8sPVCCreateReq) error
	UpdatePVC(ctx context.Context, req *model.K8sPVCUpdateReq) error
	// YAML相关方法
	CreatePVCByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error
	UpdatePVCByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error
	DeletePVC(ctx context.Context, req *model.K8sPVCDeleteReq) error

	// 批量操作

	// 高级功能（TODO实现）
	GetPVCEvents(ctx context.Context, req *model.K8sPVCEventReq) ([]*model.K8sEvent, error)
	GetPVCUsage(ctx context.Context, req *model.K8sPVCUsageReq) (*model.K8sPVCUsageInfo, error)
	ExpandPVC(ctx context.Context, req *model.K8sPVCExpandReq) error
}

type pvcService struct {
	dao        dao.ClusterDAO     // 保持对DAO的依赖
	client     client.K8sClient   // 保持向后兼容
	pvcManager manager.PVCManager // 新的依赖注入
	logger     *zap.Logger
}

// NewPVCService 创建新的 PVCService 实例
func NewPVCService(dao dao.ClusterDAO, client client.K8sClient, pvcManager manager.PVCManager, logger *zap.Logger) PVCService {
	return &pvcService{
		dao:        dao,
		client:     client,
		pvcManager: pvcManager,
		logger:     logger,
	}
}

// GetPVCList 获取PVC列表
func (p *pvcService) GetPVCList(ctx context.Context, req *model.K8sPVCListReq) ([]*model.K8sPVCEntity, error) {
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

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).List(ctx, listOptions)
	if err != nil {
		p.logger.Error("获取PVC列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PVC列表失败")
	}

	entities := make([]*model.K8sPVCEntity, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		entity := p.convertPVCToEntity(&pvc, req.ClusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetPVCsByNamespace 根据命名空间获取PVC列表
func (p *pvcService) GetPVCsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPVCEntity, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取PVC列表失败",
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sPVCEntity, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		entity := p.convertPVCToEntity(&pvc, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetPVC 获取单个PVC详情
func (p *pvcService) GetPVC(ctx context.Context, clusterID int, namespace, name string) (*model.K8sPVCEntity, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取PVC详情失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC详情失败")
	}

	return p.convertPVCToEntity(pvc, clusterID), nil
}

// GetPVCYaml 获取PVC的YAML
func (p *pvcService) GetPVCYaml(ctx context.Context, clusterID int, namespace, name string) (string, error) {
	kubeClient, err := p.client.GetKubeClient(clusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取PVC失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC失败")
	}

	yamlData, err := yaml.Marshal(pvc)
	if err != nil {
		p.logger.Error("序列化PVC为YAML失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化PVC为YAML失败")
	}

	return string(yamlData), nil
}

// CreatePVC 创建PVC
func (p *pvcService) CreatePVC(ctx context.Context, req *model.K8sPVCCreateReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.PVCYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC YAML不能为空")
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Create(ctx, req.PVCYaml, metav1.CreateOptions{})
	if err != nil {
		p.logger.Error("创建PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PVC失败")
	}

	p.logger.Info("成功创建PVC",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdatePVC 更新PVC
func (p *pvcService) UpdatePVC(ctx context.Context, req *model.K8sPVCUpdateReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.PVCYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC YAML不能为空")
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Update(ctx, req.PVCYaml, metav1.UpdateOptions{})
	if err != nil {
		p.logger.Error("更新PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PVC失败")
	}

	p.logger.Info("成功更新PVC",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeletePVC 删除PVC
func (p *pvcService) DeletePVC(ctx context.Context, req *model.K8sPVCDeleteReq) error {
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		p.logger.Error("删除PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除PVC失败")
	}

	p.logger.Info("成功删除PVC",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// GetPVCEvents 获取PVC事件
func (p *pvcService) GetPVCEvents(ctx context.Context, req *model.K8sPVCEventReq) ([]*model.K8sEvent, error) {
	if req == nil {
		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}
	fieldSelector := fmt.Sprintf("involvedObject.kind=PersistentVolumeClaim,involvedObject.name=%s", req.Name)
	events, err := kubeClient.CoreV1().Events(req.Namespace).List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		p.logger.Error("获取PVC事件失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PVC事件失败")
	}
	var result []*model.K8sEvent
	for _, ev := range events.Items {
		converted := &model.K8sEvent{
			Name:           ev.Name,
			Namespace:      ev.Namespace,
			UID:            string(ev.UID),
			ClusterID:      req.ClusterID,
			Message:        ev.Message,
			FirstTimestamp: ev.FirstTimestamp.Time,
			LastTimestamp:  ev.LastTimestamp.Time,
			Count:          int64(ev.Count),
			InvolvedObject: model.InvolvedObject{Kind: ev.InvolvedObject.Kind, Name: ev.InvolvedObject.Name, Namespace: ev.InvolvedObject.Namespace, UID: string(ev.InvolvedObject.UID), APIVersion: ev.InvolvedObject.APIVersion, FieldPath: ev.InvolvedObject.FieldPath},
			Source:         model.EventSource{Component: ev.Source.Component, Host: ev.Source.Host},
		}
		if ev.Type == "Warning" {
			converted.Type = model.EventTypeWarning
		} else {
			converted.Type = model.EventTypeNormal
		}
		result = append(result, converted)
	}
	return result, nil
}

// GetPVCUsage 获取PVC使用情况
func (p *pvcService) GetPVCUsage(ctx context.Context, req *model.K8sPVCUsageReq) (*model.K8sPVCUsageInfo, error) {
	if req == nil {
		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	return &model.K8sPVCUsageInfo{Total: "", Used: "", Available: "", UsageRate: 0}, nil
}

// ExpandPVC 扩容PVC
func (p *pvcService) ExpandPVC(ctx context.Context, req *model.K8sPVCExpandReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	if req.ClusterID <= 0 || req.Namespace == "" || req.Name == "" || req.NewCapacity == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "参数无效")
	}
	return p.pvcManager.ExpandPVC(ctx, req.ClusterID, req.Namespace, req.Name, req.NewCapacity)
}

// convertPVCToEntity 将Kubernetes PVC对象转换为实体模型
func (p *pvcService) convertPVCToEntity(pvc *corev1.PersistentVolumeClaim, clusterID int) *model.K8sPVCEntity {
	// 获取请求容量
	requestedStorage := ""
	if pvc.Spec.Resources.Requests != nil {
		if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			requestedStorage = storage.String()
		}
	}

	// 获取实际容量
	actualStorage := ""
	if pvc.Status.Capacity != nil {
		if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			actualStorage = storage.String()
		}
	}

	// 获取存储类
	storageClass := ""
	if pvc.Spec.StorageClassName != nil {
		storageClass = *pvc.Spec.StorageClassName
	}

	// 获取访问模式
	accessModes := make([]string, 0, len(pvc.Spec.AccessModes))
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取卷模式
	volumeMode := ""
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	// 计算年龄
	age := pkg.GetAge(pvc.CreationTimestamp.Time)

	return &model.K8sPVCEntity{
		Name:              pvc.Name,
		Namespace:         pvc.Namespace,
		ClusterID:         clusterID,
		UID:               string(pvc.UID),
		Status:            string(pvc.Status.Phase),
		RequestStorage:    requestedStorage,
		Capacity:          actualStorage,
		StorageClass:      storageClass,
		AccessModes:       accessModes,
		VolumeMode:        volumeMode,
		VolumeName:        pvc.Spec.VolumeName,
		Labels:            pvc.Labels,
		Annotations:       pvc.Annotations,
		CreationTimestamp: pvc.CreationTimestamp.Time,
		Age:               age,
	}
}

// CreatePVCByYaml 通过YAML创建PVC
func (p *pvcService) CreatePVCByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "通过YAML创建PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	pvc, err := k8sutils.YAMLToPVC(req.YAML)
	if err != nil {
		p.logger.Error("解析PVC YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if pvc.Namespace == "" {
		pvc.Namespace = req.Namespace
	}
	if err := k8sutils.ValidatePVC(pvc); err != nil {
		p.logger.Error("PVC配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := p.pvcManager.CreatePVC(ctx, req.ClusterID, pvc.Namespace, pvc); err != nil {
		p.logger.Error("创建PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", pvc.Namespace), zap.String("name", pvc.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PVC失败")
	}
	return nil
}

// UpdatePVCByYaml 通过YAML更新PVC
func (p *pvcService) UpdatePVCByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "通过YAML更新PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	if req.Name == "" || req.Namespace == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "资源名称与命名空间不能为空")
	}
	desired, err := k8sutils.YAMLToPVC(req.YAML)
	if err != nil {
		p.logger.Error("解析PVC YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if desired.Name == "" {
		desired.Name = req.Name
	}
	if desired.Namespace == "" {
		desired.Namespace = req.Namespace
	}
	if desired.Name != req.Name || desired.Namespace != req.Namespace {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "请求的名称/命名空间与YAML不一致")
	}
	if err := k8sutils.ValidatePVC(desired); err != nil {
		p.logger.Error("PVC配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := p.pvcManager.UpdatePVC(ctx, req.ClusterID, req.Namespace, desired); err != nil {
		p.logger.Error("更新PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PVC失败")
	}
	return nil
}
