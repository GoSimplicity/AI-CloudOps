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

// K8sPVUsageInfo PV使用信息
type K8sPVUsageInfo struct {
	Total     string  `json:"total"`      // 总容量
	Used      string  `json:"used"`       // 已使用
	Available string  `json:"available"`  // 可用
	UsageRate float64 `json:"usage_rate"` // 使用率
}

// K8sPVEntity Kubernetes PersistentVolume数据库实体
type K8sPVEntity struct {
	Model
	Name              string                 `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:PV名称"` // PV名称
	ClusterID         int                    `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                    // 所属集群ID
	UID               string                 `json:"uid" gorm:"size:100;comment:PV UID"`                                 // PV UID
	Capacity          string                 `json:"capacity" gorm:"size:50;comment:存储容量"`                               // 存储容量
	AccessModes       []string               `json:"access_modes" gorm:"type:text;serializer:json;comment:访问模式"`         // 访问模式
	ReclaimPolicy     string                 `json:"reclaim_policy" gorm:"size:50;comment:回收策略"`                         // 回收策略
	StorageClass      string                 `json:"storage_class" gorm:"size:200;comment:存储类"`                          // 存储类
	VolumeMode        string                 `json:"volume_mode" gorm:"size:50;comment:卷模式"`                             // 卷模式
	Status            string                 `json:"status" gorm:"size:50;comment:PV状态"`                                 // PV状态
	ClaimRef          map[string]string      `json:"claim_ref" gorm:"type:text;serializer:json;comment:PVC引用"`           // PVC引用
	VolumeSource      map[string]interface{} `json:"volume_source" gorm:"type:text;serializer:json;comment:卷源配置"`        // 卷源配置
	NodeAffinity      map[string]interface{} `json:"node_affinity" gorm:"type:text;serializer:json;comment:节点亲和性"`       // 节点亲和性
	Labels            map[string]string      `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                 // 标签
	Annotations       map[string]string      `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`            // 注解
	CreationTimestamp time.Time              `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                   // Kubernetes创建时间
	Age               string                 `json:"age" gorm:"-"`                                                       // 存在时间，前端计算使用
	VolumeType        string                 `json:"volume_type" gorm:"-"`                                               // 卷类型，前端计算使用
	UsedCapacity      string                 `json:"used_capacity" gorm:"-"`                                             // 已使用容量，前端计算使用
	AvailableCapacity string                 `json:"available_capacity" gorm:"-"`                                        // 可用容量，前端计算使用
}

func (k *K8sPVEntity) TableName() string {
	return "cl_k8s_pvs"
}

// K8sPVListRequest PV列表查询请求
type K8sPVListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"PV状态过滤"`                          // PV状态过滤
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`             // 存储类过滤
	AccessMode    string `json:"access_mode" form:"access_mode" comment:"访问模式过滤"`                // 访问模式过滤
	VolumeType    string `json:"volume_type" form:"volume_type" comment:"卷类型过滤"`                 // 卷类型过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sPVCreateRequest 创建PV请求
type K8sPVCreateReq struct {
	ClusterID     int                      `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Name          string                   `json:"name" binding:"required" comment:"PV名称"`          // PV名称，必填
	Capacity      string                   `json:"capacity" binding:"required" comment:"存储容量"`      // 存储容量，必填
	AccessModes   []string                 `json:"access_modes" binding:"required" comment:"访问模式"`  // 访问模式，必填
	ReclaimPolicy string                   `json:"reclaim_policy" comment:"回收策略"`                   // 回收策略
	StorageClass  string                   `json:"storage_class" comment:"存储类"`                     // 存储类
	VolumeMode    string                   `json:"volume_mode" comment:"卷模式"`                       // 卷模式
	VolumeSource  map[string]interface{}   `json:"volume_source" binding:"required" comment:"卷源配置"` // 卷源配置，必填
	NodeAffinity  map[string]interface{}   `json:"node_affinity" comment:"节点亲和性"`                   // 节点亲和性
	Labels        map[string]string        `json:"labels" comment:"标签"`                             // 标签
	Annotations   map[string]string        `json:"annotations" comment:"注解"`                        // 注解
	PVYaml        *corev1.PersistentVolume `json:"pv_yaml" comment:"PV YAML对象"`                     // PV YAML对象
}

// K8sPVUpdateRequest 更新PV请求
type K8sPVUpdateReq struct {
	ClusterID     int                      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name          string                   `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	Capacity      string                   `json:"capacity" comment:"存储容量"`                      // 存储容量
	AccessModes   []string                 `json:"access_modes" comment:"访问模式"`                  // 访问模式
	ReclaimPolicy string                   `json:"reclaim_policy" comment:"回收策略"`                // 回收策略
	StorageClass  string                   `json:"storage_class" comment:"存储类"`                  // 存储类
	VolumeMode    string                   `json:"volume_mode" comment:"卷模式"`                    // 卷模式
	VolumeSource  map[string]interface{}   `json:"volume_source" comment:"卷源配置"`                 // 卷源配置
	NodeAffinity  map[string]interface{}   `json:"node_affinity" comment:"节点亲和性"`                // 节点亲和性
	Labels        map[string]string        `json:"labels" comment:"标签"`                          // 标签
	Annotations   map[string]string        `json:"annotations" comment:"注解"`                     // 注解
	PVYaml        *corev1.PersistentVolume `json:"pv_yaml" comment:"PV YAML对象"`                  // PV YAML对象
}

// K8sPVReclaimReq PV回收请求
type K8sPVReclaimReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
}

// K8sPVDeleteRequest 删除PV请求
type K8sPVDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name               string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPVBatchDeleteRequest 批量删除PV请求
type K8sPVBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Names              []string `json:"names" binding:"required" comment:"PV名称列表"`    // PV名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPVEventRequest 获取PV事件请求
type K8sPVEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sPVUsageRequest 获取PV使用情况请求
type K8sPVUsageReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Name      string `json:"name" binding:"required" comment:"PV名称"`       // PV名称，必填
}

// K8sPVExpandRequest 扩容PV请求
type K8sPVExpandReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Name        string `json:"name" binding:"required" comment:"PV名称"`        // PV名称，必填
	NewCapacity string `json:"new_capacity" binding:"required" comment:"新容量"` // 新容量，必填
}

// K8sPVBackupRequest 备份PV请求
type K8sPVBackupReq struct {
	ClusterID   int      `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Names       []string `json:"names" binding:"required" comment:"PV名称列表"`     // PV名称列表，必填
	BackupName  string   `json:"backup_name" binding:"required" comment:"备份名称"` // 备份名称，必填
	Description string   `json:"description" comment:"备份描述"`                    // 备份描述
}

// ====================== PV响应实体 ======================

// PVEntity PV响应实体
type PVEntity struct {
	Name              string               `json:"name"`               // PV名称
	UID               string               `json:"uid"`                // PV UID
	Labels            map[string]string    `json:"labels"`             // 标签
	Annotations       map[string]string    `json:"annotations"`        // 注解
	Capacity          string               `json:"capacity"`           // 存储容量
	AccessModes       []string             `json:"access_modes"`       // 访问模式
	ReclaimPolicy     string               `json:"reclaim_policy"`     // 回收策略
	StorageClass      string               `json:"storage_class"`      // 存储类
	VolumeMode        string               `json:"volume_mode"`        // 卷模式
	Status            string               `json:"status"`             // PV状态
	ClaimRef          PVClaimRefEntity     `json:"claim_ref"`          // PVC引用
	VolumeSource      PVVolumeSourceEntity `json:"volume_source"`      // 卷源配置
	NodeAffinity      PVNodeAffinityEntity `json:"node_affinity"`      // 节点亲和性
	VolumeType        string               `json:"volume_type"`        // 卷类型
	UsedCapacity      string               `json:"used_capacity"`      // 已使用容量
	AvailableCapacity string               `json:"available_capacity"` // 可用容量
	Age               string               `json:"age"`                // 存在时间
	CreatedAt         string               `json:"created_at"`         // 创建时间
}

// PVClaimRefEntity PVC引用实体
type PVClaimRefEntity struct {
	Kind            string `json:"kind"`             // 类型
	Namespace       string `json:"namespace"`        // 命名空间
	Name            string `json:"name"`             // 名称
	UID             string `json:"uid"`              // UID
	APIVersion      string `json:"api_version"`      // API版本
	ResourceVersion string `json:"resource_version"` // 资源版本
}

// PVVolumeSourceEntity PV卷源实体
type PVVolumeSourceEntity struct {
	Type     string                 `json:"type"`      // 卷源类型
	Config   map[string]interface{} `json:"config"`    // 配置信息
	Path     string                 `json:"path"`      // 路径
	Server   string                 `json:"server"`    // 服务器
	Driver   string                 `json:"driver"`    // 驱动
	VolumeID string                 `json:"volume_id"` // 卷ID
	FSType   string                 `json:"fs_type"`   // 文件系统类型
	ReadOnly bool                   `json:"read_only"` // 是否只读
}

// PVNodeAffinityEntity PV节点亲和性实体
type PVNodeAffinityEntity struct {
	Required PVNodeSelectorEntity `json:"required"` // 必须满足的节点选择器
}

// PVNodeSelectorEntity PV节点选择器实体
type PVNodeSelectorEntity struct {
	NodeSelectorTerms []PVNodeSelectorTermEntity `json:"node_selector_terms"` // 节点选择器条件
}

// PVNodeSelectorTermEntity PV节点选择器条件实体
type PVNodeSelectorTermEntity struct {
	MatchExpressions []PVNodeSelectorRequirementEntity `json:"match_expressions"` // 匹配表达式
	MatchFields      []PVNodeSelectorRequirementEntity `json:"match_fields"`      // 匹配字段
}

// PVNodeSelectorRequirementEntity PV节点选择器要求实体
type PVNodeSelectorRequirementEntity struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符
	Values   []string `json:"values"`   // 值列表
}

// PVListResponse PV列表响应
type PVListResponse struct {
	Items      []PVEntity `json:"items"`       // PV列表
	TotalCount int        `json:"total_count"` // 总数
}

// PVDetailResponse PV详情响应
type PVDetailResponse struct {
	PV      PVEntity        `json:"pv"`      // PV信息
	YAML    string          `json:"yaml"`    // YAML内容
	Events  []PVEventEntity `json:"events"`  // 事件列表
	Usage   PVUsageEntity   `json:"usage"`   // 使用情况
	Metrics PVMetricsEntity `json:"metrics"` // 指标信息
}

// PVEventEntity PV事件实体
type PVEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// PVUsageEntity PV使用情况实体
type PVUsageEntity struct {
	IsBound        bool             `json:"is_bound"`        // 是否已绑定
	ClaimName      string           `json:"claim_name"`      // PVC名称
	ClaimNamespace string           `json:"claim_namespace"` // PVC命名空间
	UsedBy         []PVUsedByEntity `json:"used_by"`         // 使用者列表
	AccessPatterns []string         `json:"access_patterns"` // 访问模式
}

// PVUsedByEntity PV使用者实体
type PVUsedByEntity struct {
	Kind      string `json:"kind"`       // 资源类型
	Name      string `json:"name"`       // 资源名称
	Namespace string `json:"namespace"`  // 命名空间
	PodName   string `json:"pod_name"`   // Pod名称
	MountPath string `json:"mount_path"` // 挂载路径
}

// PVMetricsEntity PV指标实体
type PVMetricsEntity struct {
	CapacityBytes     int64   `json:"capacity_bytes"`      // 容量(字节)
	UsedBytes         int64   `json:"used_bytes"`          // 已使用(字节)
	AvailableBytes    int64   `json:"available_bytes"`     // 可用(字节)
	UsagePercentage   float64 `json:"usage_percentage"`    // 使用率
	InodeCapacity     int64   `json:"inode_capacity"`      // inode容量
	InodeUsed         int64   `json:"inode_used"`          // 已使用inode
	InodeUsagePercent float64 `json:"inode_usage_percent"` // inode使用率
	IOPSRead          float64 `json:"iops_read"`           // 读IOPS
	IOPSWrite         float64 `json:"iops_write"`          // 写IOPS
	ThroughputRead    float64 `json:"throughput_read"`     // 读吞吐量
	ThroughputWrite   float64 `json:"throughput_write"`    // 写吞吐量
}

// PVExpandResponse PV扩容响应
type PVExpandResponse struct {
	Name        string `json:"name"`         // PV名称
	OldCapacity string `json:"old_capacity"` // 原容量
	NewCapacity string `json:"new_capacity"` // 新容量
	Status      string `json:"status"`       // 扩容状态
	Message     string `json:"message"`      // 扩容消息
	StartTime   string `json:"start_time"`   // 开始时间
	EndTime     string `json:"end_time"`     // 结束时间
}

// PVBackupResponse PV备份响应
type PVBackupResponse struct {
	BackupName string   `json:"backup_name"` // 备份名称
	ClusterID  int      `json:"cluster_id"`  // 集群ID
	PVNames    []string `json:"pv_names"`    // PV名称列表
	BackupPath string   `json:"backup_path"` // 备份路径
	Size       string   `json:"size"`        // 备份大小
	Status     string   `json:"status"`      // 备份状态
	Message    string   `json:"message"`     // 备份消息
	CreatedAt  string   `json:"created_at"`  // 创建时间
}
