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
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// DrainOptions 驱逐节点选项
type DrainOptions struct {
	Force              int8 // 是否强制驱逐
	IgnoreDaemonSets   int8 // 是否忽略DaemonSet
	DeleteLocalData    int8 // 是否删除本地数据
	GracePeriodSeconds int  // 优雅关闭时间(秒)
	TimeoutSeconds     int  // 超时时间(秒)
}

// BuildK8sNode 构建详细的 K8sNode 模型
func BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node, kubeClient *kubernetes.Clientset, metricsClient interface{}) (*model.K8sNode, error) {
	if clusterID <= 0 {
		return nil, fmt.Errorf("无效的集群ID: %d", clusterID)
	}

	// 获取节点状态
	status := getNodeStatus(node)

	// 判断是否可调度
	schedulable := int8(1)
	if node.Spec.Unschedulable {
		schedulable = int8(2)
	}

	// 获取节点角色
	roles := getNodeRoles(node)

	// 获取节点 IP 和主机名
	internalIP := getNodeInternalIP(node)
	externalIP := getNodeExternalIP(node)
	hostname := getNodeHostname(node)

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
		HostName:         hostname,
		KubeletVersion:   node.Status.NodeInfo.KubeletVersion,
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

	return k8sNode, nil
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

// getNodeHostname 获取节点主机名
func getNodeHostname(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeHostName {
			return address.Address
		}
	}
	return node.Name // 如果没有找到主机名，返回节点名称
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

// IsSystemNamespace 判断是否为系统命名空间
func IsSystemNamespace(namespace string) bool {
	systemNamespaces := []string{
		"kube-system",
		"kube-public",
		"kube-node-lease",
		"default",
	}

	for _, ns := range systemNamespaces {
		if namespace == ns {
			return true
		}
	}
	return false
}

// ShouldSkipPodDrain 判断是否应该跳过Pod驱逐
func ShouldSkipPodDrain(pod corev1.Pod, options *DrainOptions) bool {
	// 跳过系统命名空间的Pod（除非强制）
	if options.Force != 1 && IsSystemNamespace(pod.Namespace) {
		return true
	}

	// 跳过DaemonSet Pod（除非设置忽略）
	if IsDaemonSetPod(pod) && options.IgnoreDaemonSets == 1 {
		return true
	}

	return false
}

// ValidateBasicParams 验证基础参数
func ValidateBasicParams(clusterID int, nodeName string) error {
	if clusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	return nil
}

// ValidateNodeName 验证节点名称
func ValidateNodeName(nodeName string) error {
	if nodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	return nil
}

// ValidateNodeLabelsMap 验证节点标签映射
func ValidateNodeLabelsMap(labels map[string]string) error {
	if len(labels) == 0 {
		return fmt.Errorf("标签不能为空")
	}
	return ValidateNodeLabels(labels)
}

// ValidateLabelKeys 验证标签键
func ValidateLabelKeys(labelKeys []string) error {
	if len(labelKeys) == 0 {
		return fmt.Errorf("标签键不能为空")
	}
	return nil
}

// BuildNodeTaints 构建节点污点列表
func BuildNodeTaints(taints []corev1.Taint) ([]*model.NodeTaint, int64) {
	var taintEntities []*model.NodeTaint
	for _, taint := range taints {
		taintEntity := &model.NodeTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		taintEntities = append(taintEntities, taintEntity)
	}

	return taintEntities, int64(len(taintEntities))
}
