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

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RoleService interface {
	// 基础 CRUD 操作
	GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error)
	GetRoleDetails(ctx context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error)
	CreateRole(ctx context.Context, req *model.CreateRoleReq) error
	UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error
	DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error
	CreateRoleByYaml(ctx context.Context, req *model.CreateRoleByYamlReq) error

	// YAML 操作
	GetRoleYaml(ctx context.Context, req *model.GetRoleYamlReq) (*model.K8sYaml, error)
	UpdateRoleYaml(ctx context.Context, req *model.UpdateRoleByYamlReq) error
}

type roleService struct {
	roleManager manager.RoleManager
	logger      *zap.Logger
}

func NewRoleService(roleManager manager.RoleManager, logger *zap.Logger) RoleService {
	return &roleService{
		roleManager: roleManager,
		logger:      logger,
	}
}

// GetRoleList 获取Role列表
func (s *roleService) GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error) {
	if req == nil {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("集群ID不能为空")
	}

	roleList, err := s.roleManager.GetRoleList(ctx, req.ClusterID, req.Namespace, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("GetRoleList: 获取Role列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表失败: %w", err)
	}

	// 关键字过滤
	var filteredRoles []*model.K8sRole
	if req.Keyword != "" {
		for _, role := range roleList {
			if strings.Contains(strings.ToLower(role.Name), strings.ToLower(req.Keyword)) {
				filteredRoles = append(filteredRoles, role)
			}
		}
	} else {
		filteredRoles = roleList
	}

	// 分页处理
	total := int64(len(filteredRoles))
	page := req.Page
	size := req.PageSize
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		filteredRoles = []*model.K8sRole{}
	} else if end > total {
		filteredRoles = filteredRoles[start:]
	} else {
		filteredRoles = filteredRoles[start:end]
	}

	return model.ListResp[*model.K8sRole]{
		Items: filteredRoles,
		Total: total,
	}, nil
}

// GetRoleDetails 获取Role详情
func (s *roleService) GetRoleDetails(ctx context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Role名称不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	role, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetRoleDetails: 获取Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role失败: %w", err)
	}

	roleInfo := utils.ConvertK8sRoleToRoleInfo(role, req.ClusterID)
	return &roleInfo, nil
}

// CreateRole 创建Role
func (s *roleService) CreateRole(ctx context.Context, req *model.CreateRoleReq) error {
	// 构建 Role 对象
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: utils.ConvertPolicyRulesToK8s(req.Rules),
	}

	// 使用 RoleManager 创建 Role
	err := s.roleManager.CreateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// UpdateRole 更新Role
func (s *roleService) UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error {
	// 获取现有Role
	existingRole, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}

	// 更新Role
	existingRole.Labels = req.Labels
	existingRole.Annotations = req.Annotations
	existingRole.Rules = utils.ConvertPolicyRulesToK8s(req.Rules)

	// 使用 RoleManager 更新 Role
	err = s.roleManager.UpdateRole(ctx, req.ClusterID, req.Namespace, existingRole)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeleteRole 删除Role
func (s *roleService) DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error {
	// 使用 RoleManager 删除 Role
	err := s.roleManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// CreateRoleByYaml 通过YAML创建Role
func (s *roleService) CreateRoleByYaml(ctx context.Context, req *model.CreateRoleByYamlReq) error {
	role, err := utils.YAMLToRole(req.YamlContent)
	if err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	// 设置命名空间
	// Namespace will be extracted from YAML content

	// 使用 RoleManager 创建 Role
	err = s.roleManager.CreateRole(ctx, req.ClusterID, role.Namespace, role)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	return nil
}

// GetRoleYaml 获取Role YAML
func (s *roleService) GetRoleYaml(ctx context.Context, req *model.GetRoleYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role YAML请求不能为空")
	}

	// 获取 Role
	role, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	yamlContent, err := utils.RoleToYAML(role)
	if err != nil {
		return nil, fmt.Errorf("failed to convert role to yaml: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdateRoleYaml 更新Role YAML
func (s *roleService) UpdateRoleYaml(ctx context.Context, req *model.UpdateRoleByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新Role YAML请求不能为空")
	}

	role, err := utils.YAMLToRole(req.YamlContent)
	if err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	// 确保名称和命名空间一致
	role.Name = req.Name
	role.Namespace = req.Namespace

	// 获取现有Role以保持ResourceVersion
	existingRole, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role: %w", err)
	}

	role.ResourceVersion = existingRole.ResourceVersion
	role.UID = existingRole.UID

	// 使用 RoleManager 更新 Role
	err = s.roleManager.UpdateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}
