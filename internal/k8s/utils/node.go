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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// DrainOptions 驱逐节点选项
type DrainOptions struct {
	Force              bool // 是否强制驱逐
	IgnoreDaemonSets   bool // 是否忽略DaemonSet
	DeleteLocalData    bool // 是否删除本地数据
	GracePeriodSeconds int  // 优雅关闭时间(秒)
	TimeoutSeconds     int  // 超时时间(秒)
}

// BuildK8sNode 构建详细的 K8sNode 模型
func BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node, kubeClient *kubernetes.Clientset, metricsClient *metricsClient.Clientset) (*model.K8sNode, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 获取节点状态
	status := getNodeStatus(node)

	// 判断是否可调度
	schedulable := !node.Spec.Unschedulable

	// 获取节点角色
	roles := getNodeRoles(node)

	// 获取节点 IP
	internalIP := getNodeInternalIP(node)
	externalIP := getNodeExternalIP(node)

	// 获取节点的年龄
	age := calculateAge(node.CreationTimestamp.Time)

	// 构建基础节点信息
	k8sNode := &model.K8sNode{
		Name:             node.Name,
		ClusterID:        clusterID,
		Status:           status,
		Schedulable:      schedulable,
		Roles:            roles,
		Age:              age,
		InternalIP:       internalIP,
		ExternalIP:       externalIP,
		HostName:         node.Status.NodeInfo.MachineID,
		KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
		KubeProxyVersion: node.Status.NodeInfo.KubeProxyVersion,
		ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
		OperatingSystem:  node.Status.NodeInfo.OperatingSystem,
		Architecture:     node.Status.NodeInfo.Architecture,
		KernelVersion:    node.Status.NodeInfo.KernelVersion,
		OSImage:          node.Status.NodeInfo.OSImage,
		Labels:           node.Labels,
		Annotations:      node.Annotations,
		Conditions:       node.Status.Conditions,
		Taints:           node.Spec.Taints,
		CreatedAt:        node.CreationTimestamp.Time,
		UpdatedAt:        time.Now(),
		RawNode:          &node,
	}

	// 获取节点资源信息
	if node.Status.Capacity != nil {
		k8sNode.CPU = buildResourceInfo("cpu", node.Status.Capacity, node.Status.Allocatable, kubeClient, ctx, node.Name)
		k8sNode.Memory = buildResourceInfo("memory", node.Status.Capacity, node.Status.Allocatable, kubeClient, ctx, node.Name)
		k8sNode.Storage = buildResourceInfo("storage", node.Status.Capacity, node.Status.Allocatable, kubeClient, ctx, node.Name)
		k8sNode.EphemeralStorage = buildResourceInfo("ephemeral-storage", node.Status.Capacity, node.Status.Allocatable, kubeClient, ctx, node.Name)
		k8sNode.Pods = buildPodResourceInfo(node.Status.Capacity, kubeClient, ctx, node.Name)
	}

	// 获取节点事件
	events, err := getNodeEvents(ctx, kubeClient, node.Name, 10)
	if err == nil {
		k8sNode.Events = events
	}

	return k8sNode, nil
}

// buildResourceInfo 构建资源信息
func buildResourceInfo(resourceName string, capacity, allocatable corev1.ResourceList, kubeClient *kubernetes.Clientset, ctx context.Context, nodeName string) model.NodeResource {
	resourceInfo := model.NodeResource{}

	// 获取容量信息
	if cap, ok := capacity[corev1.ResourceName(resourceName)]; ok {
		resourceInfo.Total = cap.String()
	}

	// 获取可分配信息
	if alloc, ok := allocatable[corev1.ResourceName(resourceName)]; ok {
		resourceInfo.Requests = alloc.String()
	}

	// 计算已使用量和限制量（基于Pod的请求量和限制量）
	if kubeClient != nil {
		pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + nodeName,
		})
		if err == nil {
			var used resource.Quantity
			var limits resource.Quantity

			for _, pod := range pods.Items {
				// 跳过已完成或失败的Pod
				if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					continue
				}

				for _, container := range pod.Spec.Containers {
					// 计算请求量
					if req := container.Resources.Requests; req != nil {
						if resReq := req[corev1.ResourceName(resourceName)]; !resReq.IsZero() {
							used.Add(resReq)
						}
					}

					// 计算限制量
					if limit := container.Resources.Limits; limit != nil {
						if resLimit := limit[corev1.ResourceName(resourceName)]; !resLimit.IsZero() {
							limits.Add(resLimit)
						}
					}
				}

				// 处理InitContainers
				for _, initContainer := range pod.Spec.InitContainers {
					if req := initContainer.Resources.Requests; req != nil {
						if resReq := req[corev1.ResourceName(resourceName)]; !resReq.IsZero() {
							used.Add(resReq)
						}
					}

					if limit := initContainer.Resources.Limits; limit != nil {
						if resLimit := limit[corev1.ResourceName(resourceName)]; !resLimit.IsZero() {
							limits.Add(resLimit)
						}
					}
				}
			}

			resourceInfo.Used = used.String()
			resourceInfo.Limits = limits.String()

			// 计算使用百分比（基于可分配资源）
			if allocQuantity, ok := allocatable[corev1.ResourceName(resourceName)]; ok && !allocQuantity.IsZero() {
				resourceInfo.Percent = float64(used.MilliValue()) / float64(allocQuantity.MilliValue()) * 100
				// 确保百分比不超过100%
				if resourceInfo.Percent > 100 {
					resourceInfo.Percent = 100
				}
			}
		}
	}

	return resourceInfo
}

// buildPodResourceInfo 构建Pod资源信息
func buildPodResourceInfo(capacity corev1.ResourceList, kubeClient *kubernetes.Clientset, ctx context.Context, nodeName string) model.NodeResource {
	resourceInfo := model.NodeResource{}

	// 获取Pod容量信息
	if cap, ok := capacity[corev1.ResourcePods]; ok {
		resourceInfo.Total = cap.String()
	}

	// 获取当前运行的Pod数量
	if kubeClient != nil {
		pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + nodeName,
		})
		if err == nil {
			resourceInfo.Used = fmt.Sprintf("%d", len(pods.Items))

			// 计算使用百分比
			if capQuantity, ok := capacity[corev1.ResourcePods]; ok && !capQuantity.IsZero() {
				resourceInfo.Percent = float64(len(pods.Items)) / float64(capQuantity.Value()) * 100
			}
		}
	}

	return resourceInfo
}

// getNodeStatus 获取节点状态
func getNodeStatus(node corev1.Node) model.NodeStatus {
	// 首先检查节点是否被禁止调度
	if node.Spec.Unschedulable {
		return model.NodeStatusSchedulingDisabled
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return model.NodeStatusReady
			}
			return model.NodeStatusNotReady
		}
	}
	return model.NodeStatusUnknown
}

// getNodeRoles 获取节点角色
func getNodeRoles(node corev1.Node) []string {
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}
	return roles
}

// getNodeInternalIP 获取节点内部 IP
func getNodeInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

// getNodeExternalIP 获取节点外部 IP
func getNodeExternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeExternalIP {
			return address.Address
		}
	}
	return ""
}

// calculateAge 计算年龄
func calculateAge(creationTime time.Time) string {
	age := time.Since(creationTime)
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

// getNodeEvents 获取节点事件
func getNodeEvents(ctx context.Context, kubeClient *kubernetes.Clientset, nodeName string, limit int) ([]model.NodeEvent, error) {
	eventList, err := kubeClient.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})
	if err != nil {
		return nil, err
	}

	// 按时间排序
	sort.Slice(eventList.Items, func(i, j int) bool {
		return eventList.Items[i].LastTimestamp.Time.After(eventList.Items[j].LastTimestamp.Time)
	})

	var events []model.NodeEvent
	count := 0
	for _, event := range eventList.Items {
		if limit > 0 && count >= limit {
			break
		}

		nodeEvent := model.NodeEvent{
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        event.Message,
			Component:      event.Source.Component,
			Host:           event.Source.Host,
			FirstTimestamp: event.FirstTimestamp.Time,
			LastTimestamp:  event.LastTimestamp.Time,
			Count:          event.Count,
		}
		events = append(events, nodeEvent)
		count++
	}

	return events, nil
}

// BuildNodeResources 构建节点资源信息
func BuildNodeResource(ctx context.Context, kubeClient *kubernetes.Clientset, node *corev1.Node) *model.NodeResource {
	if node == nil {
		return &model.NodeResource{}
	}

	if node.Status.Capacity == nil {
		return &model.NodeResource{}
	}

	// 获取CPU资源信息
	cpuInfo := buildResourceInfo("cpu", node.Status.Capacity, node.Status.Allocatable, kubeClient, ctx, node.Name)

	return &model.NodeResource{
		Used:     cpuInfo.Used,
		Total:    cpuInfo.Total,
		Percent:  cpuInfo.Percent,
		Requests: cpuInfo.Requests,
		Limits:   cpuInfo.Limits,
	}
}

// ValidateNodeLabels 验证节点标签
func ValidateNodeLabels(labels map[string]string) error {
	for key, value := range labels {
		if key == "" {
			return fmt.Errorf("标签键不能为空")
		}
		if len(key) > 253 {
			return fmt.Errorf("标签键长度不能超过253个字符")
		}
		if len(value) > 63 {
			return fmt.Errorf("标签值长度不能超过63个字符")
		}
	}
	return nil
}

// BuildNodeListOptions 构建节点列表查询选项
func BuildNodeListOptions(req *model.GetNodeListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	if req.LabelSelector != "" {
		options.LabelSelector = req.LabelSelector
	}

	if req.FieldSelector != "" {
		options.FieldSelector = req.FieldSelector
	}

	return options
}

// FilterNodesByNames 根据节点名称过滤
func FilterNodesByNames(nodes []corev1.Node, nodeNames []string) []corev1.Node {
	if len(nodeNames) == 0 {
		return nodes
	}

	nameSet := make(map[string]bool)
	for _, name := range nodeNames {
		nameSet[name] = true
	}

	var filtered []corev1.Node
	for _, node := range nodes {
		if nameSet[node.Name] {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// FilterNodesByStatus 根据节点状态过滤
func FilterNodesByStatus(nodes []corev1.Node, statuses []model.NodeStatus) []corev1.Node {
	if len(statuses) == 0 {
		return nodes
	}

	statusSet := make(map[model.NodeStatus]bool)
	for _, status := range statuses {
		statusSet[status] = true
	}

	var filtered []corev1.Node
	for _, node := range nodes {
		nodeStatus := getNodeStatus(node)
		if statusSet[nodeStatus] {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

// FilterNodesByRoles 根据节点角色过滤
func FilterNodesByRoles(nodes []corev1.Node, roles []string) []corev1.Node {
	if len(roles) == 0 {
		return nodes
	}

	roleSet := make(map[string]bool)
	for _, role := range roles {
		roleSet[role] = true
	}

	var filtered []corev1.Node
	for _, node := range nodes {
		nodeRoles := getNodeRoles(node)
		for _, nodeRole := range nodeRoles {
			if roleSet[nodeRole] {
				filtered = append(filtered, node)
				break
			}
		}
	}

	return filtered
}

// GetNodeStatusMessage 获取节点状态描述信息
func GetNodeStatusMessage(node corev1.Node) string {
	if node.Spec.Unschedulable {
		return "调度已禁用"
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "就绪"
			}
			if condition.Message != "" {
				return fmt.Sprintf("未就绪: %s", condition.Message)
			}
			return "未就绪"
		}
	}
	return "状态未知"
}

// IsNodeReady 判断节点是否就绪
func IsNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// BuildNodeListPagination 构建节点列表分页逻辑
func BuildNodeListPagination(nodes []corev1.Node, page, size int) ([]corev1.Node, int64) {
	total := int64(len(nodes))
	if total == 0 {
		return []corev1.Node{}, 0
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []corev1.Node{}, total
	}
	if end > total {
		end = total
	}

	return nodes[start:end], total
}

// ApplyNodeFilters 应用所有节点过滤器
func ApplyNodeFilters(nodes []corev1.Node, req *model.GetNodeListReq) []corev1.Node {
	filtered := nodes

	if len(req.NodeNames) > 0 {
		filtered = FilterNodesByNames(filtered, req.NodeNames)
	}
	if len(req.Status) > 0 {
		filtered = FilterNodesByStatus(filtered, req.Status)
	}
	if len(req.Roles) > 0 {
		filtered = FilterNodesByRoles(filtered, req.Roles)
	}

	return filtered
}

// IsDaemonSetPod 判断是否为DaemonSet Pod
func IsDaemonSetPod(pod corev1.Pod) bool {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// IsActivePod 判断Pod是否为活跃状态
func IsActivePod(pod corev1.Pod) bool {
	return pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed
}

// BuildDeleteOptions 构建删除选项
func BuildDeleteOptions(gracePeriodSeconds int) metav1.DeleteOptions {
	deleteOptions := metav1.DeleteOptions{}
	if gracePeriodSeconds > 0 {
		gracePeriod := int64(gracePeriodSeconds)
		deleteOptions.GracePeriodSeconds = &gracePeriod
	}
	return deleteOptions
}

// ShouldSkipPodDrain 判断是否应该跳过Pod驱逐
func ShouldSkipPodDrain(pod corev1.Pod, options *DrainOptions) bool {
	// 跳过系统命名空间的Pod（除非强制）
	if !options.Force && IsSystemNamespace(pod.Namespace) {
		return true
	}

	// 跳过DaemonSet Pod（除非设置忽略）
	if IsDaemonSetPod(pod) && !options.IgnoreDaemonSets {
		return true
	}

	return false
}
