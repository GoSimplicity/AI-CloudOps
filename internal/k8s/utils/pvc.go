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

package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// ConvertToPVCEntity 将 Kubernetes PVC 转换为内部 PVC 模型
func ConvertToPVCEntity(pvc *corev1.PersistentVolumeClaim, clusterID int) *model.K8sPVC {
	if pvc == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []string
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取存储类
	storageClass := ""
	if pvc.Spec.StorageClassName != nil {
		storageClass = *pvc.Spec.StorageClassName
	}

	// 获取请求容量
	requestStorage := ""
	if pvc.Spec.Resources.Requests != nil {
		if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			requestStorage = storage.String()
		}
	}

	// 获取实际容量
	capacity := ""
	if pvc.Status.Capacity != nil {
		if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = storage.String()
		}
	}

	// 获取绑定的 PV
	volumeName := pvc.Spec.VolumeName

	// 获取卷模式
	volumeMode := string(corev1.PersistentVolumeFilesystem)
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	// 转换状态为枚举
	status := convertPVCStatusToEnum(pvc.Status.Phase)

	// 获取选择器
	selector := make(map[string]string)
	if pvc.Spec.Selector != nil && pvc.Spec.Selector.MatchLabels != nil {
		selector = pvc.Spec.Selector.MatchLabels
	}

	return &model.K8sPVC{
		Name:            pvc.Name,
		Namespace:       pvc.Namespace,
		ClusterID:       clusterID,
		UID:             string(pvc.UID),
		Capacity:        capacity,
		RequestStorage:  requestStorage,
		AccessModes:     accessModes,
		StorageClass:    storageClass,
		VolumeMode:      volumeMode,
		Status:          status,
		VolumeName:      volumeName,
		Selector:        selector,
		Labels:          pvc.Labels,
		Annotations:     pvc.Annotations,
		ResourceVersion: pvc.ResourceVersion,
		CreatedAt:       pvc.CreationTimestamp.Time,
		Age:             GetPVCAge(*pvc),
		RawPVC:          pvc,
	}
}

// convertPVCStatusToEnum 转换PVC状态为枚举类型
func convertPVCStatusToEnum(phase corev1.PersistentVolumeClaimPhase) model.K8sPVCStatus {
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

// ConvertToPVCEntities 批量转换 PVC 列表
func ConvertToPVCEntities(pvcs []corev1.PersistentVolumeClaim, clusterID int) []*model.K8sPVC {
	if len(pvcs) == 0 {
		return nil
	}

	results := make([]*model.K8sPVC, 0, len(pvcs))
	for _, pvc := range pvcs {
		if entity := ConvertToPVCEntity(&pvc, clusterID); entity != nil {
			results = append(results, entity)
		}
	}
	return results
}

// BuildPVCListOptions 构建 PVC 列表查询选项
func BuildPVCListOptions(req *model.GetPVCListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 构建字段选择器用于状态过滤
	var fieldSelectors []string
	if req.Status != "" {
		// 将状态字符串转换为Kubernetes状态值
		k8sStatus := convertStatusStringToK8sPVCStatus(req.Status)
		if k8sStatus != "" {
			fieldSelectors = append(fieldSelectors, fmt.Sprintf("status.phase=%s", k8sStatus))
		}
	}

	// 构建标签选择器
	var labelSelectors []string
	for key, value := range req.Labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	if len(labelSelectors) > 0 {
		options.LabelSelector = strings.Join(labelSelectors, ",")
	}

	if len(fieldSelectors) > 0 {
		options.FieldSelector = strings.Join(fieldSelectors, ",")
	}

	return options
}

// convertStatusStringToK8sPVCStatus 将状态字符串转换为Kubernetes状态
func convertStatusStringToK8sPVCStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending":
		return string(corev1.ClaimPending)
	case "bound":
		return string(corev1.ClaimBound)
	case "lost":
		return string(corev1.ClaimLost)
	default:
		return ""
	}
}

// ValidatePVC 验证 PVC 配置
func ValidatePVC(pvc *corev1.PersistentVolumeClaim) error {
	if pvc == nil {
		return fmt.Errorf("PVC 不能为空")
	}

	if pvc.Name == "" {
		return fmt.Errorf("PVC 名称不能为空")
	}

	if pvc.Namespace == "" {
		return fmt.Errorf("PVC 命名空间不能为空")
	}

	if len(pvc.Spec.AccessModes) == 0 {
		return fmt.Errorf("PVC 访问模式不能为空")
	}

	if len(pvc.Spec.Resources.Requests) == 0 {
		return fmt.Errorf("PVC 资源请求不能为空")
	}

	return nil
}

// PVCToYAML 将 PVC 转换为 YAML
func PVCToYAML(pvc *corev1.PersistentVolumeClaim) (string, error) {
	if pvc == nil {
		return "", fmt.Errorf("PVC 不能为空")
	}

	// 清理不需要的字段
	cleanPVC := pvc.DeepCopy()
	cleanPVC.Status = corev1.PersistentVolumeClaimStatus{}
	cleanPVC.ManagedFields = nil
	cleanPVC.ResourceVersion = ""
	cleanPVC.UID = ""
	cleanPVC.CreationTimestamp = metav1.Time{}
	cleanPVC.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanPVC)
	if err != nil {
		return "", fmt.Errorf("转换为 YAML 失败: %w", err)
	}

	return string(yamlBytes), nil
}

// ConvertCreatePVCReqToPVC 将创建PVC请求转换为Kubernetes PVC对象
func ConvertCreatePVCReqToPVC(req *model.CreatePVCReq) *corev1.PersistentVolumeClaim {
	if req == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range req.Spec.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.Spec.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.Spec.VolumeMode)
		volumeMode = &vm
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			VolumeMode:  volumeMode,
		},
	}

	// 设置存储类
	if req.Spec.StorageClass != "" {
		pvc.Spec.StorageClassName = &req.Spec.StorageClass
	}

	// 设置卷名
	if req.Spec.VolumeName != "" {
		pvc.Spec.VolumeName = req.Spec.VolumeName
	}

	// 设置资源请求
	if req.Spec.RequestStorage != "" {
		pvc.Spec.Resources = corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(req.Spec.RequestStorage),
			},
		}
	}

	// 设置选择器
	if len(req.Spec.Selector) > 0 {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: req.Spec.Selector,
		}
	}

	return pvc
}

// ConvertUpdatePVCReqToPVC 将更新PVC请求转换为Kubernetes PVC对象
func ConvertUpdatePVCReqToPVC(req *model.UpdatePVCReq) *corev1.PersistentVolumeClaim {
	if req == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range req.Spec.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.Spec.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.Spec.VolumeMode)
		volumeMode = &vm
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			VolumeMode:  volumeMode,
		},
	}

	// 设置存储类
	if req.Spec.StorageClass != "" {
		pvc.Spec.StorageClassName = &req.Spec.StorageClass
	}

	// 设置卷名
	if req.Spec.VolumeName != "" {
		pvc.Spec.VolumeName = req.Spec.VolumeName
	}

	// 设置资源请求
	if req.Spec.RequestStorage != "" {
		pvc.Spec.Resources = corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(req.Spec.RequestStorage),
			},
		}
	}

	// 设置选择器
	if len(req.Spec.Selector) > 0 {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: req.Spec.Selector,
		}
	}

	return pvc
}

// YAMLToPVC 将 YAML 转换为 PVC
func YAMLToPVC(yamlContent string) (*corev1.PersistentVolumeClaim, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML 内容不能为空")
	}

	var pvc corev1.PersistentVolumeClaim
	err := yaml.Unmarshal([]byte(yamlContent), &pvc)
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return &pvc, nil
}

// FilterPVCsByStatus 根据状态过滤 PVC 列表
func FilterPVCsByStatus(pvcs []corev1.PersistentVolumeClaim, status string) []corev1.PersistentVolumeClaim {
	if status == "" {
		return pvcs
	}

	var filtered []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if string(pvc.Status.Phase) == status {
			filtered = append(filtered, pvc)
		}
	}

	return filtered
}

// GetPVCAge 获取 PVC 年龄
func GetPVCAge(pvc corev1.PersistentVolumeClaim) string {
	age := time.Since(pvc.CreationTimestamp.Time)
	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(age.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(age.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

// IsPVCBound 判断 PVC 是否已绑定
func IsPVCBound(pvc corev1.PersistentVolumeClaim) bool {
	return pvc.Status.Phase == corev1.ClaimBound
}

// IsPVCPending 判断 PVC 是否处于等待状态
func IsPVCPending(pvc corev1.PersistentVolumeClaim) bool {
	return pvc.Status.Phase == corev1.ClaimPending
}

// GetPVCStorageSize 获取 PVC 存储大小（请求大小）
func GetPVCStorageSize(pvc corev1.PersistentVolumeClaim) string {
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return ""
}

// GetPVCActualStorageSize 获取 PVC 实际存储大小
func GetPVCActualStorageSize(pvc corev1.PersistentVolumeClaim) string {
	if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		return storage.String()
	}
	return ""
}

// PaginatePVCList 对PVC列表进行分页处理
func PaginatePVCList(pvcs []*model.K8sPVC, page, size int) ([]*model.K8sPVC, int64) {
	total := int64(len(pvcs))
	if total == 0 {
		return []*model.K8sPVC{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return pvcs, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []*model.K8sPVC{}, total
	}
	if end > total {
		end = total
	}

	return pvcs[start:end], total
}

// FilterPVCsByAccessMode 根据访问模式过滤PVC列表
func FilterPVCsByAccessMode(pvcs []corev1.PersistentVolumeClaim, accessMode string) []corev1.PersistentVolumeClaim {
	if accessMode == "" {
		return pvcs
	}

	var filtered []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		for _, mode := range pvc.Spec.AccessModes {
			if string(mode) == accessMode {
				filtered = append(filtered, pvc)
				break
			}
		}
	}

	return filtered
}

// FilterPVCsByStorageClass 根据存储类过滤PVC列表
func FilterPVCsByStorageClass(pvcs []corev1.PersistentVolumeClaim, storageClass string) []corev1.PersistentVolumeClaim {
	if storageClass == "" {
		return pvcs
	}

	var filtered []corev1.PersistentVolumeClaim
	for _, pvc := range pvcs {
		if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == storageClass {
			filtered = append(filtered, pvc)
		}
	}

	return filtered
}

// GetPVCCapacity 获取PVC容量信息
func GetPVCCapacity(pvc corev1.PersistentVolumeClaim) (string, int64) {
	if pvc.Spec.Resources.Requests == nil {
		return "", 0
	}

	storage, exists := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !exists {
		return "", 0
	}

	return storage.String(), storage.Value()
}

// ValidatePVCUpdate 验证PVC更新请求
func ValidatePVCUpdate(req *model.UpdatePVCReq) error {
	if req == nil {
		return fmt.Errorf("更新PVC请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID必须大于0")
	}

	if req.Name == "" {
		return fmt.Errorf("PVC名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	return nil
}

// ValidatePVCCreate 验证PVC创建请求
func ValidatePVCCreate(req *model.CreatePVCReq) error {
	if req == nil {
		return fmt.Errorf("创建PVC请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID必须大于0")
	}

	if req.Name == "" {
		return fmt.Errorf("PVC名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Spec.RequestStorage == "" {
		return fmt.Errorf("PVC请求容量不能为空")
	}

	if len(req.Spec.AccessModes) == 0 {
		return fmt.Errorf("PVC访问模式不能为空")
	}

	return nil
}

// ComparePVCs 比较两个PVC是否相同（用于检测更新）
func ComparePVCs(pvc1, pvc2 *corev1.PersistentVolumeClaim) bool {
	if pvc1 == nil || pvc2 == nil {
		return pvc1 == pvc2
	}

	if pvc1.Name != pvc2.Name || pvc1.Namespace != pvc2.Namespace {
		return false
	}

	// 比较存储类
	if (pvc1.Spec.StorageClassName == nil) != (pvc2.Spec.StorageClassName == nil) {
		return false
	}
	if pvc1.Spec.StorageClassName != nil && *pvc1.Spec.StorageClassName != *pvc2.Spec.StorageClassName {
		return false
	}

	// 比较访问模式
	if len(pvc1.Spec.AccessModes) != len(pvc2.Spec.AccessModes) {
		return false
	}
	for i, mode := range pvc1.Spec.AccessModes {
		if mode != pvc2.Spec.AccessModes[i] {
			return false
		}
	}

	// 比较存储请求
	req1 := pvc1.Spec.Resources.Requests[corev1.ResourceStorage]
	req2 := pvc2.Spec.Resources.Requests[corev1.ResourceStorage]
	if !req1.Equal(req2) {
		return false
	}

	return true
}

// IsPVCExpandable 判断PVC是否支持扩容
func IsPVCExpandable(pvc corev1.PersistentVolumeClaim) bool {
	// 检查存储类是否支持扩容
	// 这里简化处理，实际情况下需要查询StorageClass的AllowVolumeExpansion字段
	return pvc.Spec.StorageClassName != nil && pvc.Status.Phase == corev1.ClaimBound
}

// GetPVCAccessModes 获取PVC访问模式的字符串列表
func GetPVCAccessModes(pvc corev1.PersistentVolumeClaim) []string {
	modes := make([]string, 0, len(pvc.Spec.AccessModes))
	for _, mode := range pvc.Spec.AccessModes {
		modes = append(modes, string(mode))
	}
	return modes
}

// CalculatePVCStorageUsage 计算PVC存储使用率
func CalculatePVCStorageUsage(pvc corev1.PersistentVolumeClaim) (float64, error) {
	// 获取请求大小
	requestedStorage, exists := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if !exists {
		return 0, fmt.Errorf("PVC未设置存储请求")
	}

	// 获取实际大小
	actualStorage, exists := pvc.Status.Capacity[corev1.ResourceStorage]
	if !exists {
		return 0, fmt.Errorf("PVC未绑定存储")
	}

	// 计算使用率（这里简化为100%，实际需要通过metrics获取真实使用情况）
	if actualStorage.Value() == 0 {
		return 0, nil
	}

	// 这里返回分配率而不是真正的使用率
	return float64(requestedStorage.Value()) / float64(actualStorage.Value()) * 100, nil
}
