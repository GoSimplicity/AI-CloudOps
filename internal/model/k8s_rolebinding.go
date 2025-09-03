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

// RoleBindingStatus RoleBinding状态枚举
type RoleBindingStatus int8

const (
	RoleBindingStatusActive   RoleBindingStatus = iota + 1 // 活跃
	RoleBindingStatusInactive                              // 非活跃
	RoleBindingStatusOrphaned                              // 孤立的（Role不存在）
	RoleBindingStatusError                                 // 异常
)

// K8sRoleBinding Kubernetes RoleBinding模型
type K8sRoleBinding struct {
	Name              string              `json:"name"`               // RoleBinding名称
	Namespace         string              `json:"namespace"`          // 命名空间
	ClusterID         int                 `json:"cluster_id"`         // 所属集群ID
	UID               string              `json:"uid"`                // RoleBinding UID
	Status            RoleBindingStatus   `json:"status"`             // RoleBinding状态
	Labels            map[string]string   `json:"labels"`             // RoleBinding标签
	Annotations       map[string]string   `json:"annotations"`        // RoleBinding注解
	RoleRef           RoleRef             `json:"role_ref"`           // 角色引用
	Subjects          []Subject           `json:"subjects"`           // 主体列表
	ResourceVersion   string              `json:"resource_version"`   // 资源版本
	Age               string              `json:"age"`                // 存在时间
	SubjectCount      int                 `json:"subject_count"`      // 主体数量
	IsSystemBinding   BoolValue           `json:"is_system_binding"`  // 是否为系统绑定
	IsOrphaned        BoolValue           `json:"is_orphaned"`        // 是否为孤立绑定（Role不存在）
	CreationTimestamp time.Time           `json:"creation_timestamp"` // 创建时间
	RawRoleBinding    *rbacv1.RoleBinding `json:"-"`                  // 原始RoleBinding对象，不序列化
}

// K8sRoleBindingEvent RoleBinding相关事件
type K8sRoleBindingEvent struct {
	Type      string    `json:"type"`       // 事件类型 (Normal, Warning)
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Source    string    `json:"source"`     // 事件源
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Count     int32     `json:"count"`      // 发生次数
}

// K8sRoleBindingSubjectSummary RoleBinding主体摘要
type K8sRoleBindingSubjectSummary struct {
	RoleBindingName  string       `json:"role_binding_name"` // RoleBinding名称
	Namespace        string       `json:"namespace"`         // 命名空间
	Users            []string     `json:"users"`             // 用户列表
	Groups           []string     `json:"groups"`            // 组列表
	ServiceAccounts  []string     `json:"service_accounts"`  // 服务账户列表
	EffectiveRules   []PolicyRule `json:"effective_rules"`   // 有效权限规则
	TotalPermissions int          `json:"total_permissions"` // 总权限数
	RiskAssessment   string       `json:"risk_assessment"`   // 风险评估
}

// K8sRoleBindingDependency RoleBinding依赖关系
type K8sRoleBindingDependency struct {
	RoleBindingName string    `json:"role_binding_name"` // RoleBinding名称
	Namespace       string    `json:"namespace"`         // 命名空间
	RoleName        string    `json:"role_name"`         // 依赖的Role名称
	RoleKind        string    `json:"role_kind"`         // Role类型
	RoleExists      BoolValue `json:"role_exists"`       // Role是否存在
	SubjectsExist   []string  `json:"subjects_exist"`    // 存在的主体列表
	MissingSubjects []string  `json:"missing_subjects"`  // 缺失的主体列表
	IsHealthy       BoolValue `json:"is_healthy"`        // 依赖关系是否健康
}

// K8sRoleBindingUsage RoleBinding使用分析
type K8sRoleBindingUsage struct {
	RoleBindingName  string                       `json:"role_binding_name"` // RoleBinding名称
	Namespace        string                       `json:"namespace"`         // 命名空间
	RoleName         string                       `json:"role_name"`         // 引用的Role名称
	RoleKind         string                       `json:"role_kind"`         // Role类型
	SubjectAnalysis  K8sRoleBindingSubjectSummary `json:"subject_analysis"`  // 主体分析
	DependencyStatus K8sRoleBindingDependency     `json:"dependency_status"` // 依赖状态
	AccessPatterns   []AccessPattern              `json:"access_patterns"`   // 访问模式
	SecurityRisks    []SecurityRisk               `json:"security_risks"`    // 安全风险
	Recommendations  []string                     `json:"recommendations"`   // 使用建议
	LastAnalyzed     time.Time                    `json:"last_analyzed"`     // 最后分析时间
	AnalysisScore    int                          `json:"analysis_score"`    // 分析评分
}

// AccessPattern 访问模式
type AccessPattern struct {
	Resource    string    `json:"resource"`     // 资源类型
	Verb        string    `json:"verb"`         // 操作动词
	Frequency   int       `json:"frequency"`    // 使用频率
	LastUsed    time.Time `json:"last_used"`    // 最后使用时间
	IsEffective BoolValue `json:"is_effective"` // 是否有效
}

// SecurityRisk 安全风险
type SecurityRisk struct {
	Type        string `json:"type"`        // 风险类型
	Level       string `json:"level"`       // 风险等级
	Description string `json:"description"` // 风险描述
	Impact      string `json:"impact"`      // 影响范围
	Mitigation  string `json:"mitigation"`  // 缓解建议
}
