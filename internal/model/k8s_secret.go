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

// K8sSecretEntity Kubernetes Secret数据库实体
type K8sSecretEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:Secret名称"`    // Secret名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:Secret UID"`                                    // Secret UID
	Type              string            `json:"type" gorm:"size:100;comment:Secret类型"`                                     // Secret类型
	Data              map[string][]byte `json:"data" gorm:"type:text;serializer:json;comment:加密数据"`                        // 加密数据
	StringData        map[string]string `json:"string_data" gorm:"type:text;serializer:json;comment:明文数据"`                 // 明文数据
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	DataCount         int               `json:"data_count" gorm:"-"`                                                       // 数据条目数量，前端计算使用
	Size              string            `json:"size" gorm:"-"`                                                             // 数据大小，前端计算使用
}

func (k *K8sSecretEntity) TableName() string {
	return "cl_k8s_secrets"
}

// K8sSecretListRequest Secret列表查询请求
type K8sSecretListRequest struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Type          string `json:"type" form:"type" comment:"Secret类型过滤"`                          // Secret类型过滤
	DataKey       string `json:"data_key" form:"data_key" comment:"数据键过滤"`                       // 数据键过滤
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sSecretCreateRequest 创建Secret请求
type K8sSecretCreateRequest struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
	Type        string            `json:"type" comment:"Secret类型"`                      // Secret类型
	Data        map[string][]byte `json:"data" comment:"加密数据"`                          // 加密数据
	StringData  map[string]string `json:"string_data" comment:"明文数据"`                   // 明文数据
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
	SecretYaml  *corev1.Secret    `json:"secret_yaml" comment:"Secret YAML对象"`          // Secret YAML对象
}

// K8sSecretUpdateRequest 更新Secret请求
type K8sSecretUpdateRequest struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
	Data        map[string][]byte `json:"data" comment:"加密数据"`                          // 加密数据
	StringData  map[string]string `json:"string_data" comment:"明文数据"`                   // 明文数据
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
	SecretYaml  *corev1.Secret    `json:"secret_yaml" comment:"Secret YAML对象"`          // Secret YAML对象
}

// K8sSecretDeleteRequest 删除Secret请求
type K8sSecretDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sSecretBatchDeleteRequest 批量删除Secret请求
type K8sSecretBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"Secret名称列表"` // Secret名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`      // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                        // 是否强制删除
}

// K8sSecretDataRequest 获取Secret数据请求
type K8sSecretDataRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
	Key       string `json:"key" comment:"数据键，为空则获取所有"`                    // 数据键，为空则获取所有
	Decode    bool   `json:"decode" comment:"是否解码数据"`                      // 是否解码数据
}

// K8sSecretEventRequest 获取Secret事件请求
type K8sSecretEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sSecretUsageRequest 获取Secret使用情况请求
type K8sSecretUsageRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Secret名称"`   // Secret名称，必填
}

// K8sSecretBackupRequest 备份Secret请求
type K8sSecretBackupRequest struct {
	ClusterID   int      `json:"cluster_id" binding:"required" comment:"集群ID"`  // 集群ID，必填
	Namespace   string   `json:"namespace" binding:"required" comment:"命名空间"`   // 命名空间，必填
	Names       []string `json:"names" binding:"required" comment:"Secret名称列表"` // Secret名称列表，必填
	BackupName  string   `json:"backup_name" binding:"required" comment:"备份名称"` // 备份名称，必填
	Description string   `json:"description" comment:"备份描述"`                    // 备份描述
}
