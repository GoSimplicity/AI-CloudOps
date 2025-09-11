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

// ConvertToPVEntity 将 Kubernetes PV 转换为内部 PV 模型
func ConvertToPVEntity(pv *corev1.PersistentVolume, clusterID int) *model.K8sPV {
	if pv == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []string
	for _, mode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取存储类
	storageClass := ""
	if pv.Spec.StorageClassName != "" {
		storageClass = pv.Spec.StorageClassName
	}

	// 获取容量
	capacity := ""
	if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		capacity = storage.String()
	}

	// 获取回收策略
	reclaimPolicy := string(corev1.PersistentVolumeReclaimDelete)
	if pv.Spec.PersistentVolumeReclaimPolicy != "" {
		reclaimPolicy = string(pv.Spec.PersistentVolumeReclaimPolicy)
	}

	// 获取卷模式
	volumeMode := string(corev1.PersistentVolumeFilesystem)
	if pv.Spec.VolumeMode != nil {
		volumeMode = string(*pv.Spec.VolumeMode)
	}

	// 获取绑定的 PVC（如果需要的话可以添加到模型中）
	_ = ""
	if pv.Spec.ClaimRef != nil {
		_ = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
	}

	return &model.K8sPV{
		Name:          pv.Name,
		UID:           string(pv.UID),
		ClusterID:     clusterID,
		Status:        string(pv.Status.Phase),
		Capacity:      capacity,
		AccessModes:   accessModes,
		ReclaimPolicy: reclaimPolicy,
		StorageClass:  storageClass,
		VolumeMode:    volumeMode,
		Labels:        pv.Labels,
		Annotations:   pv.Annotations,
	}
}

// ConvertToPVEntities 批量转换 PV 列表
func ConvertToPVEntities(pvs []corev1.PersistentVolume, clusterID int) []*model.K8sPV {
	if len(pvs) == 0 {
		return nil
	}

	results := make([]*model.K8sPV, 0, len(pvs))
	for _, pv := range pvs {
		if entity := ConvertToPVEntity(&pv, clusterID); entity != nil {
			results = append(results, entity)
		}
	}
	return results
}

// BuildPVListOptions 构建 PV 列表查询选项
func BuildPVListOptions(req *model.GetPVListReq) metav1.ListOptions {
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

// ValidatePV 验证 PV 配置
func ValidatePV(pv *corev1.PersistentVolume) error {
	if pv == nil {
		return fmt.Errorf("PV 不能为空")
	}

	if pv.Name == "" {
		return fmt.Errorf("PV 名称不能为空")
	}

	if len(pv.Spec.AccessModes) == 0 {
		return fmt.Errorf("PV 访问模式不能为空")
	}

	if pv.Spec.Capacity == nil || len(pv.Spec.Capacity) == 0 {
		return fmt.Errorf("PV 容量不能为空")
	}

	return nil
}

// PVToYAML 将 PV 转换为 YAML
func PVToYAML(pv *corev1.PersistentVolume) (string, error) {
	if pv == nil {
		return "", fmt.Errorf("PV 不能为空")
	}

	// 清理不需要的字段
	cleanPV := pv.DeepCopy()
	cleanPV.Status = corev1.PersistentVolumeStatus{}
	cleanPV.ManagedFields = nil
	cleanPV.ResourceVersion = ""
	cleanPV.UID = ""
	cleanPV.CreationTimestamp = metav1.Time{}
	cleanPV.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanPV)
	if err != nil {
		return "", fmt.Errorf("转换为 YAML 失败: %w", err)
	}

	return string(yamlBytes), nil
}

// YAMLToPV 将 YAML 转换为 PV
func YAMLToPV(yamlContent string) (*corev1.PersistentVolume, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML 内容不能为空")
	}

	var pv corev1.PersistentVolume
	err := yaml.Unmarshal([]byte(yamlContent), &pv)
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return &pv, nil
}

// FilterPVsByStatus 根据状态过滤 PV 列表
func FilterPVsByStatus(pvs []corev1.PersistentVolume, status string) []corev1.PersistentVolume {
	if status == "" {
		return pvs
	}

	var filtered []corev1.PersistentVolume
	for _, pv := range pvs {
		if string(pv.Status.Phase) == status {
			filtered = append(filtered, pv)
		}
	}

	return filtered
}

// GetPVAge 获取 PV 年龄
func GetPVAge(pv corev1.PersistentVolume) string {
	age := time.Since(pv.CreationTimestamp.Time)
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

// IsPVBound 判断 PV 是否已绑定
func IsPVBound(pv corev1.PersistentVolume) bool {
	return pv.Status.Phase == corev1.VolumeBound
}

// IsPVAvailable 判断 PV 是否可用
func IsPVAvailable(pv corev1.PersistentVolume) bool {
	return pv.Status.Phase == corev1.VolumeAvailable
}

// ConvertCreatePVReqToPV 将创建PV请求转换为Kubernetes PV对象
func ConvertCreatePVReqToPV(req *model.CreatePVReq) *corev1.PersistentVolume {
	if req == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range req.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换回收策略
	var reclaimPolicy corev1.PersistentVolumeReclaimPolicy
	if req.ReclaimPolicy != "" {
		reclaimPolicy = corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
	} else {
		reclaimPolicy = corev1.PersistentVolumeReclaimDelete
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.VolumeMode)
		volumeMode = &vm
	}

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			StorageClassName:              req.StorageClass,
			VolumeMode:                    volumeMode,
		},
	}

	// 设置容量
	if req.Capacity != "" {
		pv.Spec.Capacity = corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(req.Capacity),
		}
	}

	// 设置卷源 - 这里简化处理，实际可能需要更复杂的转换逻辑
	if len(req.VolumeSource) > 0 {
		// 这里需要根据具体的卷源类型进行转换
		// 暂时留空，需要根据实际需求实现
	}

	return pv
}

// ConvertUpdatePVReqToPV 将更新PV请求转换为Kubernetes PV对象
func ConvertUpdatePVReqToPV(req *model.UpdatePVReq) *corev1.PersistentVolume {
	if req == nil {
		return nil
	}

	// 转换访问模式
	var accessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range req.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 转换回收策略
	var reclaimPolicy corev1.PersistentVolumeReclaimPolicy
	if req.ReclaimPolicy != "" {
		reclaimPolicy = corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
	} else {
		reclaimPolicy = corev1.PersistentVolumeReclaimDelete
	}

	// 转换卷模式
	var volumeMode *corev1.PersistentVolumeMode
	if req.VolumeMode != "" {
		vm := corev1.PersistentVolumeMode(req.VolumeMode)
		volumeMode = &vm
	}

	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:                   accessModes,
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			StorageClassName:              req.StorageClass,
			VolumeMode:                    volumeMode,
		},
	}

	// 设置容量
	if req.Capacity != "" {
		pv.Spec.Capacity = corev1.ResourceList{
			corev1.ResourceStorage: resource.MustParse(req.Capacity),
		}
	}

	// 设置卷源 - 这里简化处理，实际可能需要更复杂的转换逻辑
	if len(req.VolumeSource) > 0 {
		// 这里需要根据具体的卷源类型进行转换
		// 暂时留空，需要根据实际需求实现
	}

	return pv
}
