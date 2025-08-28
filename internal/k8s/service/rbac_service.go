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

package service

import (
	"context"
	"fmt"
	"strings"

	authorizationv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RBACService struct {
	k8sClient client.K8sClient
}

func NewRBACService(k8sClient client.K8sClient) *RBACService {
	return &RBACService{
		k8sClient: k8sClient,
	}
}

// GetRBACStatistics 获取RBAC统计信息
func (rs *RBACService) GetRBACStatistics(ctx context.Context, clusterID int) (*model.RBACStatistics, error) {
	k8sClient, err := rs.k8sClient.GetKubeClient(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	stats := &model.RBACStatistics{}

	// 统计Roles
	roles, err := k8sClient.RbacV1().Roles("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalRoles = len(roles.Items)
	}

	// 统计ClusterRoles
	clusterRoles, err := k8sClient.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalClusterRoles = len(clusterRoles.Items)
		// 统计系统和自定义角色
		for _, cr := range clusterRoles.Items {
			if strings.HasPrefix(cr.Name, "system:") {
				stats.SystemRoles++
			} else {
				stats.CustomRoles++
			}
		}
	}

	// 统计RoleBindings
	roleBindings, err := k8sClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalRoleBindings = len(roleBindings.Items)
	}

	// 统计ClusterRoleBindings
	clusterRoleBindings, err := k8sClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err == nil {
		stats.TotalClusterRoleBindings = len(clusterRoleBindings.Items)
	}

	// 统计活跃主体
	activeSubjects := make(map[string]bool)
	if roleBindings != nil {
		for _, rb := range roleBindings.Items {
			for _, subject := range rb.Subjects {
				key := fmt.Sprintf("%s:%s:%s", subject.Kind, subject.Name, subject.Namespace)
				activeSubjects[key] = true
			}
		}
	}
	if clusterRoleBindings != nil {
		for _, crb := range clusterRoleBindings.Items {
			for _, subject := range crb.Subjects {
				key := fmt.Sprintf("%s:%s:%s", subject.Kind, subject.Name, subject.Namespace)
				activeSubjects[key] = true
			}
		}
	}
	stats.ActiveSubjects = len(activeSubjects)

	return stats, nil
}

// CheckPermissions 检查权限
func (rs *RBACService) CheckPermissions(ctx context.Context, req *model.CheckPermissionsReq) ([]model.PermissionResult, error) {
	k8sClient, err := rs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	var results []model.PermissionResult

	for _, resource := range req.Resources {
		// 构建SubjectAccessReview
		sar := &authorizationv1.SubjectAccessReview{
			Spec: authorizationv1.SubjectAccessReviewSpec{
				User:   req.Subject.Name,
				Groups: []string{req.Subject.APIGroup},
				ResourceAttributes: &authorizationv1.ResourceAttributes{
					Namespace: resource.Namespace,
					Verb:      resource.Verb,
					Resource:  resource.Resource,
				},
			},
		}

		// 如果主体是ServiceAccount，设置UID
		if req.Subject.Kind == "ServiceAccount" {
			sar.Spec.UID = req.Subject.Name
		}

		// 执行权限检查
		result, err := k8sClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
		if err != nil {
			results = append(results, model.PermissionResult{
				Namespace: resource.Namespace,
				Resource:  resource.Resource,
				Verb:      resource.Verb,
				Allowed:   false,
				Reason:    fmt.Sprintf("failed to check permission: %v", err),
			})
			continue
		}

		results = append(results, model.PermissionResult{
			Namespace: resource.Namespace,
			Resource:  resource.Resource,
			Verb:      resource.Verb,
			Allowed:   result.Status.Allowed,
			Reason:    result.Status.Reason,
		})
	}

	return results, nil
}

// GetSubjectPermissions 获取主体的有效权限列表
func (rs *RBACService) GetSubjectPermissions(ctx context.Context, req *model.SubjectPermissionsReq) (*model.SubjectPermissionsResponse, error) {
	k8sClient, err := rs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	response := &model.SubjectPermissionsResponse{
		Subject:      req.Subject,
		Permissions:  []model.PolicyRule{},
		Roles:        []string{},
		ClusterRoles: []string{},
	}

	// 获取所有RoleBindings
	roleBindings, err := k8sClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	// 检查RoleBindings中的权限
	for _, rb := range roleBindings.Items {
		if rs.isSubjectInBinding(req.Subject, rb.Subjects) {
			// 获取对应的Role
			if rb.RoleRef.Kind == "Role" {
				role, err := k8sClient.RbacV1().Roles(rb.Namespace).Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
				if err == nil {
					response.Roles = append(response.Roles, fmt.Sprintf("%s/%s", rb.Namespace, role.Name))
					for _, rule := range role.Rules {
						response.Permissions = append(response.Permissions, model.PolicyRule{
							APIGroups:       rule.APIGroups,
							Resources:       rule.Resources,
							Verbs:           rule.Verbs,
							ResourceNames:   rule.ResourceNames,
							NonResourceURLs: rule.NonResourceURLs,
						})
					}
				}
			} else if rb.RoleRef.Kind == "ClusterRole" {
				clusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, rb.RoleRef.Name, metav1.GetOptions{})
				if err == nil {
					response.ClusterRoles = append(response.ClusterRoles, clusterRole.Name)
					for _, rule := range clusterRole.Rules {
						response.Permissions = append(response.Permissions, model.PolicyRule{
							APIGroups:       rule.APIGroups,
							Resources:       rule.Resources,
							Verbs:           rule.Verbs,
							ResourceNames:   rule.ResourceNames,
							NonResourceURLs: rule.NonResourceURLs,
						})
					}
				}
			}
		}
	}

	// 获取所有ClusterRoleBindings
	clusterRoleBindings, err := k8sClient.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings: %w", err)
	}

	// 检查ClusterRoleBindings中的权限
	for _, crb := range clusterRoleBindings.Items {
		if rs.isSubjectInBinding(req.Subject, crb.Subjects) {
			clusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, crb.RoleRef.Name, metav1.GetOptions{})
			if err == nil {
				response.ClusterRoles = append(response.ClusterRoles, clusterRole.Name)
				for _, rule := range clusterRole.Rules {
					response.Permissions = append(response.Permissions, model.PolicyRule{
						APIGroups:       rule.APIGroups,
						Resources:       rule.Resources,
						Verbs:           rule.Verbs,
						ResourceNames:   rule.ResourceNames,
						NonResourceURLs: rule.NonResourceURLs,
					})
				}
			}
		}
	}

	return response, nil
}

// GetResourceVerbs 获取预定义的资源和动作列表
func (rs *RBACService) GetResourceVerbs(ctx context.Context) (*model.ResourceVerbsResponse, error) {
	// 预定义常用的Kubernetes资源和动作
	resources := []model.ResourceInfo{
		// Core resources
		{APIGroup: "", Resource: "pods", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "po"},
		{APIGroup: "", Resource: "services", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "svc"},
		{APIGroup: "", Resource: "configmaps", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "cm"},
		{APIGroup: "", Resource: "secrets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "", Resource: "persistentvolumes", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "pv"},
		{APIGroup: "", Resource: "persistentvolumeclaims", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "pvc"},
		{APIGroup: "", Resource: "namespaces", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ns"},
		{APIGroup: "", Resource: "nodes", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "", Resource: "serviceaccounts", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "sa"},
		{APIGroup: "", Resource: "events", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},

		// Apps resources
		{APIGroup: "apps", Resource: "deployments", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "deploy"},
		{APIGroup: "apps", Resource: "statefulsets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "sts"},
		{APIGroup: "apps", Resource: "daemonsets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ds"},
		{APIGroup: "apps", Resource: "replicasets", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "rs"},

		// RBAC resources
		{APIGroup: "rbac.authorization.k8s.io", Resource: "roles", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "clusterroles", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "rolebindings", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "rbac.authorization.k8s.io", Resource: "clusterrolebindings", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},

		// Networking resources
		{APIGroup: "networking.k8s.io", Resource: "ingresses", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "ing"},
		{APIGroup: "networking.k8s.io", Resource: "networkpolicies", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "netpol"},

		// Batch resources
		{APIGroup: "batch", Resource: "jobs", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}},
		{APIGroup: "batch", Resource: "cronjobs", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "cj"},

		// Autoscaling resources
		{APIGroup: "autoscaling", Resource: "horizontalpodautoscalers", Verbs: []string{"get", "list", "create", "update", "patch", "delete", "watch"}, Shortname: "hpa"},

		// Metrics resources
		{APIGroup: "metrics.k8s.io", Resource: "nodes", Verbs: []string{"get", "list"}},
		{APIGroup: "metrics.k8s.io", Resource: "pods", Verbs: []string{"get", "list"}},

		// Custom resources (example)
		{APIGroup: "*", Resource: "*", Verbs: []string{"*"}},
	}

	return &model.ResourceVerbsResponse{
		Resources: resources,
	}, nil
}

// 辅助方法：检查主体是否在绑定列表中
func (rs *RBACService) isSubjectInBinding(subject model.Subject, subjects []rbacv1.Subject) bool {
	for _, s := range subjects {
		if s.Kind == subject.Kind && s.Name == subject.Name {
			// 对于ServiceAccount，还需要检查命名空间
			if subject.Kind == "ServiceAccount" {
				return s.Namespace == subject.Namespace
			}
			return true
		}
	}
	return false
}
