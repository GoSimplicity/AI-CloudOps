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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleService struct {
	dao       dao.ClusterDAO
	k8sClient client.K8sClient
	logger    *zap.Logger
}

func NewClusterRoleService(dao dao.ClusterDAO, k8sClient client.K8sClient, logger *zap.Logger) *ClusterRoleService {
	return &ClusterRoleService{
		dao:       dao,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetClusterRoleList 获取ClusterRole列表
func (crs *ClusterRoleService) GetClusterRoleList(ctx context.Context, req *model.ClusterRoleListReq) (*model.ListResp[model.ClusterRoleInfo], error) {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 构建列表选项
	listOptions := metav1.ListOptions{}

	clusterRoles, err := k8sClient.RbacV1().ClusterRoles().List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles: %w", err)
	}

	// 转换为响应格式并过滤
	var clusterRoleInfos []model.ClusterRoleInfo
	for _, clusterRole := range clusterRoles.Items {
		clusterRoleInfo := utils.ConvertK8sClusterRoleToClusterRoleInfo(&clusterRole, req.ClusterID)

		// 关键字过滤
		if req.Keyword != "" && !strings.Contains(clusterRoleInfo.Name, req.Keyword) {
			continue
		}

		clusterRoleInfos = append(clusterRoleInfos, clusterRoleInfo)
	}

	// 排序
	sort.Slice(clusterRoleInfos, func(i, j int) bool {
		return clusterRoleInfos[i].CreationTimestamp > clusterRoleInfos[j].CreationTimestamp
	})

	// 分页
	total := len(clusterRoleInfos)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		clusterRoleInfos = []model.ClusterRoleInfo{}
	} else if end > total {
		clusterRoleInfos = clusterRoleInfos[start:]
	} else {
		clusterRoleInfos = clusterRoleInfos[start:end]
	}

	return &model.ListResp[model.ClusterRoleInfo]{
		Items: clusterRoleInfos,
		Total: int64(total),
	}, nil
}

// GetClusterRoleDetails 获取ClusterRole详情
func (crs *ClusterRoleService) GetClusterRoleDetails(ctx context.Context, req *model.ClusterRoleGetReq) (*model.ClusterRoleInfo, error) {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	clusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster role: %w", err)
	}

	clusterRoleInfo := utils.ConvertK8sClusterRoleToClusterRoleInfo(clusterRole, req.ClusterID)
	return &clusterRoleInfo, nil
}

// CreateClusterRole 创建ClusterRole
func (crs *ClusterRoleService) CreateClusterRole(ctx context.Context, req *model.CreateClusterRoleReq) error {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: utils.ConvertPolicyRulesToK8s(req.Rules),
	}

	_, err = k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	return nil
}

// UpdateClusterRole 更新ClusterRole
func (crs *ClusterRoleService) UpdateClusterRole(ctx context.Context, req *model.UpdateClusterRoleReq) error {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 如果名称发生变化，需要删除原来的ClusterRole并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原ClusterRole
		err = k8sClient.RbacV1().ClusterRoles().Delete(ctx, req.OriginalName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete original cluster role: %w", err)
		}

		// 创建新ClusterRole
		createReq := &model.CreateClusterRoleReq{
			ClusterID:   req.ClusterID,
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
			Rules:       req.Rules,
		}
		return crs.CreateClusterRole(ctx, createReq)
	}

	// 获取现有ClusterRole
	existingClusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role: %w", err)
	}

	// 更新ClusterRole
	existingClusterRole.Labels = req.Labels
	existingClusterRole.Annotations = req.Annotations
	existingClusterRole.Rules = utils.ConvertPolicyRulesToK8s(req.Rules)

	_, err = k8sClient.RbacV1().ClusterRoles().Update(ctx, existingClusterRole, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	return nil
}

// DeleteClusterRole 删除ClusterRole
func (crs *ClusterRoleService) DeleteClusterRole(ctx context.Context, req *model.DeleteClusterRoleReq) error {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	err = k8sClient.RbacV1().ClusterRoles().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster role: %w", err)
	}

	return nil
}

// BatchDeleteClusterRole 批量删除ClusterRole
func (crs *ClusterRoleService) BatchDeleteClusterRole(ctx context.Context, req *model.BatchDeleteClusterRoleReq) error {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var errors []string
	for _, name := range req.Names {
		err := k8sClient.RbacV1().ClusterRoles().Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete cluster role %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetClusterRoleYaml 获取ClusterRole的YAML配置
func (crs *ClusterRoleService) GetClusterRoleYaml(ctx context.Context, req *model.ClusterRoleGetReq) (string, error) {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return "", fmt.Errorf("failed to get k8s client: %w", err)
	}

	clusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster role: %w", err)
	}

	// 清理不需要的字段
	clusterRole.ManagedFields = nil
	clusterRole.ResourceVersion = ""
	clusterRole.UID = ""
	clusterRole.SelfLink = ""
	clusterRole.CreationTimestamp = metav1.Time{}
	clusterRole.Generation = 0

	yamlData, err := yaml.Marshal(clusterRole)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster role to yaml: %w", err)
	}

	return string(yamlData), nil
}

// UpdateClusterRoleYaml 通过YAML更新ClusterRole
func (crs *ClusterRoleService) UpdateClusterRoleYaml(ctx context.Context, req *model.ClusterRoleYamlReq) error {
	k8sClient, err := crs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var clusterRole rbacv1.ClusterRole
	err = yaml.Unmarshal([]byte(req.YamlContent), &clusterRole)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称一致
	clusterRole.Name = req.Name

	// 获取现有ClusterRole以保持ResourceVersion
	existingClusterRole, err := k8sClient.RbacV1().ClusterRoles().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role: %w", err)
	}

	clusterRole.ResourceVersion = existingClusterRole.ResourceVersion
	clusterRole.UID = existingClusterRole.UID

	_, err = k8sClient.RbacV1().ClusterRoles().Update(ctx, &clusterRole, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	return nil
}
