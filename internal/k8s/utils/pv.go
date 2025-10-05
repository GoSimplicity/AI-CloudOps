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

func ConvertToPVEntity(pv *corev1.PersistentVolume, clusterID int) *model.K8sPV {
	if pv == nil {
		return nil
	}

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

	status := convertPVStatusToEnum(pv.Status.Phase)

	// 获取绑定的 PVC 信息
	claimRef := make(map[string]string)
	if pv.Spec.ClaimRef != nil {
		claimRef["namespace"] = pv.Spec.ClaimRef.Namespace
		claimRef["name"] = pv.Spec.ClaimRef.Name
		claimRef["uid"] = string(pv.Spec.ClaimRef.UID)
	}

	volumeSource := make(map[string]interface{})
	if pv.Spec.PersistentVolumeSource.HostPath != nil {
		volumeSource["hostPath"] = map[string]interface{}{
			"path": pv.Spec.PersistentVolumeSource.HostPath.Path,
			"type": pv.Spec.PersistentVolumeSource.HostPath.Type,
		}
	}

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

func BuildPVListOptions(req *model.GetPVListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 注意：PV资源不支持通过 status.phase 进行 field selector 过滤
	// Kubernetes API 只支持 metadata.name 和 metadata.namespace
	// 状态过滤必须在应用层（service层）完成

	return options
}

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

	// 检查 ReadWriteOncePod 不能与其他访问模式混用
	hasReadWriteOncePod := false
	for _, mode := range pv.Spec.AccessModes {
		if mode == corev1.ReadWriteOncePod {
			hasReadWriteOncePod = true
			break
		}
	}
	if hasReadWriteOncePod && len(pv.Spec.AccessModes) > 1 {
		return fmt.Errorf("ReadWriteOncePod 不能与其他访问模式一起使用")
	}

	if len(pv.Spec.Capacity) == 0 {
		return fmt.Errorf("PV 容量不能为空")
	}

	// 检查必须指定卷类型
	if !hasVolumeSource(&pv.Spec.PersistentVolumeSource) {
		return fmt.Errorf("必须指定一个卷类型（如 HostPath, NFS, CephFS 等）")
	}

	return nil
}

// hasVolumeSource 检查是否指定了卷源
func hasVolumeSource(source *corev1.PersistentVolumeSource) bool {
	if source == nil {
		return false
	}

	return source.HostPath != nil ||
		source.NFS != nil ||
		source.CephFS != nil ||
		source.RBD != nil ||
		source.Glusterfs != nil ||
		source.ISCSI != nil ||
		source.FC != nil ||
		source.AWSElasticBlockStore != nil ||
		source.GCEPersistentDisk != nil ||
		source.AzureDisk != nil ||
		source.AzureFile != nil ||
		source.CSI != nil ||
		source.Local != nil
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

func ConvertCreatePVReqToPV(req *model.CreatePVReq) *corev1.PersistentVolume {
	if req == nil {
		return nil
	}

	var accessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range req.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	var reclaimPolicy corev1.PersistentVolumeReclaimPolicy
	if req.ReclaimPolicy != "" {
		reclaimPolicy = corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
	} else {
		reclaimPolicy = corev1.PersistentVolumeReclaimDelete
	}

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

	return pv
}

// ConvertCreatePVReqToPVWithValidation 将请求转换为PV对象并验证卷源
func ConvertCreatePVReqToPVWithValidation(req *model.CreatePVReq) (*corev1.PersistentVolume, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	pv := ConvertCreatePVReqToPV(req)
	if pv == nil {
		return nil, fmt.Errorf("转换PV失败")
	}

	// 设置并验证卷源
	if len(req.VolumeSource) > 0 {
		if err := convertVolumeSource(&pv.Spec.PersistentVolumeSource, req.VolumeSource); err != nil {
			return nil, fmt.Errorf("卷源配置无效: %w", err)
		}
	}

	return pv, nil
}

// convertVolumeSource 将 map 转换为 PersistentVolumeSource
func convertVolumeSource(pvSource *corev1.PersistentVolumeSource, source map[string]interface{}) error {
	// HostPath 类型
	if hostPath, ok := source["hostPath"].(map[string]interface{}); ok {
		path, pathOk := hostPath["path"].(string)
		if !pathOk || path == "" {
			return fmt.Errorf("hostPath.path 是必填字段")
		}

		pvSource.HostPath = &corev1.HostPathVolumeSource{
			Path: path,
		}

		if typeStr, ok := hostPath["type"].(string); ok && typeStr != "" {
			hostPathType := corev1.HostPathType(typeStr)
			pvSource.HostPath.Type = &hostPathType
		}
		return nil
	}

	// NFS 类型
	if nfs, ok := source["nfs"].(map[string]interface{}); ok {
		server, serverOk := nfs["server"].(string)
		path, pathOk := nfs["path"].(string)

		if !serverOk || server == "" {
			return fmt.Errorf("nfs.server 是必填字段")
		}
		if !pathOk || path == "" {
			return fmt.Errorf("nfs.path 是必填字段")
		}

		pvSource.NFS = &corev1.NFSVolumeSource{
			Server: server,
			Path:   path,
		}

		if readOnly, ok := nfs["readOnly"].(bool); ok {
			pvSource.NFS.ReadOnly = readOnly
		}
		return nil
	}

	// Local 类型
	if local, ok := source["local"].(map[string]interface{}); ok {
		path, pathOk := local["path"].(string)
		if !pathOk || path == "" {
			return fmt.Errorf("local.path 是必填字段")
		}

		pvSource.Local = &corev1.LocalVolumeSource{
			Path: path,
		}
		return nil
	}

	// CSI 类型
	if csi, ok := source["csi"].(map[string]interface{}); ok {
		driver, driverOk := csi["driver"].(string)
		volumeHandle, handleOk := csi["volumeHandle"].(string)

		if !driverOk || driver == "" {
			return fmt.Errorf("csi.driver 是必填字段")
		}
		if !handleOk || volumeHandle == "" {
			return fmt.Errorf("csi.volumeHandle 是必填字段")
		}

		pvSource.CSI = &corev1.CSIPersistentVolumeSource{
			Driver:       driver,
			VolumeHandle: volumeHandle,
		}

		if readOnly, ok := csi["readOnly"].(bool); ok {
			pvSource.CSI.ReadOnly = readOnly
		}
		if volumeAttributes, ok := csi["volumeAttributes"].(map[string]interface{}); ok {
			pvSource.CSI.VolumeAttributes = make(map[string]string)
			for k, v := range volumeAttributes {
				if strVal, ok := v.(string); ok {
					pvSource.CSI.VolumeAttributes[k] = strVal
				}
			}
		}
		return nil
	}

	// 可以继续添加其他卷类型的支持...
	return fmt.Errorf("未识别的卷源类型，支持的类型：hostPath, nfs, local, csi")
}

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
