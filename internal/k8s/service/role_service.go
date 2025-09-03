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

	"sigs.k8s.io/yaml"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	k8sutils "github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type RoleService interface {
	// 基础 CRUD 操作
	GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error)
	GetRoleDetails(ctx context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error)
	CreateRole(ctx context.Context, req *model.CreateRoleReq) error
	UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error
	DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error

	// YAML 操作
	GetRoleYaml(ctx context.Context, req *model.GetRoleYamlReq) (string, error)
	UpdateRoleYaml(ctx context.Context, req *model.UpdateRoleYamlReq) error

	// 扩展功能
	GetRoleEvents(ctx context.Context, req *model.GetRoleEventsReq) ([]*model.K8sRoleEvent, error)
	GetRoleUsage(ctx context.Context, req *model.GetRoleUsageReq) (*model.K8sRoleUsage, error)

	// 兼容性方法（保持现有API接口兼容）
	GetRoleListCompat(ctx context.Context, req *model.RoleListReq) (*model.ListResp[model.RoleInfo], error)
	GetRoleDetailsCompat(ctx context.Context, req *model.RoleGetReq) (*model.RoleInfo, error)
	GetRoleYamlCompat(ctx context.Context, req *model.RoleGetReq) (string, error)
	UpdateRoleYamlCompat(ctx context.Context, req *model.RoleYamlReq) error
}

type roleService struct {
	dao         dao.ClusterDAO
	rbacManager manager.RBACManager
	logger      *zap.Logger
}

func NewRoleService(dao dao.ClusterDAO, rbacManager manager.RBACManager, logger *zap.Logger) RoleService {
	return &roleService{
		dao:         dao,
		rbacManager: rbacManager,
		logger:      logger,
	}
}

// GetRoleList 获取Role列表（新接口）
func (r *roleService) GetRoleList(ctx context.Context, req *model.GetRoleListReq) (model.ListResp[*model.K8sRole], error) {
	if req == nil {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := k8sutils.BuildRoleListOptions(req)

	roleList, err := r.rbacManager.GetRoleListRaw(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		r.logger.Error("GetRoleList: 获取Role列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sRole]{}, fmt.Errorf("获取Role列表失败: %w", err)
	}

	// 转换为模型格式
	var k8sRoles []*model.K8sRole
	for i := range roleList.Items {
		k8sRole := k8sutils.ConvertToK8sRole(&roleList.Items[i])
		if k8sRole != nil {
			k8sRole.ClusterID = req.ClusterID

			// 关键字过滤
			if req.Keyword == "" || strings.Contains(k8sRole.Name, req.Keyword) {
				k8sRoles = append(k8sRoles, k8sRole)
			}
		}
	}

	// 分页处理
	pagedRoles, total := k8sutils.PaginateK8sRoles(k8sRoles, req.Page, req.PageSize)

	r.logger.Debug("GetRoleList: 获取Role列表成功",
		zap.Int("clusterID", req.ClusterID),
		zap.Int64("total", total),
		zap.Int("returned", len(pagedRoles)))

	return model.ListResp[*model.K8sRole]{
		Items: pagedRoles,
		Total: total,
	}, nil
}

// GetRoleDetails 获取Role详情（新接口）
func (r *roleService) GetRoleDetails(ctx context.Context, req *model.GetRoleDetailsReq) (*model.K8sRole, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Role名称不能为空")
	}

	role, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		r.logger.Error("GetRoleDetails: 获取Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role失败: %w", err)
	}

	k8sRole, err := k8sutils.BuildK8sRole(ctx, req.ClusterID, req.Namespace, *role)
	if err != nil {
		r.logger.Error("GetRoleDetails: 构建Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建Role失败: %w", err)
	}

	r.logger.Debug("GetRoleDetails: 获取Role详情成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return k8sRole, nil
}

// CreateRole 创建Role
func (r *roleService) CreateRole(ctx context.Context, req *model.CreateRoleReq) error {
	if req == nil {
		return fmt.Errorf("创建Role请求不能为空")
	}

	// 构建 Role 对象
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: k8sutils.ConvertPolicyRulesToK8s(req.Rules),
	}

	// 使用 RBACManager 创建 Role
	err := r.rbacManager.CreateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		r.logger.Error("CreateRole: 创建Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Role失败: %w", err)
	}

	r.logger.Info("CreateRole: 成功创建Role",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// UpdateRole 更新Role
func (r *roleService) UpdateRole(ctx context.Context, req *model.UpdateRoleReq) error {
	if req == nil {
		return fmt.Errorf("更新Role请求不能为空")
	}

	// 如果名称发生变化，需要删除原来的Role并创建新的
	if req.OriginalName != "" && req.OriginalName != req.Name {
		// 删除原Role
		err := r.rbacManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.OriginalName, metav1.DeleteOptions{})
		if err != nil {
			return fmt.Errorf("删除原Role失败: %w", err)
		}

		// 创建新Role
		createReq := &model.CreateRoleReq{
			ClusterID:   req.ClusterID,
			Namespace:   req.Namespace,
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
			Rules:       req.Rules,
		}
		return r.CreateRole(ctx, createReq)
	}

	// 获取现有Role
	existingRole, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("获取现有Role失败: %w", err)
	}

	// 更新Role
	existingRole.Labels = req.Labels
	existingRole.Annotations = req.Annotations
	existingRole.Rules = k8sutils.ConvertPolicyRulesToK8s(req.Rules)

	// 使用 RBACManager 更新 Role
	err = r.rbacManager.UpdateRole(ctx, req.ClusterID, req.Namespace, existingRole)
	if err != nil {
		r.logger.Error("UpdateRole: 更新Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Role失败: %w", err)
	}

	r.logger.Info("UpdateRole: 成功更新Role",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// DeleteRole 删除Role
func (r *roleService) DeleteRole(ctx context.Context, req *model.DeleteRoleReq) error {
	if req == nil {
		return fmt.Errorf("删除Role请求不能为空")
	}

	// 使用 RBACManager 删除 Role
	err := r.rbacManager.DeleteRole(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		r.logger.Error("DeleteRole: 删除Role失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Role失败: %w", err)
	}

	r.logger.Info("DeleteRole: 成功删除Role",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// GetRoleYaml 获取Role YAML
func (r *roleService) GetRoleYaml(ctx context.Context, req *model.GetRoleYamlReq) (string, error) {
	if req == nil {
		return "", fmt.Errorf("获取Role YAML请求不能为空")
	}

	// 获取 Role
	role, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return "", fmt.Errorf("获取Role失败: %w", err)
	}

	yamlContent, err := k8sutils.RoleToYAML(role)
	if err != nil {
		return "", fmt.Errorf("转换Role为YAML失败: %w", err)
	}

	return yamlContent, nil
}

// UpdateRoleYaml 更新Role YAML
func (r *roleService) UpdateRoleYaml(ctx context.Context, req *model.UpdateRoleYamlReq) error {
	if req == nil {
		return fmt.Errorf("更新Role YAML请求不能为空")
	}

	role, err := k8sutils.YAMLToRole(req.YamlContent)
	if err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 确保命名空间和名称一致
	role.Namespace = req.Namespace
	role.Name = req.Name

	// 获取现有Role以保持ResourceVersion
	existingRole, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("获取现有Role失败: %w", err)
	}

	role.ResourceVersion = existingRole.ResourceVersion
	role.UID = existingRole.UID

	// 使用 RBACManager 更新 Role
	err = r.rbacManager.UpdateRole(ctx, req.ClusterID, req.Namespace, role)
	if err != nil {
		return fmt.Errorf("更新Role失败: %w", err)
	}

	return nil
}

// GetRoleEvents 获取Role事件
func (r *roleService) GetRoleEvents(ctx context.Context, req *model.GetRoleEventsReq) ([]*model.K8sRoleEvent, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role事件请求不能为空")
	}

	events, _, err := r.rbacManager.GetRoleEvents(ctx, req.ClusterID, req.Namespace, req.Name, req.Limit)
	if err != nil {
		r.logger.Error("GetRoleEvents: 获取Role事件失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role事件失败: %w", err)
	}

	return events, nil
}

// GetRoleUsage 获取Role使用情况
func (r *roleService) GetRoleUsage(ctx context.Context, req *model.GetRoleUsageReq) (*model.K8sRoleUsage, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Role使用情况请求不能为空")
	}

	usage, err := r.rbacManager.GetRoleUsage(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		r.logger.Error("GetRoleUsage: 获取Role使用情况失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Role使用情况失败: %w", err)
	}

	return usage, nil
}

// 兼容性方法实现
func (r *roleService) GetRoleListCompat(ctx context.Context, req *model.RoleListReq) (*model.ListResp[model.RoleInfo], error) {
	// 兼容性实现
	roleList, err := r.rbacManager.GetRoleListRaw(ctx, req.ClusterID, req.Namespace, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Role列表失败: %w", err)
	}

	// 转换为响应格式并过滤
	var roleInfos []model.RoleInfo
	for _, role := range roleList.Items {
		roleInfo := k8sutils.ConvertK8sRoleToRoleInfo(&role, req.ClusterID)

		// 关键字过滤
		if req.Keyword != "" && !strings.Contains(roleInfo.Name, req.Keyword) {
			continue
		}

		roleInfos = append(roleInfos, roleInfo)
	}

	// 分页处理
	total := len(roleInfos)
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		roleInfos = []model.RoleInfo{}
	} else if end > total {
		roleInfos = roleInfos[start:]
	} else {
		roleInfos = roleInfos[start:end]
	}

	return &model.ListResp[model.RoleInfo]{
		Items: roleInfos,
		Total: int64(total),
	}, nil
}

func (r *roleService) GetRoleDetailsCompat(ctx context.Context, req *model.RoleGetReq) (*model.RoleInfo, error) {
	// 使用 RBACManager 获取 Role 详情
	role, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return nil, fmt.Errorf("获取Role失败: %w", err)
	}

	roleInfo := k8sutils.ConvertK8sRoleToRoleInfo(role, req.ClusterID)
	return &roleInfo, nil
}

func (r *roleService) GetRoleYamlCompat(ctx context.Context, req *model.RoleGetReq) (string, error) {
	// 获取 Role
	role, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return "", fmt.Errorf("获取Role失败: %w", err)
	}

	// 清理不需要的字段
	role.ManagedFields = nil
	role.ResourceVersion = ""
	role.UID = ""
	role.SelfLink = ""
	role.CreationTimestamp = metav1.Time{}
	role.Generation = 0

	yamlData, err := yaml.Marshal(role)
	if err != nil {
		return "", fmt.Errorf("转换Role为YAML失败: %w", err)
	}

	return string(yamlData), nil
}

func (r *roleService) UpdateRoleYamlCompat(ctx context.Context, req *model.RoleYamlReq) error {
	var role rbacv1.Role
	err := yaml.Unmarshal([]byte(req.YamlContent), &role)
	if err != nil {
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 确保命名空间和名称一致
	role.Namespace = req.Namespace
	role.Name = req.Name

	// 获取现有Role以保持ResourceVersion
	existingRole, err := r.rbacManager.GetRole(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		return fmt.Errorf("获取现有Role失败: %w", err)
	}

	role.ResourceVersion = existingRole.ResourceVersion
	role.UID = existingRole.UID

	// 使用 RBACManager 更新 Role
	err = r.rbacManager.UpdateRole(ctx, req.ClusterID, req.Namespace, &role)
	if err != nil {
		return fmt.Errorf("更新Role失败: %w", err)
	}

	return nil
}
