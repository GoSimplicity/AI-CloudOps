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
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

const rbacBindingPageSize = 100

type RBACService interface {
	AnalyzeRBACPermissions(ctx context.Context, req *model.AnalyzeRBACPermissionsReq) (*model.EffectivePermissions, error)
	CheckRBACPermission(ctx context.Context, req *model.CheckRBACPermissionReq) (*model.PermissionCheckResult, error)
}

// rbacService RBAC权限管理服务实现
type rbacService struct {
	roleService               RoleService
	clusterRoleService        ClusterRoleService
	roleBindingService        RoleBindingService
	clusterRoleBindingService ClusterRoleBindingService
	logger                    *zap.Logger
}

func NewRBACService(
	roleService RoleService,
	clusterRoleService ClusterRoleService,
	roleBindingService RoleBindingService,
	clusterRoleBindingService ClusterRoleBindingService,
	logger *zap.Logger,
) RBACService {
	return &rbacService{
		roleService:               roleService,
		clusterRoleService:        clusterRoleService,
		roleBindingService:        roleBindingService,
		clusterRoleBindingService: clusterRoleBindingService,
		logger:                    logger,
	}
}

// AnalyzeRBACPermissions 分析用户的有效权限
func (s *rbacService) AnalyzeRBACPermissions(ctx context.Context, req *model.AnalyzeRBACPermissionsReq) (*model.EffectivePermissions, error) {
	if req == nil {
		return nil, fmt.Errorf("分析RBAC权限请求不能为空")
	}

	result := &model.EffectivePermissions{
		Subject:     req.Subject,
		ClusterID:   req.ClusterID,
		Permissions: make(map[string][]string),
		Sources:     []model.PermissionSource{},
	}

	if err := validateRBACSubject(req.ClusterID, req.Subject); err != nil {
		return nil, err
	}

	s.logger.Info("开始分析RBAC权限",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("subject_kind", req.Subject.Kind),
		zap.String("subject_name", req.Subject.Name))

	grants, err := s.grantsForSubject(ctx, req.ClusterID, req.Subject, "", true)
	if err != nil {
		return nil, err
	}
	permissions := map[string]map[string]struct{}{}
	for _, grant := range grants {
		if !grant.contributed {
			continue
		}
		for _, rule := range grant.rules {
			for _, resource := range rule.Resources {
				for _, verb := range rule.Verbs {
					if permissions[resource] == nil {
						permissions[resource] = map[string]struct{}{}
					}
					permissions[resource][verb] = struct{}{}
				}
			}
		}
		result.Sources = append(result.Sources, grant.source)
	}
	for resource, verbs := range permissions {
		result.Permissions[resource] = sortedKeys(verbs)
	}

	s.logger.Info("RBAC权限分析完成", zap.Int("cluster_id", req.ClusterID), zap.Int("sources", len(result.Sources)))
	return result, nil
}

func (s *rbacService) CheckRBACPermission(ctx context.Context, req *model.CheckRBACPermissionReq) (*model.PermissionCheckResult, error) {
	if req == nil {
		return nil, fmt.Errorf("检查RBAC权限请求不能为空")
	}

	result := &model.PermissionCheckResult{
		Allowed: false,
	}

	if err := validateRBACSubject(req.ClusterID, req.Subject); err != nil {
		return nil, err
	}
	if req.Resource == "" || req.Verb == "" {
		return nil, fmt.Errorf("资源和动作不能为空")
	}

	s.logger.Debug("检查RBAC权限",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("subject", fmt.Sprintf("%s/%s", req.Subject.Kind, req.Subject.Name)),
		zap.String("resource", req.Resource),
		zap.String("verb", req.Verb),
		zap.String("namespace", req.Namespace))

	grants, err := s.grantsForSubject(ctx, req.ClusterID, req.Subject, req.Namespace, false)
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		for _, rule := range grant.rules {
			if ruleAllows(rule, req.Resource, req.Verb) {
				result.Allowed = true
				result.Source = grant.source.BindingName
				result.Reason = grant.source.Type + "/" + grant.source.Name
				return result, nil
			}
		}
	}
	result.Reason = "当前主体没有该资源权限"
	return result, nil
}

type rbacGrant struct {
	rules       []model.PolicyRule
	source      model.PermissionSource
	contributed bool
}

func validateRBACSubject(clusterID int, subject model.Subject) error {
	if clusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if subject.Kind == "" || subject.Name == "" {
		return fmt.Errorf("主体信息不完整")
	}
	return nil
}

// grantsForSubject 汇总主体命中的 RoleBinding 与 ClusterRoleBinding。
// allNamespaces 为真时纳入全部命名空间的 RoleBinding（权限分析）。
// 否则 namespace 为空只看 ClusterRoleBinding，非空再加入该命名空间的 RoleBinding。
func (s *rbacService) grantsForSubject(ctx context.Context, clusterID int, subject model.Subject, namespace string, allNamespaces bool) ([]rbacGrant, error) {
	var grants []rbacGrant
	if allNamespaces || namespace != "" {
		listNamespace := ""
		if !allNamespaces {
			listNamespace = namespace
		}
		bindings, err := s.listRoleBindings(ctx, clusterID, listNamespace)
		if err != nil {
			return nil, err
		}
		for _, binding := range bindings {
			if binding == nil || binding.Namespace == "" || !subjectListed(binding.Subjects, subject) {
				continue
			}
			if !allNamespaces && binding.Namespace != namespace {
				continue
			}
			grant, err := s.grantForRoleRef(ctx, clusterID, binding.Namespace, binding.Name, binding.RoleRef)
			if err != nil {
				return nil, err
			}
			if grant == nil {
				continue
			}
			grants = append(grants, *grant)
		}
	}

	clusterBindings, err := s.listClusterRoleBindings(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	for _, binding := range clusterBindings {
		if binding == nil || !subjectListed(binding.Subjects, subject) {
			continue
		}
		if binding.RoleRef.Kind != "" && binding.RoleRef.Kind != "ClusterRole" {
			s.logger.Warn("ClusterRoleBinding 引用了非 ClusterRole", zap.String("name", binding.Name), zap.String("kind", binding.RoleRef.Kind))
			continue
		}
		grant, err := s.grantForRoleRef(ctx, clusterID, "", binding.Name, binding.RoleRef)
		if err != nil {
			return nil, err
		}
		if grant == nil {
			continue
		}
		grants = append(grants, *grant)
	}
	return grants, nil
}

func (s *rbacService) grantForRoleRef(ctx context.Context, clusterID int, namespace, bindingName string, roleRef model.RoleRef) (*rbacGrant, error) {
	kind := roleRef.Kind
	if kind == "" {
		if namespace == "" {
			kind = "ClusterRole"
		} else {
			kind = "Role"
		}
	}
	var rules []model.PolicyRule
	switch kind {
	case "Role":
		if namespace == "" {
			return nil, nil
		}
		if s.roleService == nil {
			return nil, fmt.Errorf("读取Role规则失败: 服务未初始化")
		}
		role, err := s.roleService.GetRoleDetails(ctx, &model.GetRoleDetailsReq{ClusterID: clusterID, Namespace: namespace, Name: roleRef.Name})
		if err != nil {
			return nil, fmt.Errorf("读取Role规则失败: %w", err)
		}
		if role == nil {
			return nil, fmt.Errorf("读取Role规则失败: Role %s/%s 不存在", namespace, roleRef.Name)
		}
		rules = role.Rules
	case "ClusterRole":
		if s.clusterRoleService == nil {
			return nil, fmt.Errorf("读取ClusterRole规则失败: 服务未初始化")
		}
		role, err := s.clusterRoleService.GetClusterRoleDetails(ctx, &model.GetClusterRoleDetailsReq{ClusterID: clusterID, Name: roleRef.Name})
		if err != nil {
			return nil, fmt.Errorf("读取ClusterRole规则失败: %w", err)
		}
		if role == nil {
			return nil, fmt.Errorf("读取ClusterRole规则失败: ClusterRole %s 不存在", roleRef.Name)
		}
		rules = role.Rules
	default:
		s.logger.Warn("忽略无法识别的角色引用", zap.String("binding", bindingName), zap.String("kind", roleRef.Kind))
		return nil, nil
	}

	return &rbacGrant{
		rules: rules,
		source: model.PermissionSource{
			Type:        kind,
			Name:        roleRef.Name,
			Namespace:   namespace,
			BindingName: bindingName,
		},
		contributed: len(rules) > 0,
	}, nil
}

func (s *rbacService) listRoleBindings(ctx context.Context, clusterID int, namespace string) ([]*model.K8sRoleBinding, error) {
	if s.roleBindingService == nil {
		return nil, fmt.Errorf("查询RoleBinding失败: 服务未初始化")
	}
	var all []*model.K8sRoleBinding
	for page := 1; page <= 1000; page++ {
		resp, err := s.roleBindingService.GetRoleBindingList(ctx, &model.GetRoleBindingListReq{
			ListReq:   model.ListReq{Page: page, Size: rbacBindingPageSize},
			ClusterID: clusterID,
			Namespace: namespace,
		})
		if err != nil {
			return nil, fmt.Errorf("查询RoleBinding失败: %w", err)
		}
		all = append(all, resp.Items...)
		if len(resp.Items) == 0 || int64(len(all)) >= resp.Total || len(resp.Items) < rbacBindingPageSize {
			return all, nil
		}
	}
	return nil, fmt.Errorf("查询RoleBinding失败: 结果页数超出上限")
}

func (s *rbacService) listClusterRoleBindings(ctx context.Context, clusterID int) ([]*model.K8sClusterRoleBinding, error) {
	if s.clusterRoleBindingService == nil {
		return nil, fmt.Errorf("查询ClusterRoleBinding失败: 服务未初始化")
	}
	var all []*model.K8sClusterRoleBinding
	for page := 1; page <= 1000; page++ {
		resp, err := s.clusterRoleBindingService.GetClusterRoleBindingList(ctx, &model.GetClusterRoleBindingListReq{
			ListReq:   model.ListReq{Page: page, Size: rbacBindingPageSize},
			ClusterID: clusterID,
		})
		if err != nil {
			return nil, fmt.Errorf("查询ClusterRoleBinding失败: %w", err)
		}
		all = append(all, resp.Items...)
		if len(resp.Items) == 0 || int64(len(all)) >= resp.Total || len(resp.Items) < rbacBindingPageSize {
			return all, nil
		}
	}
	return nil, fmt.Errorf("查询ClusterRoleBinding失败: 结果页数超出上限")
}

func subjectListed(subjects []model.Subject, request model.Subject) bool {
	for _, subject := range subjects {
		if subject.Kind != request.Kind || subject.Name != request.Name || subject.Name == "" {
			continue
		}
		if request.Kind == "ServiceAccount" && subject.Namespace != request.Namespace {
			continue
		}
		return true
	}
	return false
}

func ruleAllows(rule model.PolicyRule, resource, verb string) bool {
	if len(rule.ResourceNames) > 0 || len(rule.Resources) == 0 {
		return false
	}
	return resourceMatches(rule.Resources, resource) && verbMatches(rule.Verbs, verb)
}

func resourceMatches(rules []string, resource string) bool {
	subresource := ""
	if slash := strings.LastIndex(resource, "/"); slash > 0 && slash < len(resource)-1 {
		subresource = resource[slash+1:]
	}
	parent := resource
	if subresource != "" {
		parent = strings.TrimSuffix(resource, "/"+subresource)
	}
	for _, ruleResource := range rules {
		if ruleResource == "*" || ruleResource == resource {
			return true
		}
		if subresource == "" {
			continue
		}
		if ruleResource == parent+"/*" || ruleResource == "*/"+subresource {
			return true
		}
	}
	return false
}

func verbMatches(rules []string, verb string) bool {
	for _, ruleVerb := range rules {
		if ruleVerb == "*" || ruleVerb == verb {
			return true
		}
	}
	return false
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
