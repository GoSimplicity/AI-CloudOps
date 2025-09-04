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
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/yaml"
)

// BuildK8sDeployment 构建K8sDeployment模型
func BuildK8sDeployment(ctx context.Context, clusterID int, deployment appsv1.Deployment) (*model.K8sDeployment, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 获取部署状态
	status := getDeploymentStatus(deployment)

	// 获取部署策略信息
	strategy := "RollingUpdate"
	maxUnavailable := ""
	maxSurge := ""
	if deployment.Spec.Strategy.Type == appsv1.RecreateDeploymentStrategyType {
		strategy = "Recreate"
	} else if deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = deployment.Spec.Strategy.RollingUpdate.MaxSurge.String()
		}
	}

	// 获取容器镜像列表
	var images []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	// 构建条件列表
	var conditions []model.DeploymentCondition
	for _, condition := range deployment.Status.Conditions {
		conditions = append(conditions, model.DeploymentCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastUpdateTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// 构建标签选择器
	selector := make(map[string]string)
	if deployment.Spec.Selector != nil && deployment.Spec.Selector.MatchLabels != nil {
		selector = deployment.Spec.Selector.MatchLabels
	}

	// 构建基础部署信息
	k8sDeployment := &model.K8sDeployment{
		Name:              deployment.Name,
		Namespace:         deployment.Namespace,
		ClusterID:         clusterID,
		UID:               string(deployment.UID),
		Replicas:          GetInt32Value(deployment.Spec.Replicas),
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		Strategy:          strategy,
		MaxUnavailable:    maxUnavailable,
		MaxSurge:          maxSurge,
		Selector:          selector,
		Labels:            deployment.Labels,
		Annotations:       deployment.Annotations,
		Images:            images,
		Status:            status,
		Conditions:        conditions,
		CreatedAt:         deployment.CreationTimestamp.Time,
		UpdatedAt:         time.Now(),
		RawDeployment:     &deployment,
	}

	return k8sDeployment, nil
}

// getDeploymentStatus 获取部署状态
func getDeploymentStatus(deployment appsv1.Deployment) model.K8sDeploymentStatus {
	// 首先检查是否处于暂停状态
	if deployment.Spec.Paused {
		return model.K8sDeploymentStatusPaused
	}

	// 如果副本数为0，认为是停止状态
	if GetInt32Value(deployment.Spec.Replicas) == 0 {
		return model.K8sDeploymentStatusStopped
	}

	// 检查部署条件
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			if condition.Status == corev1.ConditionFalse {
				return model.K8sDeploymentStatusError
			}
		}
		if condition.Type == appsv1.DeploymentAvailable {
			if condition.Status == corev1.ConditionTrue &&
				deployment.Status.ReadyReplicas == GetInt32Value(deployment.Spec.Replicas) {
				return model.K8sDeploymentStatusRunning
			}
		}
	}

	// 如果就绪副本数不等于期望副本数，认为是异常状态
	if deployment.Status.ReadyReplicas != GetInt32Value(deployment.Spec.Replicas) {
		return model.K8sDeploymentStatusError
	}

	return model.K8sDeploymentStatusRunning
}

// GetInt32Value 安全获取int32指针的值
func GetInt32Value(ptr *int32) int32 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// BuildDeploymentListOptions 构建部署列表查询选项
func BuildDeploymentListOptions(req *model.GetDeploymentListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 构建标签选择器
	var labelSelectors []string
	for key, value := range req.Labels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	if len(labelSelectors) > 0 {
		options.LabelSelector = strings.Join(labelSelectors, ",")
	}

	return options
}

// FilterDeploymentsByStatus 根据部署状态过滤
func FilterDeploymentsByStatus(deployments []appsv1.Deployment, status string) []appsv1.Deployment {
	if status == "" {
		return deployments
	}

	var filtered []appsv1.Deployment
	for _, deployment := range deployments {
		deploymentStatus := getDeploymentStatus(deployment)
		// 正确转换状态为字符串
		var statusStr string
		switch deploymentStatus {
		case model.K8sDeploymentStatusRunning:
			statusStr = "running"
		case model.K8sDeploymentStatusStopped:
			statusStr = "stopped"
		case model.K8sDeploymentStatusPaused:
			statusStr = "paused"
		case model.K8sDeploymentStatusError:
			statusStr = "error"
		default:
			statusStr = "unknown"
		}
		if strings.EqualFold(statusStr, status) {
			filtered = append(filtered, deployment)
		}
	}

	return filtered
}

// GetDeploymentEvents 获取部署事件
func GetDeploymentEvents(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, deploymentName string, limit int) ([]*model.K8sDeploymentEvent, int64, error) {
	eventList, err := kubeClient.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Deployment", deploymentName),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("获取部署事件失败: %w", err)
	}

	total := int64(len(eventList.Items))

	// 按时间排序
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.Time.After(eventList.Items[j].LastTimestamp.Time)
	})

	var events []*model.K8sDeploymentEvent
	count := 0
	for _, event := range eventList.Items {
		if limit > 0 && count >= limit {
			break
		}

		deploymentEvent := &model.K8sDeploymentEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Count:     event.Count,
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Source:    event.Source.Component,
		}
		events = append(events, deploymentEvent)
		count++
	}

	return events, total, nil
}

// GetDeploymentPods 获取部署关联的Pod列表
func GetDeploymentPods(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, deploymentName string) ([]*model.K8sPod, int64, error) {
	// 首先获取Deployment的标签选择器
	deployment, err := kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取部署信息失败: %w", err)
	}

	// 构建标签选择器
	var labelSelectors []string
	for key, value := range deployment.Spec.Selector.MatchLabels {
		labelSelectors = append(labelSelectors, fmt.Sprintf("%s=%s", key, value))
	}
	labelSelector := strings.Join(labelSelectors, ",")

	// 获取Pod列表
	podList, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("获取Pod列表失败: %w", err)
	}

	total := int64(len(podList.Items))
	var pods []*model.K8sPod
	for _, pod := range podList.Items {
		k8sPod := &model.K8sPod{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			Status:      string(pod.Status.Phase),
			NodeName:    pod.Spec.NodeName,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
		}
		pods = append(pods, k8sPod)
	}

	return pods, total, nil
}

// GetDeploymentHistory 获取部署历史版本
func GetDeploymentHistory(ctx context.Context, kubeClient *kubernetes.Clientset, namespace, deploymentName string) ([]*model.K8sDeploymentHistory, int64, error) {
	// 获取ReplicaSet列表
	replicaSets, err := kubeClient.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取ReplicaSet列表失败: %w", err)
	}

	// 获取Deployment信息
	_, err = kubeClient.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, 0, fmt.Errorf("获取部署信息失败: %w", err)
	}

	var history []*model.K8sDeploymentHistory
	for _, rs := range replicaSets.Items {
		// 检查ReplicaSet是否属于该Deployment
		for _, ownerRef := range rs.OwnerReferences {
			if ownerRef.Kind == "Deployment" && ownerRef.Name == deploymentName {
				if revisionStr, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
					revision, _ := strconv.ParseInt(revisionStr, 10, 64)
					historyItem := &model.K8sDeploymentHistory{
						Revision: revision,
						Date:     rs.CreationTimestamp.Time,
						Message:  fmt.Sprintf("ReplicaSet %s", rs.Name),
					}
					history = append(history, historyItem)
				}
				break
			}
		}
	}

	// 按版本号排序
	sort.Slice(history, func(i, j int) bool {
		return history[i].Revision > history[j].Revision
	})

	total := int64(len(history))
	return history, total, nil
}

// DeploymentToYAML 将Deployment转换为YAML
func DeploymentToYAML(deployment *appsv1.Deployment) (string, error) {
	if deployment == nil {
		return "", fmt.Errorf("deployment不能为空")
	}

	// 清理不需要的字段
	cleanDeployment := deployment.DeepCopy()
	cleanDeployment.Status = appsv1.DeploymentStatus{}
	cleanDeployment.ManagedFields = nil
	cleanDeployment.ResourceVersion = ""
	cleanDeployment.UID = ""
	cleanDeployment.CreationTimestamp = metav1.Time{}
	cleanDeployment.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanDeployment)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

// YAMLToDeployment 将YAML转换为Deployment
func YAMLToDeployment(yamlContent string) (*appsv1.Deployment, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	var deployment appsv1.Deployment
	err := yaml.Unmarshal([]byte(yamlContent), &deployment)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &deployment, nil
}

// ValidateDeployment 验证Deployment配置
func ValidateDeployment(deployment *appsv1.Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment不能为空")
	}

	if deployment.Name == "" {
		return fmt.Errorf("deployment名称不能为空")
	}

	if deployment.Namespace == "" {
		return fmt.Errorf("namespace不能为空")
	}

	if deployment.Spec.Selector == nil || len(deployment.Spec.Selector.MatchLabels) == 0 {
		return fmt.Errorf("selector不能为空")
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("至少需要一个容器")
	}

	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == "" {
			return fmt.Errorf("容器%d名称不能为空", i)
		}
		if container.Image == "" {
			return fmt.Errorf("容器%d镜像不能为空", i)
		}
	}

	return nil
}

// BuildDeploymentFromRequest 从请求构建Deployment对象
func BuildDeploymentFromRequest(req *model.CreateDeploymentReq) (*appsv1.Deployment, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	// 从请求字段构建
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &req.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: req.Labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: req.Labels,
				},
				Spec: corev1.PodSpec{},
			},
		},
	}

	// 构建容器列表
	var containers []corev1.Container
	for i, image := range req.Images {
		container := corev1.Container{
			Name:  fmt.Sprintf("container-%d", i),
			Image: image,
		}
		containers = append(containers, container)
	}
	deployment.Spec.Template.Spec.Containers = containers

	return deployment, nil
}

// IsDeploymentReady 判断Deployment是否就绪
func IsDeploymentReady(deployment appsv1.Deployment) bool {
	return deployment.Status.ReadyReplicas == GetInt32Value(deployment.Spec.Replicas) &&
		deployment.Status.UpdatedReplicas == GetInt32Value(deployment.Spec.Replicas)
}

// GetDeploymentAge 获取Deployment年龄
func GetDeploymentAge(deployment appsv1.Deployment) string {
	age := time.Since(deployment.CreationTimestamp.Time)
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

// BuildDeploymentListPagination 构建部署列表分页逻辑
func BuildDeploymentListPagination(deployments []appsv1.Deployment, page, size int) ([]appsv1.Deployment, int64) {
	total := int64(len(deployments))
	if total == 0 {
		return []appsv1.Deployment{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return deployments, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []appsv1.Deployment{}, total
	}
	if end > total {
		end = total
	}

	return deployments[start:end], total
}

// PaginateK8sDeployments 对 K8sDeployment 列表进行分页
func PaginateK8sDeployments(deployments []*model.K8sDeployment, page, size int) ([]*model.K8sDeployment, int64) {
	total := int64(len(deployments))
	if total == 0 {
		return []*model.K8sDeployment{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || size <= 0 {
		return deployments, total
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []*model.K8sDeployment{}, total
	}
	if end > total {
		end = total
	}

	return deployments[start:end], total
}

// CalculateDeploymentMetrics 计算 Deployment 指标

// ConvertToK8sDeployment 将 appsv1.Deployment 转换为 model.K8sDeployment
func ConvertToK8sDeployment(deployment *appsv1.Deployment) *model.K8sDeployment {
	if deployment == nil {
		return nil
	}

	// 获取部署状态
	status := getDeploymentStatus(*deployment)

	// 获取部署策略信息
	strategy := "RollingUpdate"
	maxUnavailable := ""
	maxSurge := ""
	if deployment.Spec.Strategy.Type == appsv1.RecreateDeploymentStrategyType {
		strategy = "Recreate"
	} else if deployment.Spec.Strategy.RollingUpdate != nil {
		if deployment.Spec.Strategy.RollingUpdate.MaxUnavailable != nil {
			maxUnavailable = deployment.Spec.Strategy.RollingUpdate.MaxUnavailable.String()
		}
		if deployment.Spec.Strategy.RollingUpdate.MaxSurge != nil {
			maxSurge = deployment.Spec.Strategy.RollingUpdate.MaxSurge.String()
		}
	}

	// 获取容器镜像列表
	var images []string
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	// 构建条件列表
	var conditions []model.DeploymentCondition
	for _, condition := range deployment.Status.Conditions {
		conditions = append(conditions, model.DeploymentCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastUpdateTime:     condition.LastUpdateTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// 构建标签选择器
	selector := make(map[string]string)
	if deployment.Spec.Selector != nil && deployment.Spec.Selector.MatchLabels != nil {
		selector = deployment.Spec.Selector.MatchLabels
	}

	return &model.K8sDeployment{
		Name:              deployment.Name,
		Namespace:         deployment.Namespace,
		UID:               string(deployment.UID),
		Replicas:          GetInt32Value(deployment.Spec.Replicas),
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		Strategy:          strategy,
		MaxUnavailable:    maxUnavailable,
		MaxSurge:          maxSurge,
		Selector:          selector,
		Labels:            deployment.Labels,
		Annotations:       deployment.Annotations,
		Images:            images,
		Status:            status,
		Conditions:        conditions,
		CreatedAt:         deployment.CreationTimestamp.Time,
		UpdatedAt:         time.Now(),
		RawDeployment:     deployment,
	}
}

// BuildDeploymentFromYaml 从YAML构建Deployment对象
func BuildDeploymentFromYaml(req *model.CreateDeploymentByYamlReq) (*appsv1.Deployment, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	deployment, err := YAMLToDeployment(req.YAML)
	if err != nil {
		return nil, err
	}

	// YAML中必须包含namespace和name信息
	if deployment.Namespace == "" {
		return nil, fmt.Errorf("YAML中必须指定namespace")
	}

	if deployment.Name == "" {
		return nil, fmt.Errorf("YAML中必须指定name")
	}

	return deployment, nil
}

// BuildDeploymentFromYamlForUpdate 从YAML构建Deployment对象用于更新
func BuildDeploymentFromYamlForUpdate(req *model.UpdateDeploymentByYamlReq) (*appsv1.Deployment, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	if req.YAML == "" {
		return nil, fmt.Errorf("YAML内容不能为空")
	}

	deployment, err := YAMLToDeployment(req.YAML)
	if err != nil {
		return nil, err
	}

	// 确保YAML中的namespace和name与请求参数一致
	if deployment.Namespace != "" && deployment.Namespace != req.Namespace {
		return nil, fmt.Errorf("YAML中的namespace (%s) 与请求参数不一致 (%s)", deployment.Namespace, req.Namespace)
	}

	if deployment.Name != "" && deployment.Name != req.Name {
		return nil, fmt.Errorf("YAML中的name (%s) 与请求参数不一致 (%s)", deployment.Name, req.Name)
	}

	// 如果YAML中没有指定，使用请求参数
	if deployment.Namespace == "" {
		deployment.Namespace = req.Namespace
	}

	if deployment.Name == "" {
		deployment.Name = req.Name
	}

	return deployment, nil
}
