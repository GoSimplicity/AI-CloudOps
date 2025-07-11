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
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type NetworkPolicyService interface {
	GetNetworkPoliciesByNamespace(ctx context.Context, id int, namespace string) ([]*networkingv1.NetworkPolicy, error)
	CreateNetworkPolicy(ctx context.Context, req *model.K8sNetworkPolicyRequest) error
	UpdateNetworkPolicy(ctx context.Context, req *model.K8sNetworkPolicyRequest) error
	DeleteNetworkPolicy(ctx context.Context, id int, namespace, networkPolicyName string) error
	BatchDeleteNetworkPolicy(ctx context.Context, id int, namespace string, networkPolicyNames []string) error
	GetNetworkPolicyYaml(ctx context.Context, id int, namespace, networkPolicyName string) (string, error)
	GetNetworkPolicyStatus(ctx context.Context, id int, namespace, networkPolicyName string) (*model.K8sNetworkPolicyStatus, error)
	GetNetworkPolicyRules(ctx context.Context, id int, namespace, networkPolicyName string) (map[string]interface{}, error)
	GetAffectedPods(ctx context.Context, id int, namespace, networkPolicyName string) ([]*corev1.Pod, error)
	ValidateNetworkPolicy(ctx context.Context, id int, req *model.K8sNetworkPolicyRequest) (map[string]interface{}, error)
}

type networkPolicyService struct {
	dao    admin.ClusterDAO
	client client.K8sClient
	logger *zap.Logger
}

// NewNetworkPolicyService 创建新的 NetworkPolicyService 实例
func NewNetworkPolicyService(dao admin.ClusterDAO, client client.K8sClient, logger *zap.Logger) NetworkPolicyService {
	return &networkPolicyService{
		dao:    dao,
		client: client,
		logger: logger,
	}
}

// GetNetworkPoliciesByNamespace 获取指定命名空间下的所有 NetworkPolicy
func (n *networkPolicyService) GetNetworkPoliciesByNamespace(ctx context.Context, id int, namespace string) ([]*networkingv1.NetworkPolicy, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	networkPolicies, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取 NetworkPolicy 列表失败", zap.Error(err), zap.Int("cluster_id", id), zap.String("namespace", namespace))
		return nil, fmt.Errorf("failed to get NetworkPolicy list: %w", err)
	}

	result := make([]*networkingv1.NetworkPolicy, len(networkPolicies.Items))
	for i := range networkPolicies.Items {
		result[i] = &networkPolicies.Items[i]
	}

	n.logger.Info("成功获取 NetworkPolicy 列表", zap.Int("cluster_id", id), zap.String("namespace", namespace), zap.Int("count", len(result)))
	return result, nil
}

// CreateNetworkPolicy 创建 NetworkPolicy
func (n *networkPolicyService) CreateNetworkPolicy(ctx context.Context, req *model.K8sNetworkPolicyRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	_, err = kubeClient.NetworkingV1().NetworkPolicies(req.Namespace).Create(ctx, req.NetworkPolicyYaml, metav1.CreateOptions{})
	if err != nil {
		n.logger.Error("创建 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to create NetworkPolicy: %w", err)
	}

	n.logger.Info("成功创建 NetworkPolicy", zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// UpdateNetworkPolicy 更新 NetworkPolicy
func (n *networkPolicyService) UpdateNetworkPolicy(ctx context.Context, req *model.K8sNetworkPolicyRequest) error {
	kubeClient, err := pkg.GetKubeClient(req.ClusterID, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	existingNetworkPolicy, err := kubeClient.NetworkingV1().NetworkPolicies(req.Namespace).Get(ctx, req.NetworkPolicyYaml.Name, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取现有 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to get existing NetworkPolicy: %w", err)
	}

	existingNetworkPolicy.Spec = req.NetworkPolicyYaml.Spec

	if _, err := kubeClient.NetworkingV1().NetworkPolicies(req.Namespace).Update(ctx, existingNetworkPolicy, metav1.UpdateOptions{}); err != nil {
		n.logger.Error("更新 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("failed to update NetworkPolicy: %w", err)
	}

	n.logger.Info("成功更新 NetworkPolicy", zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Int("cluster_id", req.ClusterID))
	return nil
}

// GetNetworkPolicyYaml 获取指定 NetworkPolicy 的 YAML 定义
func (n *networkPolicyService) GetNetworkPolicyYaml(ctx context.Context, id int, namespace, networkPolicyName string) (string, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	networkPolicy, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return "", fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	yamlData, err := yaml.Marshal(networkPolicy)
	if err != nil {
		n.logger.Error("序列化 NetworkPolicy YAML 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName))
		return "", fmt.Errorf("failed to serialize NetworkPolicy YAML: %w", err)
	}

	n.logger.Info("成功获取 NetworkPolicy YAML", zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return string(yamlData), nil
}

// BatchDeleteNetworkPolicy 批量删除 NetworkPolicy
func (n *networkPolicyService) BatchDeleteNetworkPolicy(ctx context.Context, id int, namespace string, networkPolicyNames []string) error {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(networkPolicyNames))

	for _, name := range networkPolicyNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			if err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
				n.logger.Error("删除 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", name), zap.String("namespace", namespace), zap.Int("cluster_id", id))
				errChan <- fmt.Errorf("failed to delete NetworkPolicy '%s': %w", name, err)
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
		n.logger.Error("批量删除 NetworkPolicy 部分失败", zap.Int("failed_count", len(errs)), zap.Int("total_count", len(networkPolicyNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("errors occurred while deleting NetworkPolicies: %v", errs)
	}

	n.logger.Info("成功批量删除 NetworkPolicy", zap.Int("count", len(networkPolicyNames)), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// DeleteNetworkPolicy 删除指定的 NetworkPolicy
func (n *networkPolicyService) DeleteNetworkPolicy(ctx context.Context, id int, namespace, networkPolicyName string) error {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	if err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, networkPolicyName, metav1.DeleteOptions{}); err != nil {
		n.logger.Error("删除 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return fmt.Errorf("failed to delete NetworkPolicy '%s': %w", networkPolicyName, err)
	}

	n.logger.Info("成功删除 NetworkPolicy", zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
	return nil
}

// GetNetworkPolicyStatus 获取 NetworkPolicy 状态
func (n *networkPolicyService) GetNetworkPolicyStatus(ctx context.Context, id int, namespace, networkPolicyName string) (*model.K8sNetworkPolicyStatus, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	networkPolicy, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	status := &model.K8sNetworkPolicyStatus{
		Name:              networkPolicy.Name,
		Namespace:         networkPolicy.Namespace,
		PodSelector:       &networkPolicy.Spec.PodSelector,
		PolicyTypes:       networkPolicy.Spec.PolicyTypes,
		Ingress:           networkPolicy.Spec.Ingress,
		Egress:            networkPolicy.Spec.Egress,
		CreationTimestamp: networkPolicy.CreationTimestamp.Time,
	}

	n.logger.Info("成功获取 NetworkPolicy 状态", zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("ingress_rules", len(networkPolicy.Spec.Ingress)), zap.Int("egress_rules", len(networkPolicy.Spec.Egress)), zap.Int("cluster_id", id))
	return status, nil
}

// GetNetworkPolicyRules 获取 NetworkPolicy 规则详情
func (n *networkPolicyService) GetNetworkPolicyRules(ctx context.Context, id int, namespace, networkPolicyName string) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	networkPolicy, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	rules := map[string]interface{}{
		"name":         networkPolicy.Name,
		"namespace":    networkPolicy.Namespace,
		"pod_selector": networkPolicy.Spec.PodSelector,
		"policy_types": networkPolicy.Spec.PolicyTypes,
		"ingress":      make([]map[string]interface{}, 0),
		"egress":       make([]map[string]interface{}, 0),
	}

	// 处理入站规则
	ingressRules := make([]map[string]interface{}, 0)
	for i, rule := range networkPolicy.Spec.Ingress {
		ingressRule := map[string]interface{}{
			"rule_index": i,
			"ports":      rule.Ports,
			"from":       rule.From,
		}
		ingressRules = append(ingressRules, ingressRule)
	}
	rules["ingress"] = ingressRules

	// 处理出站规则
	egressRules := make([]map[string]interface{}, 0)
	for i, rule := range networkPolicy.Spec.Egress {
		egressRule := map[string]interface{}{
			"rule_index": i,
			"ports":      rule.Ports,
			"to":         rule.To,
		}
		egressRules = append(egressRules, egressRule)
	}
	rules["egress"] = egressRules

	// 统计信息
	rules["summary"] = map[string]interface{}{
		"total_ingress_rules": len(networkPolicy.Spec.Ingress),
		"total_egress_rules":  len(networkPolicy.Spec.Egress),
		"has_ingress_policy":  contains(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeIngress),
		"has_egress_policy":   contains(networkPolicy.Spec.PolicyTypes, networkingv1.PolicyTypeEgress),
	}

	n.logger.Info("成功获取 NetworkPolicy 规则", zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("ingress_rules", len(networkPolicy.Spec.Ingress)), zap.Int("egress_rules", len(networkPolicy.Spec.Egress)), zap.Int("cluster_id", id))
	return rules, nil
}

// GetAffectedPods 获取受 NetworkPolicy 影响的 Pod
func (n *networkPolicyService) GetAffectedPods(ctx context.Context, id int, namespace, networkPolicyName string) ([]*corev1.Pod, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	networkPolicy, err := kubeClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicyName, metav1.GetOptions{})
	if err != nil {
		n.logger.Error("获取 NetworkPolicy 失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get NetworkPolicy: %w", err)
	}

	// 获取命名空间下的所有 Pod
	pods, err := kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		n.logger.Error("获取 Pod 列表失败", zap.Error(err), zap.String("namespace", namespace), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Pod list: %w", err)
	}

	// 创建标签选择器
	selector, err := metav1.LabelSelectorAsSelector(&networkPolicy.Spec.PodSelector)
	if err != nil {
		n.logger.Error("解析标签选择器失败", zap.Error(err), zap.String("network_policy_name", networkPolicyName))
		return nil, fmt.Errorf("failed to parse label selector: %w", err)
	}

	// 筛选匹配的 Pod
	var affectedPods []*corev1.Pod
	for i := range pods.Items {
		pod := &pods.Items[i]
		if selector.Matches(labels.Set(pod.Labels)) {
			affectedPods = append(affectedPods, pod)
		}
	}

	n.logger.Info("成功获取受 NetworkPolicy 影响的 Pod", zap.String("network_policy_name", networkPolicyName), zap.String("namespace", namespace), zap.Int("affected_pods", len(affectedPods)), zap.Int("total_pods", len(pods.Items)), zap.Int("cluster_id", id))
	return affectedPods, nil
}

// ValidateNetworkPolicy 验证 NetworkPolicy 配置
func (n *networkPolicyService) ValidateNetworkPolicy(ctx context.Context, id int, req *model.K8sNetworkPolicyRequest) (map[string]interface{}, error) {
	kubeClient, err := pkg.GetKubeClient(id, n.client, n.logger)
	if err != nil {
		n.logger.Error("获取 Kubernetes 客户端失败", zap.Error(err), zap.Int("cluster_id", id))
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}

	validation := map[string]interface{}{
		"valid":    true,
		"warnings": make([]string, 0),
		"errors":   make([]string, 0),
		"checks":   make(map[string]interface{}),
	}

	warnings := make([]string, 0)
	errors := make([]string, 0)
	checks := make(map[string]interface{})

	// 检查基本配置
	if req.NetworkPolicyYaml.Name == "" {
		errors = append(errors, "NetworkPolicy 名称不能为空")
	}

	if req.Namespace == "" {
		errors = append(errors, "命名空间不能为空")
	}

	// 检查标签选择器
	selector := req.NetworkPolicyYaml.Spec.PodSelector
	checks["pod_selector"] = map[string]interface{}{
		"is_empty":      len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0,
		"match_labels":  len(selector.MatchLabels),
		"match_expressions": len(selector.MatchExpressions),
	}

	if len(selector.MatchLabels) == 0 && len(selector.MatchExpressions) == 0 {
		warnings = append(warnings, "Pod 选择器为空，将匹配命名空间中的所有 Pod")
	}

	// 检查策略类型
	checks["policy_types"] = map[string]interface{}{
		"count":       len(req.NetworkPolicyYaml.Spec.PolicyTypes),
		"has_ingress": contains(req.NetworkPolicyYaml.Spec.PolicyTypes, networkingv1.PolicyTypeIngress),
		"has_egress":  contains(req.NetworkPolicyYaml.Spec.PolicyTypes, networkingv1.PolicyTypeEgress),
	}

	if len(req.NetworkPolicyYaml.Spec.PolicyTypes) == 0 {
		warnings = append(warnings, "未指定策略类型，默认行为可能不是预期的")
	}

	// 检查入站规则
	checks["ingress_rules"] = map[string]interface{}{
		"count": len(req.NetworkPolicyYaml.Spec.Ingress),
		"rules": make([]map[string]interface{}, 0),
	}

	if contains(req.NetworkPolicyYaml.Spec.PolicyTypes, networkingv1.PolicyTypeIngress) {
		if len(req.NetworkPolicyYaml.Spec.Ingress) == 0 {
			warnings = append(warnings, "指定了 Ingress 策略类型但没有定义入站规则，将拒绝所有入站流量")
		}
	}

	// 检查出站规则
	checks["egress_rules"] = map[string]interface{}{
		"count": len(req.NetworkPolicyYaml.Spec.Egress),
		"rules": make([]map[string]interface{}, 0),
	}

	if contains(req.NetworkPolicyYaml.Spec.PolicyTypes, networkingv1.PolicyTypeEgress) {
		if len(req.NetworkPolicyYaml.Spec.Egress) == 0 {
			warnings = append(warnings, "指定了 Egress 策略类型但没有定义出站规则，将拒绝所有出站流量")
		}
	}

	// 验证命名空间是否存在
	_, err = kubeClient.CoreV1().Namespaces().Get(ctx, req.Namespace, metav1.GetOptions{})
	if err != nil {
		errors = append(errors, fmt.Sprintf("命名空间 '%s' 不存在", req.Namespace))
	}

	// 检查是否已存在同名的 NetworkPolicy
	existingNP, err := kubeClient.NetworkingV1().NetworkPolicies(req.Namespace).Get(ctx, req.NetworkPolicyYaml.Name, metav1.GetOptions{})
	if err == nil {
		warnings = append(warnings, fmt.Sprintf("已存在同名的 NetworkPolicy '%s'，创建操作将失败", existingNP.Name))
		checks["existing_policy"] = map[string]interface{}{
			"exists":            true,
			"creation_timestamp": existingNP.CreationTimestamp,
		}
	} else {
		checks["existing_policy"] = map[string]interface{}{
			"exists": false,
		}
	}

	validation["warnings"] = warnings
	validation["errors"] = errors
	validation["checks"] = checks

	if len(errors) > 0 {
		validation["valid"] = false
	}

	n.logger.Info("成功验证 NetworkPolicy 配置", zap.String("network_policy_name", req.NetworkPolicyYaml.Name), zap.String("namespace", req.Namespace), zap.Bool("valid", validation["valid"].(bool)), zap.Int("warnings", len(warnings)), zap.Int("errors", len(errors)), zap.Int("cluster_id", id))
	return validation, nil
}

// 辅助函数：检查切片是否包含指定元素
func contains(slice []networkingv1.PolicyType, item networkingv1.PolicyType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}