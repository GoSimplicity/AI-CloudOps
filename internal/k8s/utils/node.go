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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// BuildK8sNode 构建详细的 K8sNode 模型
func BuildK8sNode(ctx context.Context, clusterID int, node corev1.Node, kubeClient *kubernetes.Clientset, metricsClient *metricsClient.Clientset) (*model.K8sNode, error) {
	// 获取节点状态
	status := getNodeStatus(node)

	// 判断是否可调度
	scheduleEnable := !node.Spec.Unschedulable

	// 获取节点角色
	roles := getNodeRoles(node)

	// 获取节点 IP
	ip := getNodeInternalIP(node)

	// 获取标签（作为字符串切片）
	var labels []string
	for key, value := range node.Labels {
		labels = append(labels, key+"="+value)
	}

	k8sNode := &model.K8sNode{
		Name:           node.Name,
		ClusterID:      clusterID,
		Status:         status,
		ScheduleEnable: scheduleEnable,
		Roles:          roles,
		Age:            calculateAge(node.CreationTimestamp.Time),
		IP:             ip,
		KubeletVersion: node.Status.NodeInfo.KubeletVersion,
		CriVersion:     node.Status.NodeInfo.ContainerRuntimeVersion,
		OsVersion:      node.Status.NodeInfo.OSImage,
		KernelVersion:  node.Status.NodeInfo.KernelVersion,
		Labels:         labels,
	}

	// 获取节点 Pod 数量
	pods, err := kubeClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})
	if err == nil {
		k8sNode.PodNum = len(pods.Items)
	}

	// 获取资源信息
	if node.Status.Capacity != nil {
		k8sNode.CpuCores = getResourceInfo("cpu", node.Status.Capacity, node.Status.Allocatable)
		k8sNode.MemGibs = getResourceInfo("memory", node.Status.Capacity, node.Status.Allocatable)
		k8sNode.EphemeralStorage = getResourceInfo("ephemeral-storage", node.Status.Capacity, node.Status.Allocatable)
	}

	return k8sNode, nil
}

// getNodeStatus 获取节点状态
func getNodeStatus(node corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// convertResourceList 转换资源列表
func convertResourceList(resources corev1.ResourceList) map[string]string {
	result := make(map[string]string)
	for key, value := range resources {
		result[string(key)] = value.String()
	}
	return result
}

// getNodeRoles 获取节点角色
func getNodeRoles(node corev1.Node) []string {
	var roles []string

	// 检查是否是控制平面节点
	if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
		roles = append(roles, "master")
	}
	if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
		roles = append(roles, "control-plane")
	}

	// 检查工作节点
	if len(roles) == 0 {
		roles = append(roles, "worker")
	}

	return roles
}

// getNodeInternalIP 获取节点内部IP
func getNodeInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

// calculateAge 计算存在时间
func calculateAge(creationTime time.Time) string {
	duration := time.Since(creationTime)
	days := int(duration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(duration.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(duration.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

// getResourceInfo 获取资源信息
func getResourceInfo(resourceName string, capacity, allocatable corev1.ResourceList) string {
	var capStr, allocStr string

	if cap, ok := capacity[corev1.ResourceName(resourceName)]; ok {
		capStr = cap.String()
	}
	if alloc, ok := allocatable[corev1.ResourceName(resourceName)]; ok {
		allocStr = alloc.String()
	}

	if capStr != "" && allocStr != "" {
		return fmt.Sprintf("%s/%s", allocStr, capStr)
	} else if capStr != "" {
		return capStr
	}

	return "unknown"
}
