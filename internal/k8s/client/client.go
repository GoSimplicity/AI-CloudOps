package client

import (
	"github.com/openkruise/kruise-api/client/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	metricsClient "k8s.io/metrics/pkg/client/clientset/versioned"
	"sync"
)

type K8sClient interface {
	// InitClient 初始化指定集群 ID 的 Kubernetes 客户端
	InitClient(clusterID uint, kubeConfig *rest.Config) error
	// GetKubeClient 获取指定集群 ID 的 Kubernetes 客户端
	GetKubeClient(clusterID uint) (*kubernetes.Clientset, error)
	// GetKruiseClient 获取指定集群 ID 的 Kruise 客户端
	GetKruiseClient(clusterID uint) (*versioned.Clientset, error)
	// GetMetricsClient 获取指定集群 ID 的 Metrics 客户端
	GetMetricsClient(clusterID uint) (*metricsClient.Clientset, error)
	// GetDynamicClient 获取指定集群 ID 的动态客户端
	GetDynamicClient(clusterID uint) (*dynamic.DynamicClient, error)
	// GetNamespaces 获取指定集群的命名空间
	GetNamespaces(clusterID uint) ([]string, error)
	// RecordProbeError 记录指定集群探活时遇到的错误
	RecordProbeError(clusterID uint, errMsg string)
}

// k8sClient 用于管理不同集群的客户端连接
type k8sClient struct {
	sync.RWMutex
	KubeClients       map[int]*kubernetes.Clientset    // K8s原生客户端集合
	KruiseClients     map[int]*versioned.Clientset     // Kruise扩展客户端集合
	MetricsClients    map[int]*metricsClient.Clientset // Metrics客户端集合
	DynamicClients    map[int]*dynamic.DynamicClient   // 动态客户端集合
	RestConfigs       map[int]*rest.Config             // REST配置集合
	ClusterNamespaces map[string][]string              // 集群命名空间集合
	LastProbeErrors   map[int]string                   // 集群探针错误信息
	NamespaceLock     sync.RWMutex                     // 命名空间的锁
	Logger            *zap.Logger
}

func NewK8sClient(kubeClients map[int]*kubernetes.Clientset, kruiseClients map[int]*versioned.Clientset, metricsClients map[int]*metricsClient.Clientset, dynamicClients map[int]*dynamic.DynamicClient, restConfigs map[int]*rest.Config, logger *zap.Logger) K8sClient {
	return &k8sClient{
		KubeClients:       kubeClients,
		KruiseClients:     kruiseClients,
		MetricsClients:    metricsClients,
		DynamicClients:    dynamicClients,
		RestConfigs:       restConfigs,
		ClusterNamespaces: make(map[string][]string),
		LastProbeErrors:   make(map[int]string),
		Logger:            logger,
	}
}

func (k *k8sClient) InitClient(clusterID uint, kubeConfig *rest.Config) error {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) GetKubeClient(clusterID uint) (*kubernetes.Clientset, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) GetKruiseClient(clusterID uint) (*versioned.Clientset, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) GetMetricsClient(clusterID uint) (*metricsClient.Clientset, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) GetDynamicClient(clusterID uint) (*dynamic.DynamicClient, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) GetNamespaces(clusterID uint) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (k *k8sClient) RecordProbeError(clusterID uint, errMsg string) {
	//TODO implement me
	panic("implement me")
}
