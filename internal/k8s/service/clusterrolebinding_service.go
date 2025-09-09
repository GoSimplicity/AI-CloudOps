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
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleBindingService interface {
	// 基础 CRUD 操作
	GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error)
	GetClusterRoleBindingDetails(ctx context.Context, req *model.GetClusterRoleBindingDetailsReq) (*model.K8sClusterRoleBinding, error)
	CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error
	UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error
	DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error

	// YAML 操作
	GetClusterRoleBindingYaml(ctx context.Context, req *model.GetClusterRoleBindingYamlReq) (string, error)
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

// GetClusterRoleBindingList 获取ClusterRoleBinding列表
func (c *clusterRoleBindingService) GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := metav1.ListOptions{}

	k8sClusterRoleBindings, err := c.clusterRoleBindingManager.GetClusterRoleBindingList(ctx, req.ClusterID, listOptions)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingList: 获取ClusterRoleBinding列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表失败: %w", err)
	}

	// 关键字过滤
	var filteredClusterRoleBindings []*model.K8sClusterRoleBinding
	if req.Keyword != "" {
		for _, crb := range k8sClusterRoleBindings {
			if strings.Contains(strings.ToLower(crb.Name), strings.ToLower(req.Keyword)) {
				filteredClusterRoleBindings = append(filteredClusterRoleBindings, crb)
			}
		}
	} else {
		filteredClusterRoleBindings = k8sClusterRoleBindings
	}

	// 简单分页
	total := int64(len(filteredClusterRoleBindings))
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
		filteredClusterRoleBindings = []*model.K8sClusterRoleBinding{}
	} else {
		if end > total {
			end = total
		}
		filteredClusterRoleBindings = filteredClusterRoleBindings[start:end]
	}

	c.logger.Debug("GetClusterRoleBindingList: 获取ClusterRoleBinding列表成功",
		zap.Int("clusterID", req.ClusterID),
		zap.Int64("total", total),
		zap.Int("returned", len(filteredClusterRoleBindings)))

	return model.ListResp[*model.K8sClusterRoleBinding]{
		Items: filteredClusterRoleBindings,
		Total: total,
	}, nil
}

// GetClusterRoleBindingDetails 获取ClusterRoleBinding详情
func (c *clusterRoleBindingService) GetClusterRoleBindingDetails(ctx context.Context, req *model.GetClusterRoleBindingDetailsReq) (*model.K8sClusterRoleBinding, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRoleBinding详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRoleBinding名称不能为空")
	}

	clusterRoleBinding, err := c.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingDetails: 获取ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding失败: %w", err)
	}

	k8sClusterRoleBinding := k8sutils.ConvertToK8sClusterRoleBinding(clusterRoleBinding)
	if k8sClusterRoleBinding != nil {
		k8sClusterRoleBinding.ClusterID = req.ClusterID
	}

	c.logger.Debug("GetClusterRoleBindingDetails: 获取ClusterRoleBinding详情成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return k8sClusterRoleBinding, nil
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
func (c *clusterRoleBindingService) CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error {
	// 构建 ClusterRoleBinding 对象
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		RoleRef:  k8sutils.ConvertRoleRefToK8s(req.RoleRef),
		Subjects: k8sutils.ConvertSubjectsToK8s(req.Subjects),
	}

	// 使用 ClusterRoleBindingManager 创建 ClusterRoleBinding
	err := c.clusterRoleBindingManager.CreateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	return nil
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (c *clusterRoleBindingService) UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error {
	// 获取现有ClusterRoleBinding
	existingClusterRoleBinding, err := c.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	// 更新ClusterRoleBinding
	existingClusterRoleBinding.Labels = req.Labels
	existingClusterRoleBinding.Annotations = req.Annotations
	existingClusterRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingClusterRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	// 使用 ClusterRoleBindingManager 更新 ClusterRoleBinding
	err = c.clusterRoleBindingManager.UpdateClusterRoleBinding(ctx, req.ClusterID, existingClusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (c *clusterRoleBindingService) DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error {
	// 使用 ClusterRoleBindingManager 删除 ClusterRoleBinding
	err := c.clusterRoleBindingManager.DeleteClusterRoleBinding(ctx, req.ClusterID, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster role binding: %w", err)
	}

	return nil
}

// GetClusterRoleBindingYaml 获取ClusterRoleBinding YAML
func (c *clusterRoleBindingService) GetClusterRoleBindingYaml(ctx context.Context, req *model.GetClusterRoleBindingYamlReq) (string, error) {
	if req == nil {
		return "", fmt.Errorf("获取ClusterRoleBinding YAML请求不能为空")
	}

	// 获取 ClusterRoleBinding
	clusterRoleBinding, err := c.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return "", fmt.Errorf("failed to get cluster role binding: %w", err)
	}

	yamlContent, err := k8sutils.ClusterRoleBindingToYAML(clusterRoleBinding)
	if err != nil {
		return "", fmt.Errorf("failed to convert cluster role binding to yaml: %w", err)
	}

	return yamlContent, nil
}

// UpdateClusterRoleBindingYaml 更新ClusterRoleBinding YAML
func (c *clusterRoleBindingService) UpdateClusterRoleBindingYaml(ctx context.Context, req *model.UpdateClusterRoleBindingByYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新ClusterRoleBinding YAML请求不能为空")
	}

	clusterRoleBinding, err := k8sutils.YAMLToClusterRoleBinding(req.YamlContent)
	if err != nil {
		return fmt.Errorf("failed to parse yaml: %w", err)
	}

	// 确保名称一致
	clusterRoleBinding.Name = req.Name

	// 获取现有ClusterRoleBinding以保持ResourceVersion
	existingClusterRoleBinding, err := c.clusterRoleBindingManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
	clusterRoleBinding.UID = existingClusterRoleBinding.UID

	// 使用 RBACManager 更新 ClusterRoleBinding
	err = c.clusterRoleBindingManager.UpdateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}
