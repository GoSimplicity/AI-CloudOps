package admin

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
)

type NamespaceService interface {
	// GetClusterNamespacesList 获取命名空间列表
	GetClusterNamespacesList(ctx context.Context) (map[string][]string, error)
	// GetClusterNamespacesById 获取指定集群的所有命名空间
	GetClusterNamespacesById(ctx context.Context, id int) ([]string, error)
}

type namespaceService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewNamespaceService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) NamespaceService {
	return &namespaceService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetClusterNamespacesList 获取所有集群的命名空间列表
func (n *namespaceService) GetClusterNamespacesList(ctx context.Context) (map[string][]string, error) {
	// 获取集群列表
	clusters, err := n.dao.ListAllClusters(ctx)
	if err != nil {
		n.logger.Error("获取集群列表失败", zap.Error(err))
		return nil, err
	}

	namespaceMap := make(map[string][]string)
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // 限制并发数为 10

	for _, cluster := range clusters {
		cluster := cluster // 避免闭包变量捕获问题
		g.Go(func() error {
			namespaces, err := n.GetClusterNamespacesById(ctx, cluster.ID)
			if err != nil {
				n.logger.Error("获取命名空间列表失败", zap.Error(err), zap.String("clusterName", cluster.Name))
				return err
			}

			mu.Lock()
			defer mu.Unlock()
			namespaceMap[cluster.Name] = namespaces
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		n.logger.Error("并发获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	return namespaceMap, nil
}

// GetClusterNamespacesById 获取指定集群的命名空间列表
func (n *namespaceService) GetClusterNamespacesById(ctx context.Context, id int) ([]string, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取命名空间列表
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	// 提取命名空间名称并返回
	nsList := make([]string, len(namespaces.Items))
	for i, ns := range namespaces.Items {
		nsList[i] = ns.Name
	}

	return nsList, nil
}
