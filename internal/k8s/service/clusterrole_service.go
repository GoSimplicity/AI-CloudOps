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

	"sigs.k8s.io/yaml"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleService interface {
	GetClusterRoleList(ctx context.Context, req *model.GetClusterRoleListReq) (model.ListResp[*model.K8sClusterRole], error)
	GetClusterRoleDetails(ctx context.Context, req *model.GetClusterRoleDetailsReq) (*model.K8sClusterRole, error)
	CreateClusterRole(ctx context.Context, req *model.CreateClusterRoleReq) error
	UpdateClusterRole(ctx context.Context, req *model.UpdateClusterRoleReq) error
	DeleteClusterRole(ctx context.Context, req *model.DeleteClusterRoleReq) error
	CreateClusterRoleByYaml(ctx context.Context, req *model.CreateClusterRoleByYamlReq) error
	GetClusterRoleYaml(ctx context.Context, req *model.GetClusterRoleYamlReq) (*model.K8sYaml, error)
	UpdateClusterRoleYaml(ctx context.Context, req *model.UpdateClusterRoleByYamlReq) error
}

type clusterRoleService struct {
	clusterRoleManager manager.ClusterRoleManager
	logger             *zap.Logger
}

func NewClusterRoleService(clusterRoleManager manager.ClusterRoleManager, logger *zap.Logger) ClusterRoleService {
	return &clusterRoleService{
		clusterRoleManager: clusterRoleManager,
		logger:             logger,
	}
}

func (s *clusterRoleService) GetClusterRoleList(ctx context.Context, req *model.GetClusterRoleListReq) (model.ListResp[*model.K8sClusterRole], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("获取ClusterRole列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("集群ID不能为空")
	}

	listOptions := metav1.ListOptions{}

	k8sClusterRoles, err := s.clusterRoleManager.GetClusterRoleList(ctx, req.ClusterID, listOptions)
	if err != nil {
		s.logger.Error("获取ClusterRole列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("获取ClusterRole列表失败: %w", err)
	}

	// 名称过滤（使用通用的Search字段，支持不区分大小写）
	var filteredClusterRoles []*model.K8sClusterRole
	for _, cr := range k8sClusterRoles {
		if utils.FilterByName(cr.Name, req.Search) {
			filteredClusterRoles = append(filteredClusterRoles, cr)
		}
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredClusterRoles, func(cr *model.K8sClusterRole) time.Time {
		t, _ := time.Parse(time.RFC3339, cr.CreatedAt)
		return t
	})

	// 分页处理
	pagedItems, total := utils.Paginate(filteredClusterRoles, req.Page, req.Size)

	return model.ListResp[*model.K8sClusterRole]{
		Total: total,
		Items: pagedItems,
	}, nil
}

func (s *clusterRoleService) GetClusterRoleDetails(ctx context.Context, req *model.GetClusterRoleDetailsReq) (*model.K8sClusterRole, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	clusterRole, err := s.clusterRoleManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole失败: %w", err)
	}

	k8sClusterRole := utils.ConvertClusterRoleToModel(clusterRole, req.ClusterID)
	if k8sClusterRole == nil {
		return nil, fmt.Errorf("构建ClusterRole详细信息失败")
	}

	return k8sClusterRole, nil
}

func (s *clusterRoleService) GetClusterRoleYaml(ctx context.Context, req *model.GetClusterRoleYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	clusterRole, err := s.clusterRoleManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole失败: %w", err)
	}

	yamlContent, err := utils.ClusterRoleToYAML(clusterRole)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("clusterRoleName", clusterRole.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *clusterRoleService) UpdateClusterRoleYaml(ctx context.Context, req *model.UpdateClusterRoleByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新ClusterRole YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("ClusterRole名称不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	existingClusterRole, err := s.clusterRoleManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取现有ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有ClusterRole失败: %w", err)
	}

	updatedClusterRole, err := utils.YAMLToClusterRole(req.YamlContent)
	if err != nil {
		s.logger.Error("解析YAML失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 保持必要的元数据
	updatedClusterRole.ResourceVersion = existingClusterRole.ResourceVersion
	updatedClusterRole.UID = existingClusterRole.UID

	err = s.clusterRoleManager.UpdateClusterRole(ctx, req.ClusterID, updatedClusterRole)
	if err != nil {
		s.logger.Error("更新ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新ClusterRole失败: %w", err)
	}

	return nil
}

func (s *clusterRoleService) CreateClusterRole(ctx context.Context, req *model.CreateClusterRoleReq) error {
	// 验证规则
	if len(req.Rules) == 0 {
		s.logger.Warn("创建ClusterRole时规则为空",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("ClusterRole规则不能为空")
	}

	// 记录原始规则信息
	s.logger.Info("创建ClusterRole",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name),
		zap.Int("rulesCount", len(req.Rules)))

	// 转换规则
	k8sRules := utils.ConvertPolicyRulesToK8s(req.Rules)

	// 检查转换后的规则
	if len(k8sRules) == 0 {
		s.logger.Error("规则转换后为空，可能包含无效规则",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name),
			zap.Int("originalRulesCount", len(req.Rules)))
		return fmt.Errorf("规则无效：所有规则都不符合Kubernetes RBAC要求")
	}

	s.logger.Info("规则转换成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name),
		zap.Int("validRulesCount", len(k8sRules)))

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: k8sRules,
	}

	err := s.clusterRoleManager.CreateClusterRole(ctx, req.ClusterID, clusterRole)
	if err != nil {
		s.logger.Error("创建ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	s.logger.Info("创建ClusterRole成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return nil
}

func (s *clusterRoleService) UpdateClusterRole(ctx context.Context, req *model.UpdateClusterRoleReq) error {
	// 获取现有ClusterRole
	existingClusterRole, err := s.clusterRoleManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取现有ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("failed to get existing cluster role: %w", err)
	}

	// 验证规则
	if len(req.Rules) == 0 {
		s.logger.Warn("更新ClusterRole时规则为空",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("ClusterRole规则不能为空")
	}

	// 转换规则
	k8sRules := utils.ConvertPolicyRulesToK8s(req.Rules)

	// 检查转换后的规则
	if len(k8sRules) == 0 {
		s.logger.Error("规则转换后为空，可能包含无效规则",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name),
			zap.Int("originalRulesCount", len(req.Rules)))
		return fmt.Errorf("规则无效：所有规则都不符合Kubernetes RBAC要求")
	}

	// 更新ClusterRole
	existingClusterRole.Labels = req.Labels
	existingClusterRole.Annotations = req.Annotations
	existingClusterRole.Rules = k8sRules

	err = s.clusterRoleManager.UpdateClusterRole(ctx, req.ClusterID, existingClusterRole)
	if err != nil {
		s.logger.Error("更新ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	s.logger.Info("更新ClusterRole成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return nil
}

func (s *clusterRoleService) DeleteClusterRole(ctx context.Context, req *model.DeleteClusterRoleReq) error {

	err := s.clusterRoleManager.DeleteClusterRole(ctx, req.ClusterID, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster role: %w", err)
	}

	return nil
}

func (s *clusterRoleService) CreateClusterRoleByYaml(ctx context.Context, req *model.CreateClusterRoleByYamlReq) error {
	var clusterRole rbacv1.ClusterRole
	err := yaml.Unmarshal([]byte(req.YamlContent), &clusterRole)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	err = s.clusterRoleManager.CreateClusterRole(ctx, req.ClusterID, &clusterRole)
	if err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	return nil
}
