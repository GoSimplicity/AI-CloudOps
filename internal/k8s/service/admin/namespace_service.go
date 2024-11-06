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
	// GetClusterNamespacesByName 获取指定集群的所有命名空间
	GetClusterNamespacesByName(ctx context.Context, clusterName string) ([]string, error)
}

type namespaceService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewNamespaceService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) NamespaceService {
	return &namespaceService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetClusterNamespacesList 获取所有集群的命名空间列表
func (n *namespaceService) GetClusterNamespacesList(ctx context.Context) (map[string][]string, error) {
	// 获取集群列表
	clusters, err := n.dao.ListAllClusters(ctx)
	if err != nil {
		n.l.Error("获取集群列表失败", zap.Error(err))
		return nil, err
	}

	// 初始化返回的命名空间映射
	mp := make(map[string][]string)
	var mu sync.Mutex

	// 使用 errgroup 控制并发任务
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // 限制并发数为 10

	// 启动 goroutine 获取每个集群的命名空间
	for _, cluster := range clusters {
		cluster := cluster // 避免闭包引用问题
		g.Go(func() error {
			namespaces, err := n.GetClusterNamespacesByName(ctx, cluster.Name)
			if err != nil {
				n.l.Error("获取命名空间列表失败", zap.Error(err), zap.String("clusterName", cluster.Name))
				return err
			}

			// 使用互斥锁确保并发安全
			mu.Lock()
			mp[cluster.Name] = namespaces
			mu.Unlock()
			return nil
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		n.l.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	return mp, nil
}

// GetClusterNamespacesByName 获取指定集群的命名空间列表
func (n *namespaceService) GetClusterNamespacesByName(ctx context.Context, clusterName string) ([]string, error) {
	// 获取 Kubernetes 客户端
	kubeClient, err := pkg.GetKubeClient(ctx, clusterName, n.dao, n.client, n.l)
	if err != nil {
		n.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}

	// 获取命名空间列表
	namespaces, err := kubeClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		n.l.Error("获取命名空间列表失败", zap.Error(err))
		return nil, err
	}

	// 提取命名空间名称
	var nsList []string
	for _, ns := range namespaces.Items {
		nsList = append(nsList, ns.Name)
	}

	return nsList, nil
}
