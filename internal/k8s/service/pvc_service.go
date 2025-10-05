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
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVCService interface {
	// 获取PVC列表
	GetPVCList(ctx context.Context, req *model.GetPVCListReq) (model.ListResp[*model.K8sPVC], error)
	GetPVCsByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sPVC, error)

	// 获取PVC详情
	GetPVC(ctx context.Context, req *model.GetPVCDetailsReq) (*model.K8sPVC, error)
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

// NewPVCService 创建新的 PVCService 实例
func NewPVCService(dao dao.ClusterDAO, client client.K8sClient, pvcManager manager.PVCManager, logger *zap.Logger) PVCService {
	return &pvcService{
		dao:        dao,
		client:     client,
		pvcManager: pvcManager,
		logger:     logger,
	}
}

func (s *pvcService) GetPVCList(ctx context.Context, req *model.GetPVCListReq) (model.ListResp[*model.K8sPVC], error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.K8sPVC]{}, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	listOptions := k8sutils.BuildPVCListOptions(req)

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).List(ctx, listOptions)
	if err != nil {
		s.logger.Error("获取PVC列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return model.ListResp[*model.K8sPVC]{}, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取PVC列表失败")
	}

	entities := make([]*model.K8sPVC, 0, len(pvcs.Items))
	for _, pvc := range pvcs.Items {
		entity := s.convertPVCToEntity(&pvc, req.ClusterID)
		entities = append(entities, entity)
	}
	// filters
	filtered := make([]*model.K8sPVC, 0, len(entities))
	for _, e := range entities {
		// 过滤状态
		if req.Status != "" {
			statusStr := s.pvcStatusToString(e.Status)
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

func (s *pvcService) GetPVC(ctx context.Context, req *model.GetPVCDetailsReq) (*model.K8sPVC, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PVC详情失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC详情失败")
	}

	return s.convertPVCToEntity(pvc, req.ClusterID), nil
}

func (s *pvcService) GetPVCYaml(ctx context.Context, req *model.GetPVCYamlReq) (*model.K8sYaml, error) {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取PVC失败")
	}

	yamlData, err := yaml.Marshal(pvc)
	if err != nil {
		s.logger.Error("序列化PVC为YAML失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化PVC为YAML失败")
	}

	return &model.K8sYaml{YAML: string(yamlData)}, nil
}

func (s *pvcService) CreatePVC(ctx context.Context, req *model.CreatePVCReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 将请求转换为 Kubernetes PVC 对象
	pvc := k8sutils.ConvertCreatePVCReqToPVC(req)
	if pvc == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置转换失败")
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PVC失败")
	}

	s.logger.Info("成功创建PVC",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

func (s *pvcService) UpdatePVC(ctx context.Context, req *model.UpdatePVCReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 将请求转换为 Kubernetes PVC 对象
	pvc := k8sutils.ConvertUpdatePVCReqToPVC(req)
	if pvc == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置转换失败")
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("更新PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PVC失败")
	}

	s.logger.Info("成功更新PVC",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

func (s *pvcService) DeletePVC(ctx context.Context, req *model.DeletePVCReq) error {
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除PVC失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除PVC失败")
	}

	s.logger.Info("成功删除PVC",
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
		s.logger.Error("解析PVC YAML失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceOperation, "解析YAML失败")
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
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := s.pvcManager.CreatePVC(ctx, req.ClusterID, pvc.Namespace, pvc); err != nil {
		s.logger.Error("创建PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", pvc.Namespace), zap.String("name", pvc.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建PVC失败")
	}
	return nil
}

func (s *pvcService) UpdatePVCByYaml(ctx context.Context, req *model.UpdatePVCByYamlReq) error {
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
		s.logger.Error("解析PVC YAML失败", zap.Error(err))
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
		s.logger.Error("PVC配置验证失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrInvalidParam, "PVC配置验证失败")
	}
	if err := s.pvcManager.UpdatePVC(ctx, req.ClusterID, req.Namespace, desired); err != nil {
		s.logger.Error("更新PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新PVC失败")
	}
	return nil
}

// ExpandPVC 扩容PVC
func (s *pvcService) ExpandPVC(ctx context.Context, req *model.ExpandPVCReq) error {
	if req == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "扩容PVC请求不能为空")
	}
	if req.ClusterID <= 0 {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.Namespace == "" || req.Name == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "命名空间和PVC名称不能为空")
	}
	if req.NewCapacity == "" {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "新容量不能为空")
	}

	if err := s.pvcManager.ExpandPVC(ctx, req.ClusterID, req.Namespace, req.Name, req.NewCapacity); err != nil {
		s.logger.Error("扩容PVC失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "扩容PVC失败")
	}

	s.logger.Info("成功扩容PVC", zap.Int("clusterID", req.ClusterID), zap.String("namespace", req.Namespace), zap.String("name", req.Name), zap.String("newCapacity", req.NewCapacity))
	return nil
}

func (s *pvcService) GetPVCPods(ctx context.Context, req *model.GetPVCPodsReq) (model.ListResp[*model.K8sPod], error) {
	if req == nil {
		return model.ListResp[*model.K8sPod]{}, pkg.NewBusinessError(constants.ErrInvalidParam, "获取PVC Pods请求不能为空")
	}
	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sPod]{}, pkg.NewBusinessError(constants.ErrInvalidParam, "集群ID不能为空")
	}
	if req.Namespace == "" || req.Name == "" {
		return model.ListResp[*model.K8sPod]{}, pkg.NewBusinessError(constants.ErrInvalidParam, "命名空间和PVC名称不能为空")
	}

	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return model.ListResp[*model.K8sPod]{}, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	// 获取所有Pod，然后过滤使用指定PVC的Pod
	pods, err := kubeClient.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取Pod列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return model.ListResp[*model.K8sPod]{}, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Pod列表失败")
	}

	// 过滤使用指定PVC的Pod
	var filteredPods []*model.K8sPod
	s.logger.Info("开始过滤使用指定PVC的Pod",
		zap.String("pvc_name", req.Name),
		zap.String("namespace", req.Namespace),
		zap.Int("total_pods", len(pods.Items)))

	for _, pod := range pods.Items {
		if s.podUsesPVC(&pod, req.Name) {
			s.logger.Info("找到使用PVC的Pod",
				zap.String("pod_name", pod.Name),
				zap.String("pvc_name", req.Name))
			k8sPod := s.convertPodToEntity(&pod, req.ClusterID)
			if k8sPod != nil {
				filteredPods = append(filteredPods, k8sPod)
			}
		}
	}

	s.logger.Info("PVC关联Pod查询完成",
		zap.String("pvc_name", req.Name),
		zap.Int("filtered_pods_count", len(filteredPods)))

	return model.ListResp[*model.K8sPod]{
		Items: filteredPods,
		Total: int64(len(filteredPods)),
	}, nil
}

// podUsesPVC 检查Pod是否使用指定的PVC
func (s *pvcService) podUsesPVC(pod *corev1.Pod, pvcName string) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvcName {
			return true
		}
	}
	return false
}

// convertPodToEntity 将Kubernetes Pod对象转换为实体模型
func (s *pvcService) convertPodToEntity(pod *corev1.Pod, clusterID int) *model.K8sPod {
	if pod == nil {
		return nil
	}

	k8sPod := k8sutils.ConvertToK8sPod(pod)
	if k8sPod != nil {
		// 设置集群ID
		k8sPod.ClusterID = int64(clusterID)
	}

	return k8sPod
}
