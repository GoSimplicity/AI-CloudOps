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

type RoleBindingService struct {
	dao         dao.ClusterDAO
	rbacManager manager.RBACManager
	logger      *zap.Logger
}

func NewRoleBindingService(dao dao.ClusterDAO, rbacManager manager.RBACManager, logger *zap.Logger) *RoleBindingService {
	return &RoleBindingService{
		dao:         dao,
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// GetRoleBindingList 获取RoleBinding列表
func (rbs *RoleBindingService) GetRoleBindingList(ctx context.Context, req *model.RoleBindingListReq) (*model.ListResp[model.RoleBindingInfo], error) {
	// 使用 RBACManager 获取 RoleBinding 列表
	roleBindings, err := rbs.rbacManager.GetRoleBindingList(ctx, req.ClusterID, req.Namespace, metav1.ListOptions{})
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
	// 使用 RBACManager 获取 RoleBinding 详情
	roleBinding, err := rbs.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role binding: %w", err)
	}

	roleBindingInfo := k8sutils.ConvertK8sRoleBindingToRoleBindingInfo(roleBinding, req.ClusterID)
	return &roleBindingInfo, nil
}

// CreateRoleBinding 创建RoleBinding
func (rbs *RoleBindingService) CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error {
	// 构建 RoleBinding 对象
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

	// 使用 RBACManager 创建 RoleBinding
	err := rbs.rbacManager.CreateRoleBinding(ctx, req.ClusterID, req.Namespace, roleBinding)
	if err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	return nil
}

// UpdateRoleBinding 更新RoleBinding
func (rbs *RoleBindingService) UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error {

	// 如果名称发生变化，需要删除原来的RoleBinding并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原RoleBinding
		err := rbs.rbacManager.DeleteRoleBinding(ctx, req.ClusterID, req.Namespace, req.OriginalName, metav1.DeleteOptions{})
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
	existingRoleBinding, err := rbs.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role binding: %w", err)
	}

	// 更新RoleBinding
	existingRoleBinding.Labels = req.Labels
	existingRoleBinding.Annotations = req.Annotations
	existingRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	// 使用 RBACManager 更新 RoleBinding
	err = rbs.rbacManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, existingRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update role binding: %w", err)
	}

	return nil
}

// DeleteRoleBinding 删除RoleBinding
func (rbs *RoleBindingService) DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error {

	// 使用 RBACManager 删除 RoleBinding
	err := rbs.rbacManager.DeleteRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete role binding: %w", err)
	}

	return nil
}

// BatchDeleteRoleBinding 批量删除RoleBinding
func (rbs *RoleBindingService) BatchDeleteRoleBinding(ctx context.Context, req *model.BatchDeleteRoleBindingReq) error {

	// 批量删除 RoleBinding - 按命名空间分组处理
	namespaceRoleBindings := make(map[string][]string)
	for _, binding := range req.Bindings {
		namespaceRoleBindings[binding.Namespace] = append(namespaceRoleBindings[binding.Namespace], binding.Name)
	}

	// 使用 RBACManager 分命名空间批量删除 RoleBinding
	var errors []string
	for namespace, roleBindingNames := range namespaceRoleBindings {
		err := rbs.rbacManager.BatchDeleteRoleBindings(ctx, req.ClusterID, namespace, roleBindingNames)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to delete role bindings in namespace %s: %v", namespace, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("batch delete errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetRoleBindingYaml 获取RoleBinding的YAML配置
func (rbs *RoleBindingService) GetRoleBindingYaml(ctx context.Context, req *model.RoleBindingGetReq) (string, error) {
	// 获取 RoleBinding
	roleBinding, err := rbs.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
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

	var roleBinding rbacv1.RoleBinding
	err := yaml.Unmarshal([]byte(req.YamlContent), &roleBinding)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称和命名空间一致
	roleBinding.Name = req.Name
	roleBinding.Namespace = req.Namespace

	// 获取现有RoleBinding以保持ResourceVersion
	existingRoleBinding, err := rbs.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing role binding: %w", err)
	}

	roleBinding.ResourceVersion = existingRoleBinding.ResourceVersion
	roleBinding.UID = existingRoleBinding.UID

	// 使用 RBACManager 更新 RoleBinding
	err = rbs.rbacManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, &roleBinding)
	if err != nil {
		return fmt.Errorf("failed to update role binding: %w", err)
	}

	return nil
}
