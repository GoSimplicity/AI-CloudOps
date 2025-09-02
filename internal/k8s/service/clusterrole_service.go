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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type ClusterRoleService interface {
	// 基础 CRUD 操作
	GetClusterRoleList(ctx context.Context, req *model.GetClusterRoleListReq) (model.ListResp[*model.K8sClusterRole], error)
	GetClusterRoleDetails(ctx context.Context, req *model.GetClusterRoleDetailsReq) (*model.K8sClusterRole, error)
	CreateClusterRole(ctx context.Context, req *model.CreateClusterRoleReq) error
	UpdateClusterRole(ctx context.Context, req *model.UpdateClusterRoleReq) error
	DeleteClusterRole(ctx context.Context, req *model.DeleteClusterRoleReq) error

	// YAML 操作
	GetClusterRoleYaml(ctx context.Context, req *model.GetClusterRoleYamlReq) (*model.K8sYaml, error)
	UpdateClusterRoleYaml(ctx context.Context, req *model.UpdateClusterRoleYamlReq) error

	// 扩展功能
	GetClusterRoleEvents(ctx context.Context, req *model.GetClusterRoleEventsReq) (model.ListResp[*model.K8sClusterRoleEvent], error)
	GetClusterRoleUsage(ctx context.Context, req *model.GetClusterRoleUsageReq) (*model.K8sClusterRoleUsage, error)
	GetClusterRoleMetrics(ctx context.Context, req *model.GetClusterRoleMetricsReq) (*model.K8sClusterRoleMetrics, error)

	// 兼容性方法（保持现有API接口兼容）
	GetClusterRoleListCompat(ctx context.Context, req *model.ClusterRoleListReq) (*model.ListResp[model.ClusterRoleInfo], error)
	GetClusterRoleDetailsCompat(ctx context.Context, req *model.ClusterRoleGetReq) (*model.ClusterRoleInfo, error)
	GetClusterRoleYamlCompat(ctx context.Context, req *model.ClusterRoleGetReq) (string, error)
	UpdateClusterRoleYamlCompat(ctx context.Context, req *model.ClusterRoleYamlReq) error
}

type clusterRoleService struct {
	dao         dao.ClusterDAO
	rbacManager manager.RBACManager
	logger      *zap.Logger
}

func NewClusterRoleService(dao dao.ClusterDAO, rbacManager manager.RBACManager, logger *zap.Logger) ClusterRoleService {
	return &clusterRoleService{
		dao:         dao,
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// GetClusterRoleList 获取ClusterRole列表
func (c *clusterRoleService) GetClusterRoleList(ctx context.Context, req *model.GetClusterRoleListReq) (model.ListResp[*model.K8sClusterRole], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("获取ClusterRole列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildClusterRoleListOptions(req)

	k8sClusterRoles, err := c.rbacManager.GetClusterRoleList(ctx, req.ClusterID, listOptions)
	if err != nil {
		c.logger.Error("GetClusterRoleList: 获取ClusterRole列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sClusterRole]{}, fmt.Errorf("获取ClusterRole列表失败: %w", err)
	}

	// 根据状态过滤
	var filteredClusterRoles []*model.K8sClusterRole
	if req.Status != "" {
		// 根据状态过滤
		for _, k8sClusterRole := range k8sClusterRoles {
			var statusStr string
			switch k8sClusterRole.Status {
			case model.ClusterRoleStatusActive:
				statusStr = "active"
			case model.ClusterRoleStatusInactive:
				statusStr = "inactive"
			case model.ClusterRoleStatusUnused:
				statusStr = "unused"
			case model.ClusterRoleStatusError:
				statusStr = "error"
			default:
				statusStr = "unknown"
			}
			if strings.EqualFold(statusStr, req.Status) {
				filteredClusterRoles = append(filteredClusterRoles, k8sClusterRole)
			}
		}
	} else {
		filteredClusterRoles = k8sClusterRoles
	}

	// 分页处理
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10 // 默认每页显示10条
	}

	pagedItems, total := utils.PaginateK8sClusterRoles(filteredClusterRoles, page, size)

	return model.ListResp[*model.K8sClusterRole]{
		Total: total,
		Items: pagedItems,
	}, nil
}

// GetClusterRoleListCompat 获取ClusterRole列表（兼容性方法）
func (c *clusterRoleService) GetClusterRoleListCompat(ctx context.Context, req *model.ClusterRoleListReq) (*model.ListResp[model.ClusterRoleInfo], error) {
	// 使用 RBACManager 获取 ClusterRole 列表
	clusterRoles, err := c.rbacManager.GetClusterRoleList(ctx, req.ClusterID, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster roles: %w", err)
	}

	// 转换为响应格式并过滤
	var clusterRoleInfos []model.ClusterRoleInfo
	for _, clusterRole := range clusterRoles {
		if clusterRole.RawClusterRole == nil {
			continue
		}
		clusterRoleInfo := utils.ConvertK8sClusterRoleToClusterRoleInfo(clusterRole.RawClusterRole, req.ClusterID)

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
func (c *clusterRoleService) GetClusterRoleDetails(ctx context.Context, req *model.GetClusterRoleDetailsReq) (*model.K8sClusterRole, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	clusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleDetails: 获取ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole失败: %w", err)
	}

	// 构建详细信息
	k8sClusterRole, err := utils.BuildK8sClusterRole(ctx, req.ClusterID, *clusterRole)
	if err != nil {
		c.logger.Error("GetClusterRoleDetails: 构建ClusterRole详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建ClusterRole详细信息失败: %w", err)
	}

	return k8sClusterRole, nil
}

// GetClusterRoleYaml 获取ClusterRole YAML
func (c *clusterRoleService) GetClusterRoleYaml(ctx context.Context, req *model.GetClusterRoleYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	clusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleYaml: 获取ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.ClusterRoleToYAML(clusterRole)
	if err != nil {
		c.logger.Error("GetClusterRoleYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("clusterRoleName", clusterRole.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdateClusterRoleYaml 更新ClusterRole YAML
func (c *clusterRoleService) UpdateClusterRoleYaml(ctx context.Context, req *model.UpdateClusterRoleYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新ClusterRole YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("ClusterRole名称不能为空")
	}

	if req.YamlContent == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	existingClusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("UpdateClusterRoleYaml: 获取现有ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有ClusterRole失败: %w", err)
	}

	updatedClusterRole, err := utils.YAMLToClusterRole(req.YamlContent)
	if err != nil {
		c.logger.Error("UpdateClusterRoleYaml: 解析YAML失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 保持必要的元数据
	updatedClusterRole.ResourceVersion = existingClusterRole.ResourceVersion
	updatedClusterRole.UID = existingClusterRole.UID

	err = c.rbacManager.UpdateClusterRole(ctx, req.ClusterID, updatedClusterRole)
	if err != nil {
		c.logger.Error("UpdateClusterRoleYaml: 更新ClusterRole失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return fmt.Errorf("更新ClusterRole失败: %w", err)
	}

	return nil
}

// GetClusterRoleEvents 获取ClusterRole事件
func (c *clusterRoleService) GetClusterRoleEvents(ctx context.Context, req *model.GetClusterRoleEventsReq) (model.ListResp[*model.K8sClusterRoleEvent], error) {
	if req == nil {
		return model.ListResp[*model.K8sClusterRoleEvent]{}, fmt.Errorf("获取ClusterRole事件请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sClusterRoleEvent]{}, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return model.ListResp[*model.K8sClusterRoleEvent]{}, fmt.Errorf("ClusterRole名称不能为空")
	}

	// 设置默认限制数量
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // 默认获取100个事件
	}

	events, total, err := c.rbacManager.GetClusterRoleEvents(ctx, req.ClusterID, req.Name, limit)
	if err != nil {
		c.logger.Error("GetClusterRoleEvents: 获取ClusterRole事件失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return model.ListResp[*model.K8sClusterRoleEvent]{}, fmt.Errorf("获取ClusterRole事件失败: %w", err)
	}

	return model.ListResp[*model.K8sClusterRoleEvent]{
		Total: total,
		Items: events,
	}, nil
}

// GetClusterRoleUsage 获取ClusterRole使用情况
func (c *clusterRoleService) GetClusterRoleUsage(ctx context.Context, req *model.GetClusterRoleUsageReq) (*model.K8sClusterRoleUsage, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole使用情况请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	usage, err := c.rbacManager.GetClusterRoleUsage(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleUsage: 获取ClusterRole使用情况失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole使用情况失败: %w", err)
	}

	return usage, nil
}

// GetClusterRoleMetrics 获取ClusterRole指标
func (c *clusterRoleService) GetClusterRoleMetrics(ctx context.Context, req *model.GetClusterRoleMetricsReq) (*model.K8sClusterRoleMetrics, error) {
	if req == nil {
		return nil, fmt.Errorf("获取ClusterRole指标请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("ClusterRole名称不能为空")
	}

	metrics, err := c.rbacManager.GetClusterRoleMetrics(ctx, req.ClusterID, req.Name)
	if err != nil {
		c.logger.Error("GetClusterRoleMetrics: 获取ClusterRole指标失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取ClusterRole指标失败: %w", err)
	}

	c.logger.Debug("GetClusterRoleMetrics: 成功获取ClusterRole指标",
		zap.Int("clusterID", req.ClusterID),
		zap.String("name", req.Name))

	return metrics, nil
}

// GetClusterRoleDetailsCompat 获取ClusterRole详情（兼容性方法）
func (c *clusterRoleService) GetClusterRoleDetailsCompat(ctx context.Context, req *model.ClusterRoleGetReq) (*model.ClusterRoleInfo, error) {
	// 使用 RBACManager 获取 ClusterRole 详情
	clusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster role: %w", err)
	}

	clusterRoleInfo := utils.ConvertK8sClusterRoleToClusterRoleInfo(clusterRole, req.ClusterID)
	return &clusterRoleInfo, nil
}

// CreateClusterRole 创建ClusterRole
func (c *clusterRoleService) CreateClusterRole(ctx context.Context, req *model.CreateClusterRoleReq) error {
	// 构建 ClusterRole 对象
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: utils.ConvertPolicyRulesToK8s(req.Rules),
	}

	// 使用 RBACManager 创建 ClusterRole
	err := c.rbacManager.CreateClusterRole(ctx, req.ClusterID, clusterRole)
	if err != nil {
		return fmt.Errorf("failed to create cluster role: %w", err)
	}

	return nil
}

// UpdateClusterRole 更新ClusterRole
func (c *clusterRoleService) UpdateClusterRole(ctx context.Context, req *model.UpdateClusterRoleReq) error {
	// 如果名称发生变化，需要删除原来的ClusterRole并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原ClusterRole
		err := c.rbacManager.DeleteClusterRole(ctx, req.ClusterID, req.OriginalName, metav1.DeleteOptions{})
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
		return c.CreateClusterRole(ctx, createReq)
	}

	// 获取现有ClusterRole
	existingClusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role: %w", err)
	}

	// 更新ClusterRole
	existingClusterRole.Labels = req.Labels
	existingClusterRole.Annotations = req.Annotations
	existingClusterRole.Rules = utils.ConvertPolicyRulesToK8s(req.Rules)

	// 使用 RBACManager 更新 ClusterRole
	err = c.rbacManager.UpdateClusterRole(ctx, req.ClusterID, existingClusterRole)
	if err != nil {
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	return nil
}

// DeleteClusterRole 删除ClusterRole
func (c *clusterRoleService) DeleteClusterRole(ctx context.Context, req *model.DeleteClusterRoleReq) error {
	// 使用 RBACManager 删除 ClusterRole
	err := c.rbacManager.DeleteClusterRole(ctx, req.ClusterID, req.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete cluster role: %w", err)
	}

	return nil
}

// GetClusterRoleYamlCompat 获取ClusterRole的YAML配置（兼容性方法）
func (c *clusterRoleService) GetClusterRoleYamlCompat(ctx context.Context, req *model.ClusterRoleGetReq) (string, error) {
	// 获取 ClusterRole
	clusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
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

// UpdateClusterRoleYamlCompat 通过YAML更新ClusterRole（兼容性方法）
func (c *clusterRoleService) UpdateClusterRoleYamlCompat(ctx context.Context, req *model.ClusterRoleYamlReq) error {
	var clusterRole rbacv1.ClusterRole
	err := yaml.Unmarshal([]byte(req.YamlContent), &clusterRole)
	if err != nil {
		return fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// 确保名称一致
	clusterRole.Name = req.Name

	// 获取现有ClusterRole以保持ResourceVersion
	existingClusterRole, err := c.rbacManager.GetClusterRole(ctx, req.ClusterID, req.Name)
	if err != nil {
		return fmt.Errorf("failed to get existing cluster role: %w", err)
	}

	clusterRole.ResourceVersion = existingClusterRole.ResourceVersion
	clusterRole.UID = existingClusterRole.UID

	// 使用 RBACManager 更新 ClusterRole
	err = c.rbacManager.UpdateClusterRole(ctx, req.ClusterID, &clusterRole)
	if err != nil {
		return fmt.Errorf("failed to update cluster role: %w", err)
	}

	return nil
}
