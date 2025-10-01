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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceService interface {
	CreateNamespace(ctx context.Context, req *model.K8sNamespaceCreateReq) error
	DeleteNamespace(ctx context.Context, req *model.K8sNamespaceDeleteReq) error
	GetNamespaceDetails(ctx context.Context, req *model.K8sNamespaceGetDetailsReq) (*model.K8sNamespace, error)
	UpdateNamespace(ctx context.Context, req *model.K8sNamespaceUpdateReq) error
	ListNamespaces(ctx context.Context, req *model.K8sNamespaceListReq) (model.ListResp[*model.K8sNamespace], error)
}

type namespaceService struct {
	client           client.K8sClient
	namespaceManager manager.NamespaceManager
	logger           *zap.Logger
}

func NewNamespaceService(client client.K8sClient, namespaceManager manager.NamespaceManager, logger *zap.Logger) NamespaceService {
	return &namespaceService{
		client:           client,
		namespaceManager: namespaceManager,
		logger:           logger,
	}
}

// CreateNamespace 创建命名空间
func (s *namespaceService) CreateNamespace(ctx context.Context, req *model.K8sNamespaceCreateReq) error {
	if req == nil {
		return fmt.Errorf("创建命名空间请求不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("命名空间名称不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	// 验证命名空间名称
	if err := utils.ValidateNamespaceName(req.Name); err != nil {
		s.logger.Error("CreateNamespace: 命名空间名称验证失败", zap.Error(err), zap.String("name", req.Name))
		return fmt.Errorf("命名空间名称验证失败: %w", err)
	}

	// 验证标签
	if err := utils.ValidateNodeLabelsMap(utils.ConvertKeyValueListToLabels(req.Labels)); err != nil {
		s.logger.Error("CreateNamespace: 标签验证失败", zap.Error(err))
		return fmt.Errorf("标签验证失败: %w", err)
	}

	// 验证注解
	if err := utils.ValidateAnnotations(req.Annotations); err != nil {
		s.logger.Error("CreateNamespace: 注解验证失败", zap.Error(err))
		return fmt.Errorf("注解验证失败: %w", err)
	}

	// 转换标签和注解
	labelsMap := utils.ConvertKeyValueListToLabels(req.Labels)
	annotationsMap := utils.ConvertKeyValueListToLabels(req.Annotations)

	// 创建命名空间对象
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      labelsMap,
			Annotations: annotationsMap,
		},
	}

	// 使用 NamespaceManager 创建命名空间
	_, err := s.namespaceManager.CreateNamespace(ctx, req.ClusterID, namespace)
	if err != nil {
		s.logger.Error("CreateNamespace: 创建命名空间失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return fmt.Errorf("创建命名空间失败: %w", err)
	}

	return nil
}

// DeleteNamespace 删除命名空间
func (s *namespaceService) DeleteNamespace(ctx context.Context, req *model.K8sNamespaceDeleteReq) error {
	if req == nil {
		return fmt.Errorf("删除命名空间请求不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("命名空间名称不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	deleteOptions := metav1.DeleteOptions{}

	// 设置优雅删除时间
	if req.GracePeriodSeconds != nil {
		deleteOptions.GracePeriodSeconds = req.GracePeriodSeconds
	}

	// 设置强制删除选项
	if req.Force == 1 {
		gracePeriod := int64(0)
		deleteOptions.GracePeriodSeconds = &gracePeriod
		deleteOptions.Preconditions = &metav1.Preconditions{
			UID: nil,
		}
	}

	// 使用 NamespaceManager 删除命名空间
	err := s.namespaceManager.DeleteNamespace(ctx, req.ClusterID, req.Name, deleteOptions)
	if err != nil {
		s.logger.Error("DeleteNamespace: 删除命名空间失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return fmt.Errorf("删除命名空间失败: %w", err)
	}

	return nil
}

// GetNamespaceDetails 获取命名空间详情
func (s *namespaceService) GetNamespaceDetails(ctx context.Context, req *model.K8sNamespaceGetDetailsReq) (*model.K8sNamespace, error) {
	if req == nil {
		return nil, fmt.Errorf("获取命名空间详情请求不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("命名空间名称不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群 ID 不能为空")
	}

	// 使用 NamespaceManager 获取命名空间详情
	namespace, err := s.namespaceManager.GetNamespace(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("GetNamespaceDetails: 获取命名空间详情失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return nil, fmt.Errorf("获取命名空间详情失败: %w", err)
	}

	// 使用 utils 转换标签和注解
	labels := utils.ConvertLabelsToKeyValueList(namespace.Labels)
	annotations := utils.ConvertLabelsToKeyValueList(namespace.Annotations)

	return &model.K8sNamespace{
		ClusterID:   req.ClusterID,
		Name:        namespace.Name,
		UID:         string(namespace.UID),
		Status:      utils.GetNamespaceStatus(namespace.Status.Phase),
		Phase:       string(namespace.Status.Phase),
		Labels:      labels,
		Annotations: annotations,
	}, nil
}

// UpdateNamespace 更新命名空间
func (s *namespaceService) UpdateNamespace(ctx context.Context, req *model.K8sNamespaceUpdateReq) error {
	if req == nil {
		return fmt.Errorf("更新命名空间请求不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("命名空间名称不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群 ID 不能为空")
	}

	// 验证标签
	if err := utils.ValidateNodeLabelsMap(utils.ConvertKeyValueListToLabels(req.Labels)); err != nil {
		s.logger.Error("UpdateNamespace: 标签验证失败", zap.Error(err))
		return fmt.Errorf("标签验证失败: %w", err)
	}

	// 验证注解
	if err := utils.ValidateAnnotations(req.Annotations); err != nil {
		s.logger.Error("UpdateNamespace: 注解验证失败", zap.Error(err))
		return fmt.Errorf("注解验证失败: %w", err)
	}

	// 获取现有命名空间
	namespace, err := s.namespaceManager.GetNamespace(ctx, req.ClusterID, req.Name)
	if err != nil {
		s.logger.Error("UpdateNamespace: 获取命名空间失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return fmt.Errorf("获取命名空间失败: %w", err)
	}

	// 转换标签和注解
	labelsMap := utils.ConvertKeyValueListToLabels(req.Labels)
	annotationsMap := utils.ConvertKeyValueListToLabels(req.Annotations)

	// 更新命名空间标签和注释
	namespace.Labels = labelsMap
	namespace.Annotations = annotationsMap

	// 使用 NamespaceManager 更新命名空间
	_, err = s.namespaceManager.UpdateNamespace(ctx, req.ClusterID, namespace)
	if err != nil {
		s.logger.Error("UpdateNamespace: 更新命名空间失败", zap.Error(err), zap.Int("clusterID", req.ClusterID), zap.String("name", req.Name))
		return fmt.Errorf("更新命名空间失败: %w", err)
	}

	return nil
}

// ListNamespaces 获取命名空间列表
func (s *namespaceService) ListNamespaces(ctx context.Context, req *model.K8sNamespaceListReq) (model.ListResp[*model.K8sNamespace], error) {
	if req == nil {
		return model.ListResp[*model.K8sNamespace]{}, fmt.Errorf("获取命名空间列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sNamespace]{}, fmt.Errorf("集群 ID 不能为空")
	}

	// 验证过滤参数
	if err := utils.ValidateNamespaceFilters(req); err != nil {
		s.logger.Error("ListNamespaces: 过滤参数验证失败", zap.Error(err))
		return model.ListResp[*model.K8sNamespace]{}, fmt.Errorf("过滤参数验证失败: %w", err)
	}

	// 构建查询选项
	listOptions := utils.BuildNamespaceListOptions(req)

	// 使用 NamespaceManager 获取命名空间列表
	namespaceList, err := s.namespaceManager.GetNamespaceList(ctx, req.ClusterID, listOptions)
	if err != nil {
		s.logger.Error("ListNamespaces: 获取命名空间列表失败", zap.Error(err), zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sNamespace]{}, fmt.Errorf("获取命名空间列表失败: %w", err)
	}

	namespaces := namespaceList.Items

	// 根据条件过滤命名空间
	if req.Status != "" {
		namespaces = utils.FilterNamespacesByStatus(namespaces, req.Status)
	}

	// 根据搜索关键字过滤
	if req.Search != "" {
		namespaces = utils.FilterNamespacesBySearch(namespaces, req.Search)
	}

	// 根据标签过滤（如果没有使用标签选择器）
	if len(req.Labels) > 0 && req.LabelSelector == "" {
		labelsMap := utils.ConvertKeyValueListToLabels(req.Labels)
		namespaces = utils.FilterNamespacesByLabels(namespaces, labelsMap)
	}

	// 使用工具函数进行分页处理
	pagedNamespaces, total := utils.BuildNamespaceListPagination(namespaces, req.Page, req.Size)

	// 转换为响应格式
	var items []*model.K8sNamespace
	for _, ns := range pagedNamespaces {
		labels := utils.ConvertLabelsToKeyValueList(ns.Labels)
		annotations := utils.ConvertLabelsToKeyValueList(ns.Annotations)

		k8sNamespace := &model.K8sNamespace{
			ClusterID:   req.ClusterID,
			Name:        ns.Name,
			UID:         string(ns.UID),
			Status:      utils.GetNamespaceStatus(ns.Status.Phase),
			Phase:       string(ns.Status.Phase),
			Labels:      labels,
			Annotations: annotations,
		}
		items = append(items, k8sNamespace)
	}

	return model.ListResp[*model.K8sNamespace]{
		Total: total,
		Items: items,
	}, nil
}
