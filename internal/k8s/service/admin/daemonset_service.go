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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSetService interface {
	GetDaemonSetsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.DaemonSet, error)
	CreateDaemonSet(ctx context.Context, req *model.K8sDaemonSetRequest) error
	UpdateDaemonSet(ctx context.Context, req *model.K8sDaemonSetRequest) error
	BatchDeleteDaemonSet(ctx context.Context, id int, namespace string, daemonSetNames []string) error
	DeleteDaemonSet(ctx context.Context, id int, namespace, daemonSetName string) error
	RestartDaemonSet(ctx context.Context, id int, namespace, daemonSetName string) error
	GetDaemonSetYaml(ctx context.Context, id int, namespace, daemonSetName string) (string, error)
	GetDaemonSetStatus(ctx context.Context, id int, namespace, daemonSetName string) (*model.K8sDaemonSetStatus, error)
}

type daemonSetService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewDaemonSetService 创建新的 DaemonSetService 实例
func NewDaemonSetService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) DaemonSetService {
	return &daemonSetService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetDaemonSetsByNamespace 获取指定命名空间下的所有 DaemonSet
func (d *daemonSetService) GetDaemonSetsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.DaemonSet, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	daemonSets, err := kubeClient.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get DaemonSet list: %w", err)
	}

	result := make([]*appsv1.DaemonSet, len(daemonSets.Items))
	for i := range daemonSets.Items {
		result[i] = &daemonSets.Items[i]
	}

	d.logger.Info("成功获取 DaemonSet 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateDaemonSet 创建 DaemonSet
func (d *daemonSetService) CreateDaemonSet(ctx context.Context, req *model.K8sDaemonSetRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.AppsV1().DaemonSets(req.Namespace).Create(ctx, req.DaemonSetYaml, metav1.CreateOptions{})
	if err != nil {
		d.logger.Error("创建 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", req.DaemonSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create DaemonSet: %w", err)
	}

	d.logger.Info("成功创建 DaemonSet", zap.String("daemonset_name", req.DaemonSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetDaemonSetYaml 获取指定 DaemonSet 的 YAML 定义
func (d *daemonSetService) GetDaemonSetYaml(ctx context.Context, id int, namespace, daemonSetName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get DaemonSet: %w", err)
	}

	yamlData, err := yaml.Marshal(daemonSet)
	if err != nil {
		d.logger.Error("序列化 DaemonSet YAML 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName))
		return "", fmt.Errorf("failed to serialize DaemonSet YAML: %w", err)
	}

	d.logger.Info("成功获取 DaemonSet YAML", zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// UpdateDaemonSet 更新 DaemonSet
func (d *daemonSetService) UpdateDaemonSet(ctx context.Context, req *model.K8sDaemonSetRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingDaemonSet, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Get(ctx, req.DaemonSetYaml.Name, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取现有 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", req.DaemonSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing DaemonSet: %w", err)
	}

	existingDaemonSet.Spec = req.DaemonSetYaml.Spec

	if _, err := kubeClient.AppsV1().DaemonSets(req.Namespace).Update(ctx, existingDaemonSet, metav1.UpdateOptions{}); err != nil {
		d.logger.Error("更新 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", req.DaemonSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update DaemonSet: %w", err)
	}

	d.logger.Info("成功更新 DaemonSet", zap.String("daemonset_name", req.DaemonSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// BatchDeleteDaemonSet 批量删除 DaemonSet
func (d *daemonSetService) BatchDeleteDaemonSet(ctx context.Context, id int, namespace string, daemonSetNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(daemonSetNames))

	for _, name := range daemonSetNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				d.logger.Error("删除 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete DaemonSet '%s': %w", name, err)
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
		d.logger.Error("批量删除 DaemonSet 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(daemonSetNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting DaemonSets: %v", errs)
	}

	d.logger.Info("成功批量删除 DaemonSet", zap.Int("count", len(daemonSetNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteDaemonSet 删除指定的 DaemonSet
func (d *daemonSetService) DeleteDaemonSet(ctx context.Context, id int, namespace, daemonSetName string) error {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.AppsV1().DaemonSets(namespace).Delete(ctx, daemonSetName, metav1.DeleteOptions{}); err != nil {
		d.logger.Error("删除 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete DaemonSet '%s': %w", daemonSetName, err)
	}

	d.logger.Info("成功删除 DaemonSet", zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// RestartDaemonSet 重启指定的 DaemonSet
func (d *daemonSetService) RestartDaemonSet(ctx context.Context, id int, namespace, daemonSetName string) error {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get DaemonSet '%s': %w", daemonSetName, err)
	}

	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}

	daemonSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	if _, err := kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{}); err != nil {
		d.logger.Error("重启 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to update DaemonSet '%s': %w", daemonSetName, err)
	}

	d.logger.Info("成功重启 DaemonSet", zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetDaemonSetStatus 获取 DaemonSet 状态
func (d *daemonSetService) GetDaemonSetStatus(ctx context.Context, id int, namespace, daemonSetName string) (*model.K8sDaemonSetStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, d.client, d.logger)
	if err != nil {
		d.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	daemonSet, err := kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonSetName, metav1.GetOptions{})
	if err != nil {
		d.logger.Error("获取 DaemonSet 失败", zap.Error(err), zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get DaemonSet: %w", err)
	}

	status := &model.K8sDaemonSetStatus{
		Name:                     daemonSet.Name,
		Namespace:                daemonSet.Namespace,
		DesiredNumberScheduled:   daemonSet.Status.DesiredNumberScheduled,
		CurrentNumberScheduled:   daemonSet.Status.CurrentNumberScheduled,
		NumberReady:              daemonSet.Status.NumberReady,
		UpdatedNumberScheduled:   daemonSet.Status.UpdatedNumberScheduled,
		NumberAvailable:          daemonSet.Status.NumberAvailable,
		NumberUnavailable:        daemonSet.Status.NumberUnavailable,
		NumberMisscheduled:       daemonSet.Status.NumberMisscheduled,
		ObservedGeneration:       daemonSet.Status.ObservedGeneration,
		CreationTimestamp:        daemonSet.CreationTimestamp.Time,
	}

	d.logger.Info("成功获取 DaemonSet 状态", zap.String("daemonset_name", daemonSetName), zap.String("namespace", namespace), zap.Int32("ready_replicas", daemonSet.Status.NumberReady), zap.Int("cluster_id", id))
	return status, nil
}