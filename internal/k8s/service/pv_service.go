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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
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
	// YAML相关方法
	CreatePVByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error
	UpdatePVByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error
	DeletePV(ctx context.Context, req *model.K8sPVDeleteReq) error

	// 批量操作

	// 高级功能（TODO实现）
	GetPVEvents(ctx context.Context, req *model.K8sPVEventReq) ([]*model.K8sEvent, error)
	GetPVUsage(ctx context.Context, req *model.K8sPVUsageReq) (*model.K8sPVUsageInfo, error)
	ReclaimPV(ctx context.Context, clusterID int, name string) error
}

type pvService struct {
	dao       dao.ClusterDAO    // 保持对DAO的依赖
	client    client.K8sClient  // 保持向后兼容
	pvManager manager.PVManager // 新的依赖注入
	logger    *zap.Logger
}

// NewPVService 创建新的 PVService 实例
func NewPVService(dao dao.ClusterDAO, client client.K8sClient, pvManager manager.PVManager, logger *zap.Logger) PVService {
	return &pvService{
		dao:       dao,
		client:    client,
		pvManager: pvManager,
		logger:    logger,
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

// GetPVEvents 获取PV事件
func (p *pvService) GetPVEvents(ctx context.Context, req *model.K8sPVEventReq) ([]*model.K8sEvent, error) {
	if req == nil {
		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	kubeClient, err := p.client.GetKubeClient(req.ClusterID)
	if err != nil {
		p.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}
	fieldSelector := fmt.Sprintf("involvedObject.kind=PersistentVolume,involvedObject.name=%s", req.Name)
	events, err := kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{FieldSelector: fieldSelector})
	if err != nil {
		p.logger.Error("获取PV事件失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PV事件失败")
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

// GetPVUsage 获取PV使用情况
func (p *pvService) GetPVUsage(ctx context.Context, req *model.K8sPVUsageReq) (*model.K8sPVUsageInfo, error) {
	if req == nil {
		return nil, pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	return &model.K8sPVUsageInfo{Total: "", Used: "", Available: "", UsageRate: 0}, nil
}

// ReclaimPV 回收PV
func (p *pvService) ReclaimPV(ctx context.Context, clusterID int, name string) error {
	if clusterID <= 0 || name == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "参数无效")
	}
	return p.pvManager.ReclaimPV(ctx, clusterID, name)
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

// CreatePVByYaml 通过YAML创建PV
func (p *pvService) CreatePVByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "通过YAML创建PV请求不能为空")
	}
	if req.ClusterID <= 0 {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	pv, err := utils.YAMLToPV(req.YAML)
	if err != nil {
		p.logger.Error("解析PV YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if err := utils.ValidatePV(pv); err != nil {
		p.logger.Error("PV配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV配置验证失败")
	}
	if err := p.pvManager.CreatePV(ctx, req.ClusterID, pv); err != nil {
		p.logger.Error("创建PV失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", pv.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PV失败")
	}
	return nil
}

// UpdatePVByYaml 通过YAML更新PV
func (p *pvService) UpdatePVByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "通过YAML更新PV请求不能为空")
	}
	if req.ClusterID <= 0 {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	desired, err := utils.YAMLToPV(req.YAML)
	if err != nil {
		p.logger.Error("解析PV YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if desired.Name == "" {
		desired.Name = req.Name
	}
	if desired.Name != req.Name {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "请求名称与YAML不一致")
	}
	if err := utils.ValidatePV(desired); err != nil {
		p.logger.Error("PV配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV配置验证失败")
	}
	if err := p.pvManager.UpdatePV(ctx, req.ClusterID, desired); err != nil {
		p.logger.Error("更新PV失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PV失败")
	}
	return nil
}
