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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type LimitRangeService interface {
	CreateLimitRange(ctx context.Context, req *model.K8sLimitRangeRequest) (*core.LimitRange, error)
	ListLimitRanges(ctx context.Context, clusterID int, namespace string) ([]*core.LimitRange, error)
	GetLimitRange(ctx context.Context, clusterID int, namespace, name string) (*core.LimitRange, error)
	UpdateLimitRange(ctx context.Context, req *model.K8sLimitRangeRequest) error
	DeleteLimitRange(ctx context.Context, clusterID int, namespace, name string) error
	BatchDeleteLimitRanges(ctx context.Context, clusterID int, namespace string, names []string) error
	GetLimitRangeYaml(ctx context.Context, clusterID int, namespace, name string) (string, error)
}

type limitRangeService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewLimitRangeService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) LimitRangeService {
	return &limitRangeService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// CreateLimitRange 创建 LimitRange
func (l *limitRangeService) CreateLimitRange(ctx context.Context, req *model.K8sLimitRangeRequest) (*core.LimitRange, error) {
	l.logger.Info("开始创建 LimitRange", zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if req.LimitRangeYaml == nil {
		l.logger.Error("LimitRange YAML 不能为空", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("LimitRange YAML cannot be nil")
	}

	createdLimitRange, err := kubeClient.CoreV1().LimitRanges(req.Namespace).Create(ctx, req.LimitRangeYaml, metav1.CreateOptions{})
	if err != nil {
		l.logger.Error("创建 LimitRange 失败", zap.Error(err), zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to create LimitRange: %w", err)
	}

	l.logger.Info("成功创建 LimitRange", zap.String("name", createdLimitRange.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return createdLimitRange, nil
}

// ListLimitRanges 获取 LimitRange 列表
func (l *limitRangeService) ListLimitRanges(ctx context.Context, clusterID int, namespace string) ([]*core.LimitRange, error) {
	l.logger.Info("开始获取 LimitRange 列表", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	limitRanges, err := kubeClient.CoreV1().LimitRanges(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		l.logger.Error("获取 LimitRange 列表失败", zap.Error(err), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to list LimitRanges: %w", err)
	}

	result := make([]*core.LimitRange, len(limitRanges.Items))
	for i := range limitRanges.Items {
		result[i] = &limitRanges.Items[i]
	}

	l.logger.Info("成功获取 LimitRange 列表", zap.Int("count", len(result)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return result, nil
}

// GetLimitRange 获取 LimitRange 详情
func (l *limitRangeService) GetLimitRange(ctx context.Context, clusterID int, namespace, name string) (*core.LimitRange, error) {
	l.logger.Info("开始获取 LimitRange 详情", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	limitRange, err := kubeClient.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		l.logger.Error("获取 LimitRange 详情失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get LimitRange: %w", err)
	}

	l.logger.Info("成功获取 LimitRange 详情", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return limitRange, nil
}

// UpdateLimitRange 更新 LimitRange
func (l *limitRangeService) UpdateLimitRange(ctx context.Context, req *model.K8sLimitRangeRequest) error {
	l.logger.Info("开始更新 LimitRange", zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if req.LimitRangeYaml == nil {
		l.logger.Error("LimitRange YAML 不能为空", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("LimitRange YAML cannot be nil")
	}

	existingLimitRange, err := kubeClient.CoreV1().LimitRanges(req.Namespace).Get(ctx, req.LimitRangeYaml.Name, metav1.GetOptions{})
	if err != nil {
		l.logger.Error("获取当前 LimitRange 失败", zap.Error(err), zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing LimitRange: %w", err)
	}

	existingLimitRange.Spec = req.LimitRangeYaml.Spec
	existingLimitRange.Labels = req.LimitRangeYaml.Labels
	existingLimitRange.Annotations = req.LimitRangeYaml.Annotations

	_, err = kubeClient.CoreV1().LimitRanges(req.Namespace).Update(ctx, existingLimitRange, metav1.UpdateOptions{})
	if err != nil {
		l.logger.Error("更新 LimitRange 失败", zap.Error(err), zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update LimitRange: %w", err)
	}

	l.logger.Info("成功更新 LimitRange", zap.String("name", req.LimitRangeYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// DeleteLimitRange 删除 LimitRange
func (l *limitRangeService) DeleteLimitRange(ctx context.Context, clusterID int, namespace, name string) error {
	l.logger.Info("开始删除 LimitRange", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	err = kubeClient.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		l.logger.Error("删除 LimitRange 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to delete LimitRange '%s': %w", name, err)
	}

	l.logger.Info("成功删除 LimitRange", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return nil
}

// BatchDeleteLimitRanges 批量删除 LimitRange
func (l *limitRangeService) BatchDeleteLimitRanges(ctx context.Context, clusterID int, namespace string, names []string) error {
	l.logger.Info("开始批量删除 LimitRange", zap.Strings("names", names), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(names))

	for _, name := range names {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().LimitRanges(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				l.logger.Error("删除 LimitRange 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
				errChan <- fmt.Errorf("failed to delete LimitRange '%s': %w", name, err)
			} else {
				l.logger.Info("成功删除 LimitRange", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
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
		l.logger.Error("批量删除 LimitRange 部分失败", zap.Int("error_count", len(errs)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return fmt.Errorf("errors occurred while deleting LimitRanges: %v", errs)
	}

	l.logger.Info("成功批量删除 LimitRange", zap.Int("count", len(names)), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return nil
}

// GetLimitRangeYaml 获取 LimitRange YAML
func (l *limitRangeService) GetLimitRangeYaml(ctx context.Context, clusterID int, namespace, name string) (string, error) {
	l.logger.Info("开始获取 LimitRange YAML", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	limitRange, err := kubeClient.CoreV1().LimitRanges(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		l.logger.Error("获取 LimitRange 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to get LimitRange: %w", err)
	}

	yamlData, err := yaml.Marshal(limitRange)
	if err != nil {
		l.logger.Error("序列化 LimitRange YAML 失败", zap.Error(err), zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
		return "", fmt.Errorf("failed to serialize LimitRange YAML: %w", err)
	}

	l.logger.Info("成功获取 LimitRange YAML", zap.String("name", name), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return string(yamlData), nil
}