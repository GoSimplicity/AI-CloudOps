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

type RoleBindingService struct {
	dao       dao.ClusterDAO
	k8sClient client.K8sClient
	logger    *zap.Logger
}

func NewRoleBindingService(dao dao.ClusterDAO, k8sClient client.K8sClient, logger *zap.Logger) *RoleBindingService {
	return &RoleBindingService{
		dao:       dao,
		k8sClient: k8sClient,
		logger:    logger,
	}
}

// GetRoleBindingList 获取RoleBinding列表
func (rbs *RoleBindingService) GetRoleBindingList(ctx context.Context, req *model.RoleBindingListReq) (*model.ListResp[model.RoleBindingInfo], error) {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 构建列表选项
	listOptions := metav1.ListOptions{}

	// 如果指定了命名空间，则在该命名空间下查询，否则查询所有命名空间
	var roleBindings *rbacv1.RoleBindingList
	if req.Namespace != "" {
		roleBindings, err = k8sClient.RbacV1().RoleBindings(req.Namespace).List(ctx, listOptions)
	} else {
		// 获取所有命名空间的RoleBindings
		namespaces, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list namespaces: %w", err)
		}

		allRoleBindings := &rbacv1.RoleBindingList{}
		for _, ns := range namespaces.Items {
			nsRoleBindings, err := k8sClient.RbacV1().RoleBindings(ns.Name).List(ctx, listOptions)
			if err != nil {
				continue // 跳过无权限的命名空间
			}
			allRoleBindings.Items = append(allRoleBindings.Items, nsRoleBindings.Items...)
		}
		roleBindings = allRoleBindings
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings: %w", err)
	}

	// 转换为响应格式并过滤
	var roleBindingInfos []model.RoleBindingInfo
	for _, roleBinding := range roleBindings.Items {
		roleBindingInfo := k8sutils.ConvertK8sRoleBindingToRoleBindingInfo(&roleBinding, req.ClusterID)

		// 关键字过滤
		if req.Keyword != "" && !strings.Contains(roleBindingInfo.Name, req.Keyword) {
			continue
		}

		roleBindingInfos = append(roleBindingInfos, roleBindingInfo)
	}

	// 排序
	sort.Slice(roleBindingInfos, func(i, j int) bool {
		return roleBindingInfos[i].CreationTimestamp > roleBindingInfos[j].CreationTimestamp
	})

	// 分页
	total := len(roleBindingInfos)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		roleBindingInfos = []model.RoleBindingInfo{}
	} else if end > total {
		roleBindingInfos = roleBindingInfos[start:]
	} else {
		roleBindingInfos = roleBindingInfos[start:end]
	}

	return &model.ListResp[model.RoleBindingInfo]{
		Items: roleBindingInfos,
		Total: int64(total),
	}, nil
}

// GetRoleBindingDetails 获取RoleBinding详情
func (rbs *RoleBindingService) GetRoleBindingDetails(ctx context.Context, req *model.RoleBindingGetReq) (*model.RoleBindingInfo, error) {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	roleBinding, err := k8sClient.RbacV1().RoleBindings(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get role binding: %w", err)
	}

	roleBindingInfo := k8sutils.ConvertK8sRoleBindingToRoleBindingInfo(roleBinding, req.ClusterID)
	return &roleBindingInfo, nil
}

// CreateRoleBinding 创建RoleBinding
func (rbs *RoleBindingService) CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		RoleRef:  k8sutils.ConvertRoleRefToK8s(req.RoleRef),
		Subjects: k8sutils.ConvertSubjectsToK8s(req.Subjects),
	}

	_, err = k8sClient.RbacV1().RoleBindings(req.Namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	return nil
}

// UpdateRoleBinding 更新RoleBinding
func (rbs *RoleBindingService) UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	// 如果名称发生变化，需要删除原来的RoleBinding并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原RoleBinding
		err = k8sClient.RbacV1().RoleBindings(req.Namespace).Delete(ctx, req.OriginalName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("failed to delete original role binding: %w", err)
		}

		// 创建新RoleBinding
		createReq := &model.CreateRoleBindingReq{
			ClusterID:   req.ClusterID,
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
			RoleRef:     req.RoleRef,
			Subjects:    req.Subjects,
		}
		return rbs.CreateRoleBinding(ctx, createReq)
	}

	// 获取现有RoleBinding
	existingRoleBinding, err := k8sClient.RbacV1().RoleBindings(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing role binding: %w", err)
	}

	// 更新RoleBinding
	existingRoleBinding.Labels = req.Labels
	existingRoleBinding.Annotations = req.Annotations
	existingRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	_, err = k8sClient.RbacV1().RoleBindings(req.Namespace).Update(ctx, existingRoleBinding, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update role binding: %w", err)
	}

	return nil
}

// DeleteRoleBinding 删除RoleBinding
func (rbs *RoleBindingService) DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	err = k8sClient.RbacV1().RoleBindings(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete role binding: %w", err)
	}

	return nil
}

// BatchDeleteRoleBinding 批量删除RoleBinding
func (rbs *RoleBindingService) BatchDeleteRoleBinding(ctx context.Context, req *model.BatchDeleteRoleBindingReq) error {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var errors []string
	for _, binding := range req.Bindings {
		err := k8sClient.RbacV1().RoleBindings(binding.Namespace).Delete(ctx, binding.Name, metav1.DeleteOptions{})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete role binding %s/%s: %v", binding.Namespace, binding.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetRoleBindingYaml 获取RoleBinding的YAML配置
func (rbs *RoleBindingService) GetRoleBindingYaml(ctx context.Context, req *model.RoleBindingGetReq) (string, error) {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return "", fmt.Errorf("failed to get k8s client: %w", err)
	}

	roleBinding, err := k8sClient.RbacV1().RoleBindings(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get role binding: %w", err)
	}

	// 清理不需要的字段
	roleBinding.ManagedFields = nil
	roleBinding.ResourceVersion = ""
	roleBinding.UID = ""
	roleBinding.SelfLink = ""
	roleBinding.CreationTimestamp = metav1.Time{}
	roleBinding.Generation = 0

	yamlData, err := yaml.Marshal(roleBinding)
	if err != nil {
		return "", fmt.Errorf("failed to marshal role binding to yaml: %w", err)
	}

	return string(yamlData), nil
}

// UpdateRoleBindingYaml 通过YAML更新RoleBinding
func (rbs *RoleBindingService) UpdateRoleBindingYaml(ctx context.Context, req *model.RoleBindingYamlReq) error {
	k8sClient, err := rbs.k8sClient.GetKubeClient(req.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to get k8s client: %w", err)
	}

	var roleBinding rbacv1.RoleBinding
	err = yaml.Unmarshal([]byte(req.YamlContent), &roleBinding)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称和命名空间一致
	roleBinding.Name = req.Name
	roleBinding.Namespace = req.Namespace

	// 获取现有RoleBinding以保持ResourceVersion
	existingRoleBinding, err := k8sClient.RbacV1().RoleBindings(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get existing role binding: %w", err)
	}

	roleBinding.ResourceVersion = existingRoleBinding.ResourceVersion
	roleBinding.UID = existingRoleBinding.UID

	_, err = k8sClient.RbacV1().RoleBindings(req.Namespace).Update(ctx, &roleBinding, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update role binding: %w", err)
	}

	return nil
}
