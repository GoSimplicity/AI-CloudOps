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
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

// NodeAffinityService 节点亲和性服务接口
type NodeAffinityService interface {
	SetNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error)
	GetNodeAffinity(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sNodeAffinityResponse, error)
	UpdateNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error)
	DeleteNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error)
	ValidateNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityValidationRequest) (*model.K8sNodeAffinityValidationResponse, error)
	GetNodeAffinityRecommendations(ctx context.Context, clusterID int, namespace, resourceType string) ([]string, error)
}

// PodAffinityService Pod亲和性服务接口
type PodAffinityService interface {
	SetPodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error)
	GetPodAffinity(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sPodAffinityResponse, error)
	UpdatePodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error)
	DeletePodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error)
	ValidatePodAffinity(ctx context.Context, req *model.K8sPodAffinityValidationRequest) (*model.K8sPodAffinityValidationResponse, error)
	GetTopologyDomains(ctx context.Context, clusterID int, namespace string) ([]string, error)
}

// AffinityVisualizationService 亲和性可视化服务接口
type AffinityVisualizationService interface {
	GetAffinityVisualization(ctx context.Context, req *model.K8sAffinityVisualizationRequest) (*model.K8sAffinityVisualizationResponse, error)
}

// nodeAffinityService 节点亲和性服务实现
type nodeAffinityService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewNodeAffinityService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) NodeAffinityService {
	return &nodeAffinityService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// SetNodeAffinity 设置节点亲和性
func (n *nodeAffinityService) SetNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error) {
	n.logger.Info("开始设置节点亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := n.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		n.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 设置节点亲和性
	if err := n.setNodeAffinityOnObject(obj, req); err != nil {
		n.logger.Error("设置节点亲和性失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to set node affinity: %w", err)
	}

	// 更新资源对象
	if err := n.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		n.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sNodeAffinityResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		RequiredAffinity:  req.RequiredAffinity,
		PreferredAffinity: req.PreferredAffinity,
		NodeSelector:      req.NodeSelector,
		CreationTimestamp: time.Now(),
	}

	n.logger.Info("成功设置节点亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// GetNodeAffinity 获取节点亲和性
func (n *nodeAffinityService) GetNodeAffinity(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sNodeAffinityResponse, error) {
	n.logger.Info("开始获取节点亲和性", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := n.getResourceObject(ctx, kubeClient, resourceType, namespace, resourceName)
	if err != nil {
		n.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", resourceType), zap.String("resource_name", resourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 提取节点亲和性信息
	nodeAffinity := n.extractNodeAffinityFromObject(obj)

	response := &model.K8sNodeAffinityResponse{
		ResourceType:      resourceType,
		ResourceName:      resourceName,
		Namespace:         namespace,
		RequiredAffinity:  nodeAffinity.RequiredAffinity,
		PreferredAffinity: nodeAffinity.PreferredAffinity,
		NodeSelector:      nodeAffinity.NodeSelector,
		CreationTimestamp: time.Now(),
	}

	n.logger.Info("成功获取节点亲和性", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return response, nil
}

// UpdateNodeAffinity 更新节点亲和性
func (n *nodeAffinityService) UpdateNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error) {
	n.logger.Info("开始更新节点亲和性", zap.String(
		"resource_type", req.ResourceType),
		zap.String("resource_name", req.ResourceName),
		zap.String("namespace", req.Namespace),
		zap.Int("cluster_id", req.ClusterID),
	)

	// 更新操作与设置操作相同
	return n.SetNodeAffinity(ctx, req)
}

// DeleteNodeAffinity 删除节点亲和性
func (n *nodeAffinityService) DeleteNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityRequest) (*model.K8sNodeAffinityResponse, error) {
	n.logger.Info("开始删除节点亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := n.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		n.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 删除节点亲和性
	if err := n.removeNodeAffinityFromObject(obj); err != nil {
		n.logger.Error("删除节点亲和性失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to remove node affinity: %w", err)
	}

	// 更新资源对象
	if err := n.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		n.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sNodeAffinityResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		CreationTimestamp: time.Now(),
	}

	n.logger.Info("成功删除节点亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// ValidateNodeAffinity 验证节点亲和性
func (n *nodeAffinityService) ValidateNodeAffinity(ctx context.Context, req *model.K8sNodeAffinityValidationRequest) (*model.K8sNodeAffinityValidationResponse, error) {
	n.logger.Info("开始验证节点亲和性", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取所有节点
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取节点列表失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 验证节点亲和性
	var matchingNodes []string
	var validationErrors []string
	var suggestions []string

	// 检查硬亲和性
	for _, node := range nodes.Items {
		if n.nodeMatchesAffinity(&node, req.RequiredAffinity, req.NodeSelector) {
			matchingNodes = append(matchingNodes, node.Name)
		}
	}

	// 验证亲和性规则
	if len(req.RequiredAffinity) > 0 {
		for _, term := range req.RequiredAffinity {
			if err := n.validateNodeSelectorTerm(term); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("硬亲和性规则验证失败: %s", err.Error()))
			}
		}
	}

	// 验证软亲和性
	if len(req.PreferredAffinity) > 0 {
		for _, term := range req.PreferredAffinity {
			if term.Weight < 1 || term.Weight > 100 {
				validationErrors = append(validationErrors, fmt.Sprintf("软亲和性权重必须在1-100范围内, 当前值: %d", term.Weight))
			}
			if err := n.validateNodeSelectorTerm(term.Preference); err != nil {
				validationErrors = append(validationErrors, fmt.Sprintf("软亲和性规则验证失败: %s", err.Error()))
			}
		}
	}

	valid := len(matchingNodes) > 0 && len(validationErrors) == 0
	if !valid {
		if len(matchingNodes) == 0 {
			validationErrors = append(validationErrors, "没有找到匹配的节点")
			suggestions = append(suggestions, "请检查节点标签和亲和性规则")
		}
		if len(validationErrors) > 0 {
			suggestions = append(suggestions, "请修正验证错误后重试")
		}
	}

	scheduleResult := "验证完成"
	if len(matchingNodes) > 0 {
		scheduleResult = fmt.Sprintf("找到%d个匹配节点: %s", len(matchingNodes), strings.Join(matchingNodes, ", "))
	} else {
		scheduleResult = "没有找到匹配的节点，Pod将无法调度"
	}

	response := &model.K8sNodeAffinityValidationResponse{
		Valid:            valid,
		MatchingNodes:    matchingNodes,
		ValidationErrors: validationErrors,
		Suggestions:      suggestions,
		SchedulingResult: scheduleResult,
		ValidationTime:   time.Now(),
	}

	n.logger.Info("成功验证节点亲和性", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Bool("valid", valid))
	return response, nil
}

// GetNodeAffinityRecommendations 获取节点亲和性建议
func (n *nodeAffinityService) GetNodeAffinityRecommendations(ctx context.Context, clusterID int, namespace, resourceType string) ([]string, error) {
	n.logger.Info("开始获取节点亲和性建议", zap.String("resource_type", resourceType), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	// 模拟生成建议
	recommendations := []string{
		"建议使用 kubernetes.io/arch=amd64 标签选择器",
		"建议使用 node.kubernetes.io/instance-type 标签进行节点选择",
		"建议配置软亲和性以提高调度灵活性",
		"建议使用拓扑域进行节点分布",
	}

	n.logger.Info("成功获取节点亲和性建议", zap.String("resource_type", resourceType), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID), zap.Int("count", len(recommendations)))
	return recommendations, nil
}

// 辅助方法：获取资源对象
func (n *nodeAffinityService) getResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace, name string) (runtime.Object, error) {
	switch strings.ToLower(resourceType) {
	case "pod":
		return kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		return kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "replicaset":
		return kubeClient.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset":
		return kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "daemonset":
		return kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// 辅助方法：更新资源对象
func (n *nodeAffinityService) updateResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace string, obj runtime.Object) error {
	switch strings.ToLower(resourceType) {
	case "pod":
		pod := obj.(*core.Pod)
		_, err := kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
		return err
	case "deployment":
		deployment := obj.(*appsv1.Deployment)
		_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "replicaset":
		replicaset := obj.(*appsv1.ReplicaSet)
		_, err := kubeClient.AppsV1().ReplicaSets(namespace).Update(ctx, replicaset, metav1.UpdateOptions{})
		return err
	case "statefulset":
		statefulset := obj.(*appsv1.StatefulSet)
		_, err := kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
		return err
	case "daemonset":
		daemonset := obj.(*appsv1.DaemonSet)
		_, err := kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonset, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unsupported resource type for update: %s", resourceType)
	}
}

// 辅助方法：设置节点亲和性到对象
func (n *nodeAffinityService) setNodeAffinityOnObject(obj runtime.Object, req *model.K8sNodeAffinityRequest) error {
	switch o := obj.(type) {
	case *core.Pod:
		return n.setNodeAffinityOnPod(o, req)
	case *appsv1.Deployment:
		return n.setNodeAffinityOnDeployment(o, req)
	case *appsv1.ReplicaSet:
		return n.setNodeAffinityOnReplicaSet(o, req)
	case *appsv1.StatefulSet:
		return n.setNodeAffinityOnStatefulSet(o, req)
	case *appsv1.DaemonSet:
		return n.setNodeAffinityOnDaemonSet(o, req)
	default:
		return fmt.Errorf("unsupported object type for setting node affinity")
	}
}

// 辅助方法：设置Pod的节点亲和性
func (n *nodeAffinityService) setNodeAffinityOnPod(pod *core.Pod, req *model.K8sNodeAffinityRequest) error {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &core.Affinity{}
	}
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &core.NodeAffinity{}
	}

	// 设置硬亲和性
	if len(req.RequiredAffinity) > 0 {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &core.NodeSelector{
			NodeSelectorTerms: n.convertToK8sNodeSelectorTerms(req.RequiredAffinity),
		}
	}

	// 设置软亲和性
	if len(req.PreferredAffinity) > 0 {
		pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = n.convertToK8sPreferredSchedulingTerms(req.PreferredAffinity)
	}

	// 设置节点选择器
	if len(req.NodeSelector) > 0 {
		if pod.Spec.NodeSelector == nil {
			pod.Spec.NodeSelector = make(map[string]string)
		}
		for key, value := range req.NodeSelector {
			pod.Spec.NodeSelector[key] = value
		}
	}

	return nil
}

// 辅助方法：设置Deployment的节点亲和性
func (n *nodeAffinityService) setNodeAffinityOnDeployment(deployment *appsv1.Deployment, req *model.K8sNodeAffinityRequest) error {
	if deployment.Spec.Template.Spec.Affinity == nil {
		deployment.Spec.Template.Spec.Affinity = &core.Affinity{}
	}
	if deployment.Spec.Template.Spec.Affinity.NodeAffinity == nil {
		deployment.Spec.Template.Spec.Affinity.NodeAffinity = &core.NodeAffinity{}
	}

	// 设置硬亲和性
	if len(req.RequiredAffinity) > 0 {
		deployment.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &core.NodeSelector{
			NodeSelectorTerms: n.convertToK8sNodeSelectorTerms(req.RequiredAffinity),
		}
	}

	// 设置软亲和性
	if len(req.PreferredAffinity) > 0 {
		deployment.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = n.convertToK8sPreferredSchedulingTerms(req.PreferredAffinity)
	}

	// 设置节点选择器
	if len(req.NodeSelector) > 0 {
		deployment.Spec.Template.Spec.NodeSelector = req.NodeSelector
	}

	return nil
}

// 辅助方法：设置ReplicaSet的节点亲和性
func (n *nodeAffinityService) setNodeAffinityOnReplicaSet(replicaset *appsv1.ReplicaSet, req *model.K8sNodeAffinityRequest) error {
	if replicaset.Spec.Template.Spec.Affinity == nil {
		replicaset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}
	if replicaset.Spec.Template.Spec.Affinity.NodeAffinity == nil {
		replicaset.Spec.Template.Spec.Affinity.NodeAffinity = &core.NodeAffinity{}
	}

	// 设置硬亲和性
	if len(req.RequiredAffinity) > 0 {
		replicaset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &core.NodeSelector{
			NodeSelectorTerms: n.convertToK8sNodeSelectorTerms(req.RequiredAffinity),
		}
	}

	// 设置软亲和性
	if len(req.PreferredAffinity) > 0 {
		replicaset.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = n.convertToK8sPreferredSchedulingTerms(req.PreferredAffinity)
	}

	// 设置节点选择器
	if len(req.NodeSelector) > 0 {
		replicaset.Spec.Template.Spec.NodeSelector = req.NodeSelector
	}

	return nil
}

// 辅助方法：设置StatefulSet的节点亲和性
func (n *nodeAffinityService) setNodeAffinityOnStatefulSet(statefulset *appsv1.StatefulSet, req *model.K8sNodeAffinityRequest) error {
	if statefulset.Spec.Template.Spec.Affinity == nil {
		statefulset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}
	if statefulset.Spec.Template.Spec.Affinity.NodeAffinity == nil {
		statefulset.Spec.Template.Spec.Affinity.NodeAffinity = &core.NodeAffinity{}
	}

	// 设置硬亲和性
	if len(req.RequiredAffinity) > 0 {
		statefulset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &core.NodeSelector{
			NodeSelectorTerms: n.convertToK8sNodeSelectorTerms(req.RequiredAffinity),
		}
	}

	// 设置软亲和性
	if len(req.PreferredAffinity) > 0 {
		statefulset.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = n.convertToK8sPreferredSchedulingTerms(req.PreferredAffinity)
	}

	// 设置节点选择器
	if len(req.NodeSelector) > 0 {
		statefulset.Spec.Template.Spec.NodeSelector = req.NodeSelector
	}

	return nil
}

// 辅助方法：设置DaemonSet的节点亲和性
func (n *nodeAffinityService) setNodeAffinityOnDaemonSet(daemonset *appsv1.DaemonSet, req *model.K8sNodeAffinityRequest) error {
	if daemonset.Spec.Template.Spec.Affinity == nil {
		daemonset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}
	if daemonset.Spec.Template.Spec.Affinity.NodeAffinity == nil {
		daemonset.Spec.Template.Spec.Affinity.NodeAffinity = &core.NodeAffinity{}
	}

	// 设置硬亲和性
	if len(req.RequiredAffinity) > 0 {
		daemonset.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &core.NodeSelector{
			NodeSelectorTerms: n.convertToK8sNodeSelectorTerms(req.RequiredAffinity),
		}
	}

	// 设置软亲和性
	if len(req.PreferredAffinity) > 0 {
		daemonset.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = n.convertToK8sPreferredSchedulingTerms(req.PreferredAffinity)
	}

	// 设置节点选择器
	if len(req.NodeSelector) > 0 {
		daemonset.Spec.Template.Spec.NodeSelector = req.NodeSelector
	}

	return nil
}

// 辅助方法：从对象提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromObject(obj runtime.Object) *model.K8sNodeAffinityRequest {
	switch o := obj.(type) {
	case *core.Pod:
		return n.extractNodeAffinityFromPod(o)
	case *appsv1.Deployment:
		return n.extractNodeAffinityFromDeployment(o)
	case *appsv1.ReplicaSet:
		return n.extractNodeAffinityFromReplicaSet(o)
	case *appsv1.StatefulSet:
		return n.extractNodeAffinityFromStatefulSet(o)
	case *appsv1.DaemonSet:
		return n.extractNodeAffinityFromDaemonSet(o)
	default:
		return &model.K8sNodeAffinityRequest{}
	}
}

// 辅助方法：从Pod提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromPod(pod *core.Pod) *model.K8sNodeAffinityRequest {
	req := &model.K8sNodeAffinityRequest{
		NodeSelector: pod.Spec.NodeSelector,
	}

	if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := pod.Spec.Affinity.NodeAffinity

		// 提取硬亲和性
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			req.RequiredAffinity = n.convertFromK8sNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		}

		// 提取软亲和性
		if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PreferredAffinity = n.convertFromK8sPreferredSchedulingTerms(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

// 辅助方法：从Deployment提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromDeployment(deployment *appsv1.Deployment) *model.K8sNodeAffinityRequest {
	req := &model.K8sNodeAffinityRequest{
		NodeSelector: deployment.Spec.Template.Spec.NodeSelector,
	}

	if deployment.Spec.Template.Spec.Affinity != nil && deployment.Spec.Template.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := deployment.Spec.Template.Spec.Affinity.NodeAffinity

		// 提取硬亲和性
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			req.RequiredAffinity = n.convertFromK8sNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		}

		// 提取软亲和性
		if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PreferredAffinity = n.convertFromK8sPreferredSchedulingTerms(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

// 辅助方法：从ReplicaSet提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromReplicaSet(replicaset *appsv1.ReplicaSet) *model.K8sNodeAffinityRequest {
	req := &model.K8sNodeAffinityRequest{
		NodeSelector: replicaset.Spec.Template.Spec.NodeSelector,
	}

	if replicaset.Spec.Template.Spec.Affinity != nil && replicaset.Spec.Template.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := replicaset.Spec.Template.Spec.Affinity.NodeAffinity

		// 提取硬亲和性
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			req.RequiredAffinity = n.convertFromK8sNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		}

		// 提取软亲和性
		if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PreferredAffinity = n.convertFromK8sPreferredSchedulingTerms(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

// 辅助方法：从StatefulSet提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromStatefulSet(statefulset *appsv1.StatefulSet) *model.K8sNodeAffinityRequest {
	req := &model.K8sNodeAffinityRequest{
		NodeSelector: statefulset.Spec.Template.Spec.NodeSelector,
	}

	if statefulset.Spec.Template.Spec.Affinity != nil && statefulset.Spec.Template.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := statefulset.Spec.Template.Spec.Affinity.NodeAffinity

		// 提取硬亲和性
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			req.RequiredAffinity = n.convertFromK8sNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		}

		// 提取软亲和性
		if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PreferredAffinity = n.convertFromK8sPreferredSchedulingTerms(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

// 辅助方法：从DaemonSet提取节点亲和性信息
func (n *nodeAffinityService) extractNodeAffinityFromDaemonSet(daemonset *appsv1.DaemonSet) *model.K8sNodeAffinityRequest {
	req := &model.K8sNodeAffinityRequest{
		NodeSelector: daemonset.Spec.Template.Spec.NodeSelector,
	}

	if daemonset.Spec.Template.Spec.Affinity != nil && daemonset.Spec.Template.Spec.Affinity.NodeAffinity != nil {
		nodeAffinity := daemonset.Spec.Template.Spec.Affinity.NodeAffinity

		// 提取硬亲和性
		if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			req.RequiredAffinity = n.convertFromK8sNodeSelectorTerms(nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms)
		}

		// 提取软亲和性
		if len(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PreferredAffinity = n.convertFromK8sPreferredSchedulingTerms(nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

// 辅助方法：从对象删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromObject(obj runtime.Object) error {
	switch o := obj.(type) {
	case *core.Pod:
		return n.removeNodeAffinityFromPod(o)
	case *appsv1.Deployment:
		return n.removeNodeAffinityFromDeployment(o)
	case *appsv1.ReplicaSet:
		return n.removeNodeAffinityFromReplicaSet(o)
	case *appsv1.StatefulSet:
		return n.removeNodeAffinityFromStatefulSet(o)
	case *appsv1.DaemonSet:
		return n.removeNodeAffinityFromDaemonSet(o)
	default:
		return fmt.Errorf("unsupported object type for removing node affinity")
	}
}

// 辅助方法：从Pod删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromPod(pod *core.Pod) error {
	if pod.Spec.Affinity != nil {
		pod.Spec.Affinity.NodeAffinity = nil
	}
	pod.Spec.NodeSelector = nil
	return nil
}

// 辅助方法：从Deployment删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromDeployment(deployment *appsv1.Deployment) error {
	if deployment.Spec.Template.Spec.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity.NodeAffinity = nil
	}
	deployment.Spec.Template.Spec.NodeSelector = nil
	return nil
}

// 辅助方法：从ReplicaSet删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromReplicaSet(replicaset *appsv1.ReplicaSet) error {
	if replicaset.Spec.Template.Spec.Affinity != nil {
		replicaset.Spec.Template.Spec.Affinity.NodeAffinity = nil
	}
	replicaset.Spec.Template.Spec.NodeSelector = nil
	return nil
}

// 辅助方法：从StatefulSet删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromStatefulSet(statefulset *appsv1.StatefulSet) error {
	if statefulset.Spec.Template.Spec.Affinity != nil {
		statefulset.Spec.Template.Spec.Affinity.NodeAffinity = nil
	}
	statefulset.Spec.Template.Spec.NodeSelector = nil
	return nil
}

// 辅助方法：从DaemonSet删除节点亲和性
func (n *nodeAffinityService) removeNodeAffinityFromDaemonSet(daemonset *appsv1.DaemonSet) error {
	if daemonset.Spec.Template.Spec.Affinity != nil {
		daemonset.Spec.Template.Spec.Affinity.NodeAffinity = nil
	}
	daemonset.Spec.Template.Spec.NodeSelector = nil
	return nil
}

// 辅助方法：验证节点是否符合亲和性规则
func (n *nodeAffinityService) nodeMatchesAffinity(node *core.Node, requiredAffinity []model.K8sNodeSelectorTerm, nodeSelector map[string]string) bool {
	// 检查节点选择器
	if len(nodeSelector) > 0 {
		for key, value := range nodeSelector {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				return false
			}
		}
	}

	// 检查硬亲和性
	if len(requiredAffinity) > 0 {
		for _, term := range requiredAffinity {
			if !n.nodeMatchesSelectorTerm(node, term) {
				return false
			}
		}
	}

	return true
}

// 辅助方法：验证节点是否匹配选择器条件
func (n *nodeAffinityService) nodeMatchesSelectorTerm(node *core.Node, term model.K8sNodeSelectorTerm) bool {
	// 检查标签匹配表达式
	for _, expr := range term.MatchExpressions {
		if !n.nodeMatchesExpression(node, expr) {
			return false
		}
	}

	// 检查字段匹配表达式
	for _, field := range term.MatchFields {
		if !n.nodeMatchesFieldExpression(node, field) {
			return false
		}
	}

	return true
}

// 辅助方法：验证节点是否匹配表达式
func (n *nodeAffinityService) nodeMatchesExpression(node *core.Node, expr model.K8sNodeSelectorRequirement) bool {
	nodeValue, exists := node.Labels[expr.Key]

	switch expr.Operator {
	case "In":
		if !exists {
			return false
		}
		for _, value := range expr.Values {
			if nodeValue == value {
				return true
			}
		}
		return false
	case "NotIn":
		if !exists {
			return true
		}
		for _, value := range expr.Values {
			if nodeValue == value {
				return false
			}
		}
		return true
	case "Exists":
		return exists
	case "DoesNotExist":
		return !exists
	case "Gt":
		if !exists || len(expr.Values) == 0 {
			return false
		}
		return nodeValue > expr.Values[0]
	case "Lt":
		if !exists || len(expr.Values) == 0 {
			return false
		}
		return nodeValue < expr.Values[0]
	default:
		return false
	}
}

// 辅助方法：验证节点是否匹配字段表达式
func (n *nodeAffinityService) nodeMatchesFieldExpression(node *core.Node, field model.K8sNodeSelectorRequirement) bool {
	var fieldValue string
	var exists bool

	// 获取节点字段值
	switch field.Key {
	case "metadata.name":
		fieldValue = node.Name
		exists = true
	case "spec.unschedulable":
		fieldValue = fmt.Sprintf("%t", node.Spec.Unschedulable)
		exists = true
	default:
		// 对于其他字段，尝试从标签中获取
		fieldValue, exists = node.Labels[field.Key]
	}

	switch field.Operator {
	case "In":
		if !exists {
			return false
		}
		for _, value := range field.Values {
			if fieldValue == value {
				return true
			}
		}
		return false
	case "NotIn":
		if !exists {
			return true
		}
		for _, value := range field.Values {
			if fieldValue == value {
				return false
			}
		}
		return true
	case "Exists":
		return exists
	case "DoesNotExist":
		return !exists
	default:
		return false
	}
}

// 辅助方法：验证节点选择器条件
func (n *nodeAffinityService) validateNodeSelectorTerm(term model.K8sNodeSelectorTerm) error {
	if len(term.MatchExpressions) == 0 && len(term.MatchFields) == 0 {
		return fmt.Errorf("节点选择器条件必须包含至少一个匹配表达式或字段")
	}

	// 验证匹配表达式
	for _, expr := range term.MatchExpressions {
		if err := n.validateNodeSelectorRequirement(expr); err != nil {
			return fmt.Errorf("匹配表达式验证失败: %w", err)
		}
	}

	// 验证匹配字段
	for _, field := range term.MatchFields {
		if err := n.validateNodeSelectorRequirement(field); err != nil {
			return fmt.Errorf("匹配字段验证失败: %w", err)
		}
	}

	return nil
}

// 辅助方法：验证节点选择器要求
func (n *nodeAffinityService) validateNodeSelectorRequirement(req model.K8sNodeSelectorRequirement) error {
	if req.Key == "" {
		return fmt.Errorf("键不能为空")
	}

	validOperators := []string{"In", "NotIn", "Exists", "DoesNotExist", "Gt", "Lt"}
	validOperator := false
	for _, op := range validOperators {
		if req.Operator == op {
			validOperator = true
			break
		}
	}
	if !validOperator {
		return fmt.Errorf("无效的操作符: %s", req.Operator)
	}

	// 验证值
	if req.Operator == "In" || req.Operator == "NotIn" {
		if len(req.Values) == 0 {
			return fmt.Errorf("操作符 %s 需要提供值列表", req.Operator)
		}
	} else if req.Operator == "Gt" || req.Operator == "Lt" {
		if len(req.Values) != 1 {
			return fmt.Errorf("操作符 %s 需要提供单个值", req.Operator)
		}
	} else if req.Operator == "Exists" || req.Operator == "DoesNotExist" {
		if len(req.Values) > 0 {
			return fmt.Errorf("操作符 %s 不应该提供值列表", req.Operator)
		}
	}

	return nil
}

// 辅助方法：转换为K8s NodeSelectorTerms
func (n *nodeAffinityService) convertToK8sNodeSelectorTerms(terms []model.K8sNodeSelectorTerm) []core.NodeSelectorTerm {
	var result []core.NodeSelectorTerm
	for _, term := range terms {
		k8sTerm := core.NodeSelectorTerm{
			MatchExpressions: make([]core.NodeSelectorRequirement, len(term.MatchExpressions)),
			MatchFields:      make([]core.NodeSelectorRequirement, len(term.MatchFields)),
		}

		for i, expr := range term.MatchExpressions {
			k8sTerm.MatchExpressions[i] = core.NodeSelectorRequirement{
				Key:      expr.Key,
				Operator: core.NodeSelectorOperator(expr.Operator),
				Values:   expr.Values,
			}
		}

		for i, field := range term.MatchFields {
			k8sTerm.MatchFields[i] = core.NodeSelectorRequirement{
				Key:      field.Key,
				Operator: core.NodeSelectorOperator(field.Operator),
				Values:   field.Values,
			}
		}

		result = append(result, k8sTerm)
	}
	return result
}

// 辅助方法：转换为K8s PreferredSchedulingTerms
func (n *nodeAffinityService) convertToK8sPreferredSchedulingTerms(terms []model.K8sPreferredSchedulingTerm) []core.PreferredSchedulingTerm {
	var result []core.PreferredSchedulingTerm
	for _, term := range terms {
		k8sTerm := core.PreferredSchedulingTerm{
			Weight: term.Weight,
			Preference: core.NodeSelectorTerm{
				MatchExpressions: make([]core.NodeSelectorRequirement, len(term.Preference.MatchExpressions)),
				MatchFields:      make([]core.NodeSelectorRequirement, len(term.Preference.MatchFields)),
			},
		}

		for i, expr := range term.Preference.MatchExpressions {
			k8sTerm.Preference.MatchExpressions[i] = core.NodeSelectorRequirement{
				Key:      expr.Key,
				Operator: core.NodeSelectorOperator(expr.Operator),
				Values:   expr.Values,
			}
		}

		for i, field := range term.Preference.MatchFields {
			k8sTerm.Preference.MatchFields[i] = core.NodeSelectorRequirement{
				Key:      field.Key,
				Operator: core.NodeSelectorOperator(field.Operator),
				Values:   field.Values,
			}
		}

		result = append(result, k8sTerm)
	}
	return result
}

// 辅助方法：从K8s NodeSelectorTerms转换
func (n *nodeAffinityService) convertFromK8sNodeSelectorTerms(terms []core.NodeSelectorTerm) []model.K8sNodeSelectorTerm {
	var result []model.K8sNodeSelectorTerm
	for _, term := range terms {
		modelTerm := model.K8sNodeSelectorTerm{
			MatchExpressions: make([]model.K8sNodeSelectorRequirement, len(term.MatchExpressions)),
			MatchFields:      make([]model.K8sNodeSelectorRequirement, len(term.MatchFields)),
		}

		for i, expr := range term.MatchExpressions {
			modelTerm.MatchExpressions[i] = model.K8sNodeSelectorRequirement{
				Key:      expr.Key,
				Operator: string(expr.Operator),
				Values:   expr.Values,
			}
		}

		for i, field := range term.MatchFields {
			modelTerm.MatchFields[i] = model.K8sNodeSelectorRequirement{
				Key:      field.Key,
				Operator: string(field.Operator),
				Values:   field.Values,
			}
		}

		result = append(result, modelTerm)
	}
	return result
}

// 辅助方法：从K8s PreferredSchedulingTerms转换
func (n *nodeAffinityService) convertFromK8sPreferredSchedulingTerms(terms []core.PreferredSchedulingTerm) []model.K8sPreferredSchedulingTerm {
	var result []model.K8sPreferredSchedulingTerm
	for _, term := range terms {
		modelTerm := model.K8sPreferredSchedulingTerm{
			Weight: term.Weight,
			Preference: model.K8sNodeSelectorTerm{
				MatchExpressions: make([]model.K8sNodeSelectorRequirement, len(term.Preference.MatchExpressions)),
				MatchFields:      make([]model.K8sNodeSelectorRequirement, len(term.Preference.MatchFields)),
			},
		}

		for i, expr := range term.Preference.MatchExpressions {
			modelTerm.Preference.MatchExpressions[i] = model.K8sNodeSelectorRequirement{
				Key:      expr.Key,
				Operator: string(expr.Operator),
				Values:   expr.Values,
			}
		}

		for i, field := range term.Preference.MatchFields {
			modelTerm.Preference.MatchFields[i] = model.K8sNodeSelectorRequirement{
				Key:      field.Key,
				Operator: string(field.Operator),
				Values:   field.Values,
			}
		}

		result = append(result, modelTerm)
	}
	return result
}

// podAffinityService Pod亲和性服务实现
type podAffinityService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewPodAffinityService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) PodAffinityService {
	return &podAffinityService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// SetPodAffinity 设置Pod亲和性
func (p *podAffinityService) SetPodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error) {
	p.logger.Info("开始设置Pod亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := p.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		p.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 设置Pod亲和性
	if err := p.setPodAffinityOnObject(obj, req); err != nil {
		p.logger.Error("设置Pod亲和性失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to set pod affinity: %w", err)
	}

	// 更新资源对象
	if err := p.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		p.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sPodAffinityResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		PodAffinity:       req.PodAffinity,
		PodAntiAffinity:   req.PodAntiAffinity,
		CreationTimestamp: time.Now(),
	}

	p.logger.Info("成功设置Pod亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// GetPodAffinity 获取Pod亲和性
func (p *podAffinityService) GetPodAffinity(ctx context.Context, clusterID int, namespace, resourceType, resourceName string) (*model.K8sPodAffinityResponse, error) {
	p.logger.Info("开始获取Pod亲和性", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := p.getResourceObject(ctx, kubeClient, resourceType, namespace, resourceName)
	if err != nil {
		p.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", resourceType), zap.String("resource_name", resourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 提取Pod亲和性信息
	podAffinity := p.extractPodAffinityFromObject(obj)

	response := &model.K8sPodAffinityResponse{
		ResourceType:      resourceType,
		ResourceName:      resourceName,
		Namespace:         namespace,
		PodAffinity:       podAffinity.PodAffinity,
		PodAntiAffinity:   podAffinity.PodAntiAffinity,
		TopologyKey:       podAffinity.TopologyKey,
		CreationTimestamp: time.Now(),
	}

	p.logger.Info("成功获取Pod亲和性", zap.String("resource_type", resourceType), zap.String("resource_name", resourceName), zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))
	return response, nil
}

// UpdatePodAffinity 更新Pod亲和性
func (p *podAffinityService) UpdatePodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error) {
	p.logger.Info("开始更新Pod亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	// 更新操作与设置操作相同
	return p.SetPodAffinity(ctx, req)
}

// DeletePodAffinity 删除Pod亲和性
func (p *podAffinityService) DeletePodAffinity(ctx context.Context, req *model.K8sPodAffinityRequest) (*model.K8sPodAffinityResponse, error) {
	p.logger.Info("开始删除Pod亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取资源对象
	obj, err := p.getResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, req.ResourceName)
	if err != nil {
		p.logger.Error("获取资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to get resource object: %w", err)
	}

	// 删除Pod亲和性
	if err := p.removePodAffinityFromObject(obj); err != nil {
		p.logger.Error("删除Pod亲和性失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to remove pod affinity: %w", err)
	}

	// 更新资源对象
	if err := p.updateResourceObject(ctx, kubeClient, req.ResourceType, req.Namespace, obj); err != nil {
		p.logger.Error("更新资源对象失败", zap.Error(err), zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName))
		return nil, fmt.Errorf("failed to update resource object: %w", err)
	}

	response := &model.K8sPodAffinityResponse{
		ResourceType:      req.ResourceType,
		ResourceName:      req.ResourceName,
		Namespace:         req.Namespace,
		CreationTimestamp: time.Now(),
	}

	p.logger.Info("成功删除Pod亲和性", zap.String("resource_type", req.ResourceType), zap.String("resource_name", req.ResourceName), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// ValidatePodAffinity 验证Pod亲和性
func (p *podAffinityService) ValidatePodAffinity(ctx context.Context, req *model.K8sPodAffinityValidationRequest) (*model.K8sPodAffinityValidationResponse, error) {
	p.logger.Info("开始验证Pod亲和性", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取所有Pods
	pods, err := kubeClient.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取Pod列表失败", zap.Error(err), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// 模拟验证Pod亲和性
	var affectedPods []string
	var validationErrors []string
	var suggestions []string

	for _, pod := range pods.Items {
		// 简化的验证逻辑
		if p.validatePodAgainstAffinity(&pod, req) {
			affectedPods = append(affectedPods, pod.Name)
		}
	}

	valid := len(validationErrors) == 0
	if !valid {
		validationErrors = append(validationErrors, "Pod亲和性规则验证失败")
		suggestions = append(suggestions, "请检查Pod标签和拓扑域配置")
	}

	response := &model.K8sPodAffinityValidationResponse{
		Valid:            valid,
		MatchingPods:     affectedPods,
		ValidationErrors: validationErrors,
		Suggestions:      suggestions,
		ValidationTime:   time.Now(),
	}

	p.logger.Info("成功验证Pod亲和性", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID), zap.Bool("valid", valid))
	return response, nil
}

// GetTopologyDomains 获取拓扑域信息
func (p *podAffinityService) GetTopologyDomains(ctx context.Context, clusterID int, namespace string) ([]string, error) {
	p.logger.Info("开始获取拓扑域信息", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID))

	kubeClient, err := pkg.GetKubeClient(clusterID, p.client, p.logger)
	if err != nil {
		p.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取所有节点
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		p.logger.Error("获取节点列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 提取拓扑域信息
	topologyDomains := make(map[string]bool)
	commonTopologyKeys := []string{
		"kubernetes.io/hostname",
		"topology.kubernetes.io/zone",
		"topology.kubernetes.io/region",
		"node.kubernetes.io/instance-type",
	}

	for _, node := range nodes.Items {
		for _, key := range commonTopologyKeys {
			if _, exists := node.Labels[key]; exists {
				topologyDomains[key] = true
			}
		}
	}

	var result []string
	for domain := range topologyDomains {
		result = append(result, domain)
	}

	p.logger.Info("成功获取拓扑域信息", zap.String("namespace", namespace), zap.Int("cluster_id", clusterID), zap.Int("count", len(result)))
	return result, nil
}

// Pod亲和性服务的辅助方法
func (p *podAffinityService) getResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace, name string) (runtime.Object, error) {
	switch strings.ToLower(resourceType) {
	case "pod":
		return kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	case "deployment":
		return kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "replicaset":
		return kubeClient.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "statefulset":
		return kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "daemonset":
		return kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

func (p *podAffinityService) updateResourceObject(ctx context.Context, kubeClient kubernetes.Interface, resourceType, namespace string, obj runtime.Object) error {
	switch strings.ToLower(resourceType) {
	case "pod":
		pod := obj.(*core.Pod)
		_, err := kubeClient.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
		return err
	case "deployment":
		deployment := obj.(*appsv1.Deployment)
		_, err := kubeClient.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	case "replicaset":
		replicaset := obj.(*appsv1.ReplicaSet)
		_, err := kubeClient.AppsV1().ReplicaSets(namespace).Update(ctx, replicaset, metav1.UpdateOptions{})
		return err
	case "statefulset":
		statefulset := obj.(*appsv1.StatefulSet)
		_, err := kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulset, metav1.UpdateOptions{})
		return err
	case "daemonset":
		daemonset := obj.(*appsv1.DaemonSet)
		_, err := kubeClient.AppsV1().DaemonSets(namespace).Update(ctx, daemonset, metav1.UpdateOptions{})
		return err
	default:
		return fmt.Errorf("unsupported resource type for update: %s", resourceType)
	}
}

func (p *podAffinityService) setPodAffinityOnObject(obj runtime.Object, req *model.K8sPodAffinityRequest) error {
	switch o := obj.(type) {
	case *core.Pod:
		return p.setPodAffinityOnPod(o, req)
	case *appsv1.Deployment:
		return p.setPodAffinityOnDeployment(o, req)
	case *appsv1.ReplicaSet:
		return p.setPodAffinityOnReplicaSet(o, req)
	case *appsv1.StatefulSet:
		return p.setPodAffinityOnStatefulSet(o, req)
	case *appsv1.DaemonSet:
		return p.setPodAffinityOnDaemonSet(o, req)
	default:
		return fmt.Errorf("unsupported object type for setting pod affinity")
	}
}

func (p *podAffinityService) setPodAffinityOnPod(pod *core.Pod, req *model.K8sPodAffinityRequest) error {
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &core.Affinity{}
	}

	// 设置Pod亲和性
	if len(req.PodAffinity) > 0 {
		pod.Spec.Affinity.PodAffinity = &core.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAffinity),
		}
	}

	// 设置Pod反亲和性
	if len(req.PodAntiAffinity) > 0 {
		pod.Spec.Affinity.PodAntiAffinity = &core.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAntiAffinity),
		}
	}

	return nil
}

func (p *podAffinityService) setPodAffinityOnDeployment(deployment *appsv1.Deployment, req *model.K8sPodAffinityRequest) error {
	if deployment.Spec.Template.Spec.Affinity == nil {
		deployment.Spec.Template.Spec.Affinity = &core.Affinity{}
	}

	// 设置Pod亲和性
	if len(req.PodAffinity) > 0 {
		deployment.Spec.Template.Spec.Affinity.PodAffinity = &core.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAffinity),
		}
	}

	// 设置Pod反亲和性
	if len(req.PodAntiAffinity) > 0 {
		deployment.Spec.Template.Spec.Affinity.PodAntiAffinity = &core.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAntiAffinity),
		}
	}

	return nil
}

func (p *podAffinityService) setPodAffinityOnReplicaSet(replicaset *appsv1.ReplicaSet, req *model.K8sPodAffinityRequest) error {
	if replicaset.Spec.Template.Spec.Affinity == nil {
		replicaset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}

	// 设置Pod亲和性
	if len(req.PodAffinity) > 0 {
		replicaset.Spec.Template.Spec.Affinity.PodAffinity = &core.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAffinity),
		}
	}

	// 设置Pod反亲和性
	if len(req.PodAntiAffinity) > 0 {
		replicaset.Spec.Template.Spec.Affinity.PodAntiAffinity = &core.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAntiAffinity),
		}
	}

	return nil
}

func (p *podAffinityService) setPodAffinityOnStatefulSet(statefulset *appsv1.StatefulSet, req *model.K8sPodAffinityRequest) error {
	if statefulset.Spec.Template.Spec.Affinity == nil {
		statefulset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}

	// 设置Pod亲和性
	if len(req.PodAffinity) > 0 {
		statefulset.Spec.Template.Spec.Affinity.PodAffinity = &core.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAffinity),
		}
	}

	// 设置Pod反亲和性
	if len(req.PodAntiAffinity) > 0 {
		statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity = &core.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAntiAffinity),
		}
	}

	return nil
}

func (p *podAffinityService) setPodAffinityOnDaemonSet(daemonset *appsv1.DaemonSet, req *model.K8sPodAffinityRequest) error {
	if daemonset.Spec.Template.Spec.Affinity == nil {
		daemonset.Spec.Template.Spec.Affinity = &core.Affinity{}
	}

	// 设置Pod亲和性
	if len(req.PodAffinity) > 0 {
		daemonset.Spec.Template.Spec.Affinity.PodAffinity = &core.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAffinity),
		}
	}

	// 设置Pod反亲和性
	if len(req.PodAntiAffinity) > 0 {
		daemonset.Spec.Template.Spec.Affinity.PodAntiAffinity = &core.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: p.convertToPodAffinityTerms(req.PodAntiAffinity),
		}
	}

	return nil
}

func (p *podAffinityService) extractPodAffinityFromObject(obj runtime.Object) *model.K8sPodAffinityRequest {
	switch o := obj.(type) {
	case *core.Pod:
		return p.extractPodAffinityFromPod(o)
	case *appsv1.Deployment:
		return p.extractPodAffinityFromDeployment(o)
	case *appsv1.ReplicaSet:
		return p.extractPodAffinityFromReplicaSet(o)
	case *appsv1.StatefulSet:
		return p.extractPodAffinityFromStatefulSet(o)
	case *appsv1.DaemonSet:
		return p.extractPodAffinityFromDaemonSet(o)
	default:
		return &model.K8sPodAffinityRequest{}
	}
}

func (p *podAffinityService) extractPodAffinityFromPod(pod *core.Pod) *model.K8sPodAffinityRequest {
	req := &model.K8sPodAffinityRequest{}

	if pod.Spec.Affinity != nil {
		if pod.Spec.Affinity.PodAffinity != nil && len(pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAffinity = p.convertFromPodAffinityTerms(pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
		if pod.Spec.Affinity.PodAntiAffinity != nil && len(pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAntiAffinity = p.convertFromPodAffinityTerms(pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

func (p *podAffinityService) extractPodAffinityFromDeployment(deployment *appsv1.Deployment) *model.K8sPodAffinityRequest {
	req := &model.K8sPodAffinityRequest{}

	if deployment.Spec.Template.Spec.Affinity != nil {
		if deployment.Spec.Template.Spec.Affinity.PodAffinity != nil && len(deployment.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAffinity = p.convertFromPodAffinityTerms(deployment.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
		if deployment.Spec.Template.Spec.Affinity.PodAntiAffinity != nil && len(deployment.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAntiAffinity = p.convertFromPodAffinityTerms(deployment.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

func (p *podAffinityService) extractPodAffinityFromReplicaSet(replicaset *appsv1.ReplicaSet) *model.K8sPodAffinityRequest {
	req := &model.K8sPodAffinityRequest{}

	if replicaset.Spec.Template.Spec.Affinity != nil {
		if replicaset.Spec.Template.Spec.Affinity.PodAffinity != nil && len(replicaset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAffinity = p.convertFromPodAffinityTerms(replicaset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
		if replicaset.Spec.Template.Spec.Affinity.PodAntiAffinity != nil && len(replicaset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAntiAffinity = p.convertFromPodAffinityTerms(replicaset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

func (p *podAffinityService) extractPodAffinityFromStatefulSet(statefulset *appsv1.StatefulSet) *model.K8sPodAffinityRequest {
	req := &model.K8sPodAffinityRequest{}

	if statefulset.Spec.Template.Spec.Affinity != nil {
		if statefulset.Spec.Template.Spec.Affinity.PodAffinity != nil && len(statefulset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAffinity = p.convertFromPodAffinityTerms(statefulset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
		if statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity != nil && len(statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAntiAffinity = p.convertFromPodAffinityTerms(statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

func (p *podAffinityService) extractPodAffinityFromDaemonSet(daemonset *appsv1.DaemonSet) *model.K8sPodAffinityRequest {
	req := &model.K8sPodAffinityRequest{}

	if daemonset.Spec.Template.Spec.Affinity != nil {
		if daemonset.Spec.Template.Spec.Affinity.PodAffinity != nil && len(daemonset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAffinity = p.convertFromPodAffinityTerms(daemonset.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
		if daemonset.Spec.Template.Spec.Affinity.PodAntiAffinity != nil && len(daemonset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0 {
			req.PodAntiAffinity = p.convertFromPodAffinityTerms(daemonset.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
		}
	}

	return req
}

func (p *podAffinityService) removePodAffinityFromObject(obj runtime.Object) error {
	switch o := obj.(type) {
	case *core.Pod:
		return p.removePodAffinityFromPod(o)
	case *appsv1.Deployment:
		return p.removePodAffinityFromDeployment(o)
	case *appsv1.ReplicaSet:
		return p.removePodAffinityFromReplicaSet(o)
	case *appsv1.StatefulSet:
		return p.removePodAffinityFromStatefulSet(o)
	case *appsv1.DaemonSet:
		return p.removePodAffinityFromDaemonSet(o)
	default:
		return fmt.Errorf("unsupported object type for removing pod affinity")
	}
}

func (p *podAffinityService) removePodAffinityFromPod(pod *core.Pod) error {
	if pod.Spec.Affinity != nil {
		pod.Spec.Affinity.PodAffinity = nil
		pod.Spec.Affinity.PodAntiAffinity = nil
	}
	return nil
}

func (p *podAffinityService) removePodAffinityFromDeployment(deployment *appsv1.Deployment) error {
	if deployment.Spec.Template.Spec.Affinity != nil {
		deployment.Spec.Template.Spec.Affinity.PodAffinity = nil
		deployment.Spec.Template.Spec.Affinity.PodAntiAffinity = nil
	}
	return nil
}

func (p *podAffinityService) removePodAffinityFromReplicaSet(replicaset *appsv1.ReplicaSet) error {
	if replicaset.Spec.Template.Spec.Affinity != nil {
		replicaset.Spec.Template.Spec.Affinity.PodAffinity = nil
		replicaset.Spec.Template.Spec.Affinity.PodAntiAffinity = nil
	}
	return nil
}

func (p *podAffinityService) removePodAffinityFromStatefulSet(statefulset *appsv1.StatefulSet) error {
	if statefulset.Spec.Template.Spec.Affinity != nil {
		statefulset.Spec.Template.Spec.Affinity.PodAffinity = nil
		statefulset.Spec.Template.Spec.Affinity.PodAntiAffinity = nil
	}
	return nil
}

func (p *podAffinityService) removePodAffinityFromDaemonSet(daemonset *appsv1.DaemonSet) error {
	if daemonset.Spec.Template.Spec.Affinity != nil {
		daemonset.Spec.Template.Spec.Affinity.PodAffinity = nil
		daemonset.Spec.Template.Spec.Affinity.PodAntiAffinity = nil
	}
	return nil
}

func (p *podAffinityService) validatePodAgainstAffinity(pod *core.Pod, req *model.K8sPodAffinityValidationRequest) bool {
	// 简化的验证逻辑，实际应该根据具体的亲和性规则进行验证
	return true
}

func (p *podAffinityService) convertToPodAffinityTerms(terms []model.K8sPodAffinityTerm) []core.PodAffinityTerm {
	var result []core.PodAffinityTerm
	for _, term := range terms {
		k8sTerm := core.PodAffinityTerm{
			TopologyKey: term.TopologyKey,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels:      term.LabelSelector.MatchLabels,
				MatchExpressions: make([]metav1.LabelSelectorRequirement, len(term.LabelSelector.MatchExpressions)),
			},
		}

		for i, expr := range term.LabelSelector.MatchExpressions {
			k8sTerm.LabelSelector.MatchExpressions[i] = metav1.LabelSelectorRequirement{
				Key:      expr.Key,
				Operator: metav1.LabelSelectorOperator(expr.Operator),
				Values:   expr.Values,
			}
		}

		result = append(result, k8sTerm)
	}
	return result
}

func (p *podAffinityService) convertFromPodAffinityTerms(terms []core.PodAffinityTerm) []model.K8sPodAffinityTerm {
	var result []model.K8sPodAffinityTerm
	for _, term := range terms {
		modelTerm := model.K8sPodAffinityTerm{
			TopologyKey: term.TopologyKey,
			LabelSelector: model.K8sLabelSelector{
				MatchLabels:      term.LabelSelector.MatchLabels,
				MatchExpressions: make([]model.K8sLabelSelectorRequirement, len(term.LabelSelector.MatchExpressions)),
			},
		}

		for i, expr := range term.LabelSelector.MatchExpressions {
			modelTerm.LabelSelector.MatchExpressions[i] = model.K8sLabelSelectorRequirement{
				Key:      expr.Key,
				Operator: string(expr.Operator),
				Values:   expr.Values,
			}
		}

		result = append(result, modelTerm)
	}
	return result
}

// affinityVisualizationService 亲和性可视化服务实现
type affinityVisualizationService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

func NewAffinityVisualizationService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) AffinityVisualizationService {
	return &affinityVisualizationService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetAffinityVisualization 获取亲和性可视化
func (a *affinityVisualizationService) GetAffinityVisualization(ctx context.Context, req *model.K8sAffinityVisualizationRequest) (*model.K8sAffinityVisualizationResponse, error) {
	a.logger.Info("开始获取亲和性可视化", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))

	kubeClient, err := pkg.GetKubeClient(req.ClusterID, a.client, a.logger)
	if err != nil {
		a.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	// 获取节点信息
	nodes, err := kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		a.logger.Error("获取节点列表失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 获取Pods信息
	pods, err := kubeClient.CoreV1().Pods(req.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		a.logger.Error("获取Pod列表失败", zap.Error(err), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// 构建亲和性可视化数据
	visualization := a.buildAffinityVisualization(nodes.Items, pods.Items)

	response := &model.K8sAffinityVisualizationResponse{
		ClusterID:     req.ClusterID,
		Namespace:     req.Namespace,
		Visualization: visualization,
		GeneratedTime: time.Now(),
	}

	a.logger.Info("成功获取亲和性可视化", zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return response, nil
}

// 辅助方法：构建亲和性可视化数据
func (a *affinityVisualizationService) buildAffinityVisualization(nodes []core.Node, pods []core.Pod) map[string]interface{} {
	visualization := make(map[string]interface{})

	// 节点信息
	var nodeInfo []map[string]interface{}
	for _, node := range nodes {
		nodeData := map[string]interface{}{
			"name":   node.Name,
			"labels": node.Labels,
			"taints": node.Spec.Taints,
		}
		nodeInfo = append(nodeInfo, nodeData)
	}
	visualization["nodes"] = nodeInfo

	// Pod信息
	var podInfo []map[string]interface{}
	for _, pod := range pods {
		podData := map[string]interface{}{
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"nodeName":  pod.Spec.NodeName,
			"labels":    pod.Labels,
		}

		// 提取亲和性信息
		if pod.Spec.Affinity != nil {
			affinity := make(map[string]interface{})
			if pod.Spec.Affinity.NodeAffinity != nil {
				affinity["nodeAffinity"] = pod.Spec.Affinity.NodeAffinity
			}
			if pod.Spec.Affinity.PodAffinity != nil {
				affinity["podAffinity"] = pod.Spec.Affinity.PodAffinity
			}
			if pod.Spec.Affinity.PodAntiAffinity != nil {
				affinity["podAntiAffinity"] = pod.Spec.Affinity.PodAntiAffinity
			}
			podData["affinity"] = affinity
		}

		// 提取容忍信息
		if len(pod.Spec.Tolerations) > 0 {
			podData["tolerations"] = pod.Spec.Tolerations
		}

		podInfo = append(podInfo, podData)
	}
	visualization["pods"] = podInfo

	// 添加统计信息
	visualization["summary"] = map[string]interface{}{
		"totalNodes":          len(nodes),
		"totalPods":           len(pods),
		"podsWithAffinity":    a.countPodsWithAffinity(pods),
		"podsWithTolerations": a.countPodsWithTolerations(pods),
		"nodesWithTaints":     a.countNodesWithTaints(nodes),
	}

	return visualization
}

// 辅助方法：统计有亲和性的Pod数量
func (a *affinityVisualizationService) countPodsWithAffinity(pods []core.Pod) int {
	count := 0
	for _, pod := range pods {
		if pod.Spec.Affinity != nil {
			count++
		}
	}
	return count
}

// 辅助方法：统计有容忍的Pod数量
func (a *affinityVisualizationService) countPodsWithTolerations(pods []core.Pod) int {
	count := 0
	for _, pod := range pods {
		if len(pod.Spec.Tolerations) > 0 {
			count++
		}
	}
	return count
}

// 辅助方法：统计有污点的节点数量
func (a *affinityVisualizationService) countNodesWithTaints(nodes []core.Node) int {
	count := 0
	for _, node := range nodes {
		if len(node.Spec.Taints) > 0 {
			count++
		}
	}
	return count
}
