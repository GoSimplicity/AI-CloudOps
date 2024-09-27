package service

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
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
	ListAllNodes(ctx context.Context) ([]*model.K8sNode, error)
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
	//TODO implement me
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
			if err := k.ensureNamespace(ctx1, kubeClient, ns); err != nil {
				errChan <- fmt.Errorf("确保命名空间 %s 存在失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 LimitRange
			if err := k.applyLimitRange(ctx1, kubeClient, ns, cluster); err != nil {
				errChan <- fmt.Errorf("应用 LimitRange 到命名空间 %s 失败: %w", ns, err)
				cancel()
				return
			}

			// 应用 ResourceQuota
			if err := k.applyResourceQuota(ctx1, kubeClient, ns, cluster); err != nil {
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

// ensureNamespace 确保指定的命名空间存在，如果不存在则创建
func (k *k8sService) ensureNamespace(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string) error {
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
				k.l.Error("ensureNamespace: 创建命名空间失败", zap.String("namespace", namespace), zap.Error(createErr))
				return fmt.Errorf("创建命名空间 %s 失败: %w", namespace, createErr)
			}
			k.l.Info("ensureNamespace: 命名空间创建成功", zap.String("namespace", namespace))
			return nil
		}
		k.l.Error("ensureNamespace: 获取命名空间失败", zap.String("namespace", namespace), zap.Error(err))
		return fmt.Errorf("获取命名空间 %s 失败: %w", namespace, err)
	}
	// 命名空间已存在
	k.l.Info("ensureNamespace: 命名空间已存在", zap.String("namespace", namespace))
	return nil
}

// applyLimitRange 应用 LimitRange 到指定命名空间
func (k *k8sService) applyLimitRange(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
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
			k.l.Warn("applyLimitRange: LimitRange 已存在", zap.String("namespace", namespace))
			return nil
		}
		k.l.Error("applyLimitRange: 创建 LimitRange 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 LimitRange 失败 (namespace: %s): %w", namespace, err)
	}

	k.l.Info("applyLimitRange: LimitRange 创建成功", zap.String("namespace", namespace))
	return nil
}

// applyResourceQuota 应用 ResourceQuota 到指定命名空间
func (k *k8sService) applyResourceQuota(ctx context.Context, kubeClient *kubernetes.Clientset, namespace string, cluster *model.K8sCluster) error {
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
			k.l.Warn("applyResourceQuota: ResourceQuota 已存在", zap.String("namespace", namespace))
			return nil
		}
		k.l.Error("applyResourceQuota: 创建 ResourceQuota 失败", zap.Error(err), zap.String("namespace", namespace))
		return fmt.Errorf("创建 ResourceQuota 失败 (namespace: %s): %w", namespace, err)
	}

	k.l.Info("applyResourceQuota: ResourceQuota 创建成功", zap.String("namespace", namespace))
	return nil
}

func (k *k8sService) UpdateCluster(ctx context.Context, id int, cluster *model.K8sCluster) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) BatchEnableSwitchClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) BatchDeleteClusters(ctx context.Context, ids []int) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) GetClusterByID(ctx context.Context, id int) (*model.K8sCluster, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sService) ListAllNodes(ctx context.Context) ([]*model.K8sNode, error) {
	return k.dao.ListAllNodes(ctx)
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
		k.l.Error("解析 YAML 失败", zap.Error(err))
		return err
	}

	var errs []error

	// 遍历每个节点名称
	for _, nodeName := range req.NodeNames {
		// 获取节点信息
		node, err := kubeClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			k.l.Error("获取节点信息失败", zap.String("node", nodeName), zap.Error(err))
			errs = append(errs, fmt.Errorf("获取节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		switch req.ModType {
		case "add":
			// 添加新的 Taints，避免重复
			existingTaints := node.Spec.Taints
			taintsMap := make(map[string]corev1.Taint)

			// 记录现有 taints，键为 "Key:Value:Effect" 形式
			for _, taint := range existingTaints {
				key := fmt.Sprintf("%s:%s:%s", taint.Key, taint.Value, taint.Effect)
				taintsMap[key] = taint
			}

			// 添加新的 Taints，避免重复
			for _, newTaint := range taintsToProcess {
				key := fmt.Sprintf("%s:%s:%s", newTaint.Key, newTaint.Value, newTaint.Effect)
				if _, exists := taintsMap[key]; !exists {
					existingTaints = append(existingTaints, newTaint)
				}
			}

			node.Spec.Taints = existingTaints

		case "delete":
			// 删除指定的 Taints
			taintsToDelete := make(map[string]struct{})
			for _, delTaint := range taintsToProcess {
				key := fmt.Sprintf("%s:%s:%s", delTaint.Key, delTaint.Value, delTaint.Effect)
				taintsToDelete[key] = struct{}{}
			}

			var updatedTaints []corev1.Taint

			for _, existingTaint := range node.Spec.Taints {
				key := fmt.Sprintf("%s:%s:%s", existingTaint.Key, existingTaint.Value, existingTaint.Effect)
				if _, shouldDelete := taintsToDelete[key]; !shouldDelete {
					updatedTaints = append(updatedTaints, existingTaint)
				}
			}
			node.Spec.Taints = updatedTaints

		default:
			// 处理未知的修改类型
			errMsg := fmt.Sprintf("未知的修改类型: %s", req.ModType)
			k.l.Error(errMsg, zap.String("ModType", req.ModType))
			errs = append(errs, fmt.Errorf(errMsg))
			continue
		}

		// 更新节点信息
		_, err = kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			k.l.Error("更新节点信息失败", zap.String("node", nodeName), zap.Error(err))
			errs = append(errs, fmt.Errorf("更新节点 %s 信息失败: %w", nodeName, err))
			continue
		}

		k.l.Info("成功更新节点 Taints", zap.String("node", nodeName), zap.String("ModType", req.ModType))
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
