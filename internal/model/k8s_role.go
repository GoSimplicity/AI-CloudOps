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

// RoleStatus Role状态枚举
type RoleStatus int8

const (
	RoleStatusActive   RoleStatus = iota + 1 // 活跃
	RoleStatusInactive                       // 非活跃
	RoleStatusUnused                         // 未使用
	RoleStatusError                          // 异常
)

// K8sRole Kubernetes Role模型
type K8sRole struct {
	Name              string            `json:"name"`               // Role名称
	Namespace         string            `json:"namespace"`          // 命名空间
	ClusterID         int               `json:"cluster_id"`         // 所属集群ID
	UID               string            `json:"uid"`                // Role UID
	Status            RoleStatus        `json:"status"`             // Role状态
	Labels            map[string]string `json:"labels"`             // Role标签
	Annotations       map[string]string `json:"annotations"`        // Role注解
	Rules             []PolicyRule      `json:"rules"`              // 权限规则列表
	ResourceVersion   string            `json:"resource_version"`   // 资源版本
	Age               string            `json:"age"`                // 存在时间
	BindingCount      int               `json:"binding_count"`      // 关联的RoleBinding数量
	ActiveSubjects    int               `json:"active_subjects"`    // 活跃主体数量
	IsSystemRole      BoolValue         `json:"is_system_role"`     // 是否为系统角色
	CreationTimestamp time.Time         `json:"creation_timestamp"` // 创建时间
	RawRole           *rbacv1.Role      `json:"-"`                  // 原始Role对象，不序列化
}

// K8sRoleEvent Role相关事件
type K8sRoleEvent struct {
	Type      string    `json:"type"`       // 事件类型 (Normal, Warning)
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Source    string    `json:"source"`     // 事件源
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Count     int32     `json:"count"`      // 发生次数
}

// K8sRoleUsage Role使用情况
type K8sRoleUsage struct {
	RoleName     string                  `json:"role_name"`     // Role名称
	Namespace    string                  `json:"namespace"`     // 命名空间
	Bindings     []RoleBindingSimpleInfo `json:"bindings"`      // 关联的RoleBinding
	Subjects     []Subject               `json:"subjects"`      // 所有主体
	Permissions  []PolicyRule            `json:"permissions"`   // 权限列表
	IsUsed       BoolValue               `json:"is_used"`       // 是否被使用
	RiskLevel    string                  `json:"risk_level"`    // 风险等级
	LastAccessed *time.Time              `json:"last_accessed"` // 最后访问时间
}
