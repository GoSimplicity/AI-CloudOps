package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
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
	GetNodeByName(ctx context.Context, id int, name string) (*model.K8sNode, error)
	// GetPodsByNodeID 根据 Node ID 获取 Pod 列表
	GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error)
	// CheckTaintYaml 检查 Taint Yaml 是否合法
	CheckTaintYaml(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// BatchEnableSwitchNodes 批量启用或切换 Kubernetes 节点调度
	BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error
	// UpdateNodeLabel 添加或者删除指定节点 Label
	UpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesRequest) error
	// UpdateNodeTaint 添加或者删除指定节点 Taint
	UpdateNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
	// DrainPods 删除指定 Node 上的 Pod
	DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error

	// GetClusterNamespacesList 获取命名空间列表
	GetClusterNamespacesList(ctx context.Context) (map[string][]string, error)
	// GetClusterNamespacesByName 获取指定集群的所有命名空间
	GetClusterNamespacesByName(ctx context.Context, clusterName string) ([]string, error)

	// GetPodsByNamespace 获取指定命名空间的 Pod 列表
	GetPodsByNamespace(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPod, error)
	// GetContainersByPod 获取指定 Pod 的容器列表
	GetContainersByPod(ctx context.Context, clusterID int, namespace string, podName string) ([]*model.K8sPodContainer, error)
	// GetContainerLogs 获取指定 Pod 的容器日志
	GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error)
	// GetPodYaml 获取指定 Pod 的 YAML 配置
	GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error)
	// CreatePod 创建 Pod
	CreatePod(ctx context.Context, pod *model.K8sPodRequest) error
	// UpdatePod 更新 Pod
	UpdatePod(ctx context.Context, pod *model.K8sPodRequest) error
	// DeletePod 删除 Pod
	DeletePod(ctx context.Context, clusterName, namespace, podName string) error
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

// ListAllNodes 获取指定集群的所有节点信息
func (k *k8sService) ListAllNodes(ctx context.Context, id int) ([]*model.K8sNode, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := k.client.GetKubeClient(id)
	if err != nil {
		k.l.Error("ListAllNodes: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	// 获取 Metrics 客户端
	metricsClient, err := k.client.GetMetricsClient(id)
	if err != nil {
		k.l.Error("ListAllNodes: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, constants.ErrorMetricsClientNotReady
	}

	// 获取节点列表
	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, "")
	if err != nil {
		k.l.Error("ListAllNodes: 获取节点列表失败", zap.Error(err))
		return nil, err
	}

	// 设置最大并发数
	const maxConcurrency = 10
	semaphore := make(chan struct{}, maxConcurrency)

	g, ctx := errgroup.WithContext(ctx)

	// 使用互斥锁保护对共享切片的访问
	var mu sync.Mutex
	var k8sNodes []*model.K8sNode

	// 遍历每个节点，并发处理
	for _, node := range nodes.Items {
		node := node // 捕获循环变量

		g.Go(func() error {
			// 获取 semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 构建 K8sNode 对象
			k8sNode, err := pkg.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
			if err != nil {
				k.l.Error("ListAllNodes: 构建 K8sNode 失败", zap.Error(err), zap.String("node", node.Name))
				return nil
			}

			mu.Lock()
			k8sNodes = append(k8sNodes, k8sNode)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		k.l.Error("ListAllNodes: 并发处理节点信息失败", zap.Error(err))
		return nil, err
	}

	return k8sNodes, nil
}

func (k *k8sService) GetNodeByName(ctx context.Context, id int, name string) (*model.K8sNode, error) {
	// 获取 Kubernetes 客户端和 Metrics 客户端
	kubeClient, err := k.client.GetKubeClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	metricsClient, err := k.client.GetMetricsClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Metrics 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取节点
	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}
	node := nodes.Items[0]

	// 构建 k8sNode
	return pkg.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
}

func (k *k8sService) GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error) {
	kubeClient, err := k.client.GetKubeClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	nodes, err := pkg.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}

	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, name)
	if err != nil {
		return nil, err
	}

	return pkg.BuildK8sPods(pods), nil
}

func (k *k8sService) CheckTaintYaml(ctx context.Context, req *model.TaintK8sNodesRequest) error {
	// 1. binding 校验key不为空，effect 为 NoSchedule、PreferNoSchedule、NoExecute之一

	// 2. key 不重复
	var taintsToProcess []corev1.Taint
	if err := yaml.UnmarshalStrict([]byte(req.TaintYaml), &taintsToProcess); err != nil {
		k.l.Error("解析 Taint YAML 配置失败", zap.Error(err))
		return err
	}

	taintsKey := make(map[string]struct{})

	// for _, taint := range req.Taints {
	// 	if _, ok := taintsKey[taint.Key]; ok {
	// 		return constants.ErrorTaintsKeyDuplicate
	// 	}
	// 	taintsKey[taint.Key] = struct{}{}
	// }

	for _, taint := range taintsToProcess {
		if _, ok := taintsKey[taint.Key]; ok {
			return constants.ErrorTaintsKeyDuplicate
		}
		taintsKey[taint.Key] = struct{}{}
	}

	// 3. Cluster, Node 是否存在
	cluster, err := k.dao.GetClusterByName(ctx, req.ClusterName)
	if err != nil {
		return err
	}

	// Client 是否准备好
	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		return err
	}

	// 遍历每个节点名称
	var errs []error
	for _, nodeName := range req.NodeNames {
		_, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			k.l.Error("获取节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

func (k *k8sService) BatchEnableSwitchNodes(ctx context.Context, req *model.ScheduleK8sNodesRequest) error {
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

	// 遍历每个节点名称
	var errs []error
	for _, nodeName := range req.NodeNames {
		// 获取节点信息
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			k.l.Error("获取节点信息失败", zap.Error(err))
			continue
		}

		// 更新节点调度状态
		node.Spec.Unschedulable = !req.ScheduleEnable

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("更新节点调度状态成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点调度状态时遇到以下错误: %v", errs)
	}

	return nil
}

func (k *k8sService) UpdateNodeLabel(ctx context.Context, req *model.LabelK8sNodesRequest) error {
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
			for key, value := range req.Labels {
				node.Labels[key] = value
			}

		case "del":
			for key := range req.Labels {
				delete(node.Labels, key)
			}

		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			k.l.Error(errMsg)
			errs = append(errs, errors.New(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("更新节点Label成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Labels 时遇到以下错误: %v", errs)
	}

	return nil
}

// AddNodeTaint 添加或删除 Kubernetes 节点的 Taint
func (k *k8sService) UpdateNodeTaint(ctx context.Context, req *model.TaintK8sNodesRequest) error {
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

		case "del":
			// 删除指定的 Taints
			node.Spec.Taints = pkg.RemoveTaints(node.Spec.Taints, taintsToProcess)

		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			k.l.Error(errMsg)
			errs = append(errs, errors.New(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("更新节点Taint成功", zap.String("nodeName", nodeName))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在处理节点 Taints 时遇到以下错误: %v", errs)
	}

	return nil
}

// DrainPods 驱逐指定 Node 上的 Pod
func (k *k8sService) DrainPods(ctx context.Context, req *model.K8sClusterNodesRequest) error {
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

	// 获取 pods
	pods, err := pkg.GetPodsByNodeName(ctx, kubeClient, req.NodeNames[0])
	if err != nil {
		k.l.Error("获取 Pod 列表失败", zap.Error(err))
		return err
	}

	// 创建 Eviction 对象
	eviction := &policyv1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "Eviction",
		},
		DeleteOptions: &metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
		},
	}

	var errs []error
	// 遍历每个 Pod
	for _, pod := range pods.Items {
		// 设置 Eviction 对象的 Name 和 Namespace
		eviction.Name = pod.Name
		eviction.Namespace = pod.Namespace

		// 驱逐 Pod
		if err := kubeClient.PolicyV1().Evictions(eviction.Namespace).Evict(ctx, eviction); err != nil {
			k.l.Error("驱逐 Pod 失败", zap.Error(err), zap.String("podName", pod.Name))
			errs = append(errs, fmt.Errorf("驱逐 Pod %s 失败: %w", pod.Name, err))
			continue
		}

		k.l.Info("驱逐 Pod 成功", zap.String("podName", pod.Name))
	}

	if len(errs) > 0 {
		return fmt.Errorf("在驱逐 Pod 时遇到以下错误: %v", errs)
	}

	return nil
}

// GetClusterNamespacesList 获取所有集群的命名空间列表
func (k *k8sService) GetClusterNamespacesList(ctx context.Context) (map[string][]string, error) {
	clusters, err := k.dao.ListAllClusters(ctx)
	if err != nil {
		k.l.Error("获取集群列表失败", zap.Error(err))
		return nil, err
	}

	mp := make(map[string][]string)
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10)

	for _, cluster := range clusters {
		cluster := cluster // Capture loop variable
		g.Go(func() error {
			namespaces, err := k.GetClusterNamespacesByName(ctx, cluster.Name)
			if err != nil {
				k.l.Error("获取命名空间列表失败", zap.Error(err), zap.String("clusterName", cluster.Name))
				return err
			}

			mu.Lock()
			mp[cluster.Name] = namespaces
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		k.l.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	return mp, nil
}

// GetClusterNamespacesByName 获取指定集群的所有命名空间
func (k *k8sService) GetClusterNamespacesByName(ctx context.Context, clusterName string) ([]string, error) {
	cluster, err := k.dao.GetClusterByName(ctx, clusterName)
	if err != nil {
		k.l.Error("获取集群信息失败", zap.Error(err))
		return nil, err
	}

	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		k.l.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	var nsList []string
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}

	return nsList, nil
}

// GetPodsByNamespace 获取指定命名空间的 Pod 列表，可按名称过滤
func (k *k8sService) GetPodsByNamespace(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPod, error) {
	kubeClient, err := k.client.GetKubeClient(clusterID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	listOptions := metav1.ListOptions{}
	if podName != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", podName)
	}

	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		k.l.Error("获取 Pod 列表失败", zap.Error(err))
		return nil, err
	}

	return pkg.BuildK8sPods(pods), nil
}

// GetContainersByPod 获取指定 Pod 的容器列表
func (k *k8sService) GetContainersByPod(ctx context.Context, clusterID int, namespace, podName string) ([]*model.K8sPodContainer, error) {
	kubeClient, err := k.client.GetKubeClient(clusterID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		k.l.Error("获取 Pod 失败", zap.Error(err))
		return nil, err
	}

	containers := pkg.BuildK8sContainers(pod.Spec.Containers)
	return pkg.BuildK8sContainersWithPointer(containers), nil
}

// GetContainerLogs 获取指定 Pod 的容器日志
func (k *k8sService) GetContainerLogs(ctx context.Context, clusterID int, namespace, podName, containerName string) (string, error) {
	kubeClient, err := k.client.GetKubeClient(clusterID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return "", err
	}

	req := kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    false, // 不跟随日志
		Previous:  false, // 不使用 previous
	})

	podLogs, err := req.Stream(ctx)
	if err != nil {
		k.l.Error("获取 Pod 日志失败", zap.Error(err))
		return "", err
	}
	defer podLogs.Close()

	buf := new(strings.Builder)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		k.l.Error("读取 Pod 日志失败", zap.Error(err))
		return "", err
	}

	return buf.String(), nil
}

// GetPodYaml 获取指定 Pod 的 YAML 配置
func (k *k8sService) GetPodYaml(ctx context.Context, clusterID int, namespace, podName string) (*corev1.Pod, error) {
	kubeClient, err := k.client.GetKubeClient(clusterID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	pod, err := kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		k.l.Error("获取 Pod 信息失败", zap.Error(err))
		return nil, err
	}

	return pod, nil
}

// CreatePod 创建 Pod
func (k *k8sService) CreatePod(ctx context.Context, req *model.K8sPodRequest) error {
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

	// 创建 Pod
	_, err = kubeClient.CoreV1().Pods(req.Pod.Namespace).Create(ctx, req.Pod, metav1.CreateOptions{})
	if err != nil {
		k.l.Error("创建 Pod 失败", zap.Error(err))
		return err
	}

	k.l.Info("创建 Pod 成功", zap.String("podName", req.Pod.Name))
	return nil
}

// UpdatePod 更新 Pod
func (k *k8sService) UpdatePod(ctx context.Context, req *model.K8sPodRequest) error {
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

	// 更新 Pod
	_, err = kubeClient.CoreV1().Pods(req.Pod.Namespace).Update(ctx, req.Pod, metav1.UpdateOptions{})
	if err != nil {
		k.l.Error("更新 Pod 失败", zap.Error(err))
		return err
	}

	k.l.Info("更新 Pod 成功", zap.String("podName", req.Pod.Name))
	return nil
}

// DeletePod 删除 Pod
func (k *k8sService) DeletePod(ctx context.Context, clusterName, namespace, podName string) error {
	// 获取集群信息
	cluster, err := k.dao.GetClusterByName(ctx, clusterName)
	if err != nil {
		k.l.Error("获取集群信息失败", zap.Error(err))
		return err
	}

	kubeClient, err := k.client.GetKubeClient(cluster.ID)
	if err != nil {
		k.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return err
	}

	// 删除 Pod
	err = kubeClient.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
	if err != nil {
		k.l.Error("删除 Pod 失败", zap.Error(err))
		return err
	}

	k.l.Info("删除 Pod 成功", zap.String("podName", podName))
	return nil
}
