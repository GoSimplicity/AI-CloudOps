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
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// BuildK8sSecret 构建secret模型
func BuildK8sSecret(ctx context.Context, clusterID int, secret corev1.Secret) (*model.K8sSecretEntity, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 转换 Data 字段
	data := make(map[string][]byte)
	for k, v := range secret.Data {
		data[k] = v
	}

	// 转换 StringData 字段
	stringData := make(map[string]string)
	for k, v := range secret.StringData {
		stringData[k] = v
	}

	// 转换标签
	labels := make(map[string]string)
	if secret.Labels != nil {
		for k, v := range secret.Labels {
			labels[k] = v
		}
	}

	// 转换注解
	annotations := make(map[string]string)
	if secret.Annotations != nil {
		for k, v := range secret.Annotations {
			annotations[k] = v
		}
	}

	// 计算数据条目数量
	dataCount := len(secret.Data) + len(secret.StringData)

	// 计算数据大小
	size := calculateSecretDataSize(secret)

	// 计算存在时间
	age := CalculateAge(secret.CreationTimestamp.Time)

	k8sSecret := &model.K8sSecretEntity{
		Name:              secret.Name,
		Namespace:         secret.Namespace,
		ClusterID:         clusterID,
		UID:               string(secret.UID),
		Type:              string(secret.Type),
		Data:              data,
		StringData:        stringData,
		Labels:            labels,
		Annotations:       annotations,
		CreationTimestamp: secret.CreationTimestamp.Time,
		Age:               age,
		DataCount:         dataCount,
		Size:              size,
	}

	return k8sSecret, nil
}

// ConvertToSecretEntity 转换secret实体
func ConvertToSecretEntity(secret *model.K8sSecretEntity) *model.SecretEntity {
	if secret == nil {
		return nil
	}

	return &model.SecretEntity{
		Name:        secret.Name,
		Namespace:   secret.Namespace,
		UID:         secret.UID,
		Labels:      secret.Labels,
		Annotations: secret.Annotations,
		Type:        secret.Type,
		Data:        secret.Data,
		StringData:  secret.StringData,
		DataCount:   secret.DataCount,
		Size:        secret.Size,
		Immutable:   false, // Secret 默认可变，除非特别指定
		Age:         secret.Age,
		CreatedAt:   secret.CreationTimestamp.Format(time.RFC3339),
	}
}

// BuildSecretFromRequest 从请求构建secret
func BuildSecretFromRequest(req *model.K8sSecretCreateReq) (*corev1.Secret, error) {
	if req == nil {
		return nil, fmt.Errorf("创建请求不能为空")
	}

	// 如果提供了 SecretYaml，直接使用
	if req.SecretYaml != nil {
		return req.SecretYaml, nil
	}

	// 构建 Secret 对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Type:       corev1.SecretType(req.Type),
		Data:       req.Data,
		StringData: req.StringData,
	}

	// 如果没有指定类型，默认为 Opaque
	if secret.Type == "" {
		secret.Type = corev1.SecretTypeOpaque
	}

	return secret, nil
}

// UpdateSecretFromRequest 从更新请求更新 Kubernetes Secret
func UpdateSecretFromRequest(existing *corev1.Secret, req *model.K8sSecretUpdateReq) (*corev1.Secret, error) {
	if existing == nil {
		return nil, fmt.Errorf("现有Secret不能为空")
	}
	if req == nil {
		return nil, fmt.Errorf("更新请求不能为空")
	}

	// 如果提供了 SecretYaml，直接使用
	if req.SecretYaml != nil {
		return req.SecretYaml, nil
	}

	// 创建一个副本用于更新
	updated := existing.DeepCopy()

	// 更新数据
	if req.Data != nil {
		updated.Data = req.Data
	}
	if req.StringData != nil {
		updated.StringData = req.StringData
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

// GetSecretToYAML 将 Secret 转换为 YAML 字符串
func GetSecretToYAML(ctx context.Context, clientset kubernetes.Interface, clusterID int, namespace, name string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("获取Secret失败: %v", err)
	}

	// 清理系统字段
	secret = CleanSecretForYAML(secret)

	yamlBytes, err := yaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("转换YAML失败: %v", err)
	}

	return string(yamlBytes), nil
}

// CleanSecretForYAML 清理 Secret 对象中的系统字段，用于YAML输出
func CleanSecretForYAML(secret *corev1.Secret) *corev1.Secret {
	cleaned := secret.DeepCopy()

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

// ValidateSecretData 验证 Secret 数据的有效性
func ValidateSecretData(secretType corev1.SecretType, data map[string][]byte, stringData map[string]string) error {
	switch secretType {
	case corev1.SecretTypeServiceAccountToken:
		// ServiceAccount token 应该包含特定的键
		requiredKeys := []string{"token"}
		for _, key := range requiredKeys {
			if _, exists := data[key]; !exists {
				if _, exists := stringData[key]; !exists {
					return fmt.Errorf("ServiceAccount token Secret 必须包含 %s 键", key)
				}
			}
		}
	case corev1.SecretTypeDockerConfigJson:
		// Docker config 应该包含 .dockerconfigjson 键
		if _, exists := data[".dockerconfigjson"]; !exists {
			if _, exists := stringData[".dockerconfigjson"]; !exists {
				return fmt.Errorf("Docker config Secret 必须包含 .dockerconfigjson 键")
			}
		}
	case corev1.SecretTypeTLS:
		// TLS Secret 应该包含 tls.crt 和 tls.key
		requiredKeys := []string{"tls.crt", "tls.key"}
		for _, key := range requiredKeys {
			if _, exists := data[key]; !exists {
				if _, exists := stringData[key]; !exists {
					return fmt.Errorf("TLS Secret 必须包含 %s 键", key)
				}
			}
		}
	}

	return nil
}

// GetSecretUsageInfo 获取 Secret 的使用情况
func GetSecretUsageInfo(ctx context.Context, clientset kubernetes.Interface, namespace, secretName string) (*model.SecretUsageEntity, error) {
	usage := &model.SecretUsageEntity{
		UsedByPods:            []model.SecretPodUsageEntity{},
		UsedByDeployments:     []model.SecretDeploymentUsageEntity{},
		UsedByStatefulSets:    []model.SecretStatefulSetUsageEntity{},
		UsedByDaemonSets:      []model.SecretDaemonSetUsageEntity{},
		UsedByJobs:            []model.SecretJobUsageEntity{},
		UsedByServiceAccounts: []model.SecretServiceAccountUsageEntity{},
	}

	// 检查 Pod 使用情况
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			podUsage := findSecretUsageInPod(&pod, secretName)
			usage.UsedByPods = append(usage.UsedByPods, podUsage...)
		}
	}

	// 检查 Deployment 使用情况
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deployment := range deployments.Items {
			deploymentUsage := findSecretUsageInDeployment(&deployment, secretName)
			usage.UsedByDeployments = append(usage.UsedByDeployments, deploymentUsage...)
		}
	}

	// 检查 StatefulSet 使用情况
	statefulSets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sts := range statefulSets.Items {
			stsUsage := findSecretUsageInStatefulSet(&sts, secretName)
			usage.UsedByStatefulSets = append(usage.UsedByStatefulSets, stsUsage...)
		}
	}

	// 检查 DaemonSet 使用情况
	daemonSets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range daemonSets.Items {
			dsUsage := findSecretUsageInDaemonSet(&ds, secretName)
			usage.UsedByDaemonSets = append(usage.UsedByDaemonSets, dsUsage...)
		}
	}

	// 检查 ServiceAccount 使用情况
	serviceAccounts, err := clientset.CoreV1().ServiceAccounts(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sa := range serviceAccounts.Items {
			saUsage := findSecretUsageInServiceAccount(&sa, secretName)
			usage.UsedByServiceAccounts = append(usage.UsedByServiceAccounts, saUsage...)
		}
	}

	return usage, nil
}

// FilterSecrets 根据条件过滤 Secret 列表
func FilterSecrets(secrets []model.K8sSecretEntity, req *model.GetSecretListReq) []model.K8sSecretEntity {
	var filtered []model.K8sSecretEntity

	for _, secret := range secrets {
		// 命名空间过滤
		if req.Namespace != "" && secret.Namespace != req.Namespace {
			continue
		}

		// 名称过滤
		if req.Name != "" && !strings.Contains(secret.Name, req.Name) {
			continue
		}

		// 类型过滤
		if req.Type != "" && secret.Type != req.Type {
			continue
		}

		// 数据键过滤
		if req.DataKey != "" {
			found := false
			for key := range secret.Data {
				if strings.Contains(key, req.DataKey) {
					found = true
					break
				}
			}
			if !found {
				for key := range secret.StringData {
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
		if req.LabelSelector != "" && !matchesLabelSelector(secret.Labels, req.LabelSelector) {
			continue
		}

		filtered = append(filtered, secret)
	}

	return filtered
}

// SortSecrets 对 Secret 列表进行排序
func SortSecrets(secrets []model.K8sSecretEntity, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Name < secrets[j].Name
		})
	case "namespace":
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Namespace < secrets[j].Namespace
		})
	case "type":
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].Type < secrets[j].Type
		})
	case "created":
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].CreationTimestamp.After(secrets[j].CreationTimestamp)
		})
	default:
		// 默认按创建时间倒序
		sort.Slice(secrets, func(i, j int) bool {
			return secrets[i].CreationTimestamp.After(secrets[j].CreationTimestamp)
		})
	}
}

// calculateSecretDataSize 计算 Secret 数据大小
func calculateSecretDataSize(secret corev1.Secret) string {
	totalSize := 0

	// 计算 Data 字段大小
	for _, value := range secret.Data {
		totalSize += len(value)
	}

	// 计算 StringData 字段大小
	for _, value := range secret.StringData {
		totalSize += len([]byte(value))
	}

	return FormatBytes(totalSize)
}

// findSecretUsageInPod 查找 Pod 中对 Secret 的使用
func findSecretUsageInPod(pod *corev1.Pod, secretName string) []model.SecretPodUsageEntity {
	var usage []model.SecretPodUsageEntity

	// 检查 Volume 中的使用
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == secretName {
			// 查找挂载路径
			for _, container := range pod.Spec.Containers {
				for _, mount := range container.VolumeMounts {
					if mount.Name == volume.Name {
						usage = append(usage, model.SecretPodUsageEntity{
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
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secretName {
				usage = append(usage, model.SecretPodUsageEntity{
					PodName:       pod.Name,
					Namespace:     pod.Namespace,
					UsageType:     "env",
					Keys:          []string{env.ValueFrom.SecretKeyRef.Key},
					ContainerName: container.Name,
				})
			}
		}
	}

	// 检查 ImagePullSecrets
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		if imagePullSecret.Name == secretName {
			usage = append(usage, model.SecretPodUsageEntity{
				PodName:   pod.Name,
				Namespace: pod.Namespace,
				UsageType: "imagePullSecret",
			})
		}
	}

	return usage
}

// 其他查找函数的实现...
func findSecretUsageInDeployment(deployment *appsv1.Deployment, secretName string) []model.SecretDeploymentUsageEntity {
	// 实现 Deployment 中 Secret 使用情况查找
	// 类似于 findSecretUsageInPod 的逻辑
	return []model.SecretDeploymentUsageEntity{}
}

func findSecretUsageInStatefulSet(sts *appsv1.StatefulSet, secretName string) []model.SecretStatefulSetUsageEntity {
	// 实现 StatefulSet 中 Secret 使用情况查找
	return []model.SecretStatefulSetUsageEntity{}
}

func findSecretUsageInDaemonSet(ds *appsv1.DaemonSet, secretName string) []model.SecretDaemonSetUsageEntity {
	// 实现 DaemonSet 中 Secret 使用情况查找
	return []model.SecretDaemonSetUsageEntity{}
}

func findSecretUsageInServiceAccount(sa *corev1.ServiceAccount, secretName string) []model.SecretServiceAccountUsageEntity {
	var usage []model.SecretServiceAccountUsageEntity

	// 检查 Secrets 字段
	for _, secret := range sa.Secrets {
		if secret.Name == secretName {
			usage = append(usage, model.SecretServiceAccountUsageEntity{
				ServiceAccountName: sa.Name,
				Namespace:          sa.Namespace,
				UsageType:          "token",
			})
		}
	}

	// 检查 ImagePullSecrets 字段
	for _, imagePullSecret := range sa.ImagePullSecrets {
		if imagePullSecret.Name == secretName {
			usage = append(usage, model.SecretServiceAccountUsageEntity{
				ServiceAccountName: sa.Name,
				Namespace:          sa.Namespace,
				UsageType:          "imagePullSecret",
			})
		}
	}

	return usage
}

// CalculateAge 计算资源存在时间
func CalculateAge(creationTime time.Time) string {
	return duration.HumanDuration(time.Since(creationTime))
}

// FormatBytes 格式化字节数
func FormatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := unit, 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// matchesLabelSelector 检查标签是否匹配选择器
func matchesLabelSelector(labels map[string]string, selector string) bool {
	if selector == "" {
		return true
	}

	// 简单的标签选择器实现，支持 key=value 格式
	parts := strings.Split(selector, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "=") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				value := strings.TrimSpace(kv[1])
				if labels[key] != value {
					return false
				}
			}
		} else {
			// 只检查键是否存在
			if _, exists := labels[part]; !exists {
				return false
			}
		}
	}
	return true
}
