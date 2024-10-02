package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
)

type K8sService interface {
	// ListAllClusters 获取所有 Kubernetes 集群
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
	// ListClustersForSelect 获取用于选择的 Kubernetes 集群列表
	ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error)
	// CreateCluster 创建一个新的 Kubernetes 集群
	CreateCluster(ctx context.Context, cluster *model.K8sCluster) error
	// UpdateCluster 更新指定 ID 的 Kubernetes 集群
	UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error
	// BatchEnableSwitchClusters 批量启用或切换 Kubernetes 集群调度
	BatchEnableSwitchClusters(ctx context.Context, ids []int) error
	// BatchDeleteClusters 批量删除 Kubernetes 集群
	BatchDeleteClusters(ctx context.Context, ids []int) error
	// GetClusterByID 根据 ID 获取单个 Kubernetes 集群
	GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error)

	// ListAllNodes 获取所有 Kubernetes 节点
	ListAllNodes(ctx context.Context, id int) ([]*model.K8sNode, error)
	// GetNodeByID 根据 ID 获取单个 Kubernetes 节点
	GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error)
	// GetPodsByNodeID 根据 Node ID 获取 Pod 列表
	GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error)
	// CheckTaintYaml 检查 Taint Yaml 是否合法
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// BatchEnableSwitchNodes 批量启用或切换 Kubernetes 节点调度
	BatchEnableSwitchNodes(ctx context.Context, ids []int) error
	// AddNodeLabel 添加标签到指定 Node
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	// AddNodeTaint 添加 Taint 到指定 Node
	AddNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// DeleteNodeLabel 删除指定 Node 的标签
	DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error
	// DrainPods 删除指定 Node 上的 Pod
	DrainPods(ctx context.Context, nodeID int) error
}

type k8sService struct {
	dao    dao.K8sDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewK8sService(dao dao.K8sDAO, client client.K8sClient, l *zap.Logger) K8sService {
	return &k8sService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

func (k *k8sService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	return k.dao.ListAllClusters(ctx)
}

func (k *k8sService) ListClustersForSelect(ctx context.Context) ([]*model.K8sCluster, error) {
	panic("implement me")
}

// CreateCluster 创建一个新的 Kubernetes 集群，并应用资源限制
func (k *k8sService) CreateCluster(ctx context.Context, cluster *model.K8sCluster) (err error) {
	// 创建集群记录
	if err = k.dao.CreateCluster(ctx, cluster); err != nil {
		k.l.Error("CreateCluster: 创建集群记录失败", zap.Error(err))
		return fmt.Errorf("创建集群记录失败: %w", err)
	}

	// 确保后续操作如果出现错误时回滚集群记录以防后续步骤失败
	defer func() {
		if err != nil {
			k.l.Info("CreateCluster: 回滚集群记录", zap.Int("clusterID", cluster.ID))
			if rollbackErr := k.dao.DeleteCluster(ctx, cluster.ID); rollbackErr != nil {
				k.l.Error("CreateCluster: 回滚集群记录失败", zap.Error(rollbackErr))
			}
		}
	}()

	// 解析 kubeconfig 并手动初始化 Kubernetes 客户端
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		k.l.Error("CreateCluster: 解析 kubeconfig 失败", zap.Error(err))
		return fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}
	time.Sleep(20 * time.Second)
	if err = k.client.InitClient(ctx, cluster.ID, restConfig); err != nil { // 假设 useMock=false
		k.l.Error("CreateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("初始化 Kubernetes 客户端失败: %w", err)
	}

	// 获取 Kubernetes 客户端
	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	const maxConcurrent = 5 // 最大并发数
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	// 使用一个 channel 来收集错误
	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx1, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, namespace := range cluster.RestrictedNameSpace {
		wg.Add(1)

		// 传递变量到 goroutine
		ns := namespace

		go func() {
			defer wg.Done()

			// 获取 semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 确保命名空间存在
			if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 LimitRange
			if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 ResourceQuota
			if err := pkg.ApplyResourceQuota(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}
		}()
	}

	// 等待所有 goroutines 完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	for e := range errChan {
		if e != nil {
			k.l.Error("CreateCluster: 处理命名空间时发生错误", zap.Error(e))
			return e
		}
	}

	k.l.Info("CreateCluster: 成功创建 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

func (k *k8sService) UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error {
	//已经在dao层实现回滚机制
	if err := k.dao.UpdateCluster(ctx, id, cluster); err != nil {
		k.l.Error("UpdateCluster 更新集群失败", zap.Error(err))
		return fmt.Errorf("更新集群失败: %w", err)
	}
	//初始化或更新 kubernetes 客户端
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfigContent))
	if err != nil {
		k.l.Error("UpdateCluster: 解析 kubeconfig 失败", zap.Error(err))
		return fmt.Errorf("解析 kubeconfig 失败: %w", err)
	}

	if err = k.client.InitClient(ctx, cluster.ID, restConfig); err != nil {
		k.l.Error("UpdateCluster: 初始化 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("初始化 Kubernetes 客户端失败: %w", err)
	}
	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("UpdateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return fmt.Errorf("获取 Kubernetes 客户端失败: %w", err)
	}

	const maxConcurrent = 5
	semaphore := make(chan struct{}, maxConcurrent)

	var wg sync.WaitGroup

	errChan := make(chan error, len(cluster.RestrictedNameSpace))

	ctx1, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, namespace := range cluster.RestrictedNameSpace {
		wg.Add(1)
		//传递变量到goroutine
		ns := namespace

		go func() {
			defer wg.Done()

			//获取semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			//确保命名空间存在
			if err := pkg.EnsureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}
			//应用LimitRange
			if err := pkg.ApplyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}
			if err := pkg.ApplyResourceQuota(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 ResourceQuota 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}
		}()
	}
	wg.Wait()
	close(errChan)

	//检查是否有错误
	for e := range errChan {
		if e != nil {
			k.l.Error("UpdateCluster: 处理命名空间时发生错误", zap.Error(e))
			return e
		}
	}
	k.l.Info("UpdateCluster: 成功更新 Kubernetes 集群", zap.Int("clusterID", cluster.ID))
	return nil
}

func (k *k8sService) BatchEnableSwitchClusters(ctx context.Context, ids []int) error {
	panic("implement me")
}

func (k *k8sService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	panic("implement me")
}

func (k *k8sService) ListAllNodes(ctx context.Context, id int) ([]*model.K8sNode, error) {
	kubeClient, err := k.client.GetKubeClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	metricsClient, err := k.client.GetMetricsClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, constants.ErrorMetricsClientNotReady
	}

	// 获取节点列表
	nodes, err := k.getNodesByClusterID(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var k8sNodes []*model.K8sNode
	nodeChan := make(chan *model.K8sNode, len(nodes.Items))
	doneChan := make(chan struct{}) // 用于标识数据收集完成

	for _, node := range nodes.Items {
		wg.Add(1)
		go func(node v1.Node) {
			defer wg.Done()

			// 获取节点相关的 Pod 列表
			pods, err := k.getPodsByNodeName(ctx, kubeClient, node.Name)
			if err != nil {
				k.l.Error("获取 Pod 列表失败", zap.Error(err))
				return
			}

			// 获取节点相关事件
			events, err := k.getNodeEvents(ctx, kubeClient, node.Name)
			if err != nil {
				k.l.Error("获取节点事件失败", zap.Error(err))
				return
			}

			// 获取节点的资源使用情况
			resourceInfo, err := getResource(ctx, metricsClient, node.Name, pods, &node)
			if err != nil {
				k.l.Error("获取节点资源信息失败", zap.Error(err))
				return
			}

			// 构建 k8sNode 结构体
			k8sNode := &model.K8sNode{
				Name:              node.Name,
				ClusterID:         id,
				Status:            getNodeStatus(node),
				ScheduleEnable:    isNodeSchedulable(node),
				Roles:             getNodeRoles(node),
				Age:               getNodeAge(node),
				IP:                getInternalIP(node),
				PodNum:            len(pods.Items),
				CpuRequestInfo:    resourceInfo[0],
				CpuUsageInfo:      resourceInfo[4],
				CpuLimitInfo:      resourceInfo[1],
				MemoryRequestInfo: resourceInfo[2],
				MemoryUsageInfo:   resourceInfo[5],
				MemoryLimitInfo:   resourceInfo[3],
				PodNumInfo:        resourceInfo[6],
				CpuCores:          getResourceString(node, "cpu"),
				MemGibs:           getResourceString(node, "memory"),
				EphemeralStorage:  getResourceString(node, "ephemeral-storage"),
				KubeletVersion:    node.Status.NodeInfo.KubeletVersion,
				CriVersion:        node.Status.NodeInfo.ContainerRuntimeVersion,
				OsVersion:         node.Status.NodeInfo.OSImage,
				KernelVersion:     node.Status.NodeInfo.KernelVersion,
				Labels:            getNodeLabels(node),
				Taints:            node.Spec.Taints,
				Events:            events,
				Conditions:        node.Status.Conditions,
				CreatedAt:         node.CreationTimestamp.Time,
				UpdatedAt:         time.Now(),
			}

			// 将节点数据发送到 channel
			nodeChan <- k8sNode
		}(node)
	}

	go func() {
		defer close(doneChan)
		for k8sNode := range nodeChan {
			k8sNodes = append(k8sNodes, k8sNode)
			fmt.Println(k8sNode)
		}
	}()

	wg.Wait()
	close(nodeChan)
	<-doneChan

	return k8sNodes, nil
}

func (k *k8sService) GetNodeByID(ctx context.Context, id int) (*model.K8sNode, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) GetPodsByNodeID(ctx context.Context, nodeID int) ([]*model.K8sPod, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error {
	panic("implement me")

}

func (k *k8sService) BatchEnableSwitchNodes(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error {
	//TODO implement me
	panic("implement me")
}

// AddNodeTaint 添加或删除 Kubernetes 节点的 Taint
func (k *k8sService) AddNodeTaint(ctx context.Context, req *model.TaintK8sNodesRequest) error {
	// 获取集群信息
	cluster, err := k.dao.GetClusterByName(ctx, req.ClusterName)
	if err != nil {
		k.l.Error("获取集群信息失败", zap.Error(err))
		return err
	}

	// 获取 Kubernetes 客户端
	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 解析 YAML 配置中的 Taints
	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(req.TaintYaml), &taintsToProcess); err != nil {
		k.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	var errs []error

	// 遍历每个节点名称
	for _, nodeName := range req.NodeNames {
		// 获取节点信息
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			k.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		switch req.ModType {
		case "add":
			// 添加新的 Taints
			node.Spec.Taints = pkg.MergeTaints(node.Spec.Taints, taintsToProcess)

		case "delete":
			// 删除指定的 Taints
			node.Spec.Taints = pkg.RemoveTaints(node.Spec.Taints, taintsToProcess)

		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			k.l.Error(errMsg)
			errs = append(errs, fmt.Errorf(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("更新节点信息成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

func (k *k8sService) DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DrainPods(ctx context.Context, nodeID int) error {
	//TODO implement me
	panic("implement me")
}

// 获取指定集群上的 Node 列表
func (k *k8sService) getNodesByClusterID(ctx context.Context, client *kubernetes.Clientset) (*v1.NodeList, error) {
	nodes, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		k.l.Error("获取 Node 列表失败", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

// 获取指定节点上的 Pod 列表
func (k *k8sService) getPodsByNodeName(ctx context.Context, client *kubernetes.Clientset, nodeName string) (*v1.PodList, error) {
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})

	if err != nil {
		k.l.Error("获取 Pod 列表失败", zap.String("nodeName", nodeName), zap.Error(err))
		return nil, err
	}

	return pods, nil
}

// 获取节点事件
func (k *k8sService) getNodeEvents(ctx context.Context, client *kubernetes.Clientset, nodeName string) ([]model.OneEvent, error) {
	eventlist, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s", nodeName),
	})

	if err != nil {
		// 输出错误 nodename
		k.l.Error("获取节点事件失败", zap.String("nodeName", nodeName), zap.Error(err))
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

// 获取节点资源信息
func getResource(ctx context.Context, metricsCli *metricsClient.Clientset, nodeName string, pods *v1.PodList, node *v1.Node) ([]string, error) {
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
	nodeMetrics, err := metricsCli.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %v", err)
	}

	// CPU 和内存的使用量
	cpuUsage := nodeMetrics.Usage[corev1.ResourceCPU]
	memoryUsage := nodeMetrics.Usage[corev1.ResourceMemory]

	result = append(result, fmt.Sprintf("%dm/%dm", cpuUsage.MilliValue(), cpuCapacity.MilliValue()))
	result = append(result, fmt.Sprintf("%dMi/%dMi", memoryUsage.Value()/1024/1024, memoryCapacity.Value()/1024/1024))

	// PodNumInfo
	maxPods := node.Status.Allocatable[corev1.ResourcePods]
	result = append(result, fmt.Sprintf("%d/%d", len(pods.Items), maxPods.Value()))
	return result, nil
}

// 获取节点状态
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

// 判断节点是否可调度
func isNodeSchedulable(node corev1.Node) bool {
	return !node.Spec.Unschedulable
}

// 获取节点角色
func getNodeRoles(node corev1.Node) []string {
	var roles []string
	for key := range node.Labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			roles = append(roles, role)
		}
	}
	return roles
}

// 获取节点内部IP
func getInternalIP(node corev1.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			return address.Address
		}
	}
	return ""
}

// 获取节点标签
func getNodeLabels(node corev1.Node) []string {
	var labels []string
	for key, value := range node.Labels {
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}
	return labels
}

// 获取节点资源信息
func getResourceString(node corev1.Node, resourceName string) string {
	allocatable := node.Status.Allocatable[corev1.ResourceName(resourceName)]
	return allocatable.String()
}

// 计算节点存在时间
func getNodeAge(node corev1.Node) string {
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
