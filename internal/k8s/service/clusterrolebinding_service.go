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
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleBindingService interface {
	GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error)
	GetClusterRoleBindingDetails(ctx context.Context, req *model.GetClusterRoleBindingDetailsReq) (*model.K8sClusterRoleBinding, error)
	CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error
	UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error
	DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error

	GetClusterRoleBindingYaml(ctx context.Context, req *model.GetClusterRoleBindingYamlReq) (*model.K8sYaml, error)
	CreateClusterRoleBindingByYaml(ctx context.Context, req *model.CreateClusterRoleBindingByYamlReq) error
	UpdateClusterRoleBindingYaml(ctx context.Context, req *model.UpdateClusterRoleBindingByYamlReq) error
}

type clusterRoleBindingService struct {
	clusterRoleBindingManager manager.ClusterRoleBindingManager
	logger                    *zap.Logger
}

func NewClusterRoleBindingService(clusterRoleBindingManager manager.ClusterRoleBindingManager, logger *zap.Logger) ClusterRoleBindingService {
	return &clusterRoleBindingService{
		clusterRoleBindingManager: clusterRoleBindingManager,
		logger:                    logger,
	}
}

func (s *clusterRoleBindingService) GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("集群ID不能为空")
	}

	listOptions := metav1.ListOptions{}

	k8sClusterRoleBindings, err := s.clusterRoleBindingManager.GetClusterRoleBindingList(ctx, req.ClusterID, listOptions)
	if err != nil {
		s.logger.Error("获取ClusterRoleBinding列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表失败: %w", err)
	}

	// 名称过滤（使用通用的Search字段，支持不区分大小写）
	var filteredClusterRoleBindings []*model.K8sClusterRoleBinding
	for _, crb := range k8sClusterRoleBindings {
		if k8sutils.FilterByName(crb.Name, req.Search) {
			filteredClusterRoleBindings = append(filteredClusterRoleBindings, crb)
		}
	}

	// 按创建时间排序（最新的在前）
	k8sutils.SortByCreationTime(filteredClusterRoleBindings, func(crb *model.K8sClusterRoleBinding) time.Time {
		t, _ := time.Parse(time.RFC3339, crb.CreatedAt)
		return t
	})

	// 分页处理
	pagedItems, total := k8sutils.Paginate(filteredClusterRoleBindings, req.Page, req.Size)

	s.logger.Debug("GetClusterRoleBindingList: 获取ClusterRoleBinding列表成功",
		zap.Int("clusterID", req.ClusterID),
		zap.Int64("total", total),
		zap.Int("returned", len(pagedItems)))

	return model.ListResp[*model.K8sClusterRoleBinding]{
		Items: pagedItems,
		Total: total,
	}, nil
}

func (s *clusterRoleBindingService) GetClusterRoleBindingDetails(ctx context.Context, req *model.GetClusterRoleBindingDetailsReq) (*model.K8sClusterRoleBinding, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRoleBinding详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	clusterRoleBinding, err := s.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding失败: %w", err)
	}

	k8sClusterRoleBinding := k8sutils.ConvertToK8sClusterRoleBinding(clusterRoleBinding)
	if k8sClusterRoleBinding != nil {
		k8sClusterRoleBinding.ClusterID = req.ClusterID
	}

	s.logger.Debug("GetClusterRoleBindingDetails: 获取ClusterRoleBinding详情成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return k8sClusterRoleBinding, nil
}

func (s *clusterRoleBindingService) CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("创建ClusterRoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		RoleRef:  k8sutils.ConvertRoleRefToK8s(req.RoleRef),
		Subjects: k8sutils.ConvertSubjectsToK8s(req.Subjects),
	}

	err := s.clusterRoleBindingManager.CreateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		s.logger.Error("创建ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("创建ClusterRoleBinding失败: %w", err)
	}

	return nil
}

func (s *clusterRoleBindingService) UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("更新ClusterRoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	// 获取现有ClusterRoleBinding
	existingClusterRoleBinding, err := s.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取现有ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有ClusterRoleBinding失败: %w", err)
	}

	// 更新ClusterRoleBinding
	existingClusterRoleBinding.Labels = req.Labels
	existingClusterRoleBinding.Annotations = req.Annotations
	existingClusterRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingClusterRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	err = s.clusterRoleBindingManager.UpdateClusterRoleBinding(ctx, req.ClusterID, existingClusterRoleBinding)
	if err != nil {
		s.logger.Error("更新ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新ClusterRoleBinding失败: %w", err)
	}

	return nil
}

func (s *clusterRoleBindingService) DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error {
	if req == nil {
		return fmt.Errorf("删除ClusterRoleBinding请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	err := s.clusterRoleBindingManager.DeleteClusterRoleBinding(ctx, req.ClusterID, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("删除ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("删除ClusterRoleBinding失败: %w", err)
	}

	return nil
}

func (s *clusterRoleBindingService) GetClusterRoleBindingYaml(ctx context.Context, req *model.GetClusterRoleBindingYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRoleBinding YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	clusterRoleBinding, err := s.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding失败: %w", err)
	}

	yamlContent, err := k8sutils.ClusterRoleBindingToYAML(clusterRoleBinding)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("clusterRoleBindingName", clusterRoleBinding.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *clusterRoleBindingService) CreateClusterRoleBindingByYaml(ctx context.Context, req *model.CreateClusterRoleBindingByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建ClusterRoleBinding请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建ClusterRoleBinding",
		zap.Int("cluster_id", req.ClusterID))

	clusterRoleBinding, err := k8sutils.YAMLToClusterRoleBinding(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建ClusterRoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("从YAML构建ClusterRoleBinding失败: %w", err)
	}

	err = s.clusterRoleBindingManager.CreateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		s.logger.Error("通过YAML创建ClusterRoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", clusterRoleBinding.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML创建ClusterRoleBinding失败: %w", err)
	}

	s.logger.Info("通过YAML创建ClusterRoleBinding成功",
		zap.Int("cluster_id", req.ClusterID))

	return nil
}

func (s *clusterRoleBindingService) UpdateClusterRoleBindingYaml(ctx context.Context, req *model.UpdateClusterRoleBindingByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新ClusterRoleBinding YAML请求不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新ClusterRoleBinding",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))

	clusterRoleBinding, err := k8sutils.YAMLToClusterRoleBinding(req.YamlContent)
	if err != nil {
		s.logger.Error("从YAML构建ClusterRoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("从YAML构建ClusterRoleBinding失败: %w", err)
	}

	// 确保名称一致
	clusterRoleBinding.Name = req.Name

	// 获取现有ClusterRoleBinding以保持ResourceVersion
	existingClusterRoleBinding, err := s.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("获取现有ClusterRoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("获取现有ClusterRoleBinding失败: %w", err)
	}

	clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
	clusterRoleBinding.UID = existingClusterRoleBinding.UID

	err = s.clusterRoleBindingManager.UpdateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		s.logger.Error("通过YAML更新ClusterRoleBinding失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("通过YAML更新ClusterRoleBinding失败: %w", err)
	}

	s.logger.Info("通过YAML更新ClusterRoleBinding成功",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("name", req.Name))

	return nil
}
