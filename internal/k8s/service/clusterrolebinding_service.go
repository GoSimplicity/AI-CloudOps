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
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleBindingService struct {
	dao       dao.ClusterDAO
	k8sClient client.K8sClient
	logger    *zap.Logger
}

func NewClusterRoleBindingService(dao dao.ClusterDAO, k8sClient client.K8sClient, logger *zap.Logger) *ClusterRoleBindingService {
	return &ClusterRoleBindingService{
		dao:       dao,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表
func (crbs *ClusterRoleBindingService) GetClusterRoleBindingList(ctx context.Context, req *model.ClusterRoleBindingListReq) (*model.ListResp[model.ClusterRoleBindingInfo], error) {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 构建列表选项
	listOptions := metav1.ListOptions{}

	clusterRoleBindings, err := k8sClient.RbacV1().ClusterRoleBindings().List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster role bindings: %w", err)
	}

	// 转换为响应格式并过滤
	var clusterRoleBindingInfos []model.ClusterRoleBindingInfo
	for _, clusterRoleBinding := range clusterRoleBindings.Items {
		clusterRoleBindingInfo := k8sutils.ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(&clusterRoleBinding, req.ClusterID)

		// 关键字过滤
		if req.Keyword != "" && !strings.Contains(clusterRoleBindingInfo.Name, req.Keyword) {
			continue
		}

		clusterRoleBindingInfos = append(clusterRoleBindingInfos, clusterRoleBindingInfo)
	}

	// 排序
	sort.Slice(clusterRoleBindingInfos, func(i, j int) bool {
		return clusterRoleBindingInfos[i].CreationTimestamp > clusterRoleBindingInfos[j].CreationTimestamp
	})

	// 分页
	total := len(clusterRoleBindingInfos)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		clusterRoleBindingInfos = []model.ClusterRoleBindingInfo{}
	} else if end > total {
		clusterRoleBindingInfos = clusterRoleBindingInfos[start:]
	} else {
		clusterRoleBindingInfos = clusterRoleBindingInfos[start:end]
	}

	return &model.ListResp[model.ClusterRoleBindingInfo]{
		Items: clusterRoleBindingInfos,
		Total: int64(total),
	}, nil
}

// GetClusterRoleBindingDetails 获取ClusterRoleBinding详情
func (crbs *ClusterRoleBindingService) GetClusterRoleBindingDetails(ctx context.Context, req *model.ClusterRoleBindingGetReq) (*model.ClusterRoleBindingInfo, error) {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	clusterRoleBinding, err := k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster role binding: %w", err)
	}

	clusterRoleBindingInfo := k8sutils.ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(clusterRoleBinding, req.ClusterID)
	return &clusterRoleBindingInfo, nil
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
func (crbs *ClusterRoleBindingService) CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
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

	_, err = k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	return nil
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (crbs *ClusterRoleBindingService) UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 如果名称发生变化，需要删除原来的ClusterRoleBinding并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原ClusterRoleBinding
		err = k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, req.OriginalName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete original cluster role binding: %w", err)
		}

		// 创建新ClusterRoleBinding
		createReq := &model.CreateClusterRoleBindingReq{
			ClusterID:   req.ClusterID,
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
			RoleRef:     req.RoleRef,
			Subjects:    req.Subjects,
		}
		return crbs.CreateClusterRoleBinding(ctx, createReq)
	}

	// 获取现有ClusterRoleBinding
	existingClusterRoleBinding, err := k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	// 更新ClusterRoleBinding
	existingClusterRoleBinding.Labels = req.Labels
	existingClusterRoleBinding.Annotations = req.Annotations
	existingClusterRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingClusterRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	_, err = k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, existingClusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (crbs *ClusterRoleBindingService) DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	err = k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster role binding: %w", err)
	}

	return nil
}

// BatchDeleteClusterRoleBinding 批量删除ClusterRoleBinding
func (crbs *ClusterRoleBindingService) BatchDeleteClusterRoleBinding(ctx context.Context, req *model.BatchDeleteClusterRoleBindingReq) error {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var errors []string
	for _, name := range req.Names {
		err := k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete cluster role binding %s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetClusterRoleBindingYaml 获取ClusterRoleBinding的YAML配置
func (crbs *ClusterRoleBindingService) GetClusterRoleBindingYaml(ctx context.Context, req *model.ClusterRoleBindingGetReq) (string, error) {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return "", fmt.Errorf("failed to get k8s client: %w", err)
	}

	clusterRoleBinding, err := k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster role binding: %w", err)
	}

	// 清理不需要的字段
	clusterRoleBinding.ManagedFields = nil
	clusterRoleBinding.ResourceVersion = ""
	clusterRoleBinding.UID = ""
	clusterRoleBinding.SelfLink = ""
	clusterRoleBinding.CreationTimestamp = metav1.Time{}
	clusterRoleBinding.Generation = 0

	yamlData, err := yaml.Marshal(clusterRoleBinding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster role binding to yaml: %w", err)
	}

	return string(yamlData), nil
}

// UpdateClusterRoleBindingYaml 通过YAML更新ClusterRoleBinding
func (crbs *ClusterRoleBindingService) UpdateClusterRoleBindingYaml(ctx context.Context, req *model.ClusterRoleBindingYamlReq) error {
	k8sClient, err := crbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err = yaml.Unmarshal([]byte(req.YamlContent), &clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称一致
	clusterRoleBinding.Name = req.Name

	// 获取现有ClusterRoleBinding以保持ResourceVersion
	existingClusterRoleBinding, err := k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
	clusterRoleBinding.UID = existingClusterRoleBinding.UID

	_, err = k8sClient.RbacV1().ClusterRoleBindings().Update(ctx, &clusterRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}
