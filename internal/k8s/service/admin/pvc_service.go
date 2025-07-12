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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVCService interface {
	GetPVCsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.PersistentVolumeClaim, error)
	CreatePVC(ctx context.Context, req *model.K8sPVCRequest) error
	DeletePVC(ctx context.Context, id int, namespace, pvcName string) error
	BatchDeletePVC(ctx context.Context, id int, namespace string, pvcNames []string) error
	GetPVCYaml(ctx context.Context, id int, namespace, pvcName string) (string, error)
	GetPVCStatus(ctx context.Context, id int, namespace, pvcName string) (*model.K8sPVCStatus, error)
	GetPVCBinding(ctx context.Context, id int, namespace, pvcName string) (string, error)
	GetPVCCapacityRequest(ctx context.Context, id int, namespace, pvcName string) (string, error)
}

type pvcService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewPVCService 创建新的 PVCService 实例
func NewPVCService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) PVCService {
	return &pvcService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetPVCsByNamespace 获取指定命名空间下的所有 PVC
func (p *pvcService) GetPVCsByNamespace(ctx context.Context, id int, namespace string) ([]*corev1.PersistentVolumeClaim, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvcs, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolumeClaim 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get PersistentVolumeClaim list: %w", err)
	}

	result := make([]*corev1.PersistentVolumeClaim, len(pvcs.Items))
	for i := range pvcs.Items {
		result[i] = &pvcs.Items[i]
	}

	p.logger.Info("成功获取 PersistentVolumeClaim 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreatePVC 创建 PersistentVolumeClaim
func (p *pvcService) CreatePVC(ctx context.Context, req *model.K8sPVCRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.CoreV1().PersistentVolumeClaims(req.Namespace).Create(ctx, req.PVCYaml, metav1.CreateOptions{})
	if err != nil {
		p.logger.Error("创建 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", req.PVCYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create PersistentVolumeClaim: %w", err)
	}

	p.logger.Info("成功创建 PersistentVolumeClaim", zap.String("pvc_name", req.PVCYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetPVCYaml 获取指定 PVC 的 YAML 定义
func (p *pvcService) GetPVCYaml(ctx context.Context, id int, namespace, pvcName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}

	yamlData, err := yaml.Marshal(pvc)
	if err != nil {
		p.logger.Error("序列化 PersistentVolumeClaim YAML 失败", zap.Error(err), zap.String("pvc_name", pvcName))
		return "", fmt.Errorf("failed to serialize PersistentVolumeClaim YAML: %w", err)
	}

	p.logger.Info("成功获取 PersistentVolumeClaim YAML", zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeletePVC 批量删除 PVC
func (p *pvcService) BatchDeletePVC(ctx context.Context, id int, namespace string, pvcNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(pvcNames))

	for _, name := range pvcNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				p.logger.Error("删除 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete PersistentVolumeClaim '%s': %w", name, err)
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
		p.logger.Error("批量删除 PersistentVolumeClaim 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(pvcNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting PersistentVolumeClaims: %v", errs)
	}

	p.logger.Info("成功批量删除 PersistentVolumeClaim", zap.Int("count", len(pvcNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeletePVC 删除指定的 PVC
func (p *pvcService) DeletePVC(ctx context.Context, id int, namespace, pvcName string) error {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{}); err != nil {
		p.logger.Error("删除 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete PersistentVolumeClaim '%s': %w", pvcName, err)
	}

	p.logger.Info("成功删除 PersistentVolumeClaim", zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetPVCStatus 获取 PVC 状态
func (p *pvcService) GetPVCStatus(ctx context.Context, id int, namespace, pvcName string) (*model.K8sPVCStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}

	// 转换容量映射
	capacity := make(map[corev1.ResourceName]string)
	for k, v := range pvc.Status.Capacity {
		capacity[k] = v.String()
	}

	// 获取请求的存储容量
	requestedStorage := ""
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestedStorage = storage.String()
	}

	status := &model.K8sPVCStatus{
		Name:               pvc.Name,
		Namespace:          pvc.Namespace,
		Phase:              pvc.Status.Phase,
		VolumeName:         pvc.Spec.VolumeName,
		Capacity:           capacity,
		RequestedStorage:   requestedStorage,
		StorageClass:       pvc.Spec.StorageClassName,
		VolumeMode:         pvc.Spec.VolumeMode,
		AccessModes:        pvc.Spec.AccessModes,
		CreationTimestamp:  pvc.CreationTimestamp.Time,
	}

	p.logger.Info("成功获取 PersistentVolumeClaim 状态", zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.String("phase", string(pvc.Status.Phase)), zap.Int("cluster_id", id))
	return status, nil
}

// GetPVCBinding 获取 PVC 绑定状态
func (p *pvcService) GetPVCBinding(ctx context.Context, id int, namespace, pvcName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}

	volumeName := pvc.Spec.VolumeName
	p.logger.Info("成功获取 PersistentVolumeClaim 绑定状态", zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.String("volume_name", volumeName), zap.Int("cluster_id", id))
	return volumeName, nil
}

// GetPVCCapacityRequest 获取 PVC 容量请求信息
func (p *pvcService) GetPVCCapacityRequest(ctx context.Context, id int, namespace, pvcName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolumeClaim 失败", zap.Error(err), zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get PersistentVolumeClaim: %w", err)
	}

	requestedStorage := ""
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestedStorage = storage.String()
	}

	p.logger.Info("成功获取 PersistentVolumeClaim 容量请求信息", zap.String("pvc_name", pvcName), zap.String("namespace", namespace), zap.String("requested_storage", requestedStorage), zap.Int("cluster_id", id))
	return requestedStorage, nil
}