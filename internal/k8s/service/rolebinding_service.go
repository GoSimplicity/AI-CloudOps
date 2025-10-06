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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleBindingService interface {
	GetRoleBindingList(ctx context.Context, req *model.GetRoleBindingListReq) (model.ListResp[*model.K8sRoleBinding], error)
	GetRoleBindingDetails(ctx context.Context, req *model.GetRoleBindingDetailsReq) (*model.K8sRoleBinding, error)
	CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error
	UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error
	DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error

	GetRoleBindingYaml(ctx context.Context, req *model.GetRoleBindingYamlReq) (*model.K8sYaml, error)
	CreateRoleBindingByYaml(ctx context.Context, req *model.CreateRoleBindingByYamlReq) error
	UpdateRoleBindingYaml(ctx context.Context, req *model.UpdateRoleBindingByYamlReq) error
}

type roleBindingService struct {
	roleBindingManager manager.RoleBindingManager
	logger             *zap.Logger
}

func NewRoleBindingService(roleBindingManager manager.RoleBindingManager, logger *zap.Logger) RoleBindingService {
	return &roleBindingService{
		roleBindingManager: roleBindingManager,
		logger:             logger,
	}
}

func (s *roleBindingService) GetRoleBindingList(ctx context.Context, req *model.GetRoleBindingListReq) (model.ListResp[*model.K8sRoleBinding], error) {
	if req == nil {
		return model.ListResp[*model.K8sRoleBinding]{}, fmt.Errorf("获取RoleBinding列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sRoleBinding]{}, fmt.Errorf("集群ID不能为空")
	}

	options := k8sutils.BuildRoleBindingListOptions(req)

	roleBindings, err := s.roleBindingManager.GetRoleBindingList(ctx, req.ClusterID, req.Namespace, options)
	if err != nil {
		s.logger.Error("获取RoleBinding列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sRoleBinding]{}, fmt.Errorf("获取RoleBinding列表失败: %w", err)
	}

	// 名称过滤（使用通用的Search字段，支持不区分大小写）
	var filteredRoleBindings []*model.K8sRoleBinding
	for _, rb := range roleBindings {
		if k8sutils.FilterByName(rb.Name, req.Search) {
			filteredRoleBindings = append(filteredRoleBindings, rb)
		}
	}

	// 按创建时间排序（最新的在前）
	k8sutils.SortByCreationTime(filteredRoleBindings, func(rb *model.K8sRoleBinding) time.Time {
		t, _ := time.Parse(time.RFC3339, rb.CreatedAt)
		return t
	})

	// 分页处理
	paginatedRoleBindings, total := k8sutils.Paginate(filteredRoleBindings, req.Page, req.Size)

	return model.ListResp[*model.K8sRoleBinding]{
		Items: paginatedRoleBindings,
		Total: total,
	}, nil
}

func (s *roleBindingService) GetRoleBindingDetails(ctx context.Context, req *model.GetRoleBindingDetailsReq) (*model.K8sRoleBinding, error) {
	if req == nil {
		return nil, fmt.Errorf("获取RoleBinding详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("RoleBinding名称不能为空")
	}

	roleBinding, err := s.roleBindingManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取RoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取RoleBinding失败: %w", err)
	}

	return k8sutils.ConvertK8sRoleBindingToRoleBindingInfo(roleBinding, req.ClusterID), nil
}

func (s *roleBindingService) CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("创建RoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("RoleBinding名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	roleBinding := k8sutils.ConvertToK8sRoleBinding(req)
	err := s.roleBindingManager.CreateRoleBinding(ctx, req.ClusterID, req.Namespace, roleBinding)
	if err != nil {
		s.logger.Error("创建RoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建RoleBinding失败: %w", err)
	}

	return nil
}

func (s *roleBindingService) UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("更新RoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("RoleBinding名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	roleBinding := &model.CreateRoleBindingReq{
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Name:        req.Name,
		RoleRef:     req.RoleRef,
		Subjects:    req.Subjects,
		Labels:      req.Labels,
		Annotations: req.Annotations,
	}

	err := s.roleBindingManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, k8sutils.ConvertToK8sRoleBinding(roleBinding))
	if err != nil {
		s.logger.Error("更新RoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新RoleBinding失败: %w", err)
	}

	return nil
}

func (s *roleBindingService) DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("删除RoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("RoleBinding名称不能为空")
	}

	err := s.roleBindingManager.DeleteRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除RoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除RoleBinding失败: %w", err)
	}

	return nil
}

// ======================== YAML 操作 ========================

func (s *roleBindingService) GetRoleBindingYaml(ctx context.Context, req *model.GetRoleBindingYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取RoleBinding YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("RoleBinding名称不能为空")
	}

	roleBinding, err := s.roleBindingManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取RoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取RoleBinding失败: %w", err)
	}

	yamlContent, err := k8sutils.RoleBindingToYAML(roleBinding)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("roleBindingName", roleBinding.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *roleBindingService) CreateRoleBindingByYaml(ctx context.Context, req *model.CreateRoleBindingByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建RoleBinding请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建RoleBinding",
		zap.Int("cluster_id", req.ClusterID))

	roleBinding, err := k8sutils.YAMLToRoleBinding(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建RoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建RoleBinding失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default命名空间
	if roleBinding.Namespace == "" {
		roleBinding.Namespace = "default"
		s.logger.Info("YAML中未指定namespace，使用default命名空间",
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", roleBinding.Name))
	}

	err = s.roleBindingManager.CreateRoleBinding(ctx, req.ClusterID, roleBinding.Namespace, roleBinding)
	if err != nil {
		s.logger.Error("通过YAML创建RoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", roleBinding.Namespace),
			zap.String("name", roleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML创建RoleBinding失败: %w", err)
	}

	s.logger.Info("通过YAML创建RoleBinding成功",
		zap.Int("cluster_id", req.ClusterID))

	return nil
}

func (s *roleBindingService) UpdateRoleBindingYaml(ctx context.Context, req *model.UpdateRoleBindingByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新RoleBinding YAML请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新RoleBinding",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	roleBinding, err := k8sutils.YAMLToRoleBinding(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建RoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建RoleBinding失败: %w", err)
	}

	err = s.roleBindingManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, roleBinding)
	if err != nil {
		s.logger.Error("通过YAML更新RoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML更新RoleBinding失败: %w", err)
	}

	s.logger.Info("通过YAML更新RoleBinding成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}
