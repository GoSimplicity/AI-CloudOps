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

	corev1 "k8s.io/api/core/v1"
)

// ServiceAccountStatus ServiceAccount状态枚举
type ServiceAccountStatus int8

const (
	ServiceAccountStatusActive   ServiceAccountStatus = iota + 1 // 活跃
	ServiceAccountStatusInactive                                 // 非活跃
	ServiceAccountStatusError                                    // 异常
)

// K8sServiceAccount Kubernetes ServiceAccount模型
type K8sServiceAccount struct {
	Name                         string                 `json:"name"`                            // ServiceAccount名称
	Namespace                    string                 `json:"namespace"`                       // 命名空间
	ClusterID                    int                    `json:"cluster_id"`                      // 所属集群ID
	UID                          string                 `json:"uid"`                             // ServiceAccount UID
	Status                       ServiceAccountStatus   `json:"status"`                          // ServiceAccount状态
	Labels                       map[string]string      `json:"labels"`                          // ServiceAccount标签
	Annotations                  map[string]string      `json:"annotations"`                     // ServiceAccount注解
	AutomountServiceAccountToken *BoolValue             `json:"automount_service_account_token"` // 是否自动挂载ServiceAccount Token
	Secrets                      []ServiceAccountSecret `json:"secrets"`                         // 关联的Secrets
	ImagePullSecrets             []ServiceAccountSecret `json:"image_pull_secrets"`              // 关联的ImagePullSecrets
	ResourceVersion              string                 `json:"resource_version"`                // 资源版本
	Age                          string                 `json:"age"`                             // 存在时间
	SecretsCount                 int                    `json:"secrets_count"`                   // Secrets数量
	ImagePullSecretsCount        int                    `json:"image_pull_secrets_count"`        // ImagePullSecrets数量
	RoleBindingCount             int                    `json:"role_binding_count"`              // 关联的RoleBinding数量
	ClusterRoleBindingCount      int                    `json:"cluster_role_binding_count"`      // 关联的ClusterRoleBinding数量
	IsSystemAccount              BoolValue              `json:"is_system_account"`               // 是否为系统账户
	CreationTimestamp            time.Time              `json:"creation_timestamp"`              // 创建时间
	RawServiceAccount            *corev1.ServiceAccount `json:"-"`                               // 原始ServiceAccount对象，不序列化
}

// ServiceAccountSecret ServiceAccount关联的Secret信息
type ServiceAccountSecret struct {
	Name      string `json:"name"`      // Secret名称
	Namespace string `json:"namespace"` // 命名空间
	Type      string `json:"type"`      // Secret类型
}

// K8sServiceAccountEvent ServiceAccount相关事件
type K8sServiceAccountEvent struct {
	Type      string    `json:"type"`       // 事件类型 (Normal, Warning)
	Reason    string    `json:"reason"`     // 事件原因
	Message   string    `json:"message"`    // 事件消息
	Source    string    `json:"source"`     // 事件源
	FirstTime time.Time `json:"first_time"` // 首次发生时间
	LastTime  time.Time `json:"last_time"`  // 最后发生时间
	Count     int32     `json:"count"`      // 发生次数
}

// K8sServiceAccountMetrics ServiceAccount指标信息
type K8sServiceAccountMetrics struct {
	ServiceAccountName       string    `json:"service_account_name"`        // ServiceAccount名称
	Namespace                string    `json:"namespace"`                   // 命名空间
	TotalRoleBindings        int       `json:"total_role_bindings"`         // 总RoleBinding数
	TotalClusterRoleBindings int       `json:"total_cluster_role_bindings"` // 总ClusterRoleBinding数
	TotalSecrets             int       `json:"total_secrets"`               // 总Secrets数
	TotalImagePullSecrets    int       `json:"total_image_pull_secrets"`    // 总ImagePullSecrets数
	TokensCreated            int       `json:"tokens_created"`              // 创建的Token数
	PodsUsingAccount         int       `json:"pods_using_account"`          // 使用该账户的Pod数
	IsActive                 BoolValue `json:"is_active"`                   // 是否活跃
	AutomountEnabled         BoolValue `json:"automount_enabled"`           // 是否启用自动挂载
	SecurityRisk             string    `json:"security_risk"`               // 安全风险等级 (Low, Medium, High)
	LastUsed                 time.Time `json:"last_used"`                   // 最后使用时间
	LastUpdated              time.Time `json:"last_updated"`                // 最后更新时间
}

// K8sServiceAccountUsage ServiceAccount使用情况
type K8sServiceAccountUsage struct {
	ServiceAccountName   string                         `json:"service_account_name"`  // ServiceAccount名称
	Namespace            string                         `json:"namespace"`             // 命名空间
	RoleBindings         []RoleBindingSimpleInfo        `json:"role_bindings"`         // 关联的RoleBinding
	ClusterRoleBindings  []ClusterRoleBindingSimpleInfo `json:"cluster_role_bindings"` // 关联的ClusterRoleBinding
	EffectivePermissions []PolicyRule                   `json:"effective_permissions"` // 有效权限
	Secrets              []ServiceAccountSecret         `json:"secrets"`               // 关联的Secrets
	ImagePullSecrets     []ServiceAccountSecret         `json:"image_pull_secrets"`    // 关联的ImagePullSecrets
	UsedByPods           []string                       `json:"used_by_pods"`          // 使用该账户的Pod列表
	IsUsed               BoolValue                      `json:"is_used"`               // 是否被使用
	RiskLevel            string                         `json:"risk_level"`            // 风险等级
	LastAccessed         *time.Time                     `json:"last_accessed"`         // 最后访问时间
}

// K8sServiceAccountToken ServiceAccount Token信息
type K8sServiceAccountToken struct {
	Token               string     `json:"token"`                          // Token内容
	ExpirationTimestamp *time.Time `json:"expiration_timestamp,omitempty"` // 过期时间
	Audience            []string   `json:"audience,omitempty"`             // 受众
	BoundObjectRef      *string    `json:"bound_object_ref,omitempty"`     // 绑定对象引用
	CreatedAt           time.Time  `json:"created_at"`                     // 创建时间
}
