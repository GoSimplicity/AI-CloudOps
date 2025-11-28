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
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVCService interface {
	GetPVCList(ctx context.Context, req *model.GetPVCListReq) (model.ListResp[*model.K8sPVC], error)
	GetPVCsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPVC, error)

	GetPVCDetails(ctx context.Context, req *model.GetPVCDetailsReq) (*model.K8sPVC, error)
	GetPVCYaml(ctx context.Context, req *model.GetPVCYamlReq) (*model.K8sYaml, error)

	CreatePVC(ctx context.Context, req *model.CreatePVCReq) error
	UpdatePVC(ctx context.Context, req *model.UpdatePVCReq) error
	// YAML相关方法
	CreatePVCByYaml(ctx context.Context, req *model.CreatePVCByYamlReq) error
	UpdatePVCByYaml(ctx context.Context, req *model.UpdatePVCByYamlReq) error
	DeletePVC(ctx context.Context, req *model.DeletePVCReq) error
	ExpandPVC(ctx context.Context, req *model.ExpandPVCReq) error
	GetPVCPods(ctx context.Context, req *model.GetPVCPodsReq) (model.ListResp[*model.K8sPod], error)
}

type pvcService struct {
	dao        dao.ClusterDAO     // 保持对DAO的依赖
	client     client.K8sClient   // 保持向后兼容
	pvcManager manager.PVCManager // 新的依赖注入
	logger     *zap.Logger
}

func NewPVCService(dao dao.ClusterDAO, client client.K8sClient, pvcManager manager.PVCManager, logger *zap.Logger) PVCService {
	return &pvcService{
		dao:        dao,
		client:     client,
		pvcManager: pvcManager,
		logger:     logger,
	}
}

func (s *pvcService) GetPVCList(ctx context.Context, req *model.GetPVCListReq) (model.ListResp[*model.K8sPVC], error) {
	if req == nil {
		return model.ListResp[*model.K8sPVC]{}, fmt.Errorf("获取PVC列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPVC]{}, fmt.Errorf("集群ID不能为空")
	}

	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return model.ListResp[*model.K8sPVC]{}, base.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := k8sutils.BuildPVCListOptions(req)

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取PVC列表失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return model.ListResp[*model.K8sPVC]{}, base.NewBusinessError(constants.ErrK8sResourceList, "获取PVC列表失败")
	}

	entities := make([]*model.K8sPVC, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		entity := s.convertPVCToEntity(&pvc, req.ClusterID)
		entities = append(entities, entity)
	}
	// 应用过滤条件
	filtered := make([]*model.K8sPVC, 0, len(entities))
	for _, e := range entities {
		// 过滤状态 (0表示不过滤，其他值表示具体状态)
		if req.Status != 0 && req.Status != e.Status {
			continue
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
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !k8sutils.FilterByName(e.Name, req.Search) {
			continue
		}
		filtered = append(filtered, e)
	}

	// 按创建时间排序（最新的在前）
	k8sutils.SortByCreationTime(filtered, func(pvc *model.K8sPVC) time.Time {
		return pvc.CreatedAt
	})

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
	return model.ListResp[*model.K8sPVC]{Items: filtered[start:end], Total: total}, nil
}

func (s *pvcService) GetPVCsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPVC, error) {
	kubeClient, err := s.client.GetKubeClient(clusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取PVC列表失败",
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sPVC, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		entity := s.convertPVCToEntity(&pvc, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

func (s *pvcService) GetPVCDetails(ctx context.Context, req *model.GetPVCDetailsReq) (*model.K8sPVC, error) {
	if req == nil {
		return nil, fmt.Errorf("获取PVC详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("PVC名称不能为空")
	}

	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, base.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PVC详情失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return nil, base.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC详情失败")
	}

	return s.convertPVCToEntity(pvc, req.ClusterID), nil
}

func (s *pvcService) GetPVCYaml(ctx context.Context, req *model.GetPVCYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取PVC YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("PVC名称不能为空")
	}

	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, base.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PVC失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return nil, base.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC失败")
	}

	yamlContent, err := k8sutils.PVCToYAML(pvc)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("pvcName", pvc.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *pvcService) CreatePVC(ctx context.Context, req *model.CreatePVCReq) error {
	if req == nil {
		return fmt.Errorf("创建PVC请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("PVC名称不能为空")
	}

	// 将请求转换为 Kubernetes PVC 对象
	pvc := k8sutils.ConvertCreatePVCReqToPVC(req)
	if pvc == nil {
		s.logger.Error("构建PVC对象失败",
			zap.String("name", req.Name),
			zap.String("namespace", req.Namespace))
		return base.NewBusinessError(constants.ErrInvalidParam, "PVC配置转换失败")
	}

	if err := k8sutils.ValidatePVC(pvc); err != nil {
		s.logger.Error("PVC配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.String("namespace", req.Namespace))
		return fmt.Errorf("PVC配置验证失败: %w", err)
	}

	err := s.pvcManager.CreatePVC(ctx, req.ClusterID, req.Namespace, pvc)
	if err != nil {
		s.logger.Error("创建PVC失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建PVC失败: %w", err)
	}

	s.logger.Info("成功创建PVC",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

func (s *pvcService) UpdatePVC(ctx context.Context, req *model.UpdatePVCReq) error {
	if req == nil {
		return fmt.Errorf("更新PVC请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("PVC名称不能为空")
	}

	// 将请求转换为 Kubernetes PVC 对象
	pvc := k8sutils.ConvertUpdatePVCReqToPVC(req)
	if pvc == nil {
		s.logger.Error("转换PVC更新请求失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return base.NewBusinessError(constants.ErrInvalidParam, "PVC配置转换失败")
	}

	if err := k8sutils.ValidatePVC(pvc); err != nil {
		s.logger.Error("PVC配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name),
			zap.String("namespace", req.Namespace))
		return fmt.Errorf("PVC配置验证失败: %w", err)
	}

	err := s.pvcManager.UpdatePVC(ctx, req.ClusterID, req.Namespace, pvc)
	if err != nil {
		s.logger.Error("更新PVC失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新PVC失败: %w", err)
	}

	s.logger.Info("成功更新PVC",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

func (s *pvcService) DeletePVC(ctx context.Context, req *model.DeletePVCReq) error {
	if req == nil {
		return fmt.Errorf("删除PVC请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("PVC名称不能为空")
	}

	err := s.pvcManager.DeletePVC(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除PVC失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("删除PVC失败: %w", err)
	}

	s.logger.Info("成功删除PVC",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// convertPVCToEntity 将Kubernetes PVC对象转换为实体模型
func (s *pvcService) convertPVCToEntity(pvc *corev1.PersistentVolumeClaim, clusterID int) *model.K8sPVC {
	if pvc == nil {
		return nil
	}

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
	volumeMode := string(corev1.PersistentVolumeFilesystem)
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	status := s.convertPVCStatus(pvc.Status.Phase)

	// 获取选择器
	selector := make(map[string]string)
	if pvc.Spec.Selector != nil && pvc.Spec.Selector.MatchLabels != nil {
		selector = pvc.Spec.Selector.MatchLabels
	}

	// 计算年龄
	age := k8sutils.GetPVCAge(*pvc)

	return &model.K8sPVC{
		Name:            pvc.Name,
		Namespace:       pvc.Namespace,
		ClusterID:       clusterID,
		UID:             string(pvc.UID),
		Capacity:        actualStorage,
		RequestStorage:  requestedStorage,
		AccessModes:     accessModes,
		StorageClass:    storageClass,
		VolumeMode:      volumeMode,
		Status:          status,
		VolumeName:      pvc.Spec.VolumeName,
		Selector:        selector,
		Labels:          pvc.Labels,
		Annotations:     pvc.Annotations,
		ResourceVersion: pvc.ResourceVersion,
		CreatedAt:       pvc.CreationTimestamp.Time,
		Age:             age,
		RawPVC:          pvc,
	}
}

// convertPVCStatus 转换PVC状态为枚举类型
func (s *pvcService) convertPVCStatus(phase corev1.PersistentVolumeClaimPhase) model.K8sPVCStatus {
	switch phase {
	case corev1.ClaimPending:
		return model.K8sPVCStatusPending
	case corev1.ClaimBound:
		return model.K8sPVCStatusBound
	case corev1.ClaimLost:
		return model.K8sPVCStatusLost
	default:
		return model.K8sPVCStatusUnknown
	}
}

// pvcStatusToString 将PVC状态枚举转换为字符串
func (s *pvcService) pvcStatusToString(status model.K8sPVCStatus) string {
	switch status {
	case model.K8sPVCStatusPending:
		return "Pending"
	case model.K8sPVCStatusBound:
		return "Bound"
	case model.K8sPVCStatusLost:
		return "Lost"
	case model.K8sPVCStatusTerminating:
		return "Terminating"
	default:
		return "Unknown"
	}
}

func (s *pvcService) CreatePVCByYaml(ctx context.Context, req *model.CreatePVCByYamlReq) error {
	if req == nil {
		return base.NewBusinessError(constants.ErrInvalidParam, "通过YAML创建PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return base.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return base.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	pvc, err := k8sutils.YAMLToPVC(req.YAML)
	if err != nil {
		s.logger.Error("解析PVC YAML失败", zap.Error(err))
		return base.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	// 如果YAML中没有指定namespace，使用default命名空间
	if pvc.Namespace == "" {
		pvc.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", pvc.Name))
	}
	if err := k8sutils.ValidatePVC(pvc); err != nil {
		s.logger.Error("PVC配置验证失败", zap.Error(err))
		return base.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := s.pvcManager.CreatePVC(ctx, req.ClusterID, pvc.Namespace, pvc); err != nil {
		s.logger.Error("创建PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", pvc.Namespace), zap.String("name", pvc.Name))
		return base.NewBusinessError(constants.ErrK8sResourceCreate, "创建PVC失败")
	}
	return nil
}

func (s *pvcService) UpdatePVCByYaml(ctx context.Context, req *model.UpdatePVCByYamlReq) error {
	if req == nil {
		return base.NewBusinessError(constants.ErrInvalidParam, "通过YAML更新PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return base.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.YAML == "" {
		return base.NewBusinessError(constants.ErrInvalidParam, "YAML内容不能为空")
	}
	if req.Name == "" || req.Namespace == "" {
		return base.NewBusinessError(constants.ErrInvalidParam, "资源名称与命名空间不能为空")
	}
	desired, err := k8sutils.YAMLToPVC(req.YAML)
	if err != nil {
		s.logger.Error("解析PVC YAML失败", zap.Error(err))
		return base.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
	}
	if desired.Name == "" {
		desired.Name = req.Name
	}
	if desired.Namespace == "" {
		desired.Namespace = req.Namespace
	}
	if desired.Name != req.Name || desired.Namespace != req.Namespace {
		return base.NewBusinessError(constants.ErrInvalidParam, "请求的名称/命名空间与YAML不一致")
	}
	if err := k8sutils.ValidatePVC(desired); err != nil {
		s.logger.Error("PVC配置验证失败", zap.Error(err))
		return base.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := s.pvcManager.UpdatePVC(ctx, req.ClusterID, req.Namespace, desired); err != nil {
		s.logger.Error("更新PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return base.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PVC失败")
	}
	return nil
}

// ExpandPVC 扩容PVC
func (s *pvcService) ExpandPVC(ctx context.Context, req *model.ExpandPVCReq) error {
	if req == nil {
		return base.NewBusinessError(constants.ErrInvalidParam, "扩容PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return base.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.Namespace == "" || req.Name == "" {
		return base.NewBusinessError(constants.ErrInvalidParam, "命名空间和PVC名称不能为空")
	}
	if req.NewCapacity == "" {
		return base.NewBusinessError(constants.ErrInvalidParam, "新容量不能为空")
	}

	if err := s.pvcManager.ExpandPVC(ctx, req.ClusterID, req.Namespace, req.Name, req.NewCapacity); err != nil {
		s.logger.Error("扩容PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return base.NewBusinessError(constants.ErrK8sResourceUpdate, err.Error())
	}

	s.logger.Info("成功扩容PVC", zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name), zap.String("newCapacity", req.NewCapacity))
	return nil
}

func (s *pvcService) GetPVCPods(ctx context.Context, req *model.GetPVCPodsReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, base.NewBusinessError(constants.ErrInvalidParam, "获取PVC Pods请求不能为空")
	}
	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, base.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.Namespace == "" || req.Name == "" {
		return model.ListResp[*model.K8sPod]{}, base.NewBusinessError(constants.ErrInvalidParam, "命名空间和PVC名称不能为空")
	}

	// 通过 Manager 层获取使用该 PVC 的所有 Pod
	pods, err := s.pvcManager.GetPVCPods(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取PVC关联的Pod失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("pvcName", req.Name))
		return model.ListResp[*model.K8sPod]{}, base.NewBusinessError(constants.ErrK8sResourceList, "获取PVC关联的Pod失败")
	}

	// 转换为 model.K8sPod
	k8sPods := make([]*model.K8sPod, 0, len(pods))
	for _, pod := range pods {
		k8sPod := k8sutils.ConvertToK8sPod(&pod)
		if k8sPod != nil {
			k8sPod.ClusterID = int64(req.ClusterID)
			k8sPods = append(k8sPods, k8sPod)
		}
	}

	s.logger.Info("成功获取PVC关联的Pod",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("pvcName", req.Name),
		zap.Int("count", len(k8sPods)))

	return model.ListResp[*model.K8sPod]{
		Items: k8sPods,
		Total: int64(len(k8sPods)),
	}, nil
}
