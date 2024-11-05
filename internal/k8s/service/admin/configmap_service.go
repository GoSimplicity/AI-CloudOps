package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sync"
)

type ConfigMapService interface {
	// GetConfigMapsByNamespace 获取指定命名空间的所有 ConfigMap
	GetConfigMapsByNamespace(ctx context.Context, clusterName, namespace string) ([]*corev1.ConfigMap, error)
	// CreateConfigMap 创建新的 ConfigMap
	CreateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	// UpdateConfigMap 更新已有的 ConfigMap
	UpdateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	// UpdateConfigMapData 更新指定 ConfigMap 的数据
	UpdateConfigMapData(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	// GetConfigMapYaml 获取 ConfigMap 的详细信息
	GetConfigMapYaml(ctx context.Context, clusterName, namespace, configMapName string) (*corev1.ConfigMap, error)
	// DeleteConfigMap 删除指定的 ConfigMap
	DeleteConfigMap(ctx context.Context, clusterName, namespace string, configMapNames []string) error
}

type configMapService struct {
	dao    dao.K8sDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewConfigMapService(dao dao.K8sDAO, client client.K8sClient, l *zap.Logger) ConfigMapService {
	return &configMapService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// getKubeClient 封装获取 Kubernetes 客户端的逻辑
func (c *configMapService) getKubeClient(ctx context.Context, clusterName string) (*kubernetes.Clientset, error) {
	kubeClient, err := pkg.GetKubeClient(ctx, clusterName, c.dao, c.client, c.l)
	if err != nil {
		c.l.Error("获取 Kubernetes 客户端失败", zap.Error(err))
		return nil, err
	}
	return kubeClient, nil
}

// GetConfigMapsByNamespace 获取指定命名空间的所有 ConfigMap
func (c *configMapService) GetConfigMapsByNamespace(ctx context.Context, clusterName, namespace string) ([]*corev1.ConfigMap, error) {
	kubeClient, err := c.getKubeClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	// 获取 ConfigMap 列表
	configMapList, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 列表失败", zap.Error(err))
		return nil, err
	}

	// 转换为 []*corev1.ConfigMap
	var configMaps []*corev1.ConfigMap
	for i := range configMapList.Items {
		configMaps = append(configMaps, &configMapList.Items[i])
	}

	return configMaps, nil
}

// CreateConfigMap 创建新的 ConfigMap
func (c *configMapService) CreateConfigMap(ctx context.Context, configMapResource *model.K8sConfigMapRequest) error {
	kubeClient, err := c.getKubeClient(ctx, configMapResource.ClusterName)
	if err != nil {
		return err
	}

	// 创建 ConfigMap
	_, err = kubeClient.CoreV1().ConfigMaps(configMapResource.ConfigMap.Namespace).Create(ctx, configMapResource.ConfigMap, metav1.CreateOptions{})
	if err != nil {
		c.l.Error("创建 ConfigMap 失败", zap.Error(err))
		return err
	}

	c.l.Info("创建 ConfigMap 成功", zap.String("configMapName", configMapResource.ConfigMap.Name))
	return nil
}

// UpdateConfigMap 更新已有的 ConfigMap
func (c *configMapService) UpdateConfigMap(ctx context.Context, configMapResource *model.K8sConfigMapRequest) error {
	kubeClient, err := c.getKubeClient(ctx, configMapResource.ClusterName)
	if err != nil {
		return err
	}

	// 更新 ConfigMap
	_, err = kubeClient.CoreV1().ConfigMaps(configMapResource.ConfigMap.Namespace).Update(ctx, configMapResource.ConfigMap, metav1.UpdateOptions{})
	if err != nil {
		c.l.Error("更新 ConfigMap 失败", zap.Error(err))
		return err
	}

	c.l.Info("更新 ConfigMap 成功", zap.String("configMapName", configMapResource.ConfigMap.Name))
	return nil
}

// UpdateConfigMapData 更新指定 ConfigMap 的数据
func (c *configMapService) UpdateConfigMapData(ctx context.Context, configMapResource *model.K8sConfigMapRequest) error {
	kubeClient, err := c.getKubeClient(ctx, configMapResource.ClusterName)
	if err != nil {
		return err
	}

	// 获取 ConfigMap
	configMap, err := kubeClient.CoreV1().ConfigMaps(configMapResource.ConfigMap.Namespace).Get(ctx, configMapResource.ConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 失败", zap.Error(err))
		return err
	}

	// 更新 ConfigMap 数据
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	// 更新数据
	for key, value := range configMapResource.ConfigMap.Data {
		configMap.Data[key] = value
	}

	// 更新 ConfigMap
	_, err = kubeClient.CoreV1().ConfigMaps(configMapResource.ConfigMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		c.l.Error("更新 ConfigMap 失败", zap.Error(err))
		return err
	}

	c.l.Info("更新 ConfigMap 成功", zap.String("configMapName", configMapResource.ConfigMap.Name))
	return nil
}

// GetConfigMapYaml 获取 ConfigMap 的详细信息
func (c *configMapService) GetConfigMapYaml(ctx context.Context, clusterName, namespace, configMapName string) (*corev1.ConfigMap, error) {
	kubeClient, err := c.getKubeClient(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	// 获取 ConfigMap
	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 失败", zap.Error(err))
		return nil, err
	}

	return configMap, nil
}

// DeleteConfigMap 并发删除指定的 ConfigMap 列表
func (c *configMapService) DeleteConfigMap(ctx context.Context, clusterName, namespace string, configMapNames []string) error {
	kubeClient, err := c.getKubeClient(ctx, clusterName)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	var errs []error
	var mu sync.Mutex // 用于保护 errs 数组，防止并发竞态

	// 使用 goroutines 并发删除 ConfigMap
	for _, name := range configMapNames {
		wg.Add(1)

		go func(name string) {
			defer wg.Done()

			// 删除 ConfigMap
			err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			if err != nil {
				c.l.Error("删除 ConfigMap 失败", zap.String("configMapName", name), zap.Error(err))

				// 保护 errs 数组的并发写入
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}

			c.l.Info("删除 ConfigMap 成功", zap.String("configMapName", name))
		}(name)
	}

	wg.Wait()

	// 如果有错误，返回所有错误信息
	if len(errs) > 0 {
		return fmt.Errorf("在删除 ConfigMap 时遇到以下错误: %v", errs)
	}

	return nil
}
