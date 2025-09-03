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

type ClusterRoleBindingService interface {
	// 基础 CRUD 操作
	GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error)
	GetClusterRoleBindingDetails(ctx context.Context, req *model.GetClusterRoleBindingDetailsReq) (*model.K8sClusterRoleBinding, error)
	CreateClusterRoleBinding(ctx context.Context, req *model.CreateClusterRoleBindingReq) error
	UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error
	DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error

	// YAML 操作
	GetClusterRoleBindingYaml(ctx context.Context, req *model.GetClusterRoleBindingYamlReq) (string, error)
	UpdateClusterRoleBindingYaml(ctx context.Context, req *model.UpdateClusterRoleBindingYamlReq) error

	// 扩展功能
	GetClusterRoleBindingEvents(ctx context.Context, req *model.GetClusterRoleBindingEventsReq) ([]*model.K8sClusterRoleBindingEvent, error)
	GetClusterRoleBindingUsage(ctx context.Context, req *model.GetClusterRoleBindingUsageReq) (*model.K8sClusterRoleBindingUsage, error)

	// 兼容性方法（保持现有API接口兼容）
	GetClusterRoleBindingListCompat(ctx context.Context, req *model.ClusterRoleBindingListReq) (*model.ListResp[model.ClusterRoleBindingInfo], error)
	GetClusterRoleBindingDetailsCompat(ctx context.Context, req *model.ClusterRoleBindingGetReq) (*model.ClusterRoleBindingInfo, error)
	GetClusterRoleBindingYamlCompat(ctx context.Context, req *model.ClusterRoleBindingGetReq) (string, error)
	UpdateClusterRoleBindingYamlCompat(ctx context.Context, req *model.ClusterRoleBindingYamlReq) error
}

type clusterRoleBindingService struct {
	dao         dao.ClusterDAO
	rbacManager manager.RBACManager
	logger      *zap.Logger
}

func NewClusterRoleBindingService(dao dao.ClusterDAO, rbacManager manager.RBACManager, logger *zap.Logger) ClusterRoleBindingService {
	return &clusterRoleBindingService{
		dao:         dao,
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表（新接口）
func (c *clusterRoleBindingService) GetClusterRoleBindingList(ctx context.Context, req *model.GetClusterRoleBindingListReq) (model.ListResp[*model.K8sClusterRoleBinding], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := k8sutils.BuildClusterRoleBindingListOptions(req)

	clusterRoleBindingList, err := c.rbacManager.GetClusterRoleBindingListRaw(ctx, req.ClusterID, listOptions)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingList: 获取ClusterRoleBinding列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sClusterRoleBinding]{}, fmt.Errorf("获取ClusterRoleBinding列表失败: %w", err)
	}

	// 转换为模型格式
	var k8sClusterRoleBindings []*model.K8sClusterRoleBinding
	for i := range clusterRoleBindingList.Items {
		k8sClusterRoleBinding := k8sutils.ConvertToK8sClusterRoleBinding(&clusterRoleBindingList.Items[i])
		if k8sClusterRoleBinding != nil {
			k8sClusterRoleBinding.ClusterID = req.ClusterID

			// 关键字过滤
			if req.Keyword == "" || strings.Contains(k8sClusterRoleBinding.Name, req.Keyword) {
				k8sClusterRoleBindings = append(k8sClusterRoleBindings, k8sClusterRoleBinding)
			}
		}
	}

	// 分页处理
	pagedClusterRoleBindings, total := k8sutils.PaginateK8sClusterRoleBindings(k8sClusterRoleBindings, req.Page, req.PageSize)

	c.logger.Debug("GetClusterRoleBindingList: 获取ClusterRoleBinding列表成功",
		zap.Int("clusterID", req.ClusterID),
		zap.Int64("total", total),
		zap.Int("returned", len(pagedClusterRoleBindings)))

	return model.ListResp[*model.K8sClusterRoleBinding]{
		Items: pagedClusterRoleBindings,
		Total: total,
	}, nil
}

// GetClusterRoleBindingListCompat 获取ClusterRoleBinding列表（兼容性方法）
func (c *clusterRoleBindingService) GetClusterRoleBindingListCompat(ctx context.Context, req *model.ClusterRoleBindingListReq) (*model.ListResp[model.ClusterRoleBindingInfo], error) {
	// 使用 RBACManager 获取 ClusterRoleBinding 列表
	clusterRoleBindings, err := c.rbacManager.GetClusterRoleBindingListRaw(ctx, req.ClusterID, metav1.ListOptions{})
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

// GetClusterRoleBindingDetails 获取ClusterRoleBinding详情（新接口）
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

	clusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingDetails: 获取ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding失败: %w", err)
	}

	k8sClusterRoleBinding, err := k8sutils.BuildK8sClusterRoleBinding(ctx, req.ClusterID, *clusterRoleBinding)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingDetails: 构建ClusterRoleBinding失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建ClusterRoleBinding失败: %w", err)
	}

	c.logger.Debug("GetClusterRoleBindingDetails: 获取ClusterRoleBinding详情成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return k8sClusterRoleBinding, nil
}

// GetClusterRoleBindingDetailsCompat 获取ClusterRoleBinding详情（兼容性方法）
func (c *clusterRoleBindingService) GetClusterRoleBindingDetailsCompat(ctx context.Context, req *model.ClusterRoleBindingGetReq) (*model.ClusterRoleBindingInfo, error) {
	// 使用 RBACManager 获取 ClusterRoleBinding 详情
	clusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster role binding: %w", err)
	}

	clusterRoleBindingInfo := k8sutils.ConvertK8sClusterRoleBindingToClusterRoleBindingInfo(clusterRoleBinding, req.ClusterID)
	return &clusterRoleBindingInfo, nil
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

	// 使用 RBACManager 创建 ClusterRoleBinding
	err := c.rbacManager.CreateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to create cluster role binding: %w", err)
	}

	return nil
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (c *clusterRoleBindingService) UpdateClusterRoleBinding(ctx context.Context, req *model.UpdateClusterRoleBindingReq) error {

	// 如果名称发生变化，需要删除原来的ClusterRoleBinding并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原ClusterRoleBinding
		err := c.rbacManager.DeleteClusterRoleBinding(ctx, req.ClusterID, req.OriginalName, metav1.DeleteOptions{})
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
		return c.CreateClusterRoleBinding(ctx, createReq)
	}

	// 获取现有ClusterRoleBinding
	existingClusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	// 更新ClusterRoleBinding
	existingClusterRoleBinding.Labels = req.Labels
	existingClusterRoleBinding.Annotations = req.Annotations
	existingClusterRoleBinding.RoleRef = k8sutils.ConvertRoleRefToK8s(req.RoleRef)
	existingClusterRoleBinding.Subjects = k8sutils.ConvertSubjectsToK8s(req.Subjects)

	// 使用 RBACManager 更新 ClusterRoleBinding
	err = c.rbacManager.UpdateClusterRoleBinding(ctx, req.ClusterID, existingClusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (c *clusterRoleBindingService) DeleteClusterRoleBinding(ctx context.Context, req *model.DeleteClusterRoleBindingReq) error {

	// 使用 RBACManager 删除 ClusterRoleBinding
	err := c.rbacManager.DeleteClusterRoleBinding(ctx, req.ClusterID, req.Name, metav1.DeleteOptions{})
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
	clusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
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
func (c *clusterRoleBindingService) UpdateClusterRoleBindingYaml(ctx context.Context, req *model.UpdateClusterRoleBindingYamlReq) error {
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
	existingClusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
	clusterRoleBinding.UID = existingClusterRoleBinding.UID

	// 使用 RBACManager 更新 ClusterRoleBinding
	err = c.rbacManager.UpdateClusterRoleBinding(ctx, req.ClusterID, clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}

// GetClusterRoleBindingEvents 获取ClusterRoleBinding事件
func (c *clusterRoleBindingService) GetClusterRoleBindingEvents(ctx context.Context, req *model.GetClusterRoleBindingEventsReq) ([]*model.K8sClusterRoleBindingEvent, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRoleBinding事件请求不能为空")
	}

	events, _, err := c.rbacManager.GetClusterRoleBindingEvents(ctx, req.ClusterID, req.Name, req.Limit)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingEvents: 获取ClusterRoleBinding事件失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding事件失败: %w", err)
	}

	return events, nil
}

// GetClusterRoleBindingUsage 获取ClusterRoleBinding使用情况
func (c *clusterRoleBindingService) GetClusterRoleBindingUsage(ctx context.Context, req *model.GetClusterRoleBindingUsageReq) (*model.K8sClusterRoleBindingUsage, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRoleBinding使用情况请求不能为空")
	}

	usage, err := c.rbacManager.GetClusterRoleBindingUsage(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleBindingUsage: 获取ClusterRoleBinding使用情况失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRoleBinding使用情况失败: %w", err)
	}

	return usage, nil
}

// 兼容性方法 - 保留旧接口以避免破坏性更改
// GetClusterRoleBindingYamlCompat 获取ClusterRoleBinding的YAML配置（兼容性方法）
func (c *clusterRoleBindingService) GetClusterRoleBindingYamlCompat(ctx context.Context, req *model.ClusterRoleBindingGetReq) (string, error) {
	// 获取 ClusterRoleBinding
	clusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
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

// UpdateClusterRoleBindingYamlCompat 通过YAML更新ClusterRoleBinding（兼容性方法）
func (c *clusterRoleBindingService) UpdateClusterRoleBindingYamlCompat(ctx context.Context, req *model.ClusterRoleBindingYamlReq) error {

	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err := yaml.Unmarshal([]byte(req.YamlContent), &clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称一致
	clusterRoleBinding.Name = req.Name

	// 获取现有ClusterRoleBinding以保持ResourceVersion
	existingClusterRoleBinding, err := c.rbacManager.GetClusterRoleBinding(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role binding: %w", err)
	}

	clusterRoleBinding.ResourceVersion = existingClusterRoleBinding.ResourceVersion
	clusterRoleBinding.UID = existingClusterRoleBinding.UID

	// 使用 RBACManager 更新 ClusterRoleBinding
	err = c.rbacManager.UpdateClusterRoleBinding(ctx, req.ClusterID, &clusterRoleBinding)
	if err != nil {
		return fmt.Errorf("failed to update cluster role binding: %w", err)
	}

	return nil
}
