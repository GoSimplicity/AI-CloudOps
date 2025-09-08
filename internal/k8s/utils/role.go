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

package utils

import (
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// BuildRoleListOptions 构建Role列表选项
func BuildRoleListOptions(req *model.GetRoleListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 构建选项的逻辑可以在这里添加

	return options
}

// ConvertToK8sRole 将内部模型转换为Kubernetes Role对象
func ConvertToK8sRole(req *model.CreateRoleReq) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: ConvertPolicyRulesToK8s(req.Rules),
	}
}

// PaginateK8sRoles 对Role列表进行分页（基于内部模型）
func PaginateK8sRoles(roles []*model.K8sRole, page, pageSize int) ([]*model.K8sRole, int64) {
	total := int64(len(roles))
	if total == 0 {
		return []*model.K8sRole{}, 0
	}

	if page <= 0 || pageSize <= 0 {
		return roles, total
	}

	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize)

	if start >= total {
		return []*model.K8sRole{}, total
	}

	if end > total {
		end = total
	}

	return roles[start:end], total
}

// ConvertK8sRoleToRoleInfo 将K8s Role转换为兼容的 RoleInfo
func ConvertK8sRoleToRoleInfo(role *rbacv1.Role, clusterID int) model.RoleInfo {
	if role == nil {
		return model.RoleInfo{}
	}

	ageDuration := time.Since(role.CreationTimestamp.Time)
	age := formatAge(ageDuration)

	return model.RoleInfo{
		Name:              role.Name,
		Namespace:         role.Namespace,
		ClusterID:         clusterID,
		UID:               string(role.UID),
		CreationTimestamp: role.CreationTimestamp.Time.Format(time.RFC3339),
		Labels:            role.Labels,
		Annotations:       role.Annotations,
		Rules:             ConvertK8sPolicyRulesToModel(role.Rules),
		ResourceVersion:   role.ResourceVersion,
		Age:               age,
	}
}

// ConvertRoleToModel 将 K8s Role 转换为内部模型 K8sRole
func ConvertRoleToModel(role *rbacv1.Role, clusterID int) *model.K8sRole {
	if role == nil {
		return nil
	}

	age := formatAge(time.Since(role.CreationTimestamp.Time))

	return &model.K8sRole{
		Name:              role.Name,
		Namespace:         role.Namespace,
		ClusterID:         clusterID,
		UID:               string(role.UID),
		Status:            model.RoleStatusActive,
		Labels:            role.Labels,
		Annotations:       role.Annotations,
		Rules:             ConvertK8sPolicyRulesToModel(role.Rules),
		ResourceVersion:   role.ResourceVersion,
		Age:               age,
		BindingCount:      0,
		ActiveSubjects:    0,
		IsSystemRole:      model.BoolFalse,
		CreationTimestamp: role.CreationTimestamp.Time,
		RawRole:           role,
	}
}

// RoleToYAML 将Role转换为YAML
func RoleToYAML(role *rbacv1.Role) (string, error) {
	if role == nil {
		return "", fmt.Errorf("Role不能为空")
	}

	data, err := yaml.Marshal(role)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(data), nil
}

// YAMLToRole 将YAML转换为Role
func YAMLToRole(yamlStr string) (*rbacv1.Role, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("YAML字符串不能为空")
	}

	var role rbacv1.Role
	err := yaml.Unmarshal([]byte(yamlStr), &role)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &role, nil
}

// BuildK8sRole 构建K8s Role对象
func BuildK8sRole(name, namespace string, labels, annotations model.KeyValueList, rules []model.PolicyRule) *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      ConvertKeyValueListToLabels(labels),
			Annotations: ConvertKeyValueListToLabels(annotations),
		},
		Rules: ConvertPolicyRulesToK8s(rules),
	}
}

// formatAge 将时长格式化为简短字符串（如 2d、5h、30m）
func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%dm", minutes)
}
