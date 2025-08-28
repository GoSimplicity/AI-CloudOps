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

// K8sPVCEntity Kubernetes PersistentVolumeClaim数据库实体
type K8sPVCEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:PVC名称"`       // PVC名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:PVC UID"`                                       // PVC UID
	Capacity          string            `json:"capacity" gorm:"size:50;comment:存储容量"`                                      // 存储容量
	RequestStorage    string            `json:"request_storage" gorm:"size:50;comment:请求存储"`                               // 请求存储
	AccessModes       []string          `json:"access_modes" gorm:"type:text;serializer:json;comment:访问模式"`                // 访问模式
	StorageClass      string            `json:"storage_class" gorm:"size:200;comment:存储类"`                                 // 存储类
	VolumeMode        string            `json:"volume_mode" gorm:"size:50;comment:卷模式"`                                    // 卷模式
	Status            string            `json:"status" gorm:"size:50;comment:PVC状态"`                                       // PVC状态
	VolumeName        string            `json:"volume_name" gorm:"size:200;comment:绑定的PV名称"`                               // 绑定的PV名称
	Selector          map[string]string `json:"selector" gorm:"type:text;serializer:json;comment:选择器"`                     // 选择器
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	UsedCapacity      string            `json:"used_capacity" gorm:"-"`                                                    // 已使用容量，前端计算使用
	AvailableCapacity string            `json:"available_capacity" gorm:"-"`                                               // 可用容量，前端计算使用
}

func (k *K8sPVCEntity) TableName() string {
	return "cl_k8s_pvcs"
}

// K8sPVCListRequest PVC列表查询请求
type K8sPVCListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Status        string `json:"status" form:"status" comment:"PVC状态过滤"`                         // PVC状态过滤
	StorageClass  string `json:"storage_class" form:"storage_class" comment:"存储类过滤"`             // 存储类过滤
	AccessMode    string `json:"access_mode" form:"access_mode" comment:"访问模式过滤"`                // 访问模式过滤
	VolumeName    string `json:"volume_name" form:"volume_name" comment:"PV名称过滤"`                // PV名称过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sPVCCreateRequest 创建PVC请求
type K8sPVCCreateReq struct {
	ClusterID      int                           `json:"cluster_id" binding:"required" comment:"集群ID"`      // 集群ID，必填
	Namespace      string                        `json:"namespace" binding:"required" comment:"命名空间"`       // 命名空间，必填
	Name           string                        `json:"name" binding:"required" comment:"PVC名称"`           // PVC名称，必填
	RequestStorage string                        `json:"request_storage" binding:"required" comment:"请求存储"` // 请求存储，必填
	AccessModes    []string                      `json:"access_modes" binding:"required" comment:"访问模式"`    // 访问模式，必填
	StorageClass   string                        `json:"storage_class" comment:"存储类"`                       // 存储类
	VolumeMode     string                        `json:"volume_mode" comment:"卷模式"`                         // 卷模式
	VolumeName     string                        `json:"volume_name" comment:"指定PV名称"`                      // 指定PV名称
	Selector       map[string]string             `json:"selector" comment:"选择器"`                            // 选择器
	Labels         map[string]string             `json:"labels" comment:"标签"`                               // 标签
	Annotations    map[string]string             `json:"annotations" comment:"注解"`                          // 注解
	PVCYaml        *corev1.PersistentVolumeClaim `json:"pvc_yaml" comment:"PVC YAML对象"`                     // PVC YAML对象
}

// K8sPVCUpdateRequest 更新PVC请求
type K8sPVCUpdateReq struct {
	ClusterID      int                           `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace      string                        `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name           string                        `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	RequestStorage string                        `json:"request_storage" comment:"请求存储"`               // 请求存储
	AccessModes    []string                      `json:"access_modes" comment:"访问模式"`                  // 访问模式
	StorageClass   string                        `json:"storage_class" comment:"存储类"`                  // 存储类
	VolumeMode     string                        `json:"volume_mode" comment:"卷模式"`                    // 卷模式
	VolumeName     string                        `json:"volume_name" comment:"指定PV名称"`                 // 指定PV名称
	Selector       map[string]string             `json:"selector" comment:"选择器"`                       // 选择器
	Labels         map[string]string             `json:"labels" comment:"标签"`                          // 标签
	Annotations    map[string]string             `json:"annotations" comment:"注解"`                     // 注解
	PVCYaml        *corev1.PersistentVolumeClaim `json:"pvc_yaml" comment:"PVC YAML对象"`                // PVC YAML对象
}

// K8sPVCDeleteRequest 删除PVC请求
type K8sPVCDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPVCBatchDeleteRequest 批量删除PVC请求
type K8sPVCBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"PVC名称列表"`   // PVC名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sPVCEventRequest 获取PVC事件请求
type K8sPVCEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sPVCUsageRequest 获取PVC使用情况请求
type K8sPVCUsageReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"PVC名称"`      // PVC名称，必填
}

// K8sPVCExpandRequest 扩容PVC请求
type K8sPVCExpandReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Name        string `json:"name" binding:"required" comment:"PVC名称"`       // PVC名称，必填
	NewCapacity string `json:"new_capacity" binding:"required" comment:"新容量"` // 新容量，必填
}

// K8sPVCBackupRequest 备份PVC请求
type K8sPVCBackupReq struct {
	ClusterID   int      `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace   string   `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Names       []string `json:"names" binding:"required" comment:"PVC名称列表"`    // PVC名称列表，必填
	BackupName  string   `json:"backup_name" binding:"required" comment:"备份名称"` // 备份名称，必填
	Description string   `json:"description" comment:"备份描述"`                    // 备份描述
}

// K8sPVCCloneRequest 克隆PVC请求
type K8sPVCCloneReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`     // 集群ID，必填
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`      // 命名空间，必填
	SourceName  string `json:"source_name" binding:"required" comment:"源PVC名称"`  // 源PVC名称，必填
	TargetName  string `json:"target_name" binding:"required" comment:"目标PVC名称"` // 目标PVC名称，必填
	Description string `json:"description" comment:"克隆描述"`                       // 克隆描述
}

// K8sPVCSnapshotRequest 创建PVC快照请求
type K8sPVCSnapshotReq struct {
	ClusterID    int    `json:"cluster_id" binding:"required" comment:"集群ID"`    // 集群ID，必填
	Namespace    string `json:"namespace" binding:"required" comment:"命名空间"`     // 命名空间，必填
	PVCName      string `json:"pvc_name" binding:"required" comment:"PVC名称"`     // PVC名称，必填
	SnapshotName string `json:"snapshot_name" binding:"required" comment:"快照名称"` // 快照名称，必填
	Description  string `json:"description" comment:"快照描述"`                      // 快照描述
}

// ====================== PVC响应实体 ======================

// PVCEntity PVC响应实体
type PVCEntity struct {
	Name              string            `json:"name"`               // PVC名称
	Namespace         string            `json:"namespace"`          // 命名空间
	UID               string            `json:"uid"`                // PVC UID
	Labels            map[string]string `json:"labels"`             // 标签
	Annotations       map[string]string `json:"annotations"`        // 注解
	Capacity          string            `json:"capacity"`           // 存储容量
	RequestStorage    string            `json:"request_storage"`    // 请求存储
	AccessModes       []string          `json:"access_modes"`       // 访问模式
	StorageClass      string            `json:"storage_class"`      // 存储类
	VolumeMode        string            `json:"volume_mode"`        // 卷模式
	Status            string            `json:"status"`             // PVC状态
	VolumeName        string            `json:"volume_name"`        // 绑定的PV名称
	Selector          PVCSelectorEntity `json:"selector"`           // 选择器
	UsedCapacity      string            `json:"used_capacity"`      // 已使用容量
	AvailableCapacity string            `json:"available_capacity"` // 可用容量
	Age               string            `json:"age"`                // 存在时间
	CreatedAt         string            `json:"created_at"`         // 创建时间
}

// PVCSelectorEntity PVC选择器实体
type PVCSelectorEntity struct {
	MatchLabels      map[string]string              `json:"match_labels"`      // 标签匹配
	MatchExpressions []PVCSelectorRequirementEntity `json:"match_expressions"` // 表达式匹配
}

// PVCSelectorRequirementEntity PVC选择器要求实体
type PVCSelectorRequirementEntity struct {
	Key      string   `json:"key"`      // 键
	Operator string   `json:"operator"` // 操作符
	Values   []string `json:"values"`   // 值列表
}

// PVCListResponse PVC列表响应
type PVCListResponse struct {
	Items      []PVCEntity `json:"items"`       // PVC列表
	TotalCount int         `json:"total_count"` // 总数
}

// PVCDetailResponse PVC详情响应
type PVCDetailResponse struct {
	PVC     PVCEntity        `json:"pvc"`     // PVC信息
	YAML    string           `json:"yaml"`    // YAML内容
	Events  []PVCEventEntity `json:"events"`  // 事件列表
	Usage   PVCUsageEntity   `json:"usage"`   // 使用情况
	Metrics PVCMetricsEntity `json:"metrics"` // 指标信息
	PVInfo  PVEntity         `json:"pv_info"` // 绑定的PV信息
}

// PVCEventEntity PVC事件实体
type PVCEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// PVCUsageEntity PVC使用情况实体
type PVCUsageEntity struct {
	IsBound        bool                  `json:"is_bound"`        // 是否已绑定
	PVName         string                `json:"pv_name"`         // PV名称
	UsedBy         []PVCUsedByEntity     `json:"used_by"`         // 使用者列表
	AccessPatterns []string              `json:"access_patterns"` // 访问模式
	MountPoints    []PVCMountPointEntity `json:"mount_points"`    // 挂载点列表
}

// PVCUsedByEntity PVC使用者实体
type PVCUsedByEntity struct {
	Kind      string `json:"kind"`       // 资源类型
	Name      string `json:"name"`       // 资源名称
	Namespace string `json:"namespace"`  // 命名空间
	PodName   string `json:"pod_name"`   // Pod名称
	MountPath string `json:"mount_path"` // 挂载路径
	ReadOnly  bool   `json:"read_only"`  // 是否只读
}

// PVCMountPointEntity PVC挂载点实体
type PVCMountPointEntity struct {
	PodName       string `json:"pod_name"`       // Pod名称
	ContainerName string `json:"container_name"` // 容器名称
	MountPath     string `json:"mount_path"`     // 挂载路径
	SubPath       string `json:"sub_path"`       // 子路径
	ReadOnly      bool   `json:"read_only"`      // 是否只读
}

// PVCMetricsEntity PVC指标实体
type PVCMetricsEntity struct {
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

// PVCExpandResponse PVC扩容响应
type PVCExpandResponse struct {
	Name        string `json:"name"`         // PVC名称
	Namespace   string `json:"namespace"`    // 命名空间
	OldCapacity string `json:"old_capacity"` // 原容量
	NewCapacity string `json:"new_capacity"` // 新容量
	Status      string `json:"status"`       // 扩容状态
	Message     string `json:"message"`      // 扩容消息
	StartTime   string `json:"start_time"`   // 开始时间
	EndTime     string `json:"end_time"`     // 结束时间
}

// PVCBackupResponse PVC备份响应
type PVCBackupResponse struct {
	BackupName string   `json:"backup_name"` // 备份名称
	ClusterID  int      `json:"cluster_id"`  // 集群ID
	Namespace  string   `json:"namespace"`   // 命名空间
	PVCNames   []string `json:"pvc_names"`   // PVC名称列表
	BackupPath string   `json:"backup_path"` // 备份路径
	Size       string   `json:"size"`        // 备份大小
	Status     string   `json:"status"`      // 备份状态
	Message    string   `json:"message"`     // 备份消息
	CreatedAt  string   `json:"created_at"`  // 创建时间
}

// PVCCloneResponse PVC克隆响应
type PVCCloneResponse struct {
	SourceName string `json:"source_name"` // 源PVC名称
	TargetName string `json:"target_name"` // 目标PVC名称
	Namespace  string `json:"namespace"`   // 命名空间
	ClusterID  int    `json:"cluster_id"`  // 集群ID
	Status     string `json:"status"`      // 克隆状态
	Message    string `json:"message"`     // 克隆消息
	Progress   int    `json:"progress"`    // 克隆进度(百分比)
	StartTime  string `json:"start_time"`  // 开始时间
	EndTime    string `json:"end_time"`    // 结束时间
	Size       string `json:"size"`        // 克隆数据大小
}

// PVCSnapshotResponse PVC快照响应
type PVCSnapshotResponse struct {
	SnapshotName string `json:"snapshot_name"` // 快照名称
	PVCName      string `json:"pvc_name"`      // PVC名称
	Namespace    string `json:"namespace"`     // 命名空间
	ClusterID    int    `json:"cluster_id"`    // 集群ID
	Status       string `json:"status"`        // 快照状态
	Message      string `json:"message"`       // 快照消息
	Size         string `json:"size"`          // 快照大小
	CreatedAt    string `json:"created_at"`    // 创建时间
	ReadyToUse   bool   `json:"ready_to_use"`  // 是否可用
}

// PVCStatisticsResponse PVC统计响应
type PVCStatisticsResponse struct {
	TotalPVCs         int                         `json:"total_pvcs"`         // 总PVC数
	BoundPVCs         int                         `json:"bound_pvcs"`         // 已绑定PVC数
	PendingPVCs       int                         `json:"pending_pvcs"`       // 待绑定PVC数
	LostPVCs          int                         `json:"lost_pvcs"`          // 丢失PVC数
	TotalCapacity     string                      `json:"total_capacity"`     // 总容量
	UsedCapacity      string                      `json:"used_capacity"`      // 已使用容量
	AvailableCapacity string                      `json:"available_capacity"` // 可用容量
	UsagePercentage   float64                     `json:"usage_percentage"`   // 使用率
	ByStorageClass    []PVCStorageClassStatEntity `json:"by_storage_class"`   // 按存储类统计
	ByNamespace       []PVCNamespaceStatEntity    `json:"by_namespace"`       // 按命名空间统计
	ByStatus          []PVCStatusStatEntity       `json:"by_status"`          // 按状态统计
}

// PVCStorageClassStatEntity PVC存储类统计实体
type PVCStorageClassStatEntity struct {
	StorageClass string `json:"storage_class"` // 存储类
	Count        int    `json:"count"`         // 数量
	Capacity     string `json:"capacity"`      // 容量
}

// PVCNamespaceStatEntity PVC命名空间统计实体
type PVCNamespaceStatEntity struct {
	Namespace string `json:"namespace"` // 命名空间
	Count     int    `json:"count"`     // 数量
	Capacity  string `json:"capacity"`  // 容量
}

// PVCStatusStatEntity PVC状态统计实体
type PVCStatusStatEntity struct {
	Status string `json:"status"` // 状态
	Count  int    `json:"count"`  // 数量
}
