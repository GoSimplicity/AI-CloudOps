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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RoleService struct {
	dao         dao.ClusterDAO
	rbacManager manager.RBACManager
	logger      *zap.Logger
}

func NewRoleService(dao dao.ClusterDAO, rbacManager manager.RBACManager, logger *zap.Logger) *RoleService {
	return &RoleService{
		dao:         dao,
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// GetRoleList 获取Role列表
func (rs *RoleService) GetRoleList(ctx context.Context, req *model.RoleListReq) (*model.ListResp[model.RoleInfo], error) {
	// 使用 RBACManager 获取 Role 列表
	roles, err := rs.rbacManager.GetRoleList(ctx, req.ClusterID, req.Namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	// 转换为响应格式并过滤
	var roleInfos []model.RoleInfo
	for _, role := range roles.Items {
		roleInfo := k8sutils.ConvertK8sRoleToRoleInfo(&role, req.ClusterID)

		// 关键字过滤
		if req.Keyword != "" && !strings.Contains(roleInfo.Name, req.Keyword) {
			continue
		}

		roleInfos = append(roleInfos, roleInfo)
	}

	// 排序
	sort.Slice(roleInfos, func(i, j int) bool {
		return roleInfos[i].CreationTimestamp > roleInfos[j].CreationTimestamp
	})

	// 分页
	total := len(roleInfos)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		roleInfos = []model.RoleInfo{}
	} else if end > total {
		roleInfos = roleInfos[start:]
	} else {
		roleInfos = roleInfos[start:end]
	}

	return &model.ListResp[model.RoleInfo]{
		Items: roleInfos,
		Total: int64(total),
	}, nil
}

// GetRoleDetails 获取Role详情
func (rs *RoleService) GetRoleDetails(ctx context.Context, req *model.RoleGetReq) (*model.RoleInfo, error) {
	// 使用 RBACManager 获取 Role 详情
	role, err := rs.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	roleInfo := k8sutils.ConvertK8sRoleToRoleInfo(role, req.ClusterID)
	return &roleInfo, nil
}

// CreateRole 创建Role
func (rs *RoleService) CreateRole(ctx context.Context, req *model.CreateRoleReq) error {
	// 构建 Role 对象
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: k8sutils.ConvertPolicyRulesToK8s(req.Rules),
	}

	// 使用 RBACManager 创建 Role
	err := rs.rbacManager.CreateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// UpdateRole 更新Role
func (rs *RoleService) UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error {
	// 如果名称发生变化，需要删除原来的Role并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原Role
		err := rs.rbacManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.OriginalName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete original role: %w", err)
		}

		// 创建新Role
		createReq := &model.CreateRoleReq{
			ClusterID:   req.ClusterID,
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
			Rules:       req.Rules,
		}
		return rs.CreateRole(ctx, createReq)
	}

	// 获取现有Role
	existingRole, err := rs.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}

	// 更新Role
	existingRole.Labels = req.Labels
	existingRole.Annotations = req.Annotations
	existingRole.Rules = k8sutils.ConvertPolicyRulesToK8s(req.Rules)

	// 使用 RBACManager 更新 Role
	err = rs.rbacManager.UpdateRole(ctx, req.ClusterID, req.Namespace, existingRole)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeleteRole 删除Role
func (rs *RoleService) DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error {
	// 使用 RBACManager 删除 Role
	err := rs.rbacManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// BatchDeleteRole 批量删除Role
func (rs *RoleService) BatchDeleteRole(ctx context.Context, req *model.BatchDeleteRoleReq) error {
	// 批量删除 Role - 按命名空间分组处理
	namespaceRoles := make(map[string][]string)
	for _, role := range req.Roles {
		namespaceRoles[role.Namespace] = append(namespaceRoles[role.Namespace], role.Name)
	}

	// 使用 RBACManager 分命名空间批量删除 Role
	var errors []string
	for namespace, roleNames := range namespaceRoles {
		err := rs.rbacManager.BatchDeleteRoles(ctx, req.ClusterID, namespace, roleNames)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete roles in namespace %s: %v", namespace, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetRoleYaml 获取Role的YAML配置
func (rs *RoleService) GetRoleYaml(ctx context.Context, req *model.RoleGetReq) (string, error) {
	// 获取 Role
	role, err := rs.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get role: %w", err)
	}

	// 清理不需要的字段
	role.ManagedFields = nil
	role.ResourceVersion = ""
	role.UID = ""
	role.SelfLink = ""
	role.CreationTimestamp = metav1.Time{}
	role.Generation = 0

	yamlData, err := yaml.Marshal(role)
	if err != nil {
		return "", fmt.Errorf("failed to marshal role to yaml: %w", err)
	}

	return string(yamlData), nil
}

// UpdateRoleYaml 通过YAML更新Role
func (rs *RoleService) UpdateRoleYaml(ctx context.Context, req *model.RoleYamlReq) error {
	var role rbacv1.Role
	err := yaml.Unmarshal([]byte(req.YamlContent), &role)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称和命名空间一致
	role.Name = req.Name
	role.Namespace = req.Namespace

	// 获取现有Role以保持ResourceVersion
	existingRole, err := rs.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}

	role.ResourceVersion = existingRole.ResourceVersion
	role.UID = existingRole.UID

	// 使用 RBACManager 更新 Role
	err = rs.rbacManager.UpdateRole(ctx, req.ClusterID, req.Namespace, &role)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}
