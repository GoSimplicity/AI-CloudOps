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
	requestCapacity := ""
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestCapacity = storage.String()
	}

	// 获取实际容量（如果需要可以添加到模型中）
	_ = ""
	if storage, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
		_ = storage.String()
	}

	// 获取绑定的 PV
	volumeName := pvc.Spec.VolumeName

	// 获取卷模式
	volumeMode := string(corev1.PersistentVolumeFilesystem)
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	return &model.K8sPVC{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		UID:          string(pvc.UID),
		ClusterID:    clusterID,
		Status:       string(pvc.Status.Phase),
		Capacity:     requestCapacity,
		AccessModes:  accessModes,
		StorageClass: storageClass,
		VolumeMode:   volumeMode,
		VolumeName:   volumeName,
		Labels:       pvc.Labels,
		Annotations:  pvc.Annotations,
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

	// 构建标签选择器
	if req.LabelSelector != "" {
		options.LabelSelector = req.LabelSelector
	}

	// 构建字段选择器
	if req.FieldSelector != "" {
		options.FieldSelector = req.FieldSelector
	}

	return options
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

	if pvc.Spec.Resources.Requests == nil || len(pvc.Spec.Resources.Requests) == 0 {
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
	for _, mode := range req.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.VolumeMode)
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
	if req.StorageClass != "" {
		pvc.Spec.StorageClassName = &req.StorageClass
	}

	// 设置卷名
	if req.VolumeName != "" {
		pvc.Spec.VolumeName = req.VolumeName
	}

	// 设置资源请求
	if req.RequestStorage != "" {
		pvc.Spec.Resources = corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(req.RequestStorage),
			},
		}
	}

	// 设置选择器
	if len(req.Selector) > 0 {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: req.Selector,
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
	for _, mode := range req.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.VolumeMode)
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
	if req.StorageClass != "" {
		pvc.Spec.StorageClassName = &req.StorageClass
	}

	// 设置卷名
	if req.VolumeName != "" {
		pvc.Spec.VolumeName = req.VolumeName
	}

	// 设置资源请求
	if req.RequestStorage != "" {
		pvc.Spec.Resources = corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(req.RequestStorage),
			},
		}
	}

	// 设置选择器
	if len(req.Selector) > 0 {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: req.Selector,
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
