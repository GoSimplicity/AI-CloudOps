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
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type LabelService interface {
	AddResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error)
	UpdateResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error)
	DeleteResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error)
	GetResourceLabels(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sLabelResponse, error)
	ListResourcesByLabels(ctx context.Context, req *model.K8sLabelSelectorRequest) ([]*model.K8sLabelResponse, error)
	BatchUpdateLabels(ctx context.Context, req *model.K8sLabelBatchRequest) ([]*model.K8sLabelResponse, error)
	CreateLabelPolicy(ctx context.Context, req *model.K8sLabelPolicyRequest) (*model.K8sLabelPolicyRequest, error)
	UpdateLabelPolicy(ctx context.Context, req *model.K8sLabelPolicyRequest) (*model.K8sLabelPolicyRequest, error)
	DeleteLabelPolicy(ctx context.Context, clusterID int, policyName string) error
	GetLabelPolicy(ctx context.Context, clusterID int, policyName string) (*model.K8sLabelPolicyRequest, error)
	ListLabelPolicies(ctx context.Context, clusterID int, namespace string) ([]*model.K8sLabelPolicyRequest, error)
	CheckLabelCompliance(ctx context.Context, req *model.K8sLabelComplianceRequest) ([]*model.K8sLabelComplianceResponse, error)
	GetLabelHistory(ctx context.Context, req *model.K8sLabelHistoryRequest) ([]*model.K8sLabelHistoryResponse, error)
}

type labelService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewLabelService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) LabelService {
	return &labelService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// AddResourceLabels 添加资源标签
func (l *labelService) AddResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error) {
	l.logger.Info("开始添加资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := l.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		l.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 添加标签
	if err := l.addLabelsToObject(obj, req.Labels); err != nil {
		l.logger.Error("添加标签失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to add labels: %w", err)
	}

	// 更新资源对象
	if err := l.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		l.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sLabelResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Labels:            req.Labels,
		CreationTimestamp: time.Now(),
	}

	l.logger.Info("成功添加资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// UpdateResourceLabels 更新资源标签
func (l *labelService) UpdateResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error) {
	l.logger.Info("开始更新资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := l.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		l.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 更新标签
	if err := l.updateLabelsOnObject(obj, req.Labels); err != nil {
		l.logger.Error("更新标签失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update labels: %w", err)
	}

	// 更新资源对象
	if err := l.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		l.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sLabelResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Labels:            req.Labels,
		CreationTimestamp: time.Now(),
	}

	l.logger.Info("成功更新资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// DeleteResourceLabels 删除资源标签
func (l *labelService) DeleteResourceLabels(ctx context.Context, req *model.K8sLabelRequest) (*model.K8sLabelResponse, error) {
	l.logger.Info("开始删除资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := l.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		l.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 删除标签
	if err := l.deleteLabelsFromObject(obj, req.Labels); err != nil {
		l.logger.Error("删除标签失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to delete labels: %w", err)
	}

	// 更新资源对象
	if err := l.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		l.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sLabelResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		Labels:            l.getLabelsFromObject(obj),
		CreationTimestamp: time.Now(),
	}

	l.logger.Info("成功删除资源标签", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// GetResourceLabels 获取资源标签
func (l *labelService) GetResourceLabels(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sLabelResponse, error) {
	l.logger.Info("开始获取资源标签", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := l.getResourceObject(ctx, kubeClient, resourceType, namespace, resourceName)
	if err != nil {
		l.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", resourceType), zap.String("resource_name", resourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	response := &model.K8sLabelResponse{
		ResourceType:      resourceType,
		ResourceName:      resourceName,
		Namespace:         namespace,
		Labels:            l.getLabelsFromObject(obj),
		Annotations:       l.getAnnotationsFromObject(obj),
		CreationTimestamp: l.getCreationTimestamp(obj),
	}

	l.logger.Info("成功获取资源标签", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return response, nil
}

// ListResourcesByLabels 根据标签选择器查询资源
func (l *labelService) ListResourcesByLabels(ctx context.Context, req *model.K8sLabelSelectorRequest) ([]*model.K8sLabelResponse, error) {
	l.logger.Info("开始根据标签选择器查询资源", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 构建标签选择器
	labelSelector := metav1.FormatLabelSelector(&metav1.LabelSelector{
		MatchLabels: req.LabelSelector,
	})

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
		FieldSelector: req.FieldSelector,
	}

	if req.Limit > 0 {
		listOptions.Limit = int64(req.Limit)
	}

	// 根据资源类型查询资源
	resources, err := l.listResourcesByType(ctx, kubeClient, req.ResourceType, req.Namespace, listOptions)
	if err != nil {
		l.logger.Error("查询资源失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace))
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	l.logger.Info("成功根据标签选择器查询资源", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Int("count", len(resources)))
	return resources, nil
}

// BatchUpdateLabels 批量更新标签
func (l *labelService) BatchUpdateLabels(ctx context.Context, req *model.K8sLabelBatchRequest) ([]*model.K8sLabelResponse, error) {
	l.logger.Info("开始批量更新标签", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Strings("resource_names", req.ResourceNames))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, l.client, l.logger)
	if err != nil {
		l.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	resultChan := make(chan *model.K8sLabelResponse, len(req.ResourceNames))
	errChan := make(chan error, len(req.ResourceNames))

	// 并发处理每个资源
	for _, resourceName := range req.ResourceNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			obj, err := l.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, name)
			if err != nil {
				l.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_name", name))
				errChan <- fmt.Errorf("failed to get resource %s: %w", name, err)
				return
			}

			// 根据操作类型更新标签
			switch req.Operation {
			case "add":
				if err := l.addLabelsToObject(obj, req.Labels); err != nil {
					errChan <- fmt.Errorf("failed to add labels to %s: %w", name, err)
					return
				}
			case "update":
				if err := l.updateLabelsOnObject(obj, req.Labels); err != nil {
					errChan <- fmt.Errorf("failed to update labels on %s: %w", name, err)
					return
				}
			case "delete":
				if err := l.deleteLabelsFromObject(obj, req.Labels); err != nil {
					errChan <- fmt.Errorf("failed to delete labels from %s: %w", name, err)
					return
				}
			default:
				errChan <- fmt.Errorf("unsupported operation: %s", req.Operation)
				return
			}

			// 更新资源对象
			if err := l.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
				errChan <- fmt.Errorf("failed to update resource %s: %w", name, err)
				return
			}

			resultChan <- &model.K8sLabelResponse{
				ResourceType:      req.ResourceType,
				ResourceName:      name,
				Namespace:         req.Namespace,
				Labels:            l.getLabelsFromObject(obj),
				CreationTimestamp: time.Now(),
			}
		}(resourceName)
	}

	wg.Wait()
	close(resultChan)
	close(errChan)

	// 收集错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	// 收集结果
	var results []*model.K8sLabelResponse
	for result := range resultChan {
		results = append(results, result)
	}

	if len(errs) > 0 {
		l.logger.Error("批量更新标签部分失败", zap.Int("error_count", len(errs)), zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace))
		return results, fmt.Errorf("batch update partially failed: %v", errs)
	}

	l.logger.Info("成功批量更新标签", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Int("count", len(results)))
	return results, nil
}

// CreateLabelPolicy 创建标签策略（模拟实现）
func (l *labelService) CreateLabelPolicy(ctx context.Context, req *model.K8sLabelPolicyRequest) (*model.K8sLabelPolicyRequest, error) {
	l.logger.Info("开始创建标签策略", zap.String("policy_name", req.PolicyName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	// 模拟存储策略到数据库或配置中心
	// 这里可以根据实际需求实现持久化存储

	l.logger.Info("成功创建标签策略", zap.String("policy_name", req.PolicyName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return req, nil
}

// UpdateLabelPolicy 更新标签策略（模拟实现）
func (l *labelService) UpdateLabelPolicy(ctx context.Context, req *model.K8sLabelPolicyRequest) (*model.K8sLabelPolicyRequest, error) {
	l.logger.Info("开始更新标签策略", zap.String("policy_name", req.PolicyName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	// 模拟更新策略

	l.logger.Info("成功更新标签策略", zap.String("policy_name", req.PolicyName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return req, nil
}

// DeleteLabelPolicy 删除标签策略（模拟实现）
func (l *labelService) DeleteLabelPolicy(ctx context.Context, clusterID int, policyName string) error {
	l.logger.Info("开始删除标签策略", zap.String("policy_name", policyName), zap.Int("cluster_id", clusterID))

	// 模拟删除策略

	l.logger.Info("成功删除标签策略", zap.String("policy_name", policyName), zap.Int("cluster_id", clusterID))
	return nil
}

// GetLabelPolicy 获取标签策略（模拟实现）
func (l *labelService) GetLabelPolicy(ctx context.Context, clusterID int, policyName string) (*model.K8sLabelPolicyRequest, error) {
	l.logger.Info("开始获取标签策略", zap.String("policy_name", policyName), zap.Int("cluster_id", clusterID))

	// 模拟获取策略
	policy := &model.K8sLabelPolicyRequest{
		ClusterID:   clusterID,
		PolicyName:  policyName,
		PolicyType:  "required",
		Enabled:     true,
		Description: "模拟标签策略",
	}

	l.logger.Info("成功获取标签策略", zap.String("policy_name", policyName), zap.Int("cluster_id", clusterID))
	return policy, nil
}

// ListLabelPolicies 获取标签策略列表（模拟实现）
func (l *labelService) ListLabelPolicies(ctx context.Context, clusterID int, namespace string) ([]*model.K8sLabelPolicyRequest, error) {
	l.logger.Info("开始获取标签策略列表", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	// 模拟获取策略列表
	policies := []*model.K8sLabelPolicyRequest{
		{
			ClusterID:   clusterID,
			Namespace:   namespace,
			PolicyName:  "default-policy",
			PolicyType:  "required",
			Enabled:     true,
			Description: "默认标签策略",
		},
	}

	l.logger.Info("成功获取标签策略列表", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID), zap.Int("count", len(policies)))
	return policies, nil
}

// CheckLabelCompliance 检查标签合规性（模拟实现）
func (l *labelService) CheckLabelCompliance(ctx context.Context, req *model.K8sLabelComplianceRequest) ([]*model.K8sLabelComplianceResponse, error) {
	l.logger.Info("开始检查标签合规性", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	// 模拟合规性检查
	responses := []*model.K8sLabelComplianceResponse{
		{
			ResourceType: req.ResourceType,
			ResourceName: "example-resource",
			Namespace:    req.Namespace,
			PolicyName:   req.PolicyName,
			Compliant:    true,
			CheckTime:    time.Now(),
		},
	}

	l.logger.Info("成功检查标签合规性", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Int("count", len(responses)))
	return responses, nil
}

// GetLabelHistory 获取标签历史记录（模拟实现）
func (l *labelService) GetLabelHistory(ctx context.Context, req *model.K8sLabelHistoryRequest) ([]*model.K8sLabelHistoryResponse, error) {
	l.logger.Info("开始获取标签历史记录", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	// 模拟获取历史记录
	histories := []*model.K8sLabelHistoryResponse{
		{
			ID:           1,
			ClusterID:    req.ClusterID,
			Namespace:    req.Namespace,
			ResourceType: req.ResourceType,
			ResourceName: req.ResourceName,
			Operation:    "add",
			OldLabels:    map[string]string{},
			NewLabels:    map[string]string{"app": "test"},
			ChangedBy:    "system",
			ChangeTime:   time.Now(),
			ChangeReason: "标签添加",
		},
	}

	l.logger.Info("成功获取标签历史记录", zap.String("resource_type", req.ResourceType), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Int("count", len(histories)))
	return histories, nil
}

// 辅助方法：获取资源对象
func (l *labelService) getResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace, name string) (runtime.Object, error) {
	switch strings.ToLower(resourceType) {
	case "pod":
		return kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		return kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "service":
		return kubeClient.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	case "configmap":
		return kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	case "secret":
		return kubeClient.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "daemonset":
		return kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset":
		return kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "job":
		return kubeClient.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
	case "ingress":
		return kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	case "persistentvolume":
		return kubeClient.CoreV1().PersistentVolumes().Get(ctx, name, metav1.GetOptions{})
	case "persistentvolumeclaim":
		return kubeClient.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	case "storageclass":
		return kubeClient.StorageV1().StorageClasses().Get(ctx, name, metav1.GetOptions{})
	case "networkpolicy":
		return kubeClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
	case "node":
		return kubeClient.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	case "namespace":
		return kubeClient.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// 辅助方法：根据资源类型列出资源
func (l *labelService) listResourcesByType(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace string, listOptions metav1.ListOptions) ([]*model.K8sLabelResponse, error) {
	var responses []*model.K8sLabelResponse

	switch strings.ToLower(resourceType) {
	case "pod":
		pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, pod := range pods.Items {
			responses = append(responses, &model.K8sLabelResponse{
				ResourceType:      resourceType,
				ResourceName:      pod.Name,
				Namespace:         pod.Namespace,
				Labels:            pod.Labels,
				Annotations:       pod.Annotations,
				CreationTimestamp: pod.CreationTimestamp.Time,
			})
		}
	case "deployment":
		deployments, err := kubeClient.AppsV1().Deployments(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, deployment := range deployments.Items {
			responses = append(responses, &model.K8sLabelResponse{
				ResourceType:      resourceType,
				ResourceName:      deployment.Name,
				Namespace:         deployment.Namespace,
				Labels:            deployment.Labels,
				Annotations:       deployment.Annotations,
				CreationTimestamp: deployment.CreationTimestamp.Time,
			})
		}
	case "service":
		services, err := kubeClient.CoreV1().Services(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, service := range services.Items {
			responses = append(responses, &model.K8sLabelResponse{
				ResourceType:      resourceType,
				ResourceName:      service.Name,
				Namespace:         service.Namespace,
				Labels:            service.Labels,
				Annotations:       service.Annotations,
				CreationTimestamp: service.CreationTimestamp.Time,
			})
		}
	default:
		return nil, fmt.Errorf("unsupported resource type for listing: %s", resourceType)
	}

	return responses, nil
}

// 辅助方法：更新资源对象
func (l *labelService) updateResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace string, obj runtime.Object) error {
	switch strings.ToLower(resourceType) {
	case "pod":
		pod := obj.(*core.Pod)
		_, err := kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
		return err
	case "deployment":
		deployment := obj.(*appsv1.Deployment)
		_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "service":
		service := obj.(*core.Service)
		_, err := kubeClient.CoreV1().Services(namespace).Update(ctx, service, metav1.UpdateOptions{})
		return err
	case "configmap":
		configMap := obj.(*core.ConfigMap)
		_, err := kubeClient.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
		return err
	case "secret":
		secret := obj.(*core.Secret)
		_, err := kubeClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
		return err
	case "daemonset":
		daemonSet := obj.(*appsv1.DaemonSet)
		_, err := kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
		return err
	case "statefulset":
		statefulSet := obj.(*appsv1.StatefulSet)
		_, err := kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
		return err
	case "job":
		job := obj.(*batchv1.Job)
		_, err := kubeClient.BatchV1().Jobs(namespace).Update(ctx, job, metav1.UpdateOptions{})
		return err
	case "ingress":
		ingress := obj.(*networkingv1.Ingress)
		_, err := kubeClient.NetworkingV1().Ingresses(namespace).Update(ctx, ingress, metav1.UpdateOptions{})
		return err
	case "persistentvolume":
		pv := obj.(*core.PersistentVolume)
		_, err := kubeClient.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
		return err
	case "persistentvolumeclaim":
		pvc := obj.(*core.PersistentVolumeClaim)
		_, err := kubeClient.CoreV1().PersistentVolumeClaims(namespace).Update(ctx, pvc, metav1.UpdateOptions{})
		return err
	case "storageclass":
		sc := obj.(*storagev1.StorageClass)
		_, err := kubeClient.StorageV1().StorageClasses().Update(ctx, sc, metav1.UpdateOptions{})
		return err
	case "networkpolicy":
		np := obj.(*networkingv1.NetworkPolicy)
		_, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Update(ctx, np, metav1.UpdateOptions{})
		return err
	case "node":
		node := obj.(*core.Node)
		_, err := kubeClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		return err
	case "namespace":
		ns := obj.(*core.Namespace)
		_, err := kubeClient.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unsupported resource type for update: %s", resourceType)
	}
}

// 辅助方法：添加标签到对象
func (l *labelService) addLabelsToObject(obj runtime.Object, labels map[string]string) error {
	switch o := obj.(type) {
	case *core.Pod:
		if o.Labels == nil {
			o.Labels = make(map[string]string)
		}
		for k, v := range labels {
			o.Labels[k] = v
		}
	case *appsv1.Deployment:
		if o.Labels == nil {
			o.Labels = make(map[string]string)
		}
		for k, v := range labels {
			o.Labels[k] = v
		}
	case *core.Service:
		if o.Labels == nil {
			o.Labels = make(map[string]string)
		}
		for k, v := range labels {
			o.Labels[k] = v
		}
	case *core.ConfigMap:
		if o.Labels == nil {
			o.Labels = make(map[string]string)
		}
		for k, v := range labels {
			o.Labels[k] = v
		}
	case *core.Secret:
		if o.Labels == nil {
			o.Labels = make(map[string]string)
		}
		for k, v := range labels {
			o.Labels[k] = v
		}
	default:
		return fmt.Errorf("unsupported object type for adding labels")
	}
	return nil
}

// 辅助方法：更新对象上的标签
func (l *labelService) updateLabelsOnObject(obj runtime.Object, labels map[string]string) error {
	return l.addLabelsToObject(obj, labels)
}

// 辅助方法：从对象删除标签
func (l *labelService) deleteLabelsFromObject(obj runtime.Object, labels map[string]string) error {
	switch o := obj.(type) {
	case *core.Pod:
		if o.Labels != nil {
			for k := range labels {
				delete(o.Labels, k)
			}
		}
	case *appsv1.Deployment:
		if o.Labels != nil {
			for k := range labels {
				delete(o.Labels, k)
			}
		}
	case *core.Service:
		if o.Labels != nil {
			for k := range labels {
				delete(o.Labels, k)
			}
		}
	case *core.ConfigMap:
		if o.Labels != nil {
			for k := range labels {
				delete(o.Labels, k)
			}
		}
	case *core.Secret:
		if o.Labels != nil {
			for k := range labels {
				delete(o.Labels, k)
			}
		}
	default:
		return fmt.Errorf("unsupported object type for deleting labels")
	}
	return nil
}

// 辅助方法：从对象获取标签
func (l *labelService) getLabelsFromObject(obj runtime.Object) map[string]string {
	switch o := obj.(type) {
	case *core.Pod:
		return o.Labels
	case *appsv1.Deployment:
		return o.Labels
	case *core.Service:
		return o.Labels
	case *core.ConfigMap:
		return o.Labels
	case *core.Secret:
		return o.Labels
	default:
		return nil
	}
}

// 辅助方法：从对象获取注解
func (l *labelService) getAnnotationsFromObject(obj runtime.Object) map[string]string {
	switch o := obj.(type) {
	case *core.Pod:
		return o.Annotations
	case *appsv1.Deployment:
		return o.Annotations
	case *core.Service:
		return o.Annotations
	case *core.ConfigMap:
		return o.Annotations
	case *core.Secret:
		return o.Annotations
	default:
		return nil
	}
}

// 辅助方法：从对象获取创建时间
func (l *labelService) getCreationTimestamp(obj runtime.Object) time.Time {
	switch o := obj.(type) {
	case *core.Pod:
		return o.CreationTimestamp.Time
	case *appsv1.Deployment:
		return o.CreationTimestamp.Time
	case *core.Service:
		return o.CreationTimestamp.Time
	case *core.ConfigMap:
		return o.CreationTimestamp.Time
	case *core.Secret:
		return o.CreationTimestamp.Time
	default:
		return time.Time{}
	}
}
