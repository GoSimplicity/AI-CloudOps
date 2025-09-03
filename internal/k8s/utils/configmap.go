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
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// BuildK8sConfigMap 构建详细的 K8sConfigMapEntity 模型
func BuildK8sConfigMap(ctx context.Context, clusterID int, configMap corev1.ConfigMap) (*model.K8sConfigMapEntity, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 转换 Data 字段
	data := make(map[string]string)
	for k, v := range configMap.Data {
		data[k] = v
	}

	// 转换 BinaryData 字段
	binaryData := make(map[string][]byte)
	for k, v := range configMap.BinaryData {
		binaryData[k] = v
	}

	// 转换标签
	labels := make(map[string]string)
	if configMap.Labels != nil {
		for k, v := range configMap.Labels {
			labels[k] = v
		}
	}

	// 转换注解
	annotations := make(map[string]string)
	if configMap.Annotations != nil {
		for k, v := range configMap.Annotations {
			annotations[k] = v
		}
	}

	// 计算数据条目数量
	dataCount := len(configMap.Data) + len(configMap.BinaryData)

	// 计算数据大小
	size := calculateConfigMapDataSize(configMap)

	// 计算存在时间
	age := CalculateAge(configMap.CreationTimestamp.Time)

	k8sConfigMap := &model.K8sConfigMapEntity{
		Name:              configMap.Name,
		Namespace:         configMap.Namespace,
		ClusterID:         clusterID,
		UID:               string(configMap.UID),
		Data:              data,
		BinaryData:        binaryData,
		Labels:            labels,
		Annotations:       annotations,
		CreationTimestamp: configMap.CreationTimestamp.Time,
		Age:               age,
		DataCount:         dataCount,
		Size:              size,
	}

	return k8sConfigMap, nil
}

// ConvertToConfigMapEntity 将 K8sConfigMapEntity 转换为响应实体
func ConvertToConfigMapEntity(configMap *model.K8sConfigMapEntity) *model.ConfigMapEntity {
	if configMap == nil {
		return nil
	}

	return &model.ConfigMapEntity{
		Name:        configMap.Name,
		Namespace:   configMap.Namespace,
		UID:         configMap.UID,
		Labels:      configMap.Labels,
		Annotations: configMap.Annotations,
		Data:        configMap.Data,
		BinaryData:  configMap.BinaryData,
		DataCount:   configMap.DataCount,
		Size:        configMap.Size,
		Immutable:   false, // ConfigMap 默认可变，除非特别指定
		Age:         configMap.Age,
		CreatedAt:   configMap.CreationTimestamp.Format(time.RFC3339),
	}
}

// BuildConfigMapFromRequest 从创建请求构建 Kubernetes ConfigMap
func BuildConfigMapFromRequest(req *model.K8sConfigMapCreateReq) (*corev1.ConfigMap, error) {
	if req == nil {
		return nil, fmt.Errorf("创建请求不能为空")
	}

	// 如果提供了 ConfigMapYaml，直接使用
	if req.ConfigMapYaml != nil {
		return req.ConfigMapYaml, nil
	}

	// 构建 ConfigMap 对象
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Data:       req.Data,
		BinaryData: req.BinaryData,
	}

	return configMap, nil
}

// UpdateConfigMapFromRequest 从更新请求更新 Kubernetes ConfigMap
func UpdateConfigMapFromRequest(existing *corev1.ConfigMap, req *model.K8sConfigMapUpdateReq) (*corev1.ConfigMap, error) {
	if existing == nil {
		return nil, fmt.Errorf("现有ConfigMap不能为空")
	}
	if req == nil {
		return nil, fmt.Errorf("更新请求不能为空")
	}

	// 如果提供了 ConfigMapYaml，直接使用
	if req.ConfigMapYaml != nil {
		return req.ConfigMapYaml, nil
	}

	// 创建一个副本用于更新
	updated := existing.DeepCopy()

	// 更新数据
	if req.Data != nil {
		updated.Data = req.Data
	}
	if req.BinaryData != nil {
		updated.BinaryData = req.BinaryData
	}

	// 更新标签
	if req.Labels != nil {
		updated.Labels = req.Labels
	}

	// 更新注解
	if req.Annotations != nil {
		updated.Annotations = req.Annotations
	}

	return updated, nil
}

// GetConfigMapToYAML 将 ConfigMap 转换为 YAML 字符串
func GetConfigMapToYAML(ctx context.Context, clientset kubernetes.Interface, clusterID int, namespace, name string) (string, error) {
	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取ConfigMap失败: %v", err)
	}

	// 清理系统字段
	configMap = CleanConfigMapForYAML(configMap)

	yamlBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return "", fmt.Errorf("转换YAML失败: %v", err)
	}

	return string(yamlBytes), nil
}

// CleanConfigMapForYAML 清理 ConfigMap 对象中的系统字段，用于YAML输出
func CleanConfigMapForYAML(configMap *corev1.ConfigMap) *corev1.ConfigMap {
	cleaned := configMap.DeepCopy()

	// 清理 metadata 中的系统字段
	cleaned.ObjectMeta.ResourceVersion = ""
	cleaned.ObjectMeta.UID = ""
	cleaned.ObjectMeta.SelfLink = ""
	cleaned.ObjectMeta.CreationTimestamp = metav1.Time{}
	cleaned.ObjectMeta.Generation = 0
	cleaned.ObjectMeta.ManagedFields = nil

	// 清理状态相关的注解
	if cleaned.Annotations != nil {
		delete(cleaned.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}

	return cleaned
}

// ValidateConfigMapData 验证 ConfigMap 数据的有效性
func ValidateConfigMapData(data map[string]string, binaryData map[string][]byte) error {
	// 检查键名是否有效
	for key := range data {
		if err := validateConfigMapKey(key); err != nil {
			return fmt.Errorf("Data中的键 '%s' 无效: %v", key, err)
		}
	}

	for key := range binaryData {
		if err := validateConfigMapKey(key); err != nil {
			return fmt.Errorf("BinaryData中的键 '%s' 无效: %v", key, err)
		}
	}

	// 检查总大小限制（Kubernetes 限制为 1MB）
	totalSize := 0
	for _, value := range data {
		totalSize += len([]byte(value))
	}
	for _, value := range binaryData {
		totalSize += len(value)
	}

	if totalSize > 1048576 { // 1MB
		return fmt.Errorf("ConfigMap数据总大小 %d 字节超过1MB限制", totalSize)
	}

	return nil
}

// GetConfigMapUsageInfo 获取 ConfigMap 的使用情况
func GetConfigMapUsageInfo(ctx context.Context, clientset kubernetes.Interface, namespace, configMapName string) (*model.ConfigMapUsageEntity, error) {
	usage := &model.ConfigMapUsageEntity{
		UsedByPods:         []model.ConfigMapPodUsageEntity{},
		UsedByDeployments:  []model.ConfigMapDeploymentUsageEntity{},
		UsedByStatefulSets: []model.ConfigMapStatefulSetUsageEntity{},
		UsedByDaemonSets:   []model.ConfigMapDaemonSetUsageEntity{},
		UsedByJobs:         []model.ConfigMapJobUsageEntity{},
	}

	// 检查 Pod 使用情况
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			podUsage := findConfigMapUsageInPod(&pod, configMapName)
			usage.UsedByPods = append(usage.UsedByPods, podUsage...)
		}
	}

	// 检查 Deployment 使用情况
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deployment := range deployments.Items {
			deploymentUsage := findConfigMapUsageInDeployment(&deployment, configMapName)
			usage.UsedByDeployments = append(usage.UsedByDeployments, deploymentUsage...)
		}
	}

	// 检查 StatefulSet 使用情况
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sts := range statefulSets.Items {
			stsUsage := findConfigMapUsageInStatefulSet(&sts, configMapName)
			usage.UsedByStatefulSets = append(usage.UsedByStatefulSets, stsUsage...)
		}
	}

	// 检查 DaemonSet 使用情况
	daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range daemonSets.Items {
			dsUsage := findConfigMapUsageInDaemonSet(&ds, configMapName)
			usage.UsedByDaemonSets = append(usage.UsedByDaemonSets, dsUsage...)
		}
	}

	return usage, nil
}

// FilterConfigMaps 根据条件过滤 ConfigMap 列表
func FilterConfigMaps(configMaps []model.K8sConfigMapEntity, req *model.GetConfigMapListReq) []model.K8sConfigMapEntity {
	var filtered []model.K8sConfigMapEntity

	for _, configMap := range configMaps {
		// 命名空间过滤
		if req.Namespace != "" && configMap.Namespace != req.Namespace {
			continue
		}

		// 名称过滤
		if req.Name != "" && !strings.Contains(configMap.Name, req.Name) {
			continue
		}

		// 数据键过滤
		if req.DataKey != "" {
			found := false
			for key := range configMap.Data {
				if strings.Contains(key, req.DataKey) {
					found = true
					break
				}
			}
			if !found {
				for key := range configMap.BinaryData {
					if strings.Contains(key, req.DataKey) {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
		}

		// 标签选择器过滤
		if req.LabelSelector != "" && !matchesLabelSelector(configMap.Labels, req.LabelSelector) {
			continue
		}

		filtered = append(filtered, configMap)
	}

	return filtered
}

// SortConfigMaps 对 ConfigMap 列表进行排序
func SortConfigMaps(configMaps []model.K8sConfigMapEntity, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(configMaps, func(i, j int) bool {
			return configMaps[i].Name < configMaps[j].Name
		})
	case "namespace":
		sort.Slice(configMaps, func(i, j int) bool {
			return configMaps[i].Namespace < configMaps[j].Namespace
		})
	case "created":
		sort.Slice(configMaps, func(i, j int) bool {
			return configMaps[i].CreationTimestamp.After(configMaps[j].CreationTimestamp)
		})
	default:
		// 默认按创建时间倒序
		sort.Slice(configMaps, func(i, j int) bool {
			return configMaps[i].CreationTimestamp.After(configMaps[j].CreationTimestamp)
		})
	}
}

// calculateConfigMapDataSize 计算 ConfigMap 数据大小
func calculateConfigMapDataSize(configMap corev1.ConfigMap) string {
	totalSize := 0

	// 计算 Data 字段大小
	for _, value := range configMap.Data {
		totalSize += len([]byte(value))
	}

	// 计算 BinaryData 字段大小
	for _, value := range configMap.BinaryData {
		totalSize += len(value)
	}

	return FormatBytes(totalSize)
}

// validateConfigMapKey 验证 ConfigMap 键名的有效性
func validateConfigMapKey(key string) error {
	if key == "" {
		return fmt.Errorf("键名不能为空")
	}

	// Kubernetes ConfigMap 键名规则
	if len(key) > 253 {
		return fmt.Errorf("键名长度不能超过253个字符")
	}

	// 检查是否包含无效字符
	for _, char := range key {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '-' && char != '_' && char != '.' {
			return fmt.Errorf("键名包含无效字符: %c", char)
		}
	}

	return nil
}

// findConfigMapUsageInPod 查找 Pod 中对 ConfigMap 的使用
func findConfigMapUsageInPod(pod *corev1.Pod, configMapName string) []model.ConfigMapPodUsageEntity {
	var usage []model.ConfigMapPodUsageEntity

	// 检查 Volume 中的使用
	for _, volume := range pod.Spec.Volumes {
		if volume.ConfigMap != nil && volume.ConfigMap.Name == configMapName {
			// 查找挂载路径
			for _, container := range pod.Spec.Containers {
				for _, mount := range container.VolumeMounts {
					if mount.Name == volume.Name {
						usage = append(usage, model.ConfigMapPodUsageEntity{
							PodName:       pod.Name,
							Namespace:     pod.Namespace,
							UsageType:     "volume",
							MountPath:     mount.MountPath,
							ContainerName: container.Name,
						})
					}
				}
			}
		}
	}

	// 检查环境变量中的使用
	for _, container := range pod.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil && env.ValueFrom.ConfigMapKeyRef.Name == configMapName {
				usage = append(usage, model.ConfigMapPodUsageEntity{
					PodName:       pod.Name,
					Namespace:     pod.Namespace,
					UsageType:     "env",
					Keys:          []string{env.ValueFrom.ConfigMapKeyRef.Key},
					ContainerName: container.Name,
				})
			}
		}

		// 检查 EnvFrom 中的使用
		for _, envFrom := range container.EnvFrom {
			if envFrom.ConfigMapRef != nil && envFrom.ConfigMapRef.Name == configMapName {
				usage = append(usage, model.ConfigMapPodUsageEntity{
					PodName:       pod.Name,
					Namespace:     pod.Namespace,
					UsageType:     "envFrom",
					ContainerName: container.Name,
				})
			}
		}
	}

	return usage
}

// 其他查找函数的实现...
func findConfigMapUsageInDeployment(deployment *appsv1.Deployment, configMapName string) []model.ConfigMapDeploymentUsageEntity {
	// 实现 Deployment 中 ConfigMap 使用情况查找
	// 类似于 findConfigMapUsageInPod 的逻辑
	return []model.ConfigMapDeploymentUsageEntity{}
}

func findConfigMapUsageInStatefulSet(sts *appsv1.StatefulSet, configMapName string) []model.ConfigMapStatefulSetUsageEntity {
	// 实现 StatefulSet 中 ConfigMap 使用情况查找
	return []model.ConfigMapStatefulSetUsageEntity{}
}

func findConfigMapUsageInDaemonSet(ds *appsv1.DaemonSet, configMapName string) []model.ConfigMapDaemonSetUsageEntity {
	// 实现 DaemonSet 中 ConfigMap 使用情况查找
	return []model.ConfigMapDaemonSetUsageEntity{}
}
