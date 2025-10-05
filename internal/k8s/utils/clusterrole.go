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

func ConvertClusterRoleToModel(clusterRole *rbacv1.ClusterRole, clusterID int) *model.K8sClusterRole {
	if clusterRole == nil {
		return nil
	}

	var rules []model.PolicyRule
	for _, rule := range clusterRole.Rules {
		rules = append(rules, model.PolicyRule{
			Verbs:           rule.Verbs,
			APIGroups:       rule.APIGroups,
			Resources:       rule.Resources,
			ResourceNames:   rule.ResourceNames,
			NonResourceURLs: rule.NonResourceURLs,
		})
	}

	labels := make(map[string]string)
	if clusterRole.Labels != nil {
		labels = clusterRole.Labels
	}

	annotations := make(map[string]string)
	if clusterRole.Annotations != nil {
		annotations = clusterRole.Annotations
	}

	return &model.K8sClusterRole{
		Name:        clusterRole.Name,
		UID:         string(clusterRole.UID),
		ClusterID:   clusterID,
		Labels:      labels,
		Annotations: annotations,
		Rules:       rules,
	}
}

func ConvertClusterRolesToModel(clusterRoles []rbacv1.ClusterRole, clusterID int) []*model.K8sClusterRole {
	if len(clusterRoles) == 0 {
		return nil
	}

	results := make([]*model.K8sClusterRole, 0, len(clusterRoles))
	for _, clusterRole := range clusterRoles {
		if cr := ConvertClusterRoleToModel(&clusterRole, clusterID); cr != nil {
			results = append(results, cr)
		}
	}
	return results
}

func BuildClusterRoleListQueryOptions(req *model.GetClusterRoleListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	// 暂时不设置标签选择器，需要根据实际的 GetClusterRoleListReq 结构调整

	return options
}

func ValidateClusterRole(clusterRole *rbacv1.ClusterRole) error {
	if clusterRole == nil {
		return fmt.Errorf("ClusterRole 不能为空")
	}

	if clusterRole.Name == "" {
		return fmt.Errorf("ClusterRole 名称不能为空")
	}

	for i, rule := range clusterRole.Rules {
		if err := validatePolicyRule(rule, i); err != nil {
			return fmt.Errorf("ClusterRole 规则验证失败: %w", err)
		}
	}

	return nil
}

// validatePolicyRule 验证策略规则
func validatePolicyRule(rule rbacv1.PolicyRule, index int) error {
	if len(rule.Verbs) == 0 {
		return fmt.Errorf("规则 %d: 动作(verbs)不能为空", index)
	}

	// 至少需要指定 resources 或 nonResourceURLs 中的一个
	if len(rule.Resources) == 0 && len(rule.NonResourceURLs) == 0 {
		return fmt.Errorf("规则 %d: 必须指定 resources 或 nonResourceURLs", index)
	}

	// 不能同时指定 resources 和 nonResourceURLs
	if len(rule.Resources) > 0 && len(rule.NonResourceURLs) > 0 {
		return fmt.Errorf("规则 %d: 不能同时指定 resources 和 nonResourceURLs", index)
	}

	return nil
}

func ConvertClusterRoleToYAML(clusterRole *rbacv1.ClusterRole) (string, error) {
	if clusterRole == nil {
		return "", fmt.Errorf("ClusterRole 不能为空")
	}

	// 清理不需要的字段
	cleanClusterRole := clusterRole.DeepCopy()
	cleanClusterRole.ManagedFields = nil
	cleanClusterRole.ResourceVersion = ""
	cleanClusterRole.UID = ""
	cleanClusterRole.CreationTimestamp = metav1.Time{}
	cleanClusterRole.Generation = 0

	yamlBytes, err := yaml.Marshal(cleanClusterRole)
	if err != nil {
		return "", fmt.Errorf("转换为 YAML 失败: %w", err)
	}

	return string(yamlBytes), nil
}

// ParseYAMLToClusterRole 将 YAML 转换为 ClusterRole
func ParseYAMLToClusterRole(yamlContent string) (*rbacv1.ClusterRole, error) {
	if yamlContent == "" {
		return nil, fmt.Errorf("YAML 内容不能为空")
	}

	var clusterRole rbacv1.ClusterRole
	err := yaml.Unmarshal([]byte(yamlContent), &clusterRole)
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return &clusterRole, nil
}

// FilterClusterRolesByName 根据名称过滤 ClusterRole 列表
func FilterClusterRolesByName(clusterRoles []rbacv1.ClusterRole, nameFilter string) []rbacv1.ClusterRole {
	if nameFilter == "" {
		return clusterRoles
	}

	var filtered []rbacv1.ClusterRole
	for _, cr := range clusterRoles {
		if contains(cr.Name, nameFilter) {
			filtered = append(filtered, cr)
		}
	}

	return filtered
}

func GetClusterRoleAge(clusterRole rbacv1.ClusterRole) string {
	age := time.Since(clusterRole.CreationTimestamp.Time)
	days := int(age.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(age.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(age.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

// IsSystemClusterRole 判断是否为系统 ClusterRole
func IsSystemClusterRole(clusterRole rbacv1.ClusterRole) bool {
	systemPrefixes := []string{
		"system:",
		"cluster-admin",
		"admin",
		"edit",
		"view",
	}

	for _, prefix := range systemPrefixes {
		if len(clusterRole.Name) >= len(prefix) && clusterRole.Name[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

func GetClusterRolePermissions(clusterRole rbacv1.ClusterRole) map[string][]string {
	permissions := make(map[string][]string)

	for _, rule := range clusterRole.Rules {
		for _, resource := range rule.Resources {
			if permissions[resource] == nil {
				permissions[resource] = make([]string, 0)
			}
			permissions[resource] = append(permissions[resource], rule.Verbs...)
		}

		// 处理非资源URL
		for _, url := range rule.NonResourceURLs {
			if permissions[url] == nil {
				permissions[url] = make([]string, 0)
			}
			permissions[url] = append(permissions[url], rule.Verbs...)
		}
	}

	// 去重
	for resource, verbs := range permissions {
		permissions[resource] = removeDuplicates(verbs)
	}

	return permissions
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

// findSubstring 查找子字符串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// removeDuplicates 去除字符串数组中的重复项
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := []string{}

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

func BuildClusterRoleListOptions(req *model.GetClusterRoleListReq) metav1.ListOptions {
	options := metav1.ListOptions{}

	return options
}

func ConvertToK8sClusterRole(req *model.CreateClusterRoleReq) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Rules: ConvertPolicyRulesToK8s(req.Rules),
	}
}

// PaginateK8sClusterRoles 对ClusterRole列表进行分页
func PaginateK8sClusterRoles(clusterRoles []*model.K8sClusterRole, page, pageSize int) ([]*model.K8sClusterRole, int64) {
	total := int64(len(clusterRoles))
	if total == 0 {
		return []*model.K8sClusterRole{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 || pageSize <= 0 {
		return clusterRoles, total
	}

	start := int64((page - 1) * pageSize)
	end := start + int64(pageSize)

	if start >= total {
		return []*model.K8sClusterRole{}, total
	}

	if end > total {
		end = total
	}

	return clusterRoles[start:end], total
}

func ConvertK8sClusterRoleToClusterRoleInfo(clusterRole *rbacv1.ClusterRole, clusterID int) model.K8sClusterRole {
	if clusterRole == nil {
		return model.K8sClusterRole{}
	}

	age := GetClusterRoleAge(*clusterRole)

	return model.K8sClusterRole{
		Name:            clusterRole.Name,
		ClusterID:       clusterID,
		UID:             string(clusterRole.UID),
		CreatedAt:       clusterRole.CreationTimestamp.Time.Format(time.RFC3339),
		Labels:          clusterRole.Labels,
		Annotations:     clusterRole.Annotations,
		Rules:           ConvertK8sPolicyRulesToModel(clusterRole.Rules),
		ResourceVersion: clusterRole.ResourceVersion,
		Age:             age,
	}
}

func BuildK8sClusterRole(name string, labels, annotations model.KeyValueList, rules []model.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      ConvertKeyValueListToLabels(labels),
			Annotations: ConvertKeyValueListToLabels(annotations),
		},
		Rules: ConvertPolicyRulesToK8s(rules),
	}
}

func ConvertPolicyRulesToK8s(rules []model.PolicyRule) []rbacv1.PolicyRule {
	if len(rules) == 0 {
		return nil
	}

	k8sRules := make([]rbacv1.PolicyRule, 0, len(rules))
	for _, rule := range rules {
		k8sRules = append(k8sRules, rbacv1.PolicyRule{
			Verbs:         rule.Verbs,
			APIGroups:     rule.APIGroups,
			Resources:     rule.Resources,
			ResourceNames: rule.ResourceNames,
		})
	}

	return k8sRules
}

func ConvertK8sPolicyRulesToModel(rules []rbacv1.PolicyRule) []model.PolicyRule {
	if len(rules) == 0 {
		return nil
	}

	modelRules := make([]model.PolicyRule, 0, len(rules))
	for _, rule := range rules {
		modelRules = append(modelRules, model.PolicyRule{
			Verbs:         rule.Verbs,
			APIGroups:     rule.APIGroups,
			Resources:     rule.Resources,
			ResourceNames: rule.ResourceNames,
		})
	}

	return modelRules
}

// ClusterRoleToYAML 将ClusterRole转换为YAML
func ClusterRoleToYAML(clusterRole *rbacv1.ClusterRole) (string, error) {
	if clusterRole == nil {
		return "", fmt.Errorf("ClusterRole不能为空")
	}

	data, err := yaml.Marshal(clusterRole)
	if err != nil {
		return "", fmt.Errorf("转换为YAML失败: %w", err)
	}

	return string(data), nil
}

// YAMLToClusterRole 将YAML转换为ClusterRole
func YAMLToClusterRole(yamlStr string) (*rbacv1.ClusterRole, error) {
	if yamlStr == "" {
		return nil, fmt.Errorf("YAML字符串不能为空")
	}

	var clusterRole rbacv1.ClusterRole
	err := yaml.Unmarshal([]byte(yamlStr), &clusterRole)
	if err != nil {
		return nil, fmt.Errorf("解析YAML失败: %w", err)
	}

	return &clusterRole, nil
}
