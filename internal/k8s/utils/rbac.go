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

package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

// ConvertK8sRoleToRoleInfo 将K8s Role转换为RoleInfo
func ConvertK8sRoleToRoleInfo(role *rbacv1.Role, clusterID int) model.RoleInfo {
	rules := make([]model.PolicyRule, 0, len(role.Rules))
	for _, rule := range role.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	return model.RoleInfo{
		Name:              role.Name,
		Namespace:         role.Namespace,
		ClusterID:         clusterID,
		UID:               string(role.UID),
		CreationTimestamp: role.CreationTimestamp.Format(time.RFC3339),
		Labels:            role.Labels,
		Annotations:       role.Annotations,
		Rules:             rules,
		ResourceVersion:   role.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(role.CreationTimestamp.Time)),
	}
}

// ConvertK8sClusterRoleToClusterRoleInfo 将K8s ClusterRole转换为ClusterRoleInfo
func ConvertK8sClusterRoleToClusterRoleInfo(clusterRole *rbacv1.ClusterRole, clusterID int) model.ClusterRoleInfo {
	rules := make([]model.PolicyRule, 0, len(clusterRole.Rules))
	for _, rule := range clusterRole.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	return model.ClusterRoleInfo{
		Name:              clusterRole.Name,
		ClusterID:         clusterID,
		UID:               string(clusterRole.UID),
		CreationTimestamp: clusterRole.CreationTimestamp.Format(time.RFC3339),
		Labels:            clusterRole.Labels,
		Annotations:       clusterRole.Annotations,
		Rules:             rules,
		ResourceVersion:   clusterRole.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(clusterRole.CreationTimestamp.Time)),
	}
}

// ConvertK8sRoleBindingToRoleBindingInfo 将K8s RoleBinding转换为RoleBindingInfo
func ConvertK8sRoleBindingToRoleBindingInfo(roleBinding *rbacv1.RoleBinding, clusterID int) model.RoleBindingInfo {
	subjects := make([]model.Subject, 0, len(roleBinding.Subjects))
	for _, subject := range roleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	return model.RoleBindingInfo{
		Name:              roleBinding.Name,
		Namespace:         roleBinding.Namespace,
		ClusterID:         clusterID,
		UID:               string(roleBinding.UID),
		CreationTimestamp: roleBinding.CreationTimestamp.Format(time.RFC3339),
		Labels:            roleBinding.Labels,
		Annotations:       roleBinding.Annotations,
		RoleRef: model.RoleRef{
			APIGroup: roleBinding.RoleRef.APIGroup,
			Kind:     roleBinding.RoleRef.Kind,
			Name:     roleBinding.RoleRef.Name,
		},
		Subjects:        subjects,
		ResourceVersion: roleBinding.ResourceVersion,
		Age:             duration.HumanDuration(time.Since(roleBinding.CreationTimestamp.Time)),
	}
}

// ConvertK8sClusterRoleBindingToClusterRoleBindingInfo 将K8s ClusterRoleBinding转换为ClusterRoleBindingInfo
func ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(clusterRoleBinding *rbacv1.ClusterRoleBinding, clusterID int) model.ClusterRoleBindingInfo {
	subjects := make([]model.Subject, 0, len(clusterRoleBinding.Subjects))
	for _, subject := range clusterRoleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	return model.ClusterRoleBindingInfo{
		Name:              clusterRoleBinding.Name,
		ClusterID:         clusterID,
		UID:               string(clusterRoleBinding.UID),
		CreationTimestamp: clusterRoleBinding.CreationTimestamp.Format(time.RFC3339),
		Labels:            clusterRoleBinding.Labels,
		Annotations:       clusterRoleBinding.Annotations,
		RoleRef: model.RoleRef{
			APIGroup: clusterRoleBinding.RoleRef.APIGroup,
			Kind:     clusterRoleBinding.RoleRef.Kind,
			Name:     clusterRoleBinding.RoleRef.Name,
		},
		Subjects:        subjects,
		ResourceVersion: clusterRoleBinding.ResourceVersion,
		Age:             duration.HumanDuration(time.Since(clusterRoleBinding.CreationTimestamp.Time)),
	}
}

// ConvertPolicyRulesToK8s 将模型PolicyRule转换为K8s PolicyRule
func ConvertPolicyRulesToK8s(rules []model.PolicyRule) []rbacv1.PolicyRule {
	k8sRules := make([]rbacv1.PolicyRule, 0, len(rules))
	for _, rule := range rules {
		k8sRules = append(k8sRules, rbacv1.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}
	return k8sRules
}

// ConvertRoleRefToK8s 将模型RoleRef转换为K8s RoleRef
func ConvertRoleRefToK8s(roleRef model.RoleRef) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: roleRef.APIGroup,
		Kind:     roleRef.Kind,
		Name:     roleRef.Name,
	}
}

// ConvertSubjectsToK8s 将模型Subjects转换为K8s Subjects
func ConvertSubjectsToK8s(subjects []model.Subject) []rbacv1.Subject {
	k8sSubjects := make([]rbacv1.Subject, 0, len(subjects))
	for _, subject := range subjects {
		k8sSubjects = append(k8sSubjects, rbacv1.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}
	return k8sSubjects
}

// BuildClusterRoleListOptions 构建ClusterRole列表查询选项
func BuildClusterRoleListOptions(req *model.GetClusterRoleListReq) metav1.ListOptions {
	listOptions := metav1.ListOptions{}

	// 如果有关键字搜索，添加到标签选择器
	if req.Keyword != "" {
		// Kubernetes的标签选择器不支持模糊搜索，所以我们在获取后过滤
		// 这里只是返回基本的ListOptions
	}

	return listOptions
}

// ConvertToK8sClusterRole 将 rbacv1.ClusterRole 转换为 model.K8sClusterRole
func ConvertToK8sClusterRole(clusterRole *rbacv1.ClusterRole) *model.K8sClusterRole {
	if clusterRole == nil {
		return nil
	}

	// 转换权限规则
	rules := make([]model.PolicyRule, 0, len(clusterRole.Rules))
	for _, rule := range clusterRole.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	// 确定状态
	status := model.ClusterRoleStatusActive
	if strings.HasPrefix(clusterRole.Name, "system:") {
		// 系统角色
	}

	// 转换聚合规则
	var aggregationRule *model.ClusterRoleAggregationRule
	if clusterRole.AggregationRule != nil {
		selectors := make([]model.ClusterRoleLabelSelector, 0, len(clusterRole.AggregationRule.ClusterRoleSelectors))
		for _, selector := range clusterRole.AggregationRule.ClusterRoleSelectors {
			requirements := make([]model.ClusterRoleLabelRequirement, 0, len(selector.MatchExpressions))
			for _, req := range selector.MatchExpressions {
				requirements = append(requirements, model.ClusterRoleLabelRequirement{
					Key:      req.Key,
					Operator: string(req.Operator),
					Values:   req.Values,
				})
			}
			selectors = append(selectors, model.ClusterRoleLabelSelector{
				MatchLabels:      selector.MatchLabels,
				MatchExpressions: requirements,
			})
		}
		aggregationRule = &model.ClusterRoleAggregationRule{
			ClusterRoleSelectors: selectors,
		}
	}

	return &model.K8sClusterRole{
		Name:              clusterRole.Name,
		UID:               string(clusterRole.UID),
		Status:            status,
		Labels:            clusterRole.Labels,
		Annotations:       clusterRole.Annotations,
		Rules:             rules,
		ResourceVersion:   clusterRole.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(clusterRole.CreationTimestamp.Time)),
		IsSystemRole:      model.BoolToBoolValue(strings.HasPrefix(clusterRole.Name, "system:")),
		AggregationRule:   aggregationRule,
		CreationTimestamp: clusterRole.CreationTimestamp.Time,
		RawClusterRole:    clusterRole,
	}
}

// PaginateK8sClusterRoles 分页处理ClusterRole列表
func PaginateK8sClusterRoles(clusterRoles []*model.K8sClusterRole, page, size int) ([]*model.K8sClusterRole, int64) {
	total := int64(len(clusterRoles))

	// 计算分页
	start := (page - 1) * size
	end := start + size

	if start >= len(clusterRoles) {
		return []*model.K8sClusterRole{}, total
	}

	if end > len(clusterRoles) {
		end = len(clusterRoles)
	}

	return clusterRoles[start:end], total
}

// BuildK8sClusterRole 构建K8sClusterRole详细信息
func BuildK8sClusterRole(ctx context.Context, clusterID int, clusterRole rbacv1.ClusterRole) (*model.K8sClusterRole, error) {
	k8sClusterRole := ConvertToK8sClusterRole(&clusterRole)
	if k8sClusterRole == nil {
		return nil, fmt.Errorf("failed to convert ClusterRole")
	}

	k8sClusterRole.ClusterID = clusterID

	// TODO: 这里可以添加更多的详细信息构建逻辑
	// 比如获取绑定信息、使用情况等

	return k8sClusterRole, nil
}

// ClusterRoleToYAML 将ClusterRole转换为YAML
func ClusterRoleToYAML(clusterRole *rbacv1.ClusterRole) (string, error) {
	if clusterRole == nil {
		return "", fmt.Errorf("clusterRole is nil")
	}

	// 清理不需要的字段
	cleanClusterRole := clusterRole.DeepCopy()
	cleanClusterRole.ManagedFields = nil
	cleanClusterRole.ResourceVersion = ""
	cleanClusterRole.UID = ""
	cleanClusterRole.SelfLink = ""
	cleanClusterRole.CreationTimestamp = metav1.Time{}
	cleanClusterRole.Generation = 0

	yamlData, err := yaml.Marshal(cleanClusterRole)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ClusterRole to yaml: %w", err)
	}

	return string(yamlData), nil
}

// YAMLToClusterRole 将YAML转换为ClusterRole
func YAMLToClusterRole(yamlContent string) (*rbacv1.ClusterRole, error) {
	var clusterRole rbacv1.ClusterRole
	err := yaml.Unmarshal([]byte(yamlContent), &clusterRole)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml to ClusterRole: %w", err)
	}

	return &clusterRole, nil
}

// GetClusterRoleEvents 获取ClusterRole相关事件
func GetClusterRoleEvents(ctx context.Context, client kubernetes.Interface, name string, limit int) ([]*model.K8sClusterRoleEvent, int64, error) {
	// 获取与ClusterRole相关的事件
	events, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=ClusterRole", name),
		Limit:         int64(limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	// 转换事件
	k8sEvents := make([]*model.K8sClusterRoleEvent, 0, len(events.Items))
	for _, event := range events.Items {
		k8sEvent := &model.K8sClusterRoleEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    getEventSource(event),
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Count:     event.Count,
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, int64(len(events.Items)), nil
}

// GetClusterRoleUsage 获取ClusterRole使用情况
func GetClusterRoleUsage(ctx context.Context, client kubernetes.Interface, name string) (*model.K8sClusterRoleUsage, error) {
	usage := &model.K8sClusterRoleUsage{
		ClusterRoleName: name,
		IsUsed:          model.BoolFalse,
		RiskLevel:       "Low",
	}

	// 获取所有ClusterRoleBinding
	clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ClusterRoleBindings: %w", err)
	}

	// 检查ClusterRoleBinding
	for _, crb := range clusterRoleBindings.Items {
		if crb.RoleRef.Kind == "ClusterRole" && crb.RoleRef.Name == name {
			usage.IsUsed = model.BoolTrue
			usage.ClusterBindings = append(usage.ClusterBindings, model.ClusterRoleBindingSimpleInfo{
				Name:     crb.Name,
				Subjects: convertK8sSubjectsToModel(crb.Subjects),
			})

			// 添加主体
			for _, subject := range crb.Subjects {
				usage.Subjects = append(usage.Subjects, model.Subject{
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
					APIGroup:  subject.APIGroup,
				})
			}
		}
	}

	// 获取所有RoleBinding（ClusterRole可以被RoleBinding引用）
	roleBindings, err := client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list RoleBindings: %w", err)
	}

	// 检查RoleBinding
	for _, rb := range roleBindings.Items {
		if rb.RoleRef.Kind == "ClusterRole" && rb.RoleRef.Name == name {
			usage.IsUsed = model.BoolTrue
			usage.RoleBindings = append(usage.RoleBindings, model.RoleBindingSimpleInfo{
				Name:      rb.Name,
				Namespace: rb.Namespace,
				Subjects:  convertK8sSubjectsToModel(rb.Subjects),
			})

			// 添加主体
			for _, subject := range rb.Subjects {
				usage.Subjects = append(usage.Subjects, model.Subject{
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
					APIGroup:  subject.APIGroup,
				})
			}
		}
	}

	// 获取ClusterRole权限
	clusterRole, err := client.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRole: %w", err)
	}

	// 转换权限规则
	for _, rule := range clusterRole.Rules {
		usage.Permissions = append(usage.Permissions, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	// 评估风险等级
	if strings.HasPrefix(name, "system:") {
		usage.RiskLevel = "System"
	} else if hasHighRiskPermissions(clusterRole.Rules) {
		usage.RiskLevel = "High"
	} else if len(usage.Subjects) > 10 {
		usage.RiskLevel = "Medium"
	}

	// 设置聚合角色（如果有）
	if clusterRole.AggregationRule != nil {
		// 简单实现，实际应该获取所有匹配的角色
		usage.AggregatedRoles = []string{"aggregated-role-placeholder"}
	}

	return usage, nil
}

// GetClusterRoleMetrics 获取ClusterRole指标
func GetClusterRoleMetrics(ctx context.Context, client kubernetes.Interface, name string) (*model.K8sClusterRoleMetrics, error) {
	metrics := &model.K8sClusterRoleMetrics{
		ClusterRoleName:      name,
		TotalClusterBindings: 0,
		TotalRoleBindings:    0,
		ActiveUsers:          0,
		ActiveGroups:         0,
		ServiceAccounts:      0,
		PermissionCount:      0,
		CrossNamespaceAccess: model.BoolFalse,
		ClusterWideAccess:    model.BoolTrue, // ClusterRole默认是集群级的
		SecurityRisk:         "Low",
		LastUpdated:          time.Now(),
	}

	// 获取ClusterRole
	clusterRole, err := client.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRole: %w", err)
	}

	// 计算权限数量
	metrics.PermissionCount = len(clusterRole.Rules)

	// 统计ClusterRoleBinding
	clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err == nil {
		users := make(map[string]bool)
		groups := make(map[string]bool)

		for _, crb := range clusterRoleBindings.Items {
			if crb.RoleRef.Kind == "ClusterRole" && crb.RoleRef.Name == name {
				metrics.TotalClusterBindings++

				for _, subject := range crb.Subjects {
					switch subject.Kind {
					case "User":
						users[subject.Name] = true
					case "Group":
						groups[subject.Name] = true
					case "ServiceAccount":
						metrics.ServiceAccounts++
					}
				}
			}
		}

		metrics.ActiveUsers = len(users)
		metrics.ActiveGroups = len(groups)
	}

	// 统计RoleBinding（ClusterRole可以被RoleBinding引用）
	roleBindings, err := client.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err == nil {
		namespaces := make(map[string]bool)

		for _, rb := range roleBindings.Items {
			if rb.RoleRef.Kind == "ClusterRole" && rb.RoleRef.Name == name {
				metrics.TotalRoleBindings++
				namespaces[rb.Namespace] = true
			}
		}

		if len(namespaces) > 1 {
			metrics.CrossNamespaceAccess = model.BoolTrue
		}
	}

	// 评估安全风险
	if strings.HasPrefix(name, "system:") {
		metrics.SecurityRisk = "System"
	} else if hasHighRiskPermissions(clusterRole.Rules) {
		metrics.SecurityRisk = "High"
	} else if metrics.TotalClusterBindings > 5 || metrics.ActiveUsers > 10 {
		metrics.SecurityRisk = "Medium"
	}

	return metrics, nil
}

// 辅助函数
func getEventSource(event corev1.Event) string {
	if event.Source.Component != "" {
		return event.Source.Component
	}
	if event.Source.Host != "" {
		return event.Source.Host
	}
	return "unknown"
}

func convertK8sSubjectsToModel(subjects []rbacv1.Subject) []model.Subject {
	result := make([]model.Subject, 0, len(subjects))
	for _, s := range subjects {
		result = append(result, model.Subject{
			Kind:      s.Kind,
			Name:      s.Name,
			Namespace: s.Namespace,
			APIGroup:  s.APIGroup,
		})
	}
	return result
}

func hasHighRiskPermissions(rules []rbacv1.PolicyRule) bool {
	highRiskVerbs := []string{"*", "create", "delete", "deletecollection"}
	highRiskResources := []string{"*", "nodes", "persistentvolumes", "clusterroles", "clusterrolebindings"}

	for _, rule := range rules {
		// 检查危险动作
		for _, verb := range rule.Verbs {
			for _, highRisk := range highRiskVerbs {
				if verb == highRisk {
					return true
				}
			}
		}

		// 检查危险资源
		for _, resource := range rule.Resources {
			for _, highRisk := range highRiskResources {
				if resource == highRisk {
					return true
				}
			}
		}
	}

	return false
}

// ====================== ClusterRoleBinding 工具函数 ======================

// BuildClusterRoleBindingListOptions 构建ClusterRoleBinding列表查询选项
func BuildClusterRoleBindingListOptions(req *model.GetClusterRoleBindingListReq) metav1.ListOptions {
	listOptions := metav1.ListOptions{}

	// 如果有关键字搜索，添加到标签选择器
	if req.Keyword != "" {
		// Kubernetes的标签选择器不支持模糊搜索，所以我们在获取后过滤
		// 这里只是返回基本的ListOptions
	}

	return listOptions
}

// ConvertToK8sClusterRoleBinding 将 rbacv1.ClusterRoleBinding 转换为 model.K8sClusterRoleBinding
func ConvertToK8sClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) *model.K8sClusterRoleBinding {
	if clusterRoleBinding == nil {
		return nil
	}

	// 转换主体
	subjects := make([]model.Subject, 0, len(clusterRoleBinding.Subjects))
	for _, subject := range clusterRoleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	// 转换角色引用
	roleRef := model.RoleRef{
		APIGroup: clusterRoleBinding.RoleRef.APIGroup,
		Kind:     clusterRoleBinding.RoleRef.Kind,
		Name:     clusterRoleBinding.RoleRef.Name,
	}

	// 确定状态
	status := model.ClusterRoleBindingStatusActive
	if strings.HasPrefix(clusterRoleBinding.Name, "system:") {
		// 系统角色绑定
	}

	return &model.K8sClusterRoleBinding{
		Name:                  clusterRoleBinding.Name,
		UID:                   string(clusterRoleBinding.UID),
		Status:                status,
		Labels:                clusterRoleBinding.Labels,
		Annotations:           clusterRoleBinding.Annotations,
		RoleRef:               roleRef,
		Subjects:              subjects,
		ResourceVersion:       clusterRoleBinding.ResourceVersion,
		Age:                   duration.HumanDuration(time.Since(clusterRoleBinding.CreationTimestamp.Time)),
		IsSystemBinding:       model.BoolToBoolValue(strings.HasPrefix(clusterRoleBinding.Name, "system:")),
		CreationTimestamp:     clusterRoleBinding.CreationTimestamp.Time,
		RawClusterRoleBinding: clusterRoleBinding,
	}
}

// PaginateK8sClusterRoleBindings 分页处理ClusterRoleBinding列表
func PaginateK8sClusterRoleBindings(clusterRoleBindings []*model.K8sClusterRoleBinding, page, size int) ([]*model.K8sClusterRoleBinding, int64) {
	total := int64(len(clusterRoleBindings))

	// 计算分页
	start := (page - 1) * size
	end := start + size

	if start >= len(clusterRoleBindings) {
		return []*model.K8sClusterRoleBinding{}, total
	}

	if end > len(clusterRoleBindings) {
		end = len(clusterRoleBindings)
	}

	return clusterRoleBindings[start:end], total
}

// BuildK8sClusterRoleBinding 构建K8sClusterRoleBinding详细信息
func BuildK8sClusterRoleBinding(ctx context.Context, clusterID int, clusterRoleBinding rbacv1.ClusterRoleBinding) (*model.K8sClusterRoleBinding, error) {
	k8sClusterRoleBinding := ConvertToK8sClusterRoleBinding(&clusterRoleBinding)
	if k8sClusterRoleBinding == nil {
		return nil, fmt.Errorf("failed to convert ClusterRoleBinding")
	}

	k8sClusterRoleBinding.ClusterID = clusterID

	// TODO: 这里可以添加更多的详细信息构建逻辑
	// 比如获取角色详情、主体详情等

	return k8sClusterRoleBinding, nil
}

// ClusterRoleBindingToYAML 将ClusterRoleBinding转换为YAML
func ClusterRoleBindingToYAML(clusterRoleBinding *rbacv1.ClusterRoleBinding) (string, error) {
	if clusterRoleBinding == nil {
		return "", fmt.Errorf("clusterRoleBinding is nil")
	}

	// 清理不需要的字段
	cleanClusterRoleBinding := clusterRoleBinding.DeepCopy()
	cleanClusterRoleBinding.ManagedFields = nil
	cleanClusterRoleBinding.ResourceVersion = ""
	cleanClusterRoleBinding.UID = ""
	cleanClusterRoleBinding.SelfLink = ""
	cleanClusterRoleBinding.CreationTimestamp = metav1.Time{}
	cleanClusterRoleBinding.Generation = 0

	yamlData, err := yaml.Marshal(cleanClusterRoleBinding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ClusterRoleBinding to yaml: %w", err)
	}

	return string(yamlData), nil
}

// YAMLToClusterRoleBinding 将YAML转换为ClusterRoleBinding
func YAMLToClusterRoleBinding(yamlContent string) (*rbacv1.ClusterRoleBinding, error) {
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err := yaml.Unmarshal([]byte(yamlContent), &clusterRoleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml to ClusterRoleBinding: %w", err)
	}

	return &clusterRoleBinding, nil
}

// GetClusterRoleBindingEvents 获取ClusterRoleBinding相关事件
func GetClusterRoleBindingEvents(ctx context.Context, client kubernetes.Interface, name string, limit int) ([]*model.K8sClusterRoleBindingEvent, int64, error) {
	// 获取与ClusterRoleBinding相关的事件
	events, err := client.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=ClusterRoleBinding", name),
		Limit:         int64(limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	// 转换事件
	k8sEvents := make([]*model.K8sClusterRoleBindingEvent, 0, len(events.Items))
	for _, event := range events.Items {
		k8sEvent := &model.K8sClusterRoleBindingEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    getEventSource(event),
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Count:     event.Count,
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, int64(len(events.Items)), nil
}

// GetClusterRoleBindingUsage 获取ClusterRoleBinding使用情况
func GetClusterRoleBindingUsage(ctx context.Context, client kubernetes.Interface, name string) (*model.K8sClusterRoleBindingUsage, error) {
	// 获取ClusterRoleBinding
	clusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRoleBinding: %w", err)
	}

	usage := &model.K8sClusterRoleBindingUsage{
		ClusterRoleBindingName: clusterRoleBinding.Name,
		ClusterRoleName:        clusterRoleBinding.RoleRef.Name,
		RiskLevel:              "Low",
		CreationTimestamp:      clusterRoleBinding.CreationTimestamp.Time,
		LastUpdated:            time.Now(),
	}

	// 转换主体
	usage.Subjects = convertK8sSubjectsToModel(clusterRoleBinding.Subjects)
	usage.SubjectCount = len(clusterRoleBinding.Subjects)

	// 统计不同类型的主体
	for _, subject := range clusterRoleBinding.Subjects {
		switch subject.Kind {
		case "User":
			usage.UserCount++
		case "Group":
			usage.GroupCount++
		case "ServiceAccount":
			usage.ServiceAccountCount++
		}
	}

	// 检查是否为系统绑定
	usage.IsSystemBinding = model.BoolToBoolValue(strings.HasPrefix(clusterRoleBinding.Name, "system:"))

	// 评估风险等级
	if strings.HasPrefix(clusterRoleBinding.Name, "system:") {
		usage.RiskLevel = "System"
	} else if usage.SubjectCount > 10 {
		usage.RiskLevel = "Medium"
	}

	// 获取关联的ClusterRole信息（如果存在）
	if clusterRole, err := client.RbacV1().ClusterRoles().Get(ctx, clusterRoleBinding.RoleRef.Name, metav1.GetOptions{}); err == nil {
		usage.ClusterRoleInfo = &model.ClusterRoleBindingRoleInfo{
			Name:         clusterRole.Name,
			Kind:         clusterRoleBinding.RoleRef.Kind,
			IsSystemRole: model.BoolToBoolValue(strings.HasPrefix(clusterRole.Name, "system:")),
			CreatedAt:    clusterRole.CreationTimestamp.Time,
		}

		// 转换权限规则
		for _, rule := range clusterRole.Rules {
			usage.ClusterRoleInfo.Permissions = append(usage.ClusterRoleInfo.Permissions, model.PolicyRule{
				APIGroups:       rule.APIGroups,
				Resources:       rule.Resources,
				Verbs:           rule.Verbs,
				ResourceNames:   rule.ResourceNames,
				NonResourceURLs: rule.NonResourceURLs,
			})
		}

		// 评估角色风险等级
		if hasHighRiskPermissions(clusterRole.Rules) {
			usage.RiskLevel = "High"
			usage.ClusterRoleInfo.RiskLevel = "High"
		}
	}

	return usage, nil
}

// GetClusterRoleBindingMetrics 获取ClusterRoleBinding指标
func GetClusterRoleBindingMetrics(ctx context.Context, client kubernetes.Interface, name string) (*model.K8sClusterRoleBindingMetrics, error) {
	// 获取ClusterRoleBinding
	clusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ClusterRoleBinding: %w", err)
	}

	metrics := &model.K8sClusterRoleBindingMetrics{
		ClusterRoleBindingName: clusterRoleBinding.Name,
		ClusterRoleName:        clusterRoleBinding.RoleRef.Name,
		SubjectCount:           len(clusterRoleBinding.Subjects),
		UserCount:              0,
		GroupCount:             0,
		ServiceAccountCount:    0,
		ClusterWideAccess:      model.BoolTrue, // ClusterRoleBinding默认是集群级的
		RiskLevel:              "Low",
		PermissionScope:        "Cluster",
		LastUsed:               time.Now(),
		LastUpdated:            time.Now(),
	}

	// 统计主体类型
	users := make(map[string]bool)
	groups := make(map[string]bool)

	for _, subject := range clusterRoleBinding.Subjects {
		switch subject.Kind {
		case "User":
			users[subject.Name] = true
		case "Group":
			groups[subject.Name] = true
		case "ServiceAccount":
			metrics.ServiceAccountCount++
		}
	}

	metrics.UserCount = len(users)
	metrics.GroupCount = len(groups)

	// 获取关联的ClusterRole权限
	if clusterRole, err := client.RbacV1().ClusterRoles().Get(ctx, clusterRoleBinding.RoleRef.Name, metav1.GetOptions{}); err == nil {
		// 权限数量通过统计规则数量得到
		permissionCount := 0
		for _, rule := range clusterRole.Rules {
			permissionCount += len(rule.Resources) * len(rule.Verbs)
		}

		// 评估安全风险
		if strings.HasPrefix(clusterRoleBinding.Name, "system:") {
			metrics.RiskLevel = "System"
		} else if hasHighRiskPermissions(clusterRole.Rules) {
			metrics.RiskLevel = "High"
		} else if metrics.SubjectCount > 10 || metrics.UserCount > 10 {
			metrics.RiskLevel = "Medium"
		}

		// 构建权限使用统计
		resourceTypes := make(map[string]bool)
		verbs := make(map[string]int)

		for _, rule := range clusterRole.Rules {
			for _, resource := range rule.Resources {
				resourceTypes[resource] = true
			}
			for _, verb := range rule.Verbs {
				verbs[verb]++
			}
		}

		// 转换为切片
		var resourceTypeList []string
		for resource := range resourceTypes {
			resourceTypeList = append(resourceTypeList, resource)
		}

		var mostUsedVerbs []string
		for verb := range verbs {
			mostUsedVerbs = append(mostUsedVerbs, verb)
		}

		// 添加推荐操作暂时省略，因为模型中没有这些字段
		// TODO: 需要在模型中添加 PermissionUsageStats 和 RecommendedActions 字段

		// 临时注释掉，避免编译错误
		/*
			metrics.PermissionUsageStats = &model.ClusterRoleBindingUsageStats{
				ResourceTypes: resourceTypeList,
				MostUsedVerbs: mostUsedVerbs,
			}

			// 添加推荐操作
			if metrics.RiskLevel == "High" {
				metrics.RecommendedActions = append(metrics.RecommendedActions, model.ClusterRoleBindingRecommendation{
					Type:        "security",
					Priority:    "high",
					Description: "检查过度权限，考虑使用最小权限原则",
					Action:      "review_permissions",
					Impact:      "提升安全性",
					CreatedAt:   time.Now(),
				})
			}

			if metrics.SubjectCount > 10 {
				metrics.RecommendedActions = append(metrics.RecommendedActions, model.ClusterRoleBindingRecommendation{
					Type:        "optimization",
					Priority:    "medium",
					Description: "主体数量过多，建议拆分或优化绑定",
					Action:      "optimize_bindings",
					Impact:      "提升管理性",
					CreatedAt:   time.Now(),
				})
			}
		*/
	}

	return metrics, nil
}

// ======================== RoleBinding Utils 工具函数 ========================

// BuildRoleBindingListOptions 构建RoleBinding查询选项
func BuildRoleBindingListOptions(name, labelKey string) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 名称过滤
	if name != "" {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", name)
	}

	// 标签过滤
	if labelKey != "" {
		options.LabelSelector = labelKey
	}

	return options
}

// ConvertToK8sRoleBinding 转换RoleBinding为K8s模型
func ConvertToK8sRoleBinding(roleBinding *rbacv1.RoleBinding, clusterID int) *model.K8sRoleBinding {
	if roleBinding == nil {
		return nil
	}

	// 转换主体列表
	var subjects []model.Subject
	for _, subject := range roleBinding.Subjects {
		subjects = append(subjects, model.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	// 转换角色引用
	roleRef := model.RoleRef{
		Kind:     roleBinding.RoleRef.Kind,
		Name:     roleBinding.RoleRef.Name,
		APIGroup: roleBinding.RoleRef.APIGroup,
	}

	// 计算状态
	status := model.RoleBindingStatusActive
	isOrphaned := false
	isSystemBinding := strings.HasPrefix(roleBinding.Name, "system:")

	return &model.K8sRoleBinding{
		Name:              roleBinding.Name,
		Namespace:         roleBinding.Namespace,
		ClusterID:         clusterID,
		UID:               string(roleBinding.UID),
		Status:            status,
		Labels:            roleBinding.Labels,
		Annotations:       roleBinding.Annotations,
		RoleRef:           roleRef,
		Subjects:          subjects,
		ResourceVersion:   roleBinding.ResourceVersion,
		Age:               time.Since(roleBinding.CreationTimestamp.Time).String(),
		SubjectCount:      len(subjects),
		IsSystemBinding:   model.BoolToBoolValue(isSystemBinding),
		IsOrphaned:        model.BoolToBoolValue(isOrphaned),
		CreationTimestamp: roleBinding.CreationTimestamp.Time,
		RawRoleBinding:    roleBinding,
	}
}

// PaginateK8sRoleBindings 分页处理RoleBinding列表
func PaginateK8sRoleBindings(roleBindings []*model.K8sRoleBinding, page, pageSize int) model.ListResp[*model.K8sRoleBinding] {
	total := len(roleBindings)

	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 计算分页
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return model.ListResp[*model.K8sRoleBinding]{
			Items: []*model.K8sRoleBinding{},
			Total: int64(total),
		}
	}

	if end > total {
		end = total
	}

	return model.ListResp[*model.K8sRoleBinding]{
		Items: roleBindings[start:end],
		Total: int64(total),
	}
}

// BuildK8sRoleBinding 构建RoleBinding对象
func BuildK8sRoleBinding(req *model.CreateRoleBindingReq) *rbacv1.RoleBinding {
	// 转换主体列表
	var subjects []rbacv1.Subject
	for _, subject := range req.Subjects {
		subjects = append(subjects, rbacv1.Subject{
			Kind:      subject.Kind,
			Name:      subject.Name,
			Namespace: subject.Namespace,
			APIGroup:  subject.APIGroup,
		})
	}

	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     req.RoleRef.Kind,
			Name:     req.RoleRef.Name,
			APIGroup: req.RoleRef.APIGroup,
		},
		Subjects: subjects,
	}
}

// RoleBindingToYAML 转换RoleBinding为YAML
func RoleBindingToYAML(roleBinding *rbacv1.RoleBinding) (string, error) {
	yamlData, err := yaml.Marshal(roleBinding)
	if err != nil {
		return "", fmt.Errorf("转换RoleBinding为YAML失败: %w", err)
	}
	return string(yamlData), nil
}

// YAMLToRoleBinding 从YAML转换为RoleBinding
func YAMLToRoleBinding(yamlContent string) (*rbacv1.RoleBinding, error) {
	var roleBinding rbacv1.RoleBinding
	if err := yaml.Unmarshal([]byte(yamlContent), &roleBinding); err != nil {
		return nil, fmt.Errorf("从YAML转换RoleBinding失败: %w", err)
	}
	return &roleBinding, nil
}

// GetRoleBindingEvents 获取RoleBinding事件
func GetRoleBindingEvents(ctx context.Context, client kubernetes.Interface, namespace, name string) (model.ListResp[*model.K8sRoleBindingEvent], error) {
	// 获取RoleBinding相关的事件
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=RoleBinding", name),
	})
	if err != nil {
		return model.ListResp[*model.K8sRoleBindingEvent]{}, fmt.Errorf("获取RoleBinding事件失败: %w", err)
	}

	var roleBindingEvents []*model.K8sRoleBindingEvent
	for _, event := range events.Items {
		roleBindingEvent := &model.K8sRoleBindingEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    event.Source.Component,
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Count:     event.Count,
		}
		roleBindingEvents = append(roleBindingEvents, roleBindingEvent)
	}

	return model.ListResp[*model.K8sRoleBindingEvent]{
		Items: roleBindingEvents,
		Total: int64(len(roleBindingEvents)),
	}, nil
}

// GetRoleBindingUsage 获取RoleBinding使用分析
func GetRoleBindingUsage(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sRoleBindingUsage, error) {
	// 获取RoleBinding
	roleBinding, err := client.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取RoleBinding失败: %w", err)
	}

	// 构建使用分析
	usage := &model.K8sRoleBindingUsage{
		RoleBindingName: roleBinding.Name,
		Namespace:       roleBinding.Namespace,
		RoleName:        roleBinding.RoleRef.Name,
		RoleKind:        roleBinding.RoleRef.Kind,
		LastAnalyzed:    time.Now(),
		AnalysisScore:   75, // 默认评分
	}

	// 分析主体
	subjectAnalysis := model.K8sRoleBindingSubjectSummary{
		RoleBindingName:  roleBinding.Name,
		Namespace:        roleBinding.Namespace,
		TotalPermissions: 0,
		RiskAssessment:   "Medium",
	}

	var users, groups, serviceAccounts []string
	for _, subject := range roleBinding.Subjects {
		switch subject.Kind {
		case "User":
			users = append(users, subject.Name)
		case "Group":
			groups = append(groups, subject.Name)
		case "ServiceAccount":
			serviceAccounts = append(serviceAccounts, subject.Name)
		}
	}
	subjectAnalysis.Users = users
	subjectAnalysis.Groups = groups
	subjectAnalysis.ServiceAccounts = serviceAccounts

	usage.SubjectAnalysis = subjectAnalysis

	// 依赖状态
	dependencyStatus := model.K8sRoleBindingDependency{
		RoleBindingName: roleBinding.Name,
		Namespace:       roleBinding.Namespace,
		RoleName:        roleBinding.RoleRef.Name,
		RoleKind:        roleBinding.RoleRef.Kind,
		RoleExists:      model.BoolTrue, // 简化处理
		IsHealthy:       model.BoolTrue,
	}
	usage.DependencyStatus = dependencyStatus

	// 添加建议
	usage.Recommendations = []string{
		"定期审查绑定的主体",
		"检查权限是否过度授予",
		"考虑使用最小权限原则",
	}

	return usage, nil
}

// GetRoleBindingMetrics 获取RoleBinding指标
func GetRoleBindingMetrics(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sRoleBindingMetrics, error) {
	// 获取RoleBinding
	roleBinding, err := client.RbacV1().RoleBindings(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取RoleBinding失败: %w", err)
	}

	// 计算主体统计
	var userCount, groupCount, serviceAccountCount int
	for _, subject := range roleBinding.Subjects {
		switch subject.Kind {
		case "User":
			userCount++
		case "Group":
			groupCount++
		case "ServiceAccount":
			serviceAccountCount++
		}
	}

	// 构建指标
	metrics := &model.K8sRoleBindingMetrics{
		RoleBindingName: roleBinding.Name,
		Namespace:       roleBinding.Namespace,
		RoleName:        roleBinding.RoleRef.Name,
		RoleKind:        roleBinding.RoleRef.Kind,
		SubjectCount:    len(roleBinding.Subjects),
		UserCount:       userCount,
		GroupCount:      groupCount,
		ServiceAccount:  serviceAccountCount,
		IsActive:        model.BoolTrue,
		IsOrphaned:      model.BoolFalse,
		RiskLevel:       "Medium",
		LastUsed:        time.Now().Add(-24 * time.Hour), // 模拟数据
		LastUpdated:     roleBinding.CreationTimestamp.Time,
	}

	return metrics, nil
}

// ======================== ServiceAccount Utils 工具函数 ========================

// BuildServiceAccountListOptions 构建ServiceAccount查询选项
func BuildServiceAccountListOptions(name, labelKey string) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 名称过滤
	if name != "" {
		options.FieldSelector = fmt.Sprintf("metadata.name=%s", name)
	}

	// 标签过滤
	if labelKey != "" {
		options.LabelSelector = labelKey
	}

	return options
}

// ConvertToK8sServiceAccount 转换ServiceAccount为K8s模型
func ConvertToK8sServiceAccount(serviceAccount *corev1.ServiceAccount, clusterID int) *model.K8sServiceAccount {
	if serviceAccount == nil {
		return nil
	}

	// 计算状态
	status := model.ServiceAccountStatusActive
	isDefault := serviceAccount.Name == "default"

	// 计算令牌数量和镜像拉取密钥数量
	tokenCount := len(serviceAccount.Secrets)
	imagePullSecretsCount := len(serviceAccount.ImagePullSecrets)

	// 检查是否自动挂载服务账户令牌
	autoMount := true
	if serviceAccount.AutomountServiceAccountToken != nil {
		autoMount = *serviceAccount.AutomountServiceAccountToken
	}

	// 转换 Secrets
	var secrets []model.ServiceAccountSecret
	for _, secretRef := range serviceAccount.Secrets {
		secrets = append(secrets, model.ServiceAccountSecret{
			Name:      secretRef.Name,
			Namespace: serviceAccount.Namespace,
			Type:      "Opaque",
		})
	}

	// 转换 ImagePullSecrets
	var imagePullSecrets []model.ServiceAccountSecret
	for _, imagePullSecretRef := range serviceAccount.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, model.ServiceAccountSecret{
			Name:      imagePullSecretRef.Name,
			Namespace: serviceAccount.Namespace,
			Type:      "kubernetes.io/dockerconfigjson",
		})
	}

	return &model.K8sServiceAccount{
		Name:                         serviceAccount.Name,
		Namespace:                    serviceAccount.Namespace,
		ClusterID:                    clusterID,
		UID:                          string(serviceAccount.UID),
		Status:                       status,
		Labels:                       serviceAccount.Labels,
		Annotations:                  serviceAccount.Annotations,
		AutomountServiceAccountToken: model.PtrBoolToPtrBoolValue(&autoMount),
		Secrets:                      secrets,
		ImagePullSecrets:             imagePullSecrets,
		SecretsCount:                 tokenCount,
		ImagePullSecretsCount:        imagePullSecretsCount,
		IsSystemAccount:              model.BoolToBoolValue(isDefault),
		Age:                          time.Since(serviceAccount.CreationTimestamp.Time).String(),
		CreationTimestamp:            serviceAccount.CreationTimestamp.Time,
		ResourceVersion:              serviceAccount.ResourceVersion,
		RawServiceAccount:            serviceAccount,
	}
}

// PaginateK8sServiceAccounts 分页处理ServiceAccount列表
func PaginateK8sServiceAccounts(serviceAccounts []*model.K8sServiceAccount, page, pageSize int) model.ListResp[*model.K8sServiceAccount] {
	total := len(serviceAccounts)

	// 默认分页参数
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 计算分页
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return model.ListResp[*model.K8sServiceAccount]{
			Items: []*model.K8sServiceAccount{},
			Total: int64(total),
		}
	}

	if end > total {
		end = total
	}

	return model.ListResp[*model.K8sServiceAccount]{
		Items: serviceAccounts[start:end],
		Total: int64(total),
	}
}

// BuildK8sServiceAccount 构建ServiceAccount对象
func BuildK8sServiceAccount(req *model.CreateServiceAccountReq) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
	}

	if req.AutomountServiceAccountToken != nil {
		serviceAccount.AutomountServiceAccountToken = model.PtrBoolValueToPtrBool(req.AutomountServiceAccountToken)
	}

	return serviceAccount
}

// ServiceAccountToYAML 转换ServiceAccount为YAML
func ServiceAccountToYAML(serviceAccount *corev1.ServiceAccount) (string, error) {
	// 清理不需要的字段
	serviceAccountCopy := serviceAccount.DeepCopy()
	serviceAccountCopy.ManagedFields = nil

	yamlData, err := yaml.Marshal(serviceAccountCopy)
	if err != nil {
		return "", fmt.Errorf("转换ServiceAccount为YAML失败: %w", err)
	}
	return string(yamlData), nil
}

// YAMLToServiceAccount 从YAML转换为ServiceAccount
func YAMLToServiceAccount(yamlContent string) (*corev1.ServiceAccount, error) {
	var serviceAccount corev1.ServiceAccount
	if err := yaml.Unmarshal([]byte(yamlContent), &serviceAccount); err != nil {
		return nil, fmt.Errorf("从YAML转换ServiceAccount失败: %w", err)
	}
	return &serviceAccount, nil
}

// GetServiceAccountEvents 获取ServiceAccount事件
func GetServiceAccountEvents(ctx context.Context, client kubernetes.Interface, namespace, name string) (model.ListResp[*model.K8sServiceAccountEvent], error) {
	// 获取ServiceAccount相关的事件
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=ServiceAccount", name),
	})
	if err != nil {
		return model.ListResp[*model.K8sServiceAccountEvent]{}, fmt.Errorf("获取ServiceAccount事件失败: %w", err)
	}

	var serviceAccountEvents []*model.K8sServiceAccountEvent
	for _, event := range events.Items {
		serviceAccountEvent := &model.K8sServiceAccountEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    event.Source.Component,
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Count:     event.Count,
		}
		serviceAccountEvents = append(serviceAccountEvents, serviceAccountEvent)
	}

	return model.ListResp[*model.K8sServiceAccountEvent]{
		Items: serviceAccountEvents,
		Total: int64(len(serviceAccountEvents)),
	}, nil
}

// GetServiceAccountUsage 获取ServiceAccount使用分析
func GetServiceAccountUsage(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sServiceAccountUsage, error) {
	// 获取ServiceAccount
	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取ServiceAccount失败: %w", err)
	}

	// 查找使用该ServiceAccount的Pod
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.serviceAccount=%s", name),
	})
	if err != nil {
		// 忽略错误，继续分析
		pods = &corev1.PodList{}
	}

	// 转换 Secrets
	var secrets []model.ServiceAccountSecret
	for _, secretRef := range serviceAccount.Secrets {
		secrets = append(secrets, model.ServiceAccountSecret{
			Name:      secretRef.Name,
			Namespace: serviceAccount.Namespace,
			Type:      "Opaque",
		})
	}

	// 转换 ImagePullSecrets
	var imagePullSecrets []model.ServiceAccountSecret
	for _, imagePullSecretRef := range serviceAccount.ImagePullSecrets {
		imagePullSecrets = append(imagePullSecrets, model.ServiceAccountSecret{
			Name:      imagePullSecretRef.Name,
			Namespace: serviceAccount.Namespace,
			Type:      "kubernetes.io/dockerconfigjson",
		})
	}

	// 分析使用的Pod列表
	var podNames []string
	for _, pod := range pods.Items {
		podNames = append(podNames, pod.Name)
	}

	// 构建使用分析
	usage := &model.K8sServiceAccountUsage{
		ServiceAccountName: serviceAccount.Name,
		Namespace:          serviceAccount.Namespace,
		Secrets:            secrets,
		ImagePullSecrets:   imagePullSecrets,
		UsedByPods:         podNames,
		IsUsed:             model.BoolToBoolValue(len(pods.Items) > 0),
		RiskLevel:          "Low",
	}

	// 风险评估
	if len(serviceAccount.Secrets) > 5 {
		usage.RiskLevel = "Medium"
	}
	autoMount := serviceAccount.AutomountServiceAccountToken == nil || *serviceAccount.AutomountServiceAccountToken
	if !autoMount && len(pods.Items) > 0 {
		usage.RiskLevel = "High"
	}

	return usage, nil
}

// GetServiceAccountMetrics 获取ServiceAccount指标
func GetServiceAccountMetrics(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sServiceAccountMetrics, error) {
	// 获取ServiceAccount
	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取ServiceAccount失败: %w", err)
	}

	// 查找使用该ServiceAccount的Pod
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.serviceAccount=%s", name),
	})
	if err != nil {
		pods = &corev1.PodList{}
	}

	// 构建指标
	metrics := &model.K8sServiceAccountMetrics{
		ServiceAccountName:    serviceAccount.Name,
		Namespace:             serviceAccount.Namespace,
		TotalSecrets:          len(serviceAccount.Secrets),
		TotalImagePullSecrets: len(serviceAccount.ImagePullSecrets),
		PodsUsingAccount:      len(pods.Items),
		IsActive:              model.BoolToBoolValue(len(pods.Items) > 0),
		AutomountEnabled:      model.BoolToBoolValue(serviceAccount.AutomountServiceAccountToken == nil || *serviceAccount.AutomountServiceAccountToken),
		SecurityRisk:          "Low",
		LastUsed:              time.Now().Add(-24 * time.Hour), // 模拟数据
		LastUpdated:           serviceAccount.CreationTimestamp.Time,
	}

	// 风险等级评估
	if metrics.TotalSecrets > 5 {
		metrics.SecurityRisk = "Medium"
	}
	if model.BoolValueToBool(metrics.AutomountEnabled) == false && metrics.PodsUsingAccount > 0 {
		metrics.SecurityRisk = "High"
	}

	return metrics, nil
}

// GetServiceAccountToken 获取ServiceAccount令牌
func GetServiceAccountToken(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sServiceAccountToken, error) {
	// 获取ServiceAccount
	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取ServiceAccount失败: %w", err)
	}

	// 查找关联的Secret
	var tokenSecret *corev1.Secret
	for _, secretRef := range serviceAccount.Secrets {
		secret, err := client.CoreV1().Secrets(namespace).Get(ctx, secretRef.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			tokenSecret = secret
			break
		}
	}

	if tokenSecret == nil {
		return nil, fmt.Errorf("未找到ServiceAccount令牌")
	}

	token := &model.K8sServiceAccountToken{
		Token:     string(tokenSecret.Data["token"]),
		CreatedAt: tokenSecret.CreationTimestamp.Time,
	}

	return token, nil
}

// CreateServiceAccountToken 创建ServiceAccount令牌
func CreateServiceAccountToken(ctx context.Context, client kubernetes.Interface, namespace, name string, expiryTime *int64) (*model.K8sServiceAccountToken, error) {
	// 创建TokenRequest
	tokenRequest := &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: []string{"https://kubernetes.default.svc.cluster.local"},
		},
	}

	if expiryTime != nil {
		tokenRequest.Spec.ExpirationSeconds = expiryTime
	}

	// 创建令牌
	tokenResponse, err := client.CoreV1().ServiceAccounts(namespace).CreateToken(ctx, name, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建ServiceAccount令牌失败: %w", err)
	}

	token := &model.K8sServiceAccountToken{
		Token:     tokenResponse.Status.Token,
		CreatedAt: time.Now(),
	}

	if !tokenResponse.Status.ExpirationTimestamp.IsZero() {
		token.ExpirationTimestamp = &tokenResponse.Status.ExpirationTimestamp.Time
	}

	return token, nil
}

// ======================== Role Utils 工具函数 ========================

// BuildRoleListOptions 构建Role查询选项
func BuildRoleListOptions(req *model.GetRoleListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 名称过滤
	if req.Keyword != "" {
		// Kubernetes的标签选择器不支持模糊搜索，所以我们在获取后过滤
		// 这里只是返回基本的ListOptions
	}

	return options
}

// ConvertToK8sRole 将 rbacv1.Role 转换为 model.K8sRole
func ConvertToK8sRole(role *rbacv1.Role) *model.K8sRole {
	if role == nil {
		return nil
	}

	// 转换权限规则
	rules := make([]model.PolicyRule, 0, len(role.Rules))
	for _, rule := range role.Rules {
		rules = append(rules, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	// 确定状态
	status := model.RoleStatusActive
	if strings.HasPrefix(role.Name, "system:") {
		// 系统角色
	}

	return &model.K8sRole{
		Name:              role.Name,
		Namespace:         role.Namespace,
		UID:               string(role.UID),
		Status:            status,
		Labels:            role.Labels,
		Annotations:       role.Annotations,
		Rules:             rules,
		ResourceVersion:   role.ResourceVersion,
		Age:               duration.HumanDuration(time.Since(role.CreationTimestamp.Time)),
		IsSystemRole:      model.BoolToBoolValue(strings.HasPrefix(role.Name, "system:")),
		CreationTimestamp: role.CreationTimestamp.Time,
		RawRole:           role,
	}
}

// PaginateK8sRoles 分页处理Role列表
func PaginateK8sRoles(roles []*model.K8sRole, page, size int) ([]*model.K8sRole, int64) {
	total := int64(len(roles))

	// 计算分页
	start := (page - 1) * size
	end := start + size

	if start >= len(roles) {
		return []*model.K8sRole{}, total
	}

	if end > len(roles) {
		end = len(roles)
	}

	return roles[start:end], total
}

// BuildK8sRole 构建K8sRole详细信息
func BuildK8sRole(ctx context.Context, clusterID int, namespace string, role rbacv1.Role) (*model.K8sRole, error) {
	k8sRole := ConvertToK8sRole(&role)
	if k8sRole == nil {
		return nil, fmt.Errorf("failed to convert Role")
	}

	k8sRole.ClusterID = clusterID

	// TODO: 这里可以添加更多的详细信息构建逻辑
	// 比如获取绑定信息、使用情况等

	return k8sRole, nil
}

// RoleToYAML 将Role转换为YAML
func RoleToYAML(role *rbacv1.Role) (string, error) {
	if role == nil {
		return "", fmt.Errorf("role is nil")
	}

	// 清理不需要的字段
	cleanRole := role.DeepCopy()
	cleanRole.ManagedFields = nil
	cleanRole.ResourceVersion = ""
	cleanRole.UID = ""
	cleanRole.SelfLink = ""
	cleanRole.CreationTimestamp = metav1.Time{}
	cleanRole.Generation = 0

	yamlData, err := yaml.Marshal(cleanRole)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Role to yaml: %w", err)
	}

	return string(yamlData), nil
}

// YAMLToRole 将YAML转换为Role
func YAMLToRole(yamlContent string) (*rbacv1.Role, error) {
	var role rbacv1.Role
	err := yaml.Unmarshal([]byte(yamlContent), &role)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml to Role: %w", err)
	}

	return &role, nil
}

// GetRoleEvents 获取Role相关事件
func GetRoleEvents(ctx context.Context, client kubernetes.Interface, namespace, name string, limit int) ([]*model.K8sRoleEvent, int64, error) {
	// 获取与Role相关的事件
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Role", name),
		Limit:         int64(limit),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
	}

	// 转换事件
	k8sEvents := make([]*model.K8sRoleEvent, 0, len(events.Items))
	for _, event := range events.Items {
		k8sEvent := &model.K8sRoleEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    getEventSource(event),
			FirstTime: event.FirstTimestamp.Time,
			LastTime:  event.LastTimestamp.Time,
			Count:     event.Count,
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, int64(len(events.Items)), nil
}

// GetRoleUsage 获取Role使用情况
func GetRoleUsage(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sRoleUsage, error) {
	usage := &model.K8sRoleUsage{
		RoleName:  name,
		Namespace: namespace,
		IsUsed:    model.BoolFalse,
		RiskLevel: "Low",
	}

	// 获取所有RoleBinding
	roleBindings, err := client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list RoleBindings: %w", err)
	}

	// 检查RoleBinding
	for _, rb := range roleBindings.Items {
		if rb.RoleRef.Kind == "Role" && rb.RoleRef.Name == name {
			usage.IsUsed = model.BoolTrue
			usage.Bindings = append(usage.Bindings, model.RoleBindingSimpleInfo{
				Name:      rb.Name,
				Namespace: rb.Namespace,
				Subjects:  convertK8sSubjectsToModel(rb.Subjects),
			})

			// 添加主体
			for _, subject := range rb.Subjects {
				usage.Subjects = append(usage.Subjects, model.Subject{
					Kind:      subject.Kind,
					Name:      subject.Name,
					Namespace: subject.Namespace,
					APIGroup:  subject.APIGroup,
				})
			}
		}
	}

	// 获取Role权限
	role, err := client.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Role: %w", err)
	}

	// 转换权限规则
	for _, rule := range role.Rules {
		usage.Permissions = append(usage.Permissions, model.PolicyRule{
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			Verbs:           rule.Verbs,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	// 评估风险等级
	if strings.HasPrefix(name, "system:") {
		usage.RiskLevel = "System"
	} else if hasHighRiskPermissions(role.Rules) {
		usage.RiskLevel = "High"
	} else if len(usage.Subjects) > 10 {
		usage.RiskLevel = "Medium"
	}

	return usage, nil
}

// GetRoleMetrics 获取Role指标
func GetRoleMetrics(ctx context.Context, client kubernetes.Interface, namespace, name string) (*model.K8sRoleMetrics, error) {
	metrics := &model.K8sRoleMetrics{
		RoleName:        name,
		Namespace:       namespace,
		TotalBindings:   0,
		ActiveUsers:     0,
		ActiveGroups:    0,
		ServiceAccounts: 0,
		PermissionCount: 0,
		SecurityRisk:    "Low",
		LastUpdated:     time.Now(),
	}

	// 获取Role
	role, err := client.RbacV1().Roles(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Role: %w", err)
	}

	// 计算权限数量
	metrics.PermissionCount = len(role.Rules)

	// 统计RoleBinding
	roleBindings, err := client.RbacV1().RoleBindings(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		users := make(map[string]bool)
		groups := make(map[string]bool)

		for _, rb := range roleBindings.Items {
			if rb.RoleRef.Kind == "Role" && rb.RoleRef.Name == name {
				metrics.TotalBindings++

				for _, subject := range rb.Subjects {
					switch subject.Kind {
					case "User":
						users[subject.Name] = true
					case "Group":
						groups[subject.Name] = true
					case "ServiceAccount":
						metrics.ServiceAccounts++
					}
				}
			}
		}

		metrics.ActiveUsers = len(users)
		metrics.ActiveGroups = len(groups)
	}

	// 评估安全风险
	if strings.HasPrefix(name, "system:") {
		metrics.SecurityRisk = "System"
	} else if hasHighRiskPermissions(role.Rules) {
		metrics.SecurityRisk = "High"
	} else if metrics.TotalBindings > 5 || metrics.ActiveUsers > 10 {
		metrics.SecurityRisk = "Medium"
	}

	return metrics, nil
}
