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
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RoleService interface {
	GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error)
	GetRoleDetails(ctx context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error)
	CreateRole(ctx context.Context, req *model.CreateRoleReq) error
	UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error
	DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error
	CreateRoleByYaml(ctx context.Context, req *model.CreateRoleByYamlReq) error

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

func (s *roleService) GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error) {
	if req == nil {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("集群ID不能为空")
	}

	roleList, err := s.roleManager.GetRoleList(ctx, req.ClusterID, req.Namespace, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("获取Role列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表失败: %w", err)
	}

	// 名称过滤（使用通用的Search字段，支持不区分大小写）
	var filteredRoles []*model.K8sRole
	for _, role := range roleList {
		if utils.FilterByName(role.Name, req.Search) {
			filteredRoles = append(filteredRoles, role)
		}
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredRoles, func(role *model.K8sRole) time.Time {
		t, _ := time.Parse(time.RFC3339, role.CreatedAt)
		return t
	})

	// 分页处理
	paginatedRoles, total := utils.Paginate(filteredRoles, req.Page, req.Size)

	return model.ListResp[*model.K8sRole]{
		Items: paginatedRoles,
		Total: total,
	}, nil
}

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
		s.logger.Error("获取Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role失败: %w", err)
	}

	s.logger.Info("获取到K8s Role对象",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.Int("k8sRulesCount", len(role.Rules)))

	roleInfo := utils.ConvertK8sRoleToRoleInfo(role, req.ClusterID)

	s.logger.Info("转换后的Role信息",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.Int("modelRulesCount", len(roleInfo.Rules)))

	return &roleInfo, nil
}

func (s *roleService) CreateRole(ctx context.Context, req *model.CreateRoleReq) error {
	if req == nil {
		return fmt.Errorf("创建Role请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Role名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 验证规则
	if len(req.Rules) == 0 {
		s.logger.Warn("创建Role时规则为空",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("Role规则不能为空")
	}

	// 记录原始规则信息
	s.logger.Info("创建Role",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.Int("rulesCount", len(req.Rules)))

	// 转换规则
	k8sRules := utils.ConvertPolicyRulesToK8s(req.Rules)

	// 检查转换后的规则
	if len(k8sRules) == 0 {
		s.logger.Error("规则转换后为空，可能包含无效规则",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int("originalRulesCount", len(req.Rules)))
		return fmt.Errorf("规则无效：所有规则都不符合Kubernetes RBAC要求")
	}

	s.logger.Info("规则转换成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name),
		zap.Int("validRulesCount", len(k8sRules)))

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: k8sRules,
	}

	err := s.roleManager.CreateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		s.logger.Error("创建Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Role失败: %w", err)
	}

	s.logger.Info("创建Role成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *roleService) UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error {
	if req == nil {
		return fmt.Errorf("更新Role请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Role名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 获取现有Role
	existingRole, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取现有Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有Role失败: %w", err)
	}

	// 验证规则
	if len(req.Rules) == 0 {
		s.logger.Warn("更新Role时规则为空",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("Role规则不能为空")
	}

	// 转换规则
	k8sRules := utils.ConvertPolicyRulesToK8s(req.Rules)

	// 检查转换后的规则
	if len(k8sRules) == 0 {
		s.logger.Error("规则转换后为空，可能包含无效规则",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Int("originalRulesCount", len(req.Rules)))
		return fmt.Errorf("规则无效：所有规则都不符合Kubernetes RBAC要求")
	}

	// 更新Role
	existingRole.Labels = req.Labels
	existingRole.Annotations = req.Annotations
	existingRole.Rules = k8sRules

	err = s.roleManager.UpdateRole(ctx, req.ClusterID, req.Namespace, existingRole)
	if err != nil {
		s.logger.Error("更新Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Role失败: %w", err)
	}

	return nil
}

func (s *roleService) DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error {
	if req == nil {
		return fmt.Errorf("删除Role请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Role名称不能为空")
	}

	err := s.roleManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Role失败: %w", err)
	}

	return nil
}

func (s *roleService) CreateRoleByYaml(ctx context.Context, req *model.CreateRoleByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建Role请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建Role",
		zap.Int("cluster_id", req.ClusterID))

	role, err := utils.YAMLToRole(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建Role失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Role失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default命名空间
	if role.Namespace == "" {
		role.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", role.Name))
	}

	err = s.roleManager.CreateRole(ctx, req.ClusterID, role.Namespace, role)
	if err != nil {
		s.logger.Error("通过YAML创建Role失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", role.Namespace),
			zap.String("name", role.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML创建Role失败: %w", err)
	}

	s.logger.Info("通过YAML创建Role成功",
		zap.Int("cluster_id", req.ClusterID))

	return nil
}

func (s *roleService) GetRoleYaml(ctx context.Context, req *model.GetRoleYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Role名称不能为空")
	}

	role, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role失败: %w", err)
	}

	yamlContent, err := utils.RoleToYAML(role)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("roleName", role.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *roleService) UpdateRoleYaml(ctx context.Context, req *model.UpdateRoleByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新Role YAML请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新Role",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	role, err := utils.YAMLToRole(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建Role失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建Role失败: %w", err)
	}

	// 确保名称和命名空间一致
	role.Name = req.Name
	role.Namespace = req.Namespace

	// 获取现有Role以保持ResourceVersion
	existingRole, err := s.roleManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取现有Role失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("获取现有Role失败: %w", err)
	}

	role.ResourceVersion = existingRole.ResourceVersion
	role.UID = existingRole.UID

	err = s.roleManager.UpdateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		s.logger.Error("通过YAML更新Role失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML更新Role失败: %w", err)
	}

	s.logger.Info("通过YAML更新Role成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}
