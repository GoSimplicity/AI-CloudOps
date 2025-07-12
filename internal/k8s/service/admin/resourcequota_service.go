/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package admin

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ResourceQuotaService interface {
	CreateResourceQuota(ctx context.Context, req *model.K8sResourceQuotaRequest) (*core.ResourceQuota, error)
	ListResourceQuotas(ctx context.Context, clusterID int, namespace string) ([]*core.ResourceQuota, error)
	GetResourceQuota(ctx context.Context, clusterID int, namespace, name string) (*core.ResourceQuota, error)
	UpdateResourceQuota(ctx context.Context, req *model.K8sResourceQuotaRequest) error
	DeleteResourceQuota(ctx context.Context, clusterID int, namespace, name string) error
	BatchDeleteResourceQuotas(ctx context.Context, clusterID int, namespace string, names []string) error
	GetResourceQuotaYaml(ctx context.Context, clusterID int, namespace, name string) (string, error)
	GetResourceQuotaUsage(ctx context.Context, clusterID int, namespace, name string) (*model.K8sResourceQuotaUsage, error)
}

type resourceQuotaService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewResourceQuotaService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) ResourceQuotaService {
	return &resourceQuotaService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// CreateResourceQuota 创建 ResourceQuota
func (r *resourceQuotaService) CreateResourceQuota(ctx context.Context, req *model.K8sResourceQuotaRequest) (*core.ResourceQuota, error) {
	r.logger.Info("开始创建 ResourceQuota", zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if req.ResourceQuotaYaml == nil {
		r.logger.Error("ResourceQuota YAML 不能为空", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("ResourceQuota YAML cannot be nil")
	}

	createdResourceQuota, err := kubeClient.CoreV1().ResourceQuotas(req.Namespace).Create(ctx, req.ResourceQuotaYaml, metav1.CreateOptions{})
	if err != nil {
		r.logger.Error("创建 ResourceQuota 失败", zap.Error(err), zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to create ResourceQuota: %w", err)
	}

	r.logger.Info("成功创建 ResourceQuota", zap.String("name", createdResourceQuota.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return createdResourceQuota, nil
}

// ListResourceQuotas 获取 ResourceQuota 列表
func (r *resourceQuotaService) ListResourceQuotas(ctx context.Context, clusterID int, namespace string) ([]*core.ResourceQuota, error) {
	r.logger.Info("开始获取 ResourceQuota 列表", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	resourceQuotas, err := kubeClient.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		r.logger.Error("获取 ResourceQuota 列表失败", zap.Error(err), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to list ResourceQuotas: %w", err)
	}

	result := make([]*core.ResourceQuota, len(resourceQuotas.Items))
	for i := range resourceQuotas.Items {
		result[i] = &resourceQuotas.Items[i]
	}

	r.logger.Info("成功获取 ResourceQuota 列表", zap.Int("count", len(result)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return result, nil
}

// GetResourceQuota 获取 ResourceQuota 详情
func (r *resourceQuotaService) GetResourceQuota(ctx context.Context, clusterID int, namespace, name string) (*core.ResourceQuota, error) {
	r.logger.Info("开始获取 ResourceQuota 详情", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	resourceQuota, err := kubeClient.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 ResourceQuota 详情失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get ResourceQuota: %w", err)
	}

	r.logger.Info("成功获取 ResourceQuota 详情", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return resourceQuota, nil
}

// UpdateResourceQuota 更新 ResourceQuota
func (r *resourceQuotaService) UpdateResourceQuota(ctx context.Context, req *model.K8sResourceQuotaRequest) error {
	r.logger.Info("开始更新 ResourceQuota", zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if req.ResourceQuotaYaml == nil {
		r.logger.Error("ResourceQuota YAML 不能为空", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("ResourceQuota YAML cannot be nil")
	}

	existingResourceQuota, err := kubeClient.CoreV1().ResourceQuotas(req.Namespace).Get(ctx, req.ResourceQuotaYaml.Name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取当前 ResourceQuota 失败", zap.Error(err), zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing ResourceQuota: %w", err)
	}

	existingResourceQuota.Spec = req.ResourceQuotaYaml.Spec
	existingResourceQuota.Labels = req.ResourceQuotaYaml.Labels
	existingResourceQuota.Annotations = req.ResourceQuotaYaml.Annotations

	_, err = kubeClient.CoreV1().ResourceQuotas(req.Namespace).Update(ctx, existingResourceQuota, metav1.UpdateOptions{})
	if err != nil {
		r.logger.Error("更新 ResourceQuota 失败", zap.Error(err), zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update ResourceQuota: %w", err)
	}

	r.logger.Info("成功更新 ResourceQuota", zap.String("name", req.ResourceQuotaYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// DeleteResourceQuota 删除 ResourceQuota
func (r *resourceQuotaService) DeleteResourceQuota(ctx context.Context, clusterID int, namespace, name string) error {
	r.logger.Info("开始删除 ResourceQuota", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	err = kubeClient.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("删除 ResourceQuota 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to delete ResourceQuota '%s': %w", name, err)
	}

	r.logger.Info("成功删除 ResourceQuota", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return nil
}

// BatchDeleteResourceQuotas 批量删除 ResourceQuota
func (r *resourceQuotaService) BatchDeleteResourceQuotas(ctx context.Context, clusterID int, namespace string, names []string) error {
	r.logger.Info("开始批量删除 ResourceQuota", zap.Strings("names", names), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(names))

	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().ResourceQuotas(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				r.logger.Error("删除 ResourceQuota 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
				errChan <- fmt.Errorf("failed to delete ResourceQuota '%s': %w", name, err)
			} else {
				r.logger.Info("成功删除 ResourceQuota", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
			}
		}(name)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		r.logger.Error("批量删除 ResourceQuota 部分失败", zap.Int("error_count", len(errs)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("errors occurred while deleting ResourceQuotas: %v", errs)
	}

	r.logger.Info("成功批量删除 ResourceQuota", zap.Int("count", len(names)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return nil
}

// GetResourceQuotaYaml 获取 ResourceQuota YAML
func (r *resourceQuotaService) GetResourceQuotaYaml(ctx context.Context, clusterID int, namespace, name string) (string, error) {
	r.logger.Info("开始获取 ResourceQuota YAML", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	resourceQuota, err := kubeClient.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 ResourceQuota 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to get ResourceQuota: %w", err)
	}

	yamlData, err := yaml.Marshal(resourceQuota)
	if err != nil {
		r.logger.Error("序列化 ResourceQuota YAML 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to serialize ResourceQuota YAML: %w", err)
	}

	r.logger.Info("成功获取 ResourceQuota YAML", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return string(yamlData), nil
}

// GetResourceQuotaUsage 获取配额使用情况
func (r *resourceQuotaService) GetResourceQuotaUsage(ctx context.Context, clusterID int, namespace, name string) (*model.K8sResourceQuotaUsage, error) {
	r.logger.Info("开始获取 ResourceQuota 使用情况", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, r.client, r.logger)
	if err != nil {
		r.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	resourceQuota, err := kubeClient.CoreV1().ResourceQuotas(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		r.logger.Error("获取 ResourceQuota 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get ResourceQuota: %w", err)
	}

	usage := &model.K8sResourceQuotaUsage{
		Name:              resourceQuota.Name,
		Namespace:         resourceQuota.Namespace,
		Hard:              make(map[string]string),
		Used:              make(map[string]string),
		UsagePercentage:   make(map[string]float64),
		CreationTimestamp: resourceQuota.CreationTimestamp.Time,
	}

	// 转换 Hard 限制
	for resourceName, quantity := range resourceQuota.Status.Hard {
		usage.Hard[string(resourceName)] = quantity.String()
	}

	// 转换 Used 使用量
	for resourceName, quantity := range resourceQuota.Status.Used {
		usage.Used[string(resourceName)] = quantity.String()
	}

	// 计算使用率百分比
	for resourceName := range resourceQuota.Status.Hard {
		hardQuantity := resourceQuota.Status.Hard[resourceName]
		usedQuantity := resourceQuota.Status.Used[resourceName]

		// 尝试解析数值型资源
		if hardQuantity.Value() > 0 {
			percentage := float64(usedQuantity.Value()) / float64(hardQuantity.Value()) * 100
			usage.UsagePercentage[string(resourceName)] = percentage
		} else {
			// 对于字节型资源，需要转换为字节进行计算
			hardBytes := hardQuantity.Value()
			usedBytes := usedQuantity.Value()
			if hardBytes > 0 {
				percentage := float64(usedBytes) / float64(hardBytes) * 100
				usage.UsagePercentage[string(resourceName)] = percentage
			}
		}
	}

	r.logger.Info("成功获取 ResourceQuota 使用情况", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return usage, nil
}

// calculateResourceUsagePercentage 计算资源使用率
func (r *resourceQuotaService) calculateResourceUsagePercentage(hard, used resource.Quantity) float64 {
	if hard.Value() == 0 {
		return 0.0
	}
	return float64(used.Value()) / float64(hard.Value()) * 100
}
