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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RoleBindingService interface {
	// 基础 CRUD 操作
	GetRoleBindingList(ctx context.Context, req *model.GetRoleBindingListReq) (model.ListResp[*model.K8sRoleBinding], error)
	GetRoleBindingDetails(ctx context.Context, req *model.GetRoleBindingDetailsReq) (*model.K8sRoleBinding, error)
	CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error
	UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error
	DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error

	// YAML 操作
	GetRoleBindingYaml(ctx context.Context, req *model.GetRoleBindingYamlReq) (*model.K8sYaml, error)
	UpdateRoleBindingYaml(ctx context.Context, req *model.UpdateRoleBindingYamlReq) error

	// 扩展功能
	GetRoleBindingEvents(ctx context.Context, req *model.GetRoleBindingEventsReq) (model.ListResp[*model.K8sRoleBindingEvent], error)
	GetRoleBindingUsage(ctx context.Context, req *model.GetRoleBindingUsageReq) (*model.K8sRoleBindingUsage, error)
}

type roleBindingService struct {
	rbacManager manager.RBACManager
}

func NewRoleBindingService(rbacManager manager.RBACManager) RoleBindingService {
	return &roleBindingService{
		rbacManager: rbacManager,
	}
}

// ======================== 基础 CRUD 操作 ========================

func (s *roleBindingService) GetRoleBindingList(ctx context.Context, req *model.GetRoleBindingListReq) (model.ListResp[*model.K8sRoleBinding], error) {
	// 构建查询选项
	options := k8sutils.BuildRoleBindingListOptions(req)

	// 从 Manager 获取原始 RoleBinding 列表
	roleBindings, err := s.rbacManager.GetRoleBindingList(ctx, req.ClusterID, req.Namespace, options)
	if err != nil {
		return model.ListResp[*model.K8sRoleBinding]{}, err
	}

	// 分页处理
	paginatedRoleBindings, err := k8sutils.PaginateK8sRoleBindings(roleBindings, req.Page, req.PageSize)
	if err != nil {
		return model.ListResp[*model.K8sRoleBinding]{}, err
	}

	return paginatedRoleBindings, nil
}

func (s *roleBindingService) GetRoleBindingDetails(ctx context.Context, req *model.GetRoleBindingDetailsReq) (*model.K8sRoleBinding, error) {
	roleBinding, err := s.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	return k8sutils.ConvertK8sRoleBindingToRoleBindingInfo(roleBinding, req.ClusterID), nil
}

func (s *roleBindingService) CreateRoleBinding(ctx context.Context, req *model.CreateRoleBindingReq) error {
	roleBinding := k8sutils.ConvertToK8sRoleBinding(req)
	return s.rbacManager.CreateRoleBinding(ctx, req.ClusterID, req.Namespace, roleBinding)
}

func (s *roleBindingService) UpdateRoleBinding(ctx context.Context, req *model.UpdateRoleBindingReq) error {
	roleBinding := &model.CreateRoleBindingReq{
		ClusterID:   req.ClusterID,
		Namespace:   req.Namespace,
		Name:        req.Name,
		RoleRef:     req.RoleRef,
		Subjects:    req.Subjects,
		Labels:      req.Labels,
		Annotations: req.Annotations,
	}
	return s.rbacManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, k8sutils.ConvertToK8sRoleBinding(roleBinding))
}

func (s *roleBindingService) DeleteRoleBinding(ctx context.Context, req *model.DeleteRoleBindingReq) error {
	return s.rbacManager.DeleteRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
}

// ======================== YAML 操作 ========================

func (s *roleBindingService) GetRoleBindingYaml(ctx context.Context, req *model.GetRoleBindingYamlReq) (*model.K8sYaml, error) {
	roleBinding, err := s.rbacManager.GetRoleBinding(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}

	yamlContent, err := k8sutils.RoleBindingToYAML(roleBinding)
	if err != nil {
		return nil, err
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *roleBindingService) UpdateRoleBindingYaml(ctx context.Context, req *model.UpdateRoleBindingYamlReq) error {
	roleBinding, err := k8sutils.YAMLToRoleBinding(req.Yaml)
	if err != nil {
		return err
	}

	return s.rbacManager.UpdateRoleBinding(ctx, req.ClusterID, req.Namespace, roleBinding)
}

// ======================== 扩展功能 ========================

func (s *roleBindingService) GetRoleBindingEvents(ctx context.Context, req *model.GetRoleBindingEventsReq) (model.ListResp[*model.K8sRoleBindingEvent], error) {
	events, err := s.rbacManager.GetRoleBindingEvents(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return model.ListResp[*model.K8sRoleBindingEvent]{}, err
	}
	return events, nil
}

func (s *roleBindingService) GetRoleBindingUsage(ctx context.Context, req *model.GetRoleBindingUsageReq) (*model.K8sRoleBindingUsage, error) {
	usage, err := s.rbacManager.GetRoleBindingUsage(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, err
	}
	return usage, nil
}
