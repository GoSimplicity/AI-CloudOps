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

	// 转换状态为枚举
	status := convertPVStatusToEnum(pv.Status.Phase)

	// 获取绑定的 PVC 信息
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

	// 获取节点亲和性
	nodeAffinity := make(map[string]interface{})
	if pv.Spec.NodeAffinity != nil && pv.Spec.NodeAffinity.Required != nil {
		nodeAffinity["required"] = "true"
	}

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
		ResourceVersion: pv.ResourceVersion,
		CreatedAt:       pv.CreationTimestamp.Time,
		Age:             GetPVAge(*pv),
		RawPV:           pv,
	}
}

// convertPVStatusToEnum 转换PV状态为枚举类型
func convertPVStatusToEnum(phase corev1.PersistentVolumePhase) model.K8sPVStatus {
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

	// 构建字段选择器用于状态过滤
	var fieldSelectors []string
	if req.Status != "" {
		// 将状态字符串转换为Kubernetes状态值
		k8sStatus := convertStatusStringToK8sStatus(req.Status)
		if k8sStatus != "" {
			fieldSelectors = append(fieldSelectors, fmt.Sprintf("status.phase=%s", k8sStatus))
		}
	}

	if len(fieldSelectors) > 0 {
		options.FieldSelector = strings.Join(fieldSelectors, ",")
	}

	return options
}

// convertStatusStringToK8sStatus 将状态字符串转换为Kubernetes状态
func convertStatusStringToK8sStatus(status string) string {
	switch strings.ToLower(status) {
	case "available":
		return string(corev1.VolumeAvailable)
	case "bound":
		return string(corev1.VolumeBound)
	case "released":
		return string(corev1.VolumeReleased)
	case "failed":
		return string(corev1.VolumeFailed)
	default:
		return ""
	}
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

	if len(pv.Spec.Capacity) == 0 {
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
// 基于现有PV对象更新可变字段，保留不可变字段
func ConvertUpdatePVReqToPV(req *model.UpdatePVReq, existingPV *corev1.PersistentVolume) *corev1.PersistentVolume {
	if req == nil || existingPV == nil {
		return nil
	}

	// 深拷贝现有PV对象
	pv := existingPV.DeepCopy()

	// 更新可变的metadata字段
	if req.Labels != nil {
		pv.ObjectMeta.Labels = req.Labels
	}
	if req.Annotations != nil {
		pv.ObjectMeta.Annotations = req.Annotations
	}

	// 更新容量（如果提供）
	if req.Capacity != "" {
		if pv.Spec.Capacity == nil {
			pv.Spec.Capacity = corev1.ResourceList{}
		}
		pv.Spec.Capacity[corev1.ResourceStorage] = resource.MustParse(req.Capacity)
	}

	// 更新访问模式（如果提供）
	if len(req.AccessModes) > 0 {
		var accessModes []corev1.PersistentVolumeAccessMode
		for _, mode := range req.AccessModes {
			accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
		}
		pv.Spec.AccessModes = accessModes
	}

	// 更新回收策略（如果提供）
	if req.ReclaimPolicy != "" {
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
	}

	// 更新存储类（如果提供）
	if req.StorageClass != "" {
		pv.Spec.StorageClassName = req.StorageClass
	}

	// 注意: PersistentVolumeSource 和某些VolumeMode在PV创建后是不可变的，因此不更新这些字段
	// 这避免了 "spec.persistentvolumesource: Forbidden: spec.persistentvolumesource is immutable after creation" 错误

	return pv
}

// PaginatePVList 对PV列表进行分页处理
func PaginatePVList(pvs []*model.K8sPV, page, size int) ([]*model.K8sPV, int64) {
	total := int64(len(pvs))
	if total == 0 {
		return []*model.K8sPV{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return pvs, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []*model.K8sPV{}, total
	}
	if end > total {
		end = total
	}

	return pvs[start:end], total
}

// FilterPVsByAccessMode 根据访问模式过滤PV列表
func FilterPVsByAccessMode(pvs []corev1.PersistentVolume, accessMode string) []corev1.PersistentVolume {
	if accessMode == "" {
		return pvs
	}

	var filtered []corev1.PersistentVolume
	for _, pv := range pvs {
		for _, mode := range pv.Spec.AccessModes {
			if string(mode) == accessMode {
				filtered = append(filtered, pv)
				break
			}
		}
	}

	return filtered
}

// FilterPVsByStorageClass 根据存储类过滤PV列表
func FilterPVsByStorageClass(pvs []corev1.PersistentVolume, storageClass string) []corev1.PersistentVolume {
	if storageClass == "" {
		return pvs
	}

	var filtered []corev1.PersistentVolume
	for _, pv := range pvs {
		if pv.Spec.StorageClassName == storageClass {
			filtered = append(filtered, pv)
		}
	}

	return filtered
}

// GetPVCapacity 获取PV容量信息
func GetPVCapacity(pv corev1.PersistentVolume) (string, int64) {
	if pv.Spec.Capacity == nil {
		return "", 0
	}

	storage, exists := pv.Spec.Capacity[corev1.ResourceStorage]
	if !exists {
		return "", 0
	}

	return storage.String(), storage.Value()
}

// ValidatePVUpdate 验证PV更新请求
func ValidatePVUpdate(req *model.UpdatePVReq) error {
	if req == nil {
		return fmt.Errorf("更新PV请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID必须大于0")
	}

	if req.Name == "" {
		return fmt.Errorf("PV名称不能为空")
	}

	return nil
}

// ValidatePVCreate 验证PV创建请求
func ValidatePVCreate(req *model.CreatePVReq) error {
	if req == nil {
		return fmt.Errorf("创建PV请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID必须大于0")
	}

	if req.Name == "" {
		return fmt.Errorf("PV名称不能为空")
	}

	if req.Capacity == "" {
		return fmt.Errorf("PV容量不能为空")
	}

	if len(req.AccessModes) == 0 {
		return fmt.Errorf("PV访问模式不能为空")
	}

	if len(req.VolumeSource) == 0 {
		return fmt.Errorf("PV卷源不能为空")
	}

	return nil
}
