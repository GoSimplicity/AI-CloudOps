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

	"strings"

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
	GetPVList(ctx context.Context, req *model.GetPVListReq) (model.ListResp[*model.K8sPV], error)
	GetPVsByCluster(ctx context.Context, clusterID int) ([]*model.K8sPV, error)

	// 获取PV详情
	GetPV(ctx context.Context, clusterID int, name string) (*model.K8sPV, error)
	GetPVYaml(ctx context.Context, clusterID int, name string) (string, error)

	// PV操作
	CreatePV(ctx context.Context, req *model.CreatePVReq) error
	UpdatePV(ctx context.Context, req *model.UpdatePVReq) error
	// YAML相关方法
	CreatePVByYaml(ctx context.Context, req *model.CreatePVByYamlReq) error
	UpdatePVByYaml(ctx context.Context, req *model.UpdatePVByYamlReq) error
	DeletePV(ctx context.Context, req *model.DeletePVReq) error

	// 高级功能
	ReclaimPV(ctx context.Context, req *model.ReclaimPVReq) error
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
func (s *pvService) GetPVList(ctx context.Context, req *model.GetPVListReq) (model.ListResp[*model.K8sPV], error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.K8sPV]{}, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := utils.BuildPVListOptions(req)

	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取PV列表失败", zap.Error(err))
		return model.ListResp[*model.K8sPV]{}, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PV列表失败")
	}

	entities := make([]*model.K8sPV, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		entity := s.convertPVToEntity(&pv, req.ClusterID)
		if entity != nil {
			entities = append(entities, entity)
		}
	}
	// optional filters
	filtered := make([]*model.K8sPV, 0, len(entities))
	for _, e := range entities {
		// 过滤状态
		if req.Status != "" {
			statusStr := s.pvStatusToString(e.Status)
			if !strings.EqualFold(statusStr, req.Status) {
				continue
			}
		}
		// 过滤访问模式
		if req.AccessMode != "" {
			ok := false
			for _, m := range e.AccessModes {
				if strings.EqualFold(m, req.AccessMode) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		// 过滤卷类型
		if req.VolumeType != "" && !strings.EqualFold(e.VolumeMode, req.VolumeType) {
			continue
		}
		filtered = append(filtered, e)
	}
	// pagination
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	start := (page - 1) * size
	end := start + size
	total := int64(len(filtered))
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}
	return model.ListResp[*model.K8sPV]{Items: filtered[start:end], Total: total}, nil
}

// GetPVsByCluster 根据集群获取PV列表
func (s *pvService) GetPVsByCluster(ctx context.Context, clusterID int) ([]*model.K8sPV, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取PV列表失败", zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sPV, 0, len(pvs.Items))
	for _, pv := range pvs.Items {
		entity := s.convertPVToEntity(&pv, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetPV 获取单个PV详情
func (s *pvService) GetPV(ctx context.Context, clusterID int, name string) (*model.K8sPV, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PV详情失败",
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PV详情失败")
	}

	return s.convertPVToEntity(pv, clusterID), nil
}

// GetPVYaml 获取PV的YAML
func (s *pvService) GetPVYaml(ctx context.Context, clusterID int, name string) (string, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PV失败",
			zap.String("name", name),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PV失败")
	}

	yamlData, err := yaml.Marshal(pv)
	if err != nil {
		s.logger.Error("序列化PV为YAML失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化PV为YAML失败")
	}

	return string(yamlData), nil
}

// CreatePV 创建PV
func (s *pvService) CreatePV(ctx context.Context, req *model.CreatePVReq) error {
	// 将请求转换为 Kubernetes PV 对象
	pv := utils.ConvertCreatePVReqToPV(req)
	return s.pvManager.CreatePV(ctx, req.ClusterID, pv)
}

// UpdatePV 更新PV
func (s *pvService) UpdatePV(ctx context.Context, req *model.UpdatePVReq) error {
	// 先获取现有的PV对象
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	existingPV, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取现有PV失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取现有PV失败")
	}

	// 基于现有PV对象更新可变字段
	pv := utils.ConvertUpdatePVReqToPV(req, existingPV)
	if pv == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "无效的更新请求")
	}

	return s.pvManager.UpdatePV(ctx, req.ClusterID, pv)
}

// DeletePV 删除PV
func (s *pvService) DeletePV(ctx context.Context, req *model.DeletePVReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.CoreV1().PersistentVolumes().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除PV失败",
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除PV失败")
	}

	s.logger.Info("成功删除PV", zap.String("name", req.Name))
	return nil
}

// ReclaimPV 回收PV
func (s *pvService) ReclaimPV(ctx context.Context, req *model.ReclaimPVReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "请求不能为空")
	}
	if req.ClusterID <= 0 || req.Name == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "参数无效")
	}
	return s.pvManager.ReclaimPV(ctx, req.ClusterID, req.Name)
}

// convertPVToEntity 将Kubernetes PV对象转换为实体模型
func (s *pvService) convertPVToEntity(pv *corev1.PersistentVolume, clusterID int) *model.K8sPV {
	if pv == nil {
		return nil
	}

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

	// 转换状态为枚举类型
	status := s.convertPVStatus(pv.Status.Phase)

	// 获取卷模式
	volumeMode := string(corev1.PersistentVolumeFilesystem)
	if pv.Spec.VolumeMode != nil {
		volumeMode = string(*pv.Spec.VolumeMode)
	}

	// 获取绑定信息
	claimRef := make(map[string]string)
	if pv.Spec.ClaimRef != nil {
		claimRef["namespace"] = pv.Spec.ClaimRef.Namespace
		claimRef["name"] = pv.Spec.ClaimRef.Name
		claimRef["uid"] = string(pv.Spec.ClaimRef.UID)
	}

	// 获取卷源配置
	volumeSource := make(map[string]interface{})
	if pv.Spec.PersistentVolumeSource.HostPath != nil {
		volumeSource["hostPath"] = map[string]interface{}{
			"path": pv.Spec.PersistentVolumeSource.HostPath.Path,
			"type": pv.Spec.PersistentVolumeSource.HostPath.Type,
		}
	}
	// 其他卷源类型可以根据需要添加

	// 获取节点亲和性
	nodeAffinity := make(map[string]interface{})
	if pv.Spec.NodeAffinity != nil && pv.Spec.NodeAffinity.Required != nil {
		// 简化节点亲和性处理，可根据需要扩展
		nodeAffinity["required"] = "true"
	}

	// 计算年龄
	age := utils.GetPVAge(*pv)

	// 获取资源版本
	resourceVersion := pv.ResourceVersion

	return &model.K8sPV{
		Name:            pv.Name,
		ClusterID:       clusterID,
		UID:             string(pv.UID),
		Capacity:        capacity,
		AccessModes:     accessModes,
		ReclaimPolicy:   reclaimPolicy,
		StorageClass:    storageClass,
		VolumeMode:      volumeMode,
		Status:          status,
		ClaimRef:        claimRef,
		VolumeSource:    volumeSource,
		NodeAffinity:    nodeAffinity,
		Labels:          pv.Labels,
		Annotations:     pv.Annotations,
		ResourceVersion: resourceVersion,
		CreatedAt:       pv.CreationTimestamp.Time,
		Age:             age,
		RawPV:           pv,
	}
}

// convertPVStatus 转换PV状态为枚举类型
func (s *pvService) convertPVStatus(phase corev1.PersistentVolumePhase) model.K8sPVStatus {
	switch phase {
	case corev1.VolumeAvailable:
		return model.K8sPVStatusAvailable
	case corev1.VolumeBound:
		return model.K8sPVStatusBound
	case corev1.VolumeReleased:
		return model.K8sPVStatusReleased
	case corev1.VolumeFailed:
		return model.K8sPVStatusFailed
	default:
		return model.K8sPVStatusUnknown
	}
}

// pvStatusToString 将PV状态枚举转换为字符串
func (s *pvService) pvStatusToString(status model.K8sPVStatus) string {
	switch status {
	case model.K8sPVStatusAvailable:
		return "Available"
	case model.K8sPVStatusBound:
		return "Bound"
	case model.K8sPVStatusReleased:
		return "Released"
	case model.K8sPVStatusFailed:
		return "Failed"
	default:
		return "Unknown"
	}
}

// CreatePVByYaml 通过YAML创建PV
func (s *pvService) CreatePVByYaml(ctx context.Context, req *model.CreatePVByYamlReq) error {
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
		s.logger.Error("解析PV YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if err := utils.ValidatePV(pv); err != nil {
		s.logger.Error("PV配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV配置验证失败")
	}
	if err := s.pvManager.CreatePV(ctx, req.ClusterID, pv); err != nil {
		s.logger.Error("创建PV失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", pv.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PV失败")
	}
	return nil
}

// UpdatePVByYaml 通过YAML更新PV
func (s *pvService) UpdatePVByYaml(ctx context.Context, req *model.UpdatePVByYamlReq) error {
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
		s.logger.Error("解析PV YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if desired.Name == "" {
		desired.Name = req.Name
	}
	if desired.Name != req.Name {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "请求名称与YAML不一致")
	}
	if err := utils.ValidatePV(desired); err != nil {
		s.logger.Error("PV配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PV配置验证失败")
	}
	if err := s.pvManager.UpdatePV(ctx, req.ClusterID, desired); err != nil {
		s.logger.Error("更新PV失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PV失败")
	}
	return nil
}
