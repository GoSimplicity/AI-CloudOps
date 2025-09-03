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

package model

import (
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
)

// ClusterRoleStatus ClusterRole状态枚举
type ClusterRoleStatus int8

const (
	ClusterRoleStatusActive   ClusterRoleStatus = iota + 1 // 活跃
	ClusterRoleStatusInactive                              // 非活跃
	ClusterRoleStatusUnused                                // 未使用
	ClusterRoleStatusError                                 // 异常
)

// K8sClusterRole Kubernetes ClusterRole模型
type K8sClusterRole struct {
	Name                string                      `json:"name"`                  // ClusterRole名称
	ClusterID           int                         `json:"cluster_id"`            // 所属集群ID
	UID                 string                      `json:"uid"`                   // ClusterRole UID
	Status              ClusterRoleStatus           `json:"status"`                // ClusterRole状态
	Labels              map[string]string           `json:"labels"`                // ClusterRole标签
	Annotations         map[string]string           `json:"annotations"`           // ClusterRole注解
	Rules               []PolicyRule                `json:"rules"`                 // 权限规则列表
	ResourceVersion     string                      `json:"resource_version"`      // 资源版本
	Age                 string                      `json:"age"`                   // 存在时间
	ClusterBindingCount int                         `json:"cluster_binding_count"` // 关联的ClusterRoleBinding数量
	RoleBindingCount    int                         `json:"role_binding_count"`    // 关联的RoleBinding数量（ClusterRole可以被RoleBinding引用）
	ActiveSubjects      int                         `json:"active_subjects"`       // 活跃主体数量
	IsSystemRole        BoolValue                   `json:"is_system_role"`        // 是否为系统角色
	AggregationRule     *ClusterRoleAggregationRule `json:"aggregation_rule"`      // 聚合规则
	CreationTimestamp   time.Time                   `json:"creation_timestamp"`    // 创建时间
	RawClusterRole      *rbacv1.ClusterRole         `json:"-"`                     // 原始ClusterRole对象，不序列化
}

// ClusterRoleAggregationRule ClusterRole聚合规则
type ClusterRoleAggregationRule struct {
	ClusterRoleSelectors []ClusterRoleLabelSelector `json:"cluster_role_selectors"` // ClusterRole选择器列表
}

// ClusterRoleLabelSelector ClusterRole标签选择器
type ClusterRoleLabelSelector struct {
	MatchLabels      map[string]string             `json:"match_labels,omitempty"`      // 匹配标签
	MatchExpressions []ClusterRoleLabelRequirement `json:"match_expressions,omitempty"` // 匹配表达式
}

// ClusterRoleLabelRequirement ClusterRole标签要求
type ClusterRoleLabelRequirement struct {
	Key      string   `json:"key"`      // 标签键
	Operator string   `json:"operator"` // 操作符 (In, NotIn, Exists, DoesNotExist)
	Values   []string `json:"values"`   // 标签值列表
}

// K8sClusterRoleEvent ClusterRole相关事件
type K8sClusterRoleEvent struct {
	Type      string    `json:"type"`       // 事件类型 (Normal, Warning)
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Source    string    `json:"source"`     // 事件源
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Count     int32     `json:"count"`      // 发生次数
}

// K8sClusterRoleUsage ClusterRole使用情况
type K8sClusterRoleUsage struct {
	ClusterRoleName string                         `json:"cluster_role_name"` // ClusterRole名称
	ClusterBindings []ClusterRoleBindingSimpleInfo `json:"cluster_bindings"`  // 关联的ClusterRoleBinding
	RoleBindings    []RoleBindingSimpleInfo        `json:"role_bindings"`     // 关联的RoleBinding
	Subjects        []Subject                      `json:"subjects"`          // 所有主体
	Permissions     []PolicyRule                   `json:"permissions"`       // 权限列表
	IsUsed          BoolValue                      `json:"is_used"`           // 是否被使用
	RiskLevel       string                         `json:"risk_level"`        // 风险等级
	LastAccessed    *time.Time                     `json:"last_accessed"`     // 最后访问时间
	AggregatedRoles []string                       `json:"aggregated_roles"`  // 聚合的角色列表（如果有聚合规则）
}
