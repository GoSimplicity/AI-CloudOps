package k8s

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

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

const QuotaName = "compute-quota"

// EnsureNamespace 确保指定的命名空间存在，如果不存在则创建
func EnsureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
	// 获取命名空间
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// 如果命名空间不存在，则创建
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, createErr := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if createErr != nil {
				// 创建失败日志
				log.Printf("EnsureNamespace: 创建命名空间失败 %s: %v", namespace, createErr)
				return fmt.Errorf("创建命名空间 %s 失败: %w", namespace, createErr)
			}
			// 创建成功日志
			log.Printf("EnsureNamespace: 命名空间创建成功 %s", namespace)
			return nil
		}
		// 获取命名空间失败日志
		log.Printf("EnsureNamespace: 获取命名空间失败 %s: %v", namespace, err)
		return fmt.Errorf("获取命名空间 %s 失败: %w", namespace, err)
	}
	// 命名空间已存在日志
	log.Printf("EnsureNamespace: 命名空间已存在 %s", namespace)
	return nil
}

// ApplyLimitRange 应用 LimitRange 到指定命名空间
func ApplyLimitRange(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-limits",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cluster.CpuLimit),
						corev1.ResourceMemory: resource.MustParse(cluster.MemoryLimit),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse(cluster.CpuRequest),
						corev1.ResourceMemory: resource.MustParse(cluster.MemoryRequest),
					},
				},
			},
		},
	}

	// 创建 LimitRange
	_, err := kubeClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// 如果 LimitRange 已存在，跳过创建
			log.Printf("ApplyLimitRange: LimitRange 已存在 %s/%s，跳过创建", namespace, limitRange.Name)
			return nil
		}
		// 处理其他错误
		log.Printf("ApplyLimitRange: 创建 LimitRange 失败 %s/%s: %v", namespace, limitRange.Name, err)
		return fmt.Errorf("创建 LimitRange 失败 (namespace: %s, cpuLimit: %s, memoryLimit: %s): %w",
			namespace, cluster.CpuLimit, cluster.MemoryLimit, err)
	}

	log.Printf("ApplyLimitRange: LimitRange 创建成功 %s/%s", namespace, limitRange.Name)
	return nil
}

// ApplyResourceQuota 应用 ResourceQuota 到指定命名空间
func ApplyResourceQuota(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      QuotaName,
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    resource.MustParse(cluster.CpuRequest),
				corev1.ResourceRequestsMemory: resource.MustParse(cluster.MemoryRequest),
				corev1.ResourceLimitsCPU:      resource.MustParse(cluster.CpuLimit),
				corev1.ResourceLimitsMemory:   resource.MustParse(cluster.MemoryLimit),
			},
		},
	}

	// 创建 ResourceQuota
	_, err := kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// 如果 ResourceQuota 已存在，跳过创建
			log.Printf("ApplyResourceQuota: ResourceQuota 已存在 %s/%s，跳过创建", namespace, resourceQuota.Name)
			return nil
		}
		// 处理其他错误
		log.Printf("ApplyResourceQuota: 创建 ResourceQuota 失败 %s/%s: %v", namespace, resourceQuota.Name, err)
		return fmt.Errorf("创建 ResourceQuota 失败 (namespace: %s, cpuRequest: %s, memoryRequest: %s, cpuLimit: %s, memoryLimit: %s): %w",
			namespace, cluster.CpuRequest, cluster.MemoryRequest, cluster.CpuLimit, cluster.MemoryLimit, err)
	}

	log.Printf("ApplyResourceQuota: ResourceQuota 创建成功 %s/%s", namespace, resourceQuota.Name)
	return nil
}

// GetTaintsMapFromTaints 将 taints 转换为键为 "Key:Value:Effect" 的 map
func GetTaintsMapFromTaints(taints []corev1.Taint) map[string]corev1.Taint {
	taintsMap := make(map[string]corev1.Taint)
	for _, taint := range taints {
		key := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, taint.Effect)
		taintsMap[key] = taint
	}
	return taintsMap
}

// buildTaintKey 构建 taint 的唯一 key
func buildTaintKey(taint corev1.Taint) string {
	return strings.Join([]string{taint.Key, taint.Value, string(taint.Effect)}, ":")
}

// MergeTaints 合并新的 taints，避免重复
func MergeTaints(existingTaints []corev1.Taint, newTaints []corev1.Taint) []corev1.Taint {
	taintsMap := GetTaintsMapFromTaints(existingTaints)

	for _, newTaint := range newTaints {
		key := buildTaintKey(newTaint)
		if _, exists := taintsMap[key]; !exists {
			existingTaints = append(existingTaints, newTaint)
		}
	}
	return existingTaints
}

// RemoveTaints 从现有的 taints 中删除指定的 taints
func RemoveTaints(existingTaints []corev1.Taint, taintsToDelete []corev1.Taint) []corev1.Taint {
	taintsToDeleteMap := GetTaintsMapFromTaints(taintsToDelete)

	var updatedTaints []corev1.Taint
	for _, existingTaint := range existingTaints {
		key := buildTaintKey(existingTaint)
		if _, shouldDelete := taintsToDeleteMap[key]; !shouldDelete {
			updatedTaints = append(updatedTaints, existingTaint)
		}
	}
	return updatedTaints
}

// GetNodesByName 获取指定集群上的 Node 列表
func GetNodesByName(ctx context.Context, client *kubernetes.Clientset, nodeName string) (*corev1.NodeList, error) {
	if nodeName != "" {
		// 获取单个节点
		node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			log.Printf("获取 Node 失败 (nodeName: %s): %v", nodeName, err)
			return nil, fmt.Errorf("获取 Node %s 失败: %w", nodeName, err)
		}
		// 将单个节点转换为 NodeList
		nodeList := &corev1.NodeList{
			Items: []corev1.Node{*node},
		}
		log.Printf("获取单个 Node 成功 (nodeName: %s)", nodeName)
		return nodeList, nil
	}

	// 获取所有节点
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("获取 Node 列表失败: %v", err)
		return nil, fmt.Errorf("获取 Node 列表失败: %w", err)
	}

	log.Printf("获取所有 Nodes 成功, 总数: %d", len(nodes.Items))
	return nodes, nil
}

// GetPodsByNodeName 获取指定节点上的 Pod 列表
func GetPodsByNodeName(ctx context.Context, client *kubernetes.Clientset, nodeName string) (*corev1.PodList, error) {
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})

	if err != nil {
		log.Printf("获取 Pod 列表失败 (nodeName: %s): %v", nodeName, err)
		return nil, fmt.Errorf("获取 Pod 列表失败 (nodeName: %s): %w", nodeName, err)
	}

	log.Printf("成功获取节点 %s 上的 Pod 列表, 总数: %d", nodeName, len(pods.Items))
	return pods, nil
}

// GetNodeEvents 获取节点事件
func GetNodeEvents(ctx context.Context, client *kubernetes.Clientset, nodeName string) ([]model.OneEvent, error) {
	eventlist, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})

	if err != nil {
		log.Printf("获取节点事件失败 (nodeName: %s): %v", nodeName, err)
		return nil, fmt.Errorf("获取节点事件失败 (nodeName: %s): %w", nodeName, err)
	}

	// 转换为 OneEvent 模型
	var oneEvents []model.OneEvent
	for _, event := range eventlist.Items {
		oneEvent := model.OneEvent{
			Type:      event.Type,
			Component: event.Source.Component,
			Reason:    event.Reason,
			Message:   event.Message,
			FirstTime: event.FirstTimestamp.Format(time.RFC3339),
			LastTime:  event.LastTimestamp.Format(time.RFC3339),
			Object:    fmt.Sprintf("kind:%s name:%s", event.InvolvedObject.Kind, event.InvolvedObject.Name),
			Count:     int(event.Count),
		}

		// 处理空时间戳，避免 nil 引用错误
		if event.FirstTimestamp.IsZero() {
			oneEvent.FirstTime = "N/A"
		}
		if event.LastTimestamp.IsZero() {
			oneEvent.LastTime = "N/A"
		}

		oneEvents = append(oneEvents, oneEvent)
	}

	log.Printf("成功获取节点 %s 的事件, 总数: %d", nodeName, len(oneEvents))
	return oneEvents, nil
}

// GetNodeResource 获取节点资源信息
func GetNodeResource(ctx context.Context, metricsCli *metricsClient.Clientset, nodeName string, pods *corev1.PodList, node *corev1.Node) ([]string, error) {
	// 计算 CPU 和内存的请求和限制
	var totalCPURequest, totalCPULimit, totalMemoryRequest, totalMemoryLimit int64
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if cpuRequest, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				totalCPURequest += cpuRequest.MilliValue()
			} else {
				log.Printf("Pod %s: Missing CPU request", pod.Name)
			}
			if cpuLimit, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
				totalCPULimit += cpuLimit.MilliValue()
			} else {
				log.Printf("Pod %s: Missing CPU limit", pod.Name)
			}
			if memoryRequest, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				totalMemoryRequest += memoryRequest.Value()
			} else {
				log.Printf("Pod %s: Missing memory request", pod.Name)
			}
			if memoryLimit, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
				totalMemoryLimit += memoryLimit.Value()
			} else {
				log.Printf("Pod %s: Missing memory limit", pod.Name)
			}
		}
	}

	var result []string

	// 获取节点的总 CPU 和内存容量
	cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
	memoryCapacity := node.Status.Capacity[corev1.ResourceMemory]

	// CPU Request 和 Limit 信息
	result = append(result, fmt.Sprintf("CPU Request: %dm / %dm", totalCPURequest, cpuCapacity.MilliValue()))
	result = append(result, fmt.Sprintf("CPU Limit: %dm / %dm", totalCPULimit, cpuCapacity.MilliValue()))

	// Memory Request 和 Limit 信息（单位：MiB）
	result = append(result, fmt.Sprintf("Memory Request: %dMi / %dMi", totalMemoryRequest/1024/1024, memoryCapacity.Value()/1024/1024))
	result = append(result, fmt.Sprintf("Memory Limit: %dMi / %dMi", totalMemoryLimit/1024/1024, memoryCapacity.Value()/1024/1024))

	// 获取节点资源使用情况
	// TODO: need Metrics-Server
	// nodeMetrics, err := metricsCli.MetricsV1alpha1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get node metrics: %v", err)
	// }

	// Mock data for testing
	cpuUsage := resource.NewMilliQuantity(100, resource.DecimalSI)
	memoryUsage := resource.NewQuantity(1024*1024*100, resource.BinarySI)

	// CPU 和内存的使用量（单位：m，MiB）
	result = append(result, fmt.Sprintf("CPU Usage: %dm / %dm", cpuUsage.MilliValue(), cpuCapacity.MilliValue()))
	result = append(result, fmt.Sprintf("Memory Usage: %dMi / %dMi", memoryUsage.Value()/1024/1024, memoryCapacity.Value()/1024/1024))

	// Pod 数量信息
	maxPods := node.Status.Allocatable[corev1.ResourcePods]
	result = append(result, fmt.Sprintf("Pods: %d / %d", len(pods.Items), maxPods.Value()))

	// 返回结果
	return result, nil
}

// GetNodeStatus 获取节点状态
func GetNodeStatus(node corev1.Node) string {
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

// IsNodeSchedulable 判断节点是否可调度
func IsNodeSchedulable(node corev1.Node) bool {
	return !node.Spec.Unschedulable
}

// GetNodeRoles 获取节点角色
func GetNodeRoles(node corev1.Node) []string {
	var roles []string
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	return roles
}

// GetInternalIP 获取节点内部IP
func GetInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

// GetNodeLabels 获取节点标签
func GetNodeLabels(node corev1.Node) []string {
	var labels []string
	for key, value := range node.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}
	return labels
}

// GetResourceString 获取节点资源信息
func GetResourceString(node corev1.Node, resourceName string) string {
	allocatable := node.Status.Allocatable[corev1.ResourceName(resourceName)]
	return allocatable.String()
}

// GetNodeAge 计算节点存在时间
func GetNodeAge(node corev1.Node) string {
	// 获取节点的创建时间
	creationTime := node.CreationTimestamp.Time

	// 计算当前时间与创建时间的差值
	duration := time.Since(creationTime)

	// 将差值转换为天数、小时数等格式
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24

	// 返回节点存在时间的字符串表示
	return fmt.Sprintf("%dd%dh", days, hours)
}

func BuildK8sNode(ctx context.Context, id int, node corev1.Node, kubeClient *kubernetes.Clientset, metricsClient *metricsClient.Clientset) (*model.K8sNode, error) {
	// 获取节点相关的 Pod 列表
	pods, err := GetPodsByNodeName(ctx, kubeClient, node.Name)
	if err != nil {
		log.Printf("获取节点 Pod 列表失败 %s: %v", node.Name, err)
		return nil, err
	}

	// 获取节点相关事件
	events, err := GetNodeEvents(ctx, kubeClient, node.Name)
	if err != nil {
		log.Printf("获取节点事件失败 %s: %v", node.Name, err)
		return nil, err
	}

	// 获取节点的资源使用情况
	resourceInfo, err := GetNodeResource(ctx, metricsClient, node.Name, pods, &node)
	if err != nil {
		log.Printf("获取节点资源使用情况失败 %s: %v", node.Name, err)
		return nil, err
	}

	// 构建 k8sNode 结构体
	k8sNode := &model.K8sNode{
		Name:              node.Name,
		ClusterID:         id,
		Status:            GetNodeStatus(node),
		ScheduleEnable:    IsNodeSchedulable(node),
		Roles:             GetNodeRoles(node),
		Age:               GetNodeAge(node),
		IP:                GetInternalIP(node),
		PodNum:            len(pods.Items),
		CpuRequestInfo:    resourceInfo[0],
		CpuUsageInfo:      resourceInfo[4],
		CpuLimitInfo:      resourceInfo[1],
		MemoryRequestInfo: resourceInfo[2],
		MemoryUsageInfo:   resourceInfo[5],
		MemoryLimitInfo:   resourceInfo[3],
		PodNumInfo:        resourceInfo[6],
		CpuCores:          GetResourceString(node, "cpu"),
		MemGibs:           GetResourceString(node, "memory"),
		EphemeralStorage:  GetResourceString(node, "ephemeral-storage"),
		KubeletVersion:    node.Status.NodeInfo.KubeletVersion,
		CriVersion:        node.Status.NodeInfo.ContainerRuntimeVersion,
		OsVersion:         node.Status.NodeInfo.OSImage,
		KernelVersion:     node.Status.NodeInfo.KernelVersion,
		Labels:            GetNodeLabels(node),
		Taints:            node.Spec.Taints,
		Events:            events,
		Annotation:        node.Annotations,
		Conditions:        node.Status.Conditions,
		CreatedAt:         node.CreationTimestamp.Time,
		UpdatedAt:         time.Now(),
	}

	return k8sNode, nil
}

// BuildK8sPods BuildK8sNodes 构建 K8sNode 列表
func BuildK8sPods(pods *corev1.PodList) []*model.K8sPod {
	var k8sPods []*model.K8sPod

	for _, pod := range pods.Items {
		k8sPod := &model.K8sPod{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			NodeName:    pod.Spec.NodeName,
			Status:      string(pod.Status.Phase),
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			Containers:  BuildK8sContainers(pod.Spec.Containers),
		}
		k8sPods = append(k8sPods, k8sPod)
	}

	return k8sPods
}

// BuildK8sContainers 构建 K8sContainer 列表
func BuildK8sContainers(containers []corev1.Container) []model.K8sPodContainer {
	k8sContainers := make([]model.K8sPodContainer, 0, len(containers)) // 预分配切片容量，避免重复内存分配

	// 遍历所有容器并构建 K8sPodContainer
	for _, container := range containers {
		newContainer := model.K8sPodContainer{
			Name:    container.Name,
			Image:   container.Image,
			Command: model.StringList(container.Command),
			Args:    model.StringList(container.Args),
			Envs:    make([]model.K8sEnvVar, len(container.Env)), // 直接预分配大小
			Ports:   make([]model.K8sContainerPort, len(container.Ports)),
			Resources: model.ResourceRequirements{
				Requests: model.K8sResourceList{
					CPU:    container.Resources.Requests.Cpu().String(),
					Memory: container.Resources.Requests.Memory().String(),
				},
				Limits: model.K8sResourceList{
					CPU:    container.Resources.Limits.Cpu().String(),
					Memory: container.Resources.Limits.Memory().String(),
				},
			},
			VolumeMounts:    make([]model.K8sVolumeMount, len(container.VolumeMounts)),
			ImagePullPolicy: string(container.ImagePullPolicy),
		}

		// 构建 LivenessProbe 和 ReadinessProbe
		buildProbeIfNeeded(container.LivenessProbe, &newContainer.LivenessProbe)
		buildProbeIfNeeded(container.ReadinessProbe, &newContainer.ReadinessProbe)

		// 构建环境变量列表
		for i, env := range container.Env {
			newContainer.Envs[i] = model.K8sEnvVar{
				Name:  env.Name,
				Value: env.Value,
			}
		}

		// 构建容器端口列表
		for i, port := range container.Ports {
			newContainer.Ports[i] = model.K8sContainerPort{
				Name:          port.Name,
				ContainerPort: int(port.ContainerPort),
				Protocol:      string(port.Protocol),
			}
		}

		// 构建挂载卷列表
		for i, volumeMount := range container.VolumeMounts {
			newContainer.VolumeMounts[i] = model.K8sVolumeMount{
				Name:      volumeMount.Name,
				MountPath: volumeMount.MountPath,
				ReadOnly:  volumeMount.ReadOnly,
				SubPath:   volumeMount.SubPath,
			}
		}

		// 将新容器添加到列表中
		k8sContainers = append(k8sContainers, newContainer)
	}

	return k8sContainers
}

// buildProbeIfNeeded 构建探针（LivenessProbe 或 ReadinessProbe）
func buildProbeIfNeeded(probe *corev1.Probe, result **model.K8sProbe) {
	if probe != nil {
		*result = &model.K8sProbe{
			HTTPGet: &model.K8sHTTPGetAction{
				Path:   probe.HTTPGet.Path,
				Port:   probe.HTTPGet.Port.IntValue(),
				Scheme: string(probe.HTTPGet.Scheme),
			},
			InitialDelaySeconds: int(probe.InitialDelaySeconds),
			PeriodSeconds:       int(probe.PeriodSeconds),
			TimeoutSeconds:      int(probe.TimeoutSeconds),
			SuccessThreshold:    int(probe.SuccessThreshold),
			FailureThreshold:    int(probe.FailureThreshold),
		}
	}
}

// BuildK8sContainersWithPointer 转换普通切片为指针切片
func BuildK8sContainersWithPointer(k8sContainers []model.K8sPodContainer) []*model.K8sPodContainer {
	pointerSlice := make([]*model.K8sPodContainer, len(k8sContainers))
	for i := 0; i < len(k8sContainers); i++ {
		pointerSlice[i] = &k8sContainers[i]
	}
	return pointerSlice
}

// GetResourceName 根据 Kind 获取资源名称
func GetResourceName(kind string) string {
	switch kind {
	case "Pod":
		return "pods"
	case "Service":
		return "services"
	case "Deployment":
		return "deployments"
	//TODO: 添加其他资源类型
	default:
		return strings.ToLower(kind) + "s"
	}
}

// GetKubeClient 获取 Kubernetes 客户端
func GetKubeClient(clusterId int, client client.K8sClient, l *zap.Logger) (*kubernetes.Clientset, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := client.GetKubeClient(clusterId)
	if err != nil {
		l.Error("获取 Kubernetes 客户端失败", zap.String("clusterID", fmt.Sprintf("%d", clusterId)), zap.Error(err))
		return nil, fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	return kubeClient, nil
}

func InitAadGetKubeClient(ctx context.Context, cluster *model.K8sCluster, logger *zap.Logger, client client.K8sClient) (*kubernetes.Clientset, error) {
	// 解析 kubeconfig 并手动初始化 Kubernetes 客户端
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		logger.Error("CreateCluster: 解析 kubeconfig 失败", zap.Error(err))
		return nil, err
	}

	// 初始化 Kubernetes 客户端
	if err = client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		logger.Error("CreateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取 Kubernetes 客户端
	kubeClient, err := client.GetKubeClient(cluster.ID)
	if err != nil {
		logger.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	return kubeClient, err
}

func GetKubeAndMetricsClient(id int, logger *zap.Logger, client client.K8sClient) (*kubernetes.Clientset, *metricsClient.Clientset, error) {
	kc, err := client.GetKubeClient(id)
	if err != nil {
		logger.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, nil, err
	}

	mc, err := client.GetMetricsClient(id)
	if err != nil {
		logger.Error("CreateCluster: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, nil, err
	}
	return kc, mc, nil
}

func GetDynamicClient(ctx context.Context, id int, clusterDao admin.ClusterDAO, client client.K8sClient) (*dynamic.DynamicClient, error) {
	cluster, err := clusterDao.GetClusterByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("集群不存在: %w", err)
	}

	dynClient, err := client.GetDynamicClient(cluster.ID)
	if err != nil {
		return nil, fmt.Errorf("无法获取动态客户端: %w", err)
	}

	return dynClient, nil
}

// GetPodResources 获取 Pod 资源
func GetPodResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, pod := range pods.Items {
		resources = append(resources, model.Resource{
			Type:         "Pod",
			Name:         pod.Name,
			Namespace:    pod.Namespace,
			Status:       string(pod.Status.Phase),
			CreationTime: pod.CreationTimestamp.Time,
		})
	}

	return resources, nil
}

// GetServiceResources 获取 Service 资源
func GetServiceResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, service := range services.Items {
		resources = append(resources, model.Resource{
			Type:         "Service",
			Name:         service.Name,
			Namespace:    service.Namespace,
			Status:       "Active", // TODO: 自定义处理
			CreationTime: service.CreationTimestamp.Time,
		})
	}

	return resources, nil
}

// GetDeploymentResources 获取 Deployment 资源
func GetDeploymentResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, deployment := range deployments.Items {
		status := "Unknown"
		for _, condition := range deployment.Status.Conditions {
			if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
				status = "Available"
				break
			}
		}

		resources = append(resources, model.Resource{
			Type:         "Deployment",
			Name:         deployment.Name,
			Namespace:    deployment.Namespace,
			Status:       status,
			CreationTime: deployment.CreationTimestamp.Time,
		})
	}

	return resources, nil
}

// GetReplicaSetResources 获取 ReplicaSet 资源
func GetReplicaSetResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	rs, err := kubeClient.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, rsItem := range rs.Items {
		status := "NotReady"
		if rsItem.Status.ReadyReplicas == rsItem.Status.Replicas {
			status = "Ready"
		}
		resources = append(resources, model.Resource{
			Type:         "ReplicaSet",
			Name:         rsItem.Name,
			Namespace:    rsItem.Namespace,
			Status:       status,
			CreationTime: rsItem.CreationTimestamp.Time,
		})
	}

	return resources, nil
}

// GetStatefulSetResources 获取 StatefulSet 资源
func GetStatefulSetResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	ss, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, ssItem := range ss.Items {
		status := "NotReady"
		if ssItem.Status.ReadyReplicas == ssItem.Status.Replicas {
			status = "Ready"
		}

		resources = append(resources, model.Resource{
			Type:         "StatefulSet",
			Name:         ssItem.Name,
			Namespace:    ssItem.Namespace,
			Status:       status,
			CreationTime: ssItem.CreationTimestamp.Time,
		})
	}

	return resources, nil
}

// GetDaemonSetResources 获取 DaemonSet 资源
func GetDaemonSetResources(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) ([]model.Resource, error) {
	ds, err := kubeClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []model.Resource

	for _, dsItem := range ds.Items {
		status := "NotReady"
		if dsItem.Status.NumberReady == dsItem.Status.DesiredNumberScheduled {
			status = "Ready"
		}

		resources = append(resources, model.Resource{
			Type:         "DaemonSet",
			Name:         dsItem.Name,
			Namespace:    dsItem.Namespace,
			Status:       status,
			CreationTime: dsItem.CreationTimestamp.Time,
		})
	}

	return resources, nil
}
