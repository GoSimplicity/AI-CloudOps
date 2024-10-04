package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/clientcmd"
	"sync"
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
	// AddNodeLabel 添加标签到指定 Node
	AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error
	// UpdateNodeTaint 添加或者删除指定节点 Taint
	UpdateNodeTaint(ctx context.Context, taint *model.TaintK8sNodesRequest) error
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
	nodes, err := k8s.GetNodesByClusterID(ctx, kubeClient, "")
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
			k8sNode, err := k8s.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
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
	nodes, err := k8s.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}
	node := nodes.Items[0]

	// 构建 k8sNode
	return k8s.BuildK8sNode(ctx, id, node, kubeClient, metricsClient)
}

func (k *k8sService) GetPodsByNodeName(ctx context.Context, id int, name string) ([]*model.K8sPod, error) {
	kubeClient, err := k.client.GetKubeClient(id)
	if err != nil {
		k.l.Error("CreateCluster: 获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	// 获取节点
	nodes, err := k8s.GetNodesByClusterID(ctx, kubeClient, name)
	if err != nil || len(nodes.Items) == 0 {
		return nil, constants.ErrorNodeNotFound
	}

	pods, err := k8s.GetPodsByNodeName(ctx, kubeClient, name)
	if err != nil {
		return nil, err
	}

	var k8sPods []*model.K8sPod
	for _, pod := range pods.Items {
		k8sPod := &model.K8sPod{
			Name:        pod.Name,
			Namespace:   pod.Namespace,
			NodeName:    pod.Spec.NodeName,
			Status:      string(pod.Status.Phase),
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			Containers:  make([]model.K8sPodContainer, 0),
		}

		for _, container := range pod.Spec.Containers {
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

			k8sPod.Containers = append(k8sPod.Containers, newContainer)
		}

		k8sPods = append(k8sPods, k8sPod)
	}

	return k8sPods, nil
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

func (k *k8sService) AddNodeLabel(ctx context.Context, nodeID int, labelKey, labelValue string) error {
	//TODO implement me
	panic("implement me")
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

		case "delete":
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

func (k *k8sService) DeleteNodeLabel(ctx context.Context, nodeID int, labelKey string) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) DrainPods(ctx context.Context, nodeID int) error {
	//TODO implement me
	panic("implement me")
}
