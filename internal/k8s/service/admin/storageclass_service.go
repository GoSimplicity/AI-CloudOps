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
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StorageClassService interface {
	GetStorageClasses(ctx context.Context, id int) ([]*storagev1.StorageClass, error)
	CreateStorageClass(ctx context.Context, req *model.K8sStorageClassRequest) error
	DeleteStorageClass(ctx context.Context, id int, storageClassName string) error
	BatchDeleteStorageClass(ctx context.Context, id int, storageClassNames []string) error
	GetStorageClassYaml(ctx context.Context, id int, storageClassName string) (string, error)
	GetStorageClassStatus(ctx context.Context, id int, storageClassName string) (*model.K8sStorageClassStatus, error)
	GetStorageClassConfig(ctx context.Context, id int, storageClassName string) (map[string]string, error)
	GetDefaultStorageClass(ctx context.Context, id int) (*storagev1.StorageClass, error)
}

type storageClassService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewStorageClassService 创建新的 StorageClassService 实例
func NewStorageClassService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) StorageClassService {
	return &storageClassService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetStorageClasses 获取所有 StorageClass
func (s *storageClassService) GetStorageClasses(ctx context.Context, id int) ([]*storagev1.StorageClass, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	storageClasses, err := kubeClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 StorageClass 列表失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get StorageClass list: %w", err)
	}

	result := make([]*storagev1.StorageClass, len(storageClasses.Items))
	for i := range storageClasses.Items {
		result[i] = &storageClasses.Items[i]
	}

	s.logger.Info("成功获取 StorageClass 列表", zap.Int("cluster_id", id), zap.Int("count", len(result)))
	return result, nil
}

// CreateStorageClass 创建 StorageClass
func (s *storageClassService) CreateStorageClass(ctx context.Context, req *model.K8sStorageClassRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.StorageV1().StorageClasses().Create(ctx, req.StorageClassYaml, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", req.StorageClassYaml.Name), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create StorageClass: %w", err)
	}

	s.logger.Info("成功创建 StorageClass", zap.String("storage_class_name", req.StorageClassYaml.Name), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetStorageClassYaml 获取指定 StorageClass 的 YAML 定义
func (s *storageClassService) GetStorageClassYaml(ctx context.Context, id int, storageClassName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	storageClass, err := kubeClient.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get StorageClass: %w", err)
	}

	yamlData, err := yaml.Marshal(storageClass)
	if err != nil {
		s.logger.Error("序列化 StorageClass YAML 失败", zap.Error(err), zap.String("storage_class_name", storageClassName))
		return "", fmt.Errorf("failed to serialize StorageClass YAML: %w", err)
	}

	s.logger.Info("成功获取 StorageClass YAML", zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeleteStorageClass 批量删除 StorageClass
func (s *storageClassService) BatchDeleteStorageClass(ctx context.Context, id int, storageClassNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(storageClassNames))

	for _, name := range storageClassNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.StorageV1().StorageClasses().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				s.logger.Error("删除 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", name), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete StorageClass '%s': %w", name, err)
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
		s.logger.Error("批量删除 StorageClass 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(storageClassNames)), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting StorageClasses: %v", errs)
	}

	s.logger.Info("成功批量删除 StorageClass", zap.Int("count", len(storageClassNames)), zap.Int("cluster_id", id))
	return nil
}

// DeleteStorageClass 删除指定的 StorageClass
func (s *storageClassService) DeleteStorageClass(ctx context.Context, id int, storageClassName string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.StorageV1().StorageClasses().Delete(ctx, storageClassName, metav1.DeleteOptions{}); err != nil {
		s.logger.Error("删除 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete StorageClass '%s': %w", storageClassName, err)
	}

	s.logger.Info("成功删除 StorageClass", zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
	return nil
}

// GetStorageClassStatus 获取 StorageClass 状态
func (s *storageClassService) GetStorageClassStatus(ctx context.Context, id int, storageClassName string) (*model.K8sStorageClassStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	storageClass, err := kubeClient.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get StorageClass: %w", err)
	}

	status := &model.K8sStorageClassStatus{
		Name:                 storageClass.Name,
		Provisioner:          storageClass.Provisioner,
		Parameters:           storageClass.Parameters,
		ReclaimPolicy:        storageClass.ReclaimPolicy,
		VolumeBindingMode:    storageClass.VolumeBindingMode,
		AllowVolumeExpansion: storageClass.AllowVolumeExpansion,
		CreationTimestamp:    storageClass.CreationTimestamp.Time,
	}

	s.logger.Info("成功获取 StorageClass 状态", zap.String("storage_class_name", storageClassName), zap.String("provisioner", storageClass.Provisioner), zap.Int("cluster_id", id))
	return status, nil
}

// GetStorageClassConfig 获取 StorageClass 配置参数
func (s *storageClassService) GetStorageClassConfig(ctx context.Context, id int, storageClassName string) (map[string]string, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	storageClass, err := kubeClient.StorageV1().StorageClasses().Get(ctx, storageClassName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StorageClass 失败", zap.Error(err), zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get StorageClass: %w", err)
	}

	s.logger.Info("成功获取 StorageClass 配置参数", zap.String("storage_class_name", storageClassName), zap.Int("cluster_id", id))
	return storageClass.Parameters, nil
}

// GetDefaultStorageClass 获取默认存储类
func (s *storageClassService) GetDefaultStorageClass(ctx context.Context, id int) (*storagev1.StorageClass, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	storageClasses, err := kubeClient.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 StorageClass 列表失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get StorageClass list: %w", err)
	}

	for _, sc := range storageClasses.Items {
		if sc.Annotations != nil {
			if isDefault, exists := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; exists && isDefault == "true" {
				s.logger.Info("成功获取默认 StorageClass", zap.String("storage_class_name", sc.Name), zap.Int("cluster_id", id))
				return &sc, nil
			}
		}
	}

	s.logger.Info("未找到默认 StorageClass", zap.Int("cluster_id", id))
	return nil, fmt.Errorf("no default StorageClass found")
}