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
)

// K8sNamespaceEntity Kubernetes 命名空间数据库实体
type K8sNamespaceEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:命名空间名称"`   // 命名空间名称
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                        // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:命名空间UID"`                                    // 命名空间UID
	Status            string            `json:"status" gorm:"size:50;comment:命名空间状态"`                                   // 命名空间状态
	Phase             string            `json:"phase" gorm:"size:50;comment:命名空间阶段"`                                    // 命名空间阶段
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                     // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                       // Kubernetes创建时间
	ResourceQuota     *ResourceQuota    `json:"resource_quota,omitempty" gorm:"type:text;serializer:json;comment:资源配额"` // 资源配额
	Age               string            `json:"age" gorm:"-"`                                                           // 存在时间，前端计算使用
}

func (k *K8sNamespaceEntity) TableName() string {
	return "cl_k8s_namespaces"
}

// K8sNamespaceListRequest 命名空间列表查询请求
type K8sNamespaceListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sNamespaceCreateRequest 创建命名空间请求
type K8sNamespaceCreateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"`           // 集群ID，必填
	Name        string            `json:"name" binding:"required,min=1,max=200" comment:"命名空间名称"` // 命名空间名称，必填
	Labels      map[string]string `json:"labels" comment:"标签"`                                    // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                               // 注解
}

// K8sNamespaceUpdateRequest 更新命名空间请求
type K8sNamespaceUpdateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name        string            `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
}

// K8sNamespaceDeleteRequest 删除命名空间请求
type K8sNamespaceDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name               string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sNamespaceBatchDeleteRequest 批量删除命名空间请求
type K8sNamespaceBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Names              []string `json:"names" binding:"required" comment:"命名空间名称列表"`  // 命名空间名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sNamespaceResourceRequest 获取命名空间资源请求
type K8sNamespaceResourceReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
}

// K8sNamespaceQuotaRequest 设置命名空间资源配额请求
type K8sNamespaceQuotaReq struct {
	ClusterID     int            `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Name          string         `json:"name" binding:"required" comment:"命名空间名称"`         // 命名空间名称，必填
	ResourceQuota *ResourceQuota `json:"resource_quota" binding:"required" comment:"资源配额"` // 资源配额，必填
}

// K8sNamespaceEventRequest 获取命名空间事件请求
type K8sNamespaceEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"命名空间名称"`     // 命名空间名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// ====================== Namespace响应实体 ======================

// NamespaceEntity Namespace响应实体
type NamespaceEntity struct {
	Name            string                        `json:"name"`             // 命名空间名称
	UID             string                        `json:"uid"`              // 命名空间UID
	Labels          map[string]string             `json:"labels"`           // 标签
	Annotations     map[string]string             `json:"annotations"`      // 注解
	Status          string                        `json:"status"`           // 命名空间状态
	Phase           string                        `json:"phase"`            // 命名空间阶段
	ResourceQuota   *NamespaceResourceQuotaEntity `json:"resource_quota"`   // 资源配额
	LimitRanges     []NamespaceLimitRangeEntity   `json:"limit_ranges"`     // 限制范围
	NetworkPolicies []string                      `json:"network_policies"` // 网络策略
	Age             string                        `json:"age"`              // 存在时间
	CreatedAt       string                        `json:"created_at"`       // 创建时间
}

// NamespaceResourceQuotaEntity 命名空间资源配额实体
type NamespaceResourceQuotaEntity struct {
	Name          string                        `json:"name"`           // 配额名称
	Hard          map[string]string             `json:"hard"`           // 硬限制
	Used          map[string]string             `json:"used"`           // 已使用
	Scopes        []string                      `json:"scopes"`         // 作用域
	ScopeSelector *NamespaceScopeSelectorEntity `json:"scope_selector"` // 作用域选择器
}

// NamespaceScopeSelectorEntity 作用域选择器实体
type NamespaceScopeSelectorEntity struct {
	MatchExpressions []NamespaceScopeExpressionEntity `json:"match_expressions"` // 匹配表达式
}

// NamespaceScopeExpressionEntity 作用域表达式实体
type NamespaceScopeExpressionEntity struct {
	ScopeName string   `json:"scope_name"` // 作用域名称
	Operator  string   `json:"operator"`   // 操作符
	Values    []string `json:"values"`     // 值列表
}

// NamespaceLimitRangeEntity 命名空间限制范围实体
type NamespaceLimitRangeEntity struct {
	Name   string                          `json:"name"`   // 限制范围名称
	Limits []NamespaceLimitRangeItemEntity `json:"limits"` // 限制项
}

// NamespaceLimitRangeItemEntity 限制范围项实体
type NamespaceLimitRangeItemEntity struct {
	Type                 string            `json:"type"`                    // 资源类型
	Max                  map[string]string `json:"max"`                     // 最大值
	Min                  map[string]string `json:"min"`                     // 最小值
	Default              map[string]string `json:"default"`                 // 默认值
	DefaultRequest       map[string]string `json:"default_request"`         // 默认请求
	MaxLimitRequestRatio map[string]string `json:"max_limit_request_ratio"` // 最大限制请求比例
}

// NamespaceListResponse Namespace列表响应
type NamespaceListResponse struct {
	Items      []NamespaceEntity `json:"items"`       // Namespace列表
	TotalCount int               `json:"total_count"` // 总数
}

// NamespaceDetailResponse Namespace详情响应
type NamespaceDetailResponse struct {
	Namespace   NamespaceEntity          `json:"namespace"`   // Namespace信息
	YAML        string                   `json:"yaml"`        // YAML内容
	Events      []NamespaceEventEntity   `json:"events"`      // 事件列表
	Resources   NamespaceResourcesEntity `json:"resources"`   // 资源统计
	Pods        []PodEntity              `json:"pods"`        // Pod列表
	Services    []ServiceEntity          `json:"services"`    // Service列表
	Deployments []DeploymentEntity       `json:"deployments"` // Deployment列表
}

// NamespaceEventEntity Namespace事件实体
type NamespaceEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// NamespaceResourcesEntity 命名空间资源统计实体
type NamespaceResourcesEntity struct {
	Pods         NamespaceResourceCountEntity `json:"pods"`         // Pod统计
	Services     NamespaceResourceCountEntity `json:"services"`     // Service统计
	Deployments  NamespaceResourceCountEntity `json:"deployments"`  // Deployment统计
	StatefulSets NamespaceResourceCountEntity `json:"statefulsets"` // StatefulSet统计
	DaemonSets   NamespaceResourceCountEntity `json:"daemonsets"`   // DaemonSet统计
	Jobs         NamespaceResourceCountEntity `json:"jobs"`         // Job统计
	CronJobs     NamespaceResourceCountEntity `json:"cronjobs"`     // CronJob统计
	ConfigMaps   NamespaceResourceCountEntity `json:"configmaps"`   // ConfigMap统计
	Secrets      NamespaceResourceCountEntity `json:"secrets"`      // Secret统计
	PVCs         NamespaceResourceCountEntity `json:"pvcs"`         // PVC统计
	Ingresses    NamespaceResourceCountEntity `json:"ingresses"`    // Ingress统计
}

// NamespaceResourceCountEntity 资源计数实体
type NamespaceResourceCountEntity struct {
	Total   int `json:"total"`   // 总数
	Running int `json:"running"` // 运行中
	Pending int `json:"pending"` // 等待中
	Failed  int `json:"failed"`  // 失败
}

// NamespaceQuotaResponse 设置配额响应
type NamespaceQuotaResponse struct {
	NamespaceName string                        `json:"namespace_name"` // 命名空间名称
	QuotaName     string                        `json:"quota_name"`     // 配额名称
	ResourceQuota *NamespaceResourceQuotaEntity `json:"resource_quota"` // 资源配额
	Status        string                        `json:"status"`         // 操作状态
	Message       string                        `json:"message"`        // 操作消息
}
