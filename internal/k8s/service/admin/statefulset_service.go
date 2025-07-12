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

type StatefulSetService interface {
	GetStatefulSetsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.StatefulSet, error)
	CreateStatefulSet(ctx context.Context, req *model.K8sStatefulSetRequest) error
	UpdateStatefulSet(ctx context.Context, req *model.K8sStatefulSetRequest) error
	BatchDeleteStatefulSet(ctx context.Context, id int, namespace string, statefulSetNames []string) error
	DeleteStatefulSet(ctx context.Context, id int, namespace, statefulSetName string) error
	RestartStatefulSet(ctx context.Context, id int, namespace, statefulSetName string) error
	ScaleStatefulSet(ctx context.Context, req *model.K8sStatefulSetScaleRequest) error
	GetStatefulSetYaml(ctx context.Context, id int, namespace, statefulSetName string) (string, error)
	GetStatefulSetStatus(ctx context.Context, id int, namespace, statefulSetName string) (*model.K8sStatefulSetStatus, error)
}

type statefulSetService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewStatefulSetService 创建新的 StatefulSetService 实例
func NewStatefulSetService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) StatefulSetService {
	return &statefulSetService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetStatefulSetsByNamespace 获取指定命名空间下的所有 StatefulSet
func (s *statefulSetService) GetStatefulSetsByNamespace(ctx context.Context, id int, namespace string) ([]*appsv1.StatefulSet, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	statefulSets, err := kubeClient.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取 StatefulSet 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get StatefulSet list: %w", err)
	}

	result := make([]*appsv1.StatefulSet, len(statefulSets.Items))
	for i := range statefulSets.Items {
		result[i] = &statefulSets.Items[i]
	}

	s.logger.Info("成功获取 StatefulSet 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateStatefulSet 创建 StatefulSet
func (s *statefulSetService) CreateStatefulSet(ctx context.Context, req *model.K8sStatefulSetRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.AppsV1().StatefulSets(req.Namespace).Create(ctx, req.StatefulSetYaml, metav1.CreateOptions{})
	if err != nil {
		s.logger.Error("创建 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", req.StatefulSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create StatefulSet: %w", err)
	}

	s.logger.Info("成功创建 StatefulSet", zap.String("statefulset_name", req.StatefulSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetStatefulSetYaml 获取指定 StatefulSet 的 YAML 定义
func (s *statefulSetService) GetStatefulSetYaml(ctx context.Context, id int, namespace, statefulSetName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	yamlData, err := yaml.Marshal(statefulSet)
	if err != nil {
		s.logger.Error("序列化 StatefulSet YAML 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName))
		return "", fmt.Errorf("failed to serialize StatefulSet YAML: %w", err)
	}

	s.logger.Info("成功获取 StatefulSet YAML", zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// UpdateStatefulSet 更新 StatefulSet
func (s *statefulSetService) UpdateStatefulSet(ctx context.Context, req *model.K8sStatefulSetRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingStatefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.StatefulSetYaml.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取现有 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", req.StatefulSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing StatefulSet: %w", err)
	}

	existingStatefulSet.Spec = req.StatefulSetYaml.Spec

	if _, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, existingStatefulSet, metav1.UpdateOptions{}); err != nil {
		s.logger.Error("更新 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", req.StatefulSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update StatefulSet: %w", err)
	}

	s.logger.Info("成功更新 StatefulSet", zap.String("statefulset_name", req.StatefulSetYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// BatchDeleteStatefulSet 批量删除 StatefulSet
func (s *statefulSetService) BatchDeleteStatefulSet(ctx context.Context, id int, namespace string, statefulSetNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(statefulSetNames))

	for _, name := range statefulSetNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				s.logger.Error("删除 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete StatefulSet '%s': %w", name, err)
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
		s.logger.Error("批量删除 StatefulSet 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(statefulSetNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting StatefulSets: %v", errs)
	}

	s.logger.Info("成功批量删除 StatefulSet", zap.Int("count", len(statefulSetNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteStatefulSet 删除指定的 StatefulSet
func (s *statefulSetService) DeleteStatefulSet(ctx context.Context, id int, namespace, statefulSetName string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.AppsV1().StatefulSets(namespace).Delete(ctx, statefulSetName, metav1.DeleteOptions{}); err != nil {
		s.logger.Error("删除 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete StatefulSet '%s': %w", statefulSetName, err)
	}

	s.logger.Info("成功删除 StatefulSet", zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// RestartStatefulSet 重启指定的 StatefulSet
func (s *statefulSetService) RestartStatefulSet(ctx context.Context, id int, namespace, statefulSetName string) error {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get StatefulSet '%s': %w", statefulSetName, err)
	}

	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = make(map[string]string)
	}

	statefulSet.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	if _, err := kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
		s.logger.Error("重启 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to update StatefulSet '%s': %w", statefulSetName, err)
	}

	s.logger.Info("成功重启 StatefulSet", zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// ScaleStatefulSet 扩缩容 StatefulSet
func (s *statefulSetService) ScaleStatefulSet(ctx context.Context, req *model.K8sStatefulSetScaleRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Get(ctx, req.StatefulSetName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", req.StatefulSetName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get StatefulSet '%s': %w", req.StatefulSetName, err)
	}

	statefulSet.Spec.Replicas = &req.Replicas

	if _, err := kubeClient.AppsV1().StatefulSets(req.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{}); err != nil {
		s.logger.Error("扩缩容 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", req.StatefulSetName), zap.String("namespace", req.Namespace), zap.Int32("replicas", req.Replicas), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to scale StatefulSet '%s': %w", req.StatefulSetName, err)
	}

	s.logger.Info("成功扩缩容 StatefulSet", zap.String("statefulset_name", req.StatefulSetName), zap.String("namespace", req.Namespace), zap.Int32("replicas", req.Replicas), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetStatefulSetStatus 获取 StatefulSet 状态
func (s *statefulSetService) GetStatefulSetStatus(ctx context.Context, id int, namespace, statefulSetName string) (*model.K8sStatefulSetStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, s.client, s.logger)
	if err != nil {
		s.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	statefulSet, err := kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, statefulSetName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("获取 StatefulSet 失败", zap.Error(err), zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get StatefulSet: %w", err)
	}

	status := &model.K8sStatefulSetStatus{
		Name:               statefulSet.Name,
		Namespace:          statefulSet.Namespace,
		Replicas:           *statefulSet.Spec.Replicas,
		ReadyReplicas:      statefulSet.Status.ReadyReplicas,
		CurrentReplicas:    statefulSet.Status.CurrentReplicas,
		UpdatedReplicas:    statefulSet.Status.UpdatedReplicas,
		AvailableReplicas:  statefulSet.Status.AvailableReplicas,
		CurrentRevision:    statefulSet.Status.CurrentRevision,
		UpdateRevision:     statefulSet.Status.UpdateRevision,
		ObservedGeneration: statefulSet.Status.ObservedGeneration,
		CreationTimestamp:  statefulSet.CreationTimestamp.Time,
	}

	s.logger.Info("成功获取 StatefulSet 状态", zap.String("statefulset_name", statefulSetName), zap.String("namespace", namespace), zap.Int32("ready_replicas", statefulSet.Status.ReadyReplicas), zap.Int32("current_replicas", statefulSet.Status.CurrentReplicas), zap.Int("cluster_id", id))
	return status, nil
}