package k8s

import (
	"context"
	"fmt"
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

// EnsureNamespace 确保指定的命名空间存在，如果不存在则创建
func EnsureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// 创建命名空间
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, createErr := kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if createErr != nil {
				log.Printf("EnsureNamespace: 创建命名空间失败 %s: %v", namespace, createErr)
				return fmt.Errorf("创建命名空间 %s 失败: %w", namespace, createErr)
			}
			log.Printf("EnsureNamespace: 命名空间创建成功 %s", namespace)
			return nil
		}
		log.Printf("EnsureNamespace: 获取命名空间失败 %s: %v", namespace, err)
		return fmt.Errorf("获取命名空间 %s 失败: %w", namespace, err)
	}
	// 命名空间已存在
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

	_, err := kubeClient.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Printf("ApplyLimitRange: LimitRange 已存在 %s", namespace)
			return nil
		}
		log.Printf("ApplyLimitRange: 创建 LimitRange 失败 %s: %v", namespace, err)
		return fmt.Errorf("创建 LimitRange 失败 (namespace: %s): %w", namespace, err)
	}

	log.Printf("ApplyLimitRange: LimitRange 创建成功 %s", namespace)
	return nil
}

// ApplyResourceQuota 应用 ResourceQuota 到指定命名空间
func ApplyResourceQuota(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "compute-quota",
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

	_, err := kubeClient.CoreV1().ResourceQuotas(namespace).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			log.Printf("ApplyResourceQuota: ResourceQuota 已存在 %s", namespace)
			return nil
		}
		log.Printf("ApplyResourceQuota: 创建 ResourceQuota 失败 %s: %v", namespace, err)
		return fmt.Errorf("创建 ResourceQuota 失败 (namespace: %s): %w", namespace, err)
	}

	log.Printf("ApplyResourceQuota: ResourceQuota 创建成功 %s", namespace)
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

// MergeTaints 合并新的 taints，避免重复
func MergeTaints(existingTaints []corev1.Taint, newTaints []corev1.Taint) []corev1.Taint {
	taintsMap := GetTaintsMapFromTaints(existingTaints)

	for _, newTaint := range newTaints {
		key := fmt.Sprintf("%s:%s:%s", newTaint.Key, newTaint.Value, newTaint.Effect)
		if _, exists := taintsMap[key]; !exists {
			existingTaints = append(existingTaints, newTaint)
		}
	}
	return existingTaints
}

// RemoveTaints 从现有的 taints 中删除指定的 taints
func RemoveTaints(existingTaints []corev1.Taint, taintsToDelete []corev1.Taint) []corev1.Taint {
	taintsMap := GetTaintsMapFromTaints(taintsToDelete)

	var updatedTaints []corev1.Taint
	for _, existingTaint := range existingTaints {
		key := fmt.Sprintf("%s:%s:%s", existingTaint.Key, existingTaint.Value, existingTaint.Effect)
		if _, shouldDelete := taintsMap[key]; !shouldDelete {
			updatedTaints = append(updatedTaints, existingTaint)
		}
	}
	return updatedTaints
}

// GetNodesByClusterID 获取指定集群上的 Node 列表
func GetNodesByClusterID(ctx context.Context, client *kubernetes.Clientset, name string) (*corev1.NodeList, error) {
	if name != "" {
		node, err := client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log.Printf("获取 Node 失败 %s: %v", name, err)
			return nil, err
		}
		// 将单个节点转换为 NodeList
		nodeList := &corev1.NodeList{
			Items: []corev1.Node{*node},
		}

		return nodeList, nil
	}

	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("获取 Node 列表失败: %v", err)
		return nil, err
	}

	return nodes, nil
}

// GetPodsByNodeName 获取指定节点上的 Pod 列表
func GetPodsByNodeName(ctx context.Context, client *kubernetes.Clientset, nodeName string) (*corev1.PodList, error) {
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})

	if err != nil {
		log.Printf("获取 Pod 列表失败 %s: %v", nodeName, err)
		return nil, err
	}

	return pods, nil
}

// GetNodeEvents 获取节点事件
func GetNodeEvents(ctx context.Context, client *kubernetes.Clientset, nodeName string) ([]model.OneEvent, error) {
	eventlist, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})

	if err != nil {
		log.Printf("获取节点事件失败 %s: %v", nodeName, err)
		return nil, err
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
		oneEvents = append(oneEvents, oneEvent)
	}

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
			}
			if cpuLimit, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
				totalCPULimit += cpuLimit.MilliValue()
			}
			if memoryRequest, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				totalMemoryRequest += memoryRequest.Value()
			}
			if memoryLimit, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
				totalMemoryLimit += memoryLimit.Value()
			}
		}
	}

	var result []string

	// 获取节点的总 CPU 和内存容量
	cpuCapacity := node.Status.Capacity[corev1.ResourceCPU]
	memoryCapacity := node.Status.Capacity[corev1.ResourceMemory]

	// CpuRequestInfo
	result = append(result, fmt.Sprintf("%dm/%dm", totalCPURequest, cpuCapacity.MilliValue()))
	// CpuLimitInfo
	result = append(result, fmt.Sprintf("%dm/%dm", totalCPULimit, cpuCapacity.MilliValue()))
	// MemoryRequestInfo
	result = append(result, fmt.Sprintf("%dMi/%dMi", totalMemoryRequest/1024/1024, memoryCapacity.Value()/1024/1024))
	// MemoryLimitInfo
	result = append(result, fmt.Sprintf("%dMi/%dMi", totalMemoryLimit/1024/1024, memoryCapacity.Value()/1024/1024))

	// 获取节点资源使用情况
	// TODO need Metrics-Server
	// nodeMetrics, err := metricsCli.MetricsV1alpha1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get node metrics: %v", err)
	// }

	// // CPU 和内存的使用量
	// cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
	// memoryUsage := nodeMetrics.Usage[corev1.ResourceMemory]

	// mock data
	cpuUsage := resource.NewMilliQuantity(100, resource.DecimalSI)
	memoryUsage := resource.NewQuantity(1024*1024*100, resource.BinarySI)

	result = append(result, fmt.Sprintf("%dm/%dm", cpuUsage.MilliValue(), cpuCapacity.MilliValue()))
	result = append(result, fmt.Sprintf("%dMi/%dMi", memoryUsage.Value()/1024/1024, memoryCapacity.Value()/1024/1024))

	// PodNumInfo
	maxPods := node.Status.Allocatable[corev1.ResourcePods]
	result = append(result, fmt.Sprintf("%d/%d", len(pods.Items), maxPods.Value()))
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

// BuildK8sNodes 构建 K8sNode 列表
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

		// for _, container := range pod.Spec.Containers {
		// 	newContainer := model.K8sPodContainer{
		// 		Name:    container.Name,
		// 		Image:   container.Image,
		// 		Command: model.StringList(container.Command),
		// 		Args:    model.StringList(container.Args),
		// 		Envs:    make([]model.K8sEnvVar, 0),
		// 		Ports:   make([]model.K8sContainerPort, 0),
		// 		Resources: model.ResourceRequirements{
		// 			Requests: model.K8sResourceList{
		// 				CPU:    container.Resources.Requests.Cpu().String(),
		// 				Memory: container.Resources.Requests.Memory().String(),
		// 			},
		// 			Limits: model.K8sResourceList{
		// 				CPU:    container.Resources.Limits.Cpu().String(),
		// 				Memory: container.Resources.Limits.Memory().String(),
		// 			},
		// 		},
		// 		VolumeMounts:    make([]model.K8sVolumeMount, 0),
		// 		ImagePullPolicy: string(container.ImagePullPolicy),
		// 	}

		// 	if container.LivenessProbe != nil {
		// 		newContainer.LivenessProbe = &model.K8sProbe{
		// 			HTTPGet: &model.K8sHTTPGetAction{
		// 				Path:   container.LivenessProbe.HTTPGet.Path,
		// 				Port:   container.LivenessProbe.HTTPGet.Port.IntValue(),
		// 				Scheme: string(container.LivenessProbe.HTTPGet.Scheme),
		// 			},
		// 			InitialDelaySeconds: int(container.LivenessProbe.InitialDelaySeconds),
		// 			PeriodSeconds:       int(container.LivenessProbe.PeriodSeconds),
		// 			TimeoutSeconds:      int(container.LivenessProbe.TimeoutSeconds),
		// 			SuccessThreshold:    int(container.LivenessProbe.SuccessThreshold),
		// 			FailureThreshold:    int(container.LivenessProbe.FailureThreshold),
		// 		}
		// 	}

		// 	if container.ReadinessProbe != nil {
		// 		newContainer.ReadinessProbe = &model.K8sProbe{
		// 			HTTPGet: &model.K8sHTTPGetAction{
		// 				Path:   container.ReadinessProbe.HTTPGet.Path,
		// 				Port:   container.ReadinessProbe.HTTPGet.Port.IntValue(),
		// 				Scheme: string(container.ReadinessProbe.HTTPGet.Scheme),
		// 			},
		// 			InitialDelaySeconds: int(container.ReadinessProbe.InitialDelaySeconds),
		// 			PeriodSeconds:       int(container.ReadinessProbe.PeriodSeconds),
		// 			TimeoutSeconds:      int(container.ReadinessProbe.TimeoutSeconds),
		// 			SuccessThreshold:    int(container.ReadinessProbe.SuccessThreshold),
		// 			FailureThreshold:    int(container.ReadinessProbe.FailureThreshold),
		// 		}
		// 	}

		// 	for _, env := range container.Env {
		// 		newContainer.Envs = append(newContainer.Envs, model.K8sEnvVar{
		// 			Name:  env.Name,
		// 			Value: env.Value,
		// 		})
		// 	}

		// 	for _, port := range container.Ports {
		// 		newContainer.Ports = append(newContainer.Ports, model.K8sContainerPort{
		// 			Name:          port.Name,
		// 			ContainerPort: int(port.ContainerPort),
		// 			Protocol:      string(port.Protocol),
		// 		})
		// 	}

		// 	for _, volumeMount := range container.VolumeMounts {
		// 		newContainer.VolumeMounts = append(newContainer.VolumeMounts, model.K8sVolumeMount{
		// 			Name:      volumeMount.Name,
		// 			MountPath: volumeMount.MountPath,
		// 			ReadOnly:  volumeMount.ReadOnly,
		// 			SubPath:   volumeMount.SubPath,
		// 		})
		// 	}

		// 	k8sPod.Containers = append(k8sPod.Containers, newContainer)
		// }

		k8sPods = append(k8sPods, k8sPod)
	}

	return k8sPods
}

// BuildK8sContainers 构建 K8sContainer 列表
func BuildK8sContainers(containers []corev1.Container) []model.K8sPodContainer {
	var k8sContainers []model.K8sPodContainer
	for _, container := range containers {
		newContainer := model.K8sPodContainer{
			Name:    container.Name,
			Image:   container.Image,
			Command: model.StringList(container.Command),
			Args:    model.StringList(container.Args),
			Envs:    make([]model.K8sEnvVar, 0),
			Ports:   make([]model.K8sContainerPort, 0),
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
			VolumeMounts:    make([]model.K8sVolumeMount, 0),
			ImagePullPolicy: string(container.ImagePullPolicy),
		}

		if container.LivenessProbe != nil {
			newContainer.LivenessProbe = &model.K8sProbe{
				HTTPGet: &model.K8sHTTPGetAction{
					Path:   container.LivenessProbe.HTTPGet.Path,
					Port:   container.LivenessProbe.HTTPGet.Port.IntValue(),
					Scheme: string(container.LivenessProbe.HTTPGet.Scheme),
				},
				InitialDelaySeconds: int(container.LivenessProbe.InitialDelaySeconds),
				PeriodSeconds:       int(container.LivenessProbe.PeriodSeconds),
				TimeoutSeconds:      int(container.LivenessProbe.TimeoutSeconds),
				SuccessThreshold:    int(container.LivenessProbe.SuccessThreshold),
				FailureThreshold:    int(container.LivenessProbe.FailureThreshold),
			}
		}

		if container.ReadinessProbe != nil {
			newContainer.ReadinessProbe = &model.K8sProbe{
				HTTPGet: &model.K8sHTTPGetAction{
					Path:   container.ReadinessProbe.HTTPGet.Path,
					Port:   container.ReadinessProbe.HTTPGet.Port.IntValue(),
					Scheme: string(container.ReadinessProbe.HTTPGet.Scheme),
				},
				InitialDelaySeconds: int(container.ReadinessProbe.InitialDelaySeconds),
				PeriodSeconds:       int(container.ReadinessProbe.PeriodSeconds),
				TimeoutSeconds:      int(container.ReadinessProbe.TimeoutSeconds),
				SuccessThreshold:    int(container.ReadinessProbe.SuccessThreshold),
				FailureThreshold:    int(container.ReadinessProbe.FailureThreshold),
			}
		}

		for _, env := range container.Env {
			newContainer.Envs = append(newContainer.Envs, model.K8sEnvVar{
				Name:  env.Name,
				Value: env.Value,
			})
		}

		for _, port := range container.Ports {
			newContainer.Ports = append(newContainer.Ports, model.K8sContainerPort{
				Name:          port.Name,
				ContainerPort: int(port.ContainerPort),
				Protocol:      string(port.Protocol),
			})
		}

		for _, volumeMount := range container.VolumeMounts {
			newContainer.VolumeMounts = append(newContainer.VolumeMounts, model.K8sVolumeMount{
				Name:      volumeMount.Name,
				MountPath: volumeMount.MountPath,
				ReadOnly:  volumeMount.ReadOnly,
				SubPath:   volumeMount.SubPath,
			})
		}

		k8sContainers = append(k8sContainers, newContainer)
	}

	return k8sContainers
}

// BuildK8sContainersWithPointer 转换普通切片为指针切片
func BuildK8sContainersWithPointer(k8sContainers []model.K8sPodContainer) []*model.K8sPodContainer {
	pointerSlice := make([]*model.K8sPodContainer, len(k8sContainers))
	for i := range k8sContainers {
		pointerSlice[i] = &k8sContainers[i]
	}
	return pointerSlice
}
