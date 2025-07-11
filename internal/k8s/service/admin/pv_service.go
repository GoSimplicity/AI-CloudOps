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

type PVService interface {
	GetPVs(ctx context.Context, id int) ([]*corev1.PersistentVolume, error)
	CreatePV(ctx context.Context, req *model.K8sPVRequest) error
	DeletePV(ctx context.Context, id int, pvName string) error
	BatchDeletePV(ctx context.Context, id int, pvNames []string) error
	GetPVYaml(ctx context.Context, id int, pvName string) (string, error)
	GetPVStatus(ctx context.Context, id int, pvName string) (*model.K8sPVStatus, error)
	GetPVCapacity(ctx context.Context, id int, pvName string) (map[corev1.ResourceName]string, error)
}

type pvService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewPVService 创建新的 PVService 实例
func NewPVService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) PVService {
	return &pvService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetPVs 获取所有 PersistentVolume
func (p *pvService) GetPVs(ctx context.Context, id int) ([]*corev1.PersistentVolume, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pvs, err := kubeClient.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolume 列表失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get PersistentVolume list: %w", err)
	}

	result := make([]*corev1.PersistentVolume, len(pvs.Items))
	for i := range pvs.Items {
		result[i] = &pvs.Items[i]
	}

	p.logger.Info("成功获取 PersistentVolume 列表", zap.Int("cluster_id", id), zap.Int("count", len(result)))
	return result, nil
}

// CreatePV 创建 PersistentVolume
func (p *pvService) CreatePV(ctx context.Context, req *model.K8sPVRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.CoreV1().PersistentVolumes().Create(ctx, req.PVYaml, metav1.CreateOptions{})
	if err != nil {
		p.logger.Error("创建 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", req.PVYaml.Name), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create PersistentVolume: %w", err)
	}

	p.logger.Info("成功创建 PersistentVolume", zap.String("pv_name", req.PVYaml.Name), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetPVYaml 获取指定 PersistentVolume 的 YAML 定义
func (p *pvService) GetPVYaml(ctx context.Context, id int, pvName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", pvName), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get PersistentVolume: %w", err)
	}

	yamlData, err := yaml.Marshal(pv)
	if err != nil {
		p.logger.Error("序列化 PersistentVolume YAML 失败", zap.Error(err), zap.String("pv_name", pvName))
		return "", fmt.Errorf("failed to serialize PersistentVolume YAML: %w", err)
	}

	p.logger.Info("成功获取 PersistentVolume YAML", zap.String("pv_name", pvName), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeletePV 批量删除 PersistentVolume
func (p *pvService) BatchDeletePV(ctx context.Context, id int, pvNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(pvNames))

	for _, name := range pvNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				p.logger.Error("删除 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", name), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete PersistentVolume '%s': %w", name, err)
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
		p.logger.Error("批量删除 PersistentVolume 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(pvNames)), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting PersistentVolumes: %v", errs)
	}

	p.logger.Info("成功批量删除 PersistentVolume", zap.Int("count", len(pvNames)), zap.Int("cluster_id", id))
	return nil
}

// DeletePV 删除指定的 PersistentVolume
func (p *pvService) DeletePV(ctx context.Context, id int, pvName string) error {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.CoreV1().PersistentVolumes().Delete(ctx, pvName, metav1.DeleteOptions{}); err != nil {
		p.logger.Error("删除 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", pvName), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete PersistentVolume '%s': %w", pvName, err)
	}

	p.logger.Info("成功删除 PersistentVolume", zap.String("pv_name", pvName), zap.Int("cluster_id", id))
	return nil
}

// GetPVStatus 获取 PersistentVolume 状态
func (p *pvService) GetPVStatus(ctx context.Context, id int, pvName string) (*model.K8sPVStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", pvName), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get PersistentVolume: %w", err)
	}

	// 转换容量映射
	capacity := make(map[corev1.ResourceName]string)
	for k, v := range pv.Spec.Capacity {
		capacity[k] = v.String()
	}

	status := &model.K8sPVStatus{
		Name:               pv.Name,
		Capacity:           capacity,
		Phase:              pv.Status.Phase,
		ClaimRef:           pv.Spec.ClaimRef,
		ReclaimPolicy:      pv.Spec.PersistentVolumeReclaimPolicy,
		StorageClass:       pv.Spec.StorageClassName,
		VolumeMode:         pv.Spec.VolumeMode,
		AccessModes:        pv.Spec.AccessModes,
		CreationTimestamp:  pv.CreationTimestamp.Time,
	}

	p.logger.Info("成功获取 PersistentVolume 状态", zap.String("pv_name", pvName), zap.String("phase", string(pv.Status.Phase)), zap.Int("cluster_id", id))
	return status, nil
}

// GetPVCapacity 获取 PersistentVolume 容量信息
func (p *pvService) GetPVCapacity(ctx context.Context, id int, pvName string) (map[corev1.ResourceName]string, error) {
	kubeClient, err := pkg.GetKubeClient(id, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	pv, err := kubeClient.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		p.logger.Error("获取 PersistentVolume 失败", zap.Error(err), zap.String("pv_name", pvName), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get PersistentVolume: %w", err)
	}

	capacity := make(map[corev1.ResourceName]string)
	for k, v := range pv.Spec.Capacity {
		capacity[k] = v.String()
	}

	p.logger.Info("成功获取 PersistentVolume 容量信息", zap.String("pv_name", pvName), zap.Int("cluster_id", id))
	return capacity, nil
}