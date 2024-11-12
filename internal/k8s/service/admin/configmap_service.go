package admin

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapService interface {
	GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error)
	CreateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	UpdateConfigMap(ctx context.Context, configMap *model.K8sConfigMapRequest) error
	GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error)
	BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error
}

type configMapService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewConfigMapService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) ConfigMapService {
	return &configMapService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

// GetConfigMapsByNamespace 获取命名空间的所有 ConfigMap
func (c *configMapService) GetConfigMapsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.l)
	if err != nil {
		return nil, err
	}

	// 获取configMap列表
	configMapList, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 列表失败", zap.Error(err))
		return nil, err
	}

	// 将configMap列表转换为数组
	configMaps := make([]*corev1.ConfigMap, len(configMapList.Items))
	for i := range configMapList.Items {
		configMaps[i] = &configMapList.Items[i]
	}

	return configMaps, nil
}

// CreateConfigMap 创建新的 ConfigMap
func (c *configMapService) CreateConfigMap(ctx context.Context, configMapRequest *model.K8sConfigMapRequest) error {
	kubeClient, err := pkg.GetKubeClient(configMapRequest.ClusterId, c.client, c.l)
	if err != nil {
		return err
	}

	// 创建 ConfigMap
	_, err = kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Create(ctx, configMapRequest.ConfigMap, metav1.CreateOptions{})
	if err != nil {
		c.l.Error("创建 ConfigMap 失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateConfigMap 更新 ConfigMap
func (c *configMapService) UpdateConfigMap(ctx context.Context, configMapRequest *model.K8sConfigMapRequest) error {
	kubeClient, err := pkg.GetKubeClient(configMapRequest.ClusterId, c.client, c.l)
	if err != nil {
		return err
	}

	// 获取单个 ConfigMap
	configMap, err := kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Get(ctx, configMapRequest.ConfigMap.Name, metav1.GetOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 失败", zap.Error(err))
		return err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	for key, value := range configMapRequest.ConfigMap.Data {
		configMap.Data[key] = value
	}

	_, err = kubeClient.CoreV1().ConfigMaps(configMapRequest.ConfigMap.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		c.l.Error("更新 ConfigMap 数据失败", zap.Error(err))
		return err
	}

	return nil
}

// GetConfigMapYaml 获取 ConfigMap 详情
func (c *configMapService) GetConfigMapYaml(ctx context.Context, id int, namespace, configMapName string) (*corev1.ConfigMap, error) {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.l)
	if err != nil {
		return nil, err
	}

	configMap, err := kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		c.l.Error("获取 ConfigMap 失败", zap.Error(err))
		return nil, err
	}

	return configMap, nil
}

// BatchDeleteConfigMap 批量删除指定的 ConfigMap
func (c *configMapService) BatchDeleteConfigMap(ctx context.Context, id int, namespace string, configMapNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, c.client, c.l)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(configMapNames))

	// 并发删除 ConfigMap
	for _, name := range configMapNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			err := kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
			if err != nil {
				c.l.Error("删除 ConfigMap 失败", zap.String("configMapName", name), zap.Error(err))
				errCh <- err
			} else {
				c.l.Info("删除 ConfigMap 成功", zap.String("configMapName", name))
			}
		}(name)
	}

	wg.Wait()
	close(errCh)

	// 检查是否有错误
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("在删除 ConfigMap 时遇到错误: %v", errs)
	}

	return nil
}
