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

type ClusterRoleBindingStatus int8

const (
	ClusterRoleBindingStatusActive   ClusterRoleBindingStatus = iota + 1 // 活跃
	ClusterRoleBindingStatusInactive                                     // 非活跃
	ClusterRoleBindingStatusOrphaned                                     // 孤立的（ClusterRole不存在）
	ClusterRoleBindingStatusError                                        // 异常
)

// K8sClusterRoleBinding Kubernetes ClusterRoleBinding模型
type K8sClusterRoleBinding struct {
	Name                    string                     `json:"name"`                       // ClusterRoleBinding名称
	ClusterID               int                        `json:"cluster_id"`                 // 所属集群ID
	UID                     string                     `json:"uid"`                        // ClusterRoleBinding UID
	Status                  ClusterRoleBindingStatus   `json:"status"`                     // ClusterRoleBinding状态
	Labels                  map[string]string          `json:"labels"`                     // ClusterRoleBinding标签
	Annotations             map[string]string          `json:"annotations"`                // ClusterRoleBinding注解
	RoleRef                 RoleRef                    `json:"role_ref"`                   // 角色引用
	Subjects                []Subject                  `json:"subjects"`                   // 主体列表
	ResourceVersion         string                     `json:"resource_version"`           // 资源版本
	Age                     string                     `json:"age"`                        // 存在时间
	SubjectCount            int                        `json:"subject_count"`              // 主体数量
	IsSystemBinding         BoolValue                  `json:"is_system_binding"`          // 是否为系统绑定
	IsOrphaned              BoolValue                  `json:"is_orphaned"`                // 是否为孤立绑定（ClusterRole不存在）
	GrantsClusterWideAccess BoolValue                  `json:"grants_cluster_wide_access"` // 是否授予集群级访问权限
	SecurityRisk            string                     `json:"security_risk"`              // 安全风险等级
	CreationTimestamp       time.Time                  `json:"creation_timestamp"`         // 创建时间
	RawClusterRoleBinding   *rbacv1.ClusterRoleBinding `json:"-"`                          // 原始ClusterRoleBinding对象，不序列化
}

// K8sClusterRoleBindingSubjectSummary ClusterRoleBinding主体摘要
type K8sClusterRoleBindingSubjectSummary struct {
	ClusterRoleBindingName string       `json:"cluster_role_binding_name"` // ClusterRoleBinding名称
	Users                  []string     `json:"users"`                     // 用户列表
	Groups                 []string     `json:"groups"`                    // 组列表
	ServiceAccounts        []Subject    `json:"service_accounts"`          // 服务账户列表（包含命名空间信息）
	EffectiveRules         []PolicyRule `json:"effective_rules"`           // 有效权限规则
	TotalPermissions       int          `json:"total_permissions"`         // 总权限数
	ClusterWideRules       []PolicyRule `json:"cluster_wide_rules"`        // 集群级权限规则
	NamespaceRules         []PolicyRule `json:"namespace_rules"`           // 命名空间级权限规则
	NonResourceRules       []PolicyRule `json:"non_resource_rules"`        // 非资源权限规则
	RiskAssessment         string       `json:"risk_assessment"`           // 风险评估
	PrivilegeEscalation    BoolValue    `json:"privilege_escalation"`      // 是否存在权限提升风险
}

// K8sClusterRoleBindingDependency ClusterRoleBinding依赖关系
type K8sClusterRoleBindingDependency struct {
	ClusterRoleBindingName string    `json:"cluster_role_binding_name"` // ClusterRoleBinding名称
	ClusterRoleName        string    `json:"cluster_role_name"`         // 依赖的ClusterRole名称
	ClusterRoleExists      BoolValue `json:"cluster_role_exists"`       // ClusterRole是否存在
	SubjectsExist          []string  `json:"subjects_exist"`            // 存在的主体列表
	MissingSubjects        []string  `json:"missing_subjects"`          // 缺失的主体列表
	OrphanedNamespaces     []string  `json:"orphaned_namespaces"`       // 孤立的命名空间（ServiceAccount所在命名空间不存在）
	IsHealthy              BoolValue `json:"is_healthy"`                // 依赖关系是否健康
	SecurityImplications   []string  `json:"security_implications"`     // 安全影响
}

// K8sClusterRoleBindingSecurityAudit ClusterRoleBinding安全审计
type K8sClusterRoleBindingSecurityAudit struct {
	ClusterRoleBindingName   string    `json:"cluster_role_binding_name"`  // ClusterRoleBinding名称
	RiskScore                int       `json:"risk_score"`                 // 风险评分 (0-100)
	SecurityViolations       []string  `json:"security_violations"`        // 安全违规
	PrivilegeEscalationRisks []string  `json:"privilege_escalation_risks"` // 权限提升风险
	OverPrivilegedSubjects   []string  `json:"over_privileged_subjects"`   // 权限过大的主体
	UnusedPermissions        []string  `json:"unused_permissions"`         // 未使用的权限
	RecommendedActions       []string  `json:"recommended_actions"`        // 建议操作
	ComplianceStatus         string    `json:"compliance_status"`          // 合规状态
	LastAuditTime            time.Time `json:"last_audit_time"`            // 最后审计时间
}

// K8sClusterRoleBindingUsage ClusterRoleBinding使用情况
type K8sClusterRoleBindingUsage struct {
	ClusterRoleBindingName string                           `json:"cluster_role_binding_name"`    // ClusterRoleBinding名称
	ClusterRoleName        string                           `json:"cluster_role_name"`            // 关联的ClusterRole名称
	Subjects               []Subject                        `json:"subjects"`                     // 绑定的主体列表
	SubjectCount           int                              `json:"subject_count"`                // 主体数量
	UserCount              int                              `json:"user_count"`                   // 用户数量
	GroupCount             int                              `json:"group_count"`                  // 用户组数量
	ServiceAccountCount    int                              `json:"service_account_count"`        // ServiceAccount数量
	IsSystemBinding        BoolValue                        `json:"is_system_binding"`            // 是否为系统绑定
	RiskLevel              string                           `json:"risk_level"`                   // 风险等级
	ClusterRoleInfo        *ClusterRoleBindingRoleInfo      `json:"cluster_role_info,omitempty"`  // 关联的ClusterRole信息
	ConflictDetection      *ClusterRoleBindingConflictCheck `json:"conflict_detection,omitempty"` // 冲突检测信息
	CreationTimestamp      time.Time                        `json:"creation_timestamp"`           // 创建时间
	LastUpdated            time.Time                        `json:"last_updated"`                 // 最后更新时间
}

// K8sClusterRoleBindingEvent ClusterRoleBinding事件信息
type K8sClusterRoleBindingEvent struct {
	Type      string    `json:"type"`       // 事件类型 (Normal, Warning)
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Source    string    `json:"source"`     // 事件源
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Count     int32     `json:"count"`      // 发生次数
}

// ClusterRoleBindingRoleInfo ClusterRoleBinding关联的角色信息
type ClusterRoleBindingRoleInfo struct {
	Name         string       `json:"name"`           // 角色名称
	Kind         string       `json:"kind"`           // 角色类型
	Permissions  []PolicyRule `json:"permissions"`    // 权限列表
	RiskLevel    string       `json:"risk_level"`     // 权限风险等级
	IsSystemRole BoolValue    `json:"is_system_role"` // 是否为系统角色
	CreatedAt    time.Time    `json:"created_at"`     // 角色创建时间
}

// ClusterRoleBindingConflictCheck ClusterRoleBinding冲突检测
type ClusterRoleBindingConflictCheck struct {
	HasConflicts      BoolValue                         `json:"has_conflicts"`                // 是否存在冲突
	ConflictBindings  []ClusterRoleBindingConflictInfo  `json:"conflict_bindings,omitempty"`  // 冲突的绑定列表
	DuplicateSubjects []ClusterRoleBindingDuplicateInfo `json:"duplicate_subjects,omitempty"` // 重复的主体
	OverPermissions   []string                          `json:"over_permissions,omitempty"`   // 过度权限列表
}

// ClusterRoleBindingConflictInfo 冲突绑定信息
type ClusterRoleBindingConflictInfo struct {
	Name         string `json:"name"`          // 冲突绑定名称
	ConflictType string `json:"conflict_type"` // 冲突类型
	Description  string `json:"description"`   // 冲突描述
	Severity     string `json:"severity"`      // 严重程度
}

// ClusterRoleBindingDuplicateInfo 重复主体信息
type ClusterRoleBindingDuplicateInfo struct {
	Subject       Subject  `json:"subject"`        // 重复的主体
	OtherBindings []string `json:"other_bindings"` // 其他包含该主体的绑定
	Reason        string   `json:"reason"`         // 重复原因
}

// ClusterRoleBindingUsageStats ClusterRoleBinding使用统计
type ClusterRoleBindingUsageStats struct {
	ResourceTypes      []string                          `json:"resource_types"`                 // 涉及的资源类型
	MostUsedVerbs      []string                          `json:"most_used_verbs"`                // 最常用的动作
	AccessPatterns     []ClusterRoleBindingAccessPattern `json:"access_patterns,omitempty"`      // 访问模式
	UnusedPermissions  []string                          `json:"unused_permissions,omitempty"`   // 未使用的权限
	HighRiskOperations []string                          `json:"high_risk_operations,omitempty"` // 高风险操作
}

// ClusterRoleBindingAccessPattern ClusterRoleBinding访问模式
type ClusterRoleBindingAccessPattern struct {
	Resource     string    `json:"resource"`      // 资源
	Verbs        []string  `json:"verbs"`         // 动作
	Frequency    int       `json:"frequency"`     // 频率
	LastAccessed time.Time `json:"last_accessed"` // 最后访问时间
}

// ClusterRoleBindingRecommendation ClusterRoleBinding推荐操作
type ClusterRoleBindingRecommendation struct {
	Type        string    `json:"type"`        // 推荐类型
	Priority    string    `json:"priority"`    // 优先级
	Description string    `json:"description"` // 描述
	Action      string    `json:"action"`      // 建议操作
	Impact      string    `json:"impact"`      // 影响
	CreatedAt   time.Time `json:"created_at"`  // 推荐生成时间
}
