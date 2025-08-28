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

// ====================== ServiceAccount Response结构体 ======================

// K8sServiceAccountResponse ServiceAccount响应结构
type K8sServiceAccountResponse struct {
	Name                         string                 `json:"name"`                            // ServiceAccount名称
	UID                          string                 `json:"uid"`                             // UID
	Namespace                    string                 `json:"namespace"`                       // 命名空间
	ClusterID                    int                    `json:"cluster_id"`                      // 集群ID
	Labels                       map[string]string      `json:"labels"`                          // 标签
	Annotations                  map[string]string      `json:"annotations"`                     // 注解
	CreationTimestamp            time.Time              `json:"creation_timestamp"`              // 创建时间
	Age                          string                 `json:"age"`                             // 存在时间
	SecretsCount                 int                    `json:"secrets_count"`                   // Secrets数量
	ImagePullSecretsCount        int                    `json:"image_pull_secrets_count"`        // ImagePullSecrets数量
	AutomountServiceAccountToken *bool                  `json:"automount_service_account_token"` // 是否自动挂载ServiceAccount Token
	Secrets                      []ServiceAccountSecret `json:"secrets,omitempty"`               // Secrets列表
	ImagePullSecrets             []ServiceAccountSecret `json:"image_pull_secrets,omitempty"`    // ImagePullSecrets列表
	Token                        string                 `json:"token,omitempty"`                 // Token（详情页时返回）
	CACert                       string                 `json:"ca_cert,omitempty"`               // CA证书（详情页时返回）
}

// ServiceAccountSecret ServiceAccount关联的Secret信息
type ServiceAccountSecret struct {
	Name      string `json:"name"`      // Secret名称
	Namespace string `json:"namespace"` // 命名空间
	Type      string `json:"type"`      // Secret类型
}

// ====================== ServiceAccount请求结构体 ======================

// ServiceAccountListReq ServiceAccount列表查询请求
type ServiceAccountListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// ServiceAccountCreateReq ServiceAccount创建请求
type ServiceAccountCreateReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`                         // 集群ID，必填
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`                          // 命名空间，必填
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`                   // ServiceAccount名称，必填
	Labels                       map[string]string `json:"labels" comment:"标签"`                                                  // 标签
	Annotations                  map[string]string `json:"annotations" comment:"注解"`                                             // 注解
	AutomountServiceAccountToken *bool             `json:"automount_service_account_token" comment:"是否自动挂载ServiceAccount Token"` // 是否自动挂载ServiceAccount Token
	ImagePullSecrets             []string          `json:"image_pull_secrets" comment:"ImagePullSecrets列表"`                      // ImagePullSecrets列表
}

// ServiceAccountUpdateReq ServiceAccount更新请求
type ServiceAccountUpdateReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`                         // 集群ID，必填
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`                          // 命名空间，必填
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`                   // ServiceAccount名称，必填
	Labels                       map[string]string `json:"labels" comment:"标签"`                                                  // 标签
	Annotations                  map[string]string `json:"annotations" comment:"注解"`                                             // 注解
	AutomountServiceAccountToken *bool             `json:"automount_service_account_token" comment:"是否自动挂载ServiceAccount Token"` // 是否自动挂载ServiceAccount Token
	ImagePullSecrets             []string          `json:"image_pull_secrets" comment:"ImagePullSecrets列表"`                      // ImagePullSecrets列表
}

// ServiceAccountDeleteReq ServiceAccount删除请求
type ServiceAccountDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"ServiceAccount名称"` // ServiceAccount名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间"`              // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                             // 是否强制删除
}

// ServiceAccountBatchDeleteReq ServiceAccount批量删除请求
type ServiceAccountBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`          // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`           // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"ServiceAccount名称列表"` // ServiceAccount名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间"`                 // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                                // 是否强制删除
}

// ServiceAccountStatisticsReq ServiceAccount统计信息请求
type ServiceAccountStatisticsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间，可选
}

// ServiceAccountStatisticsResp ServiceAccount统计信息响应
type ServiceAccountStatisticsResp struct {
	TotalCount                int `json:"total_count"`                   // 总数量
	ActiveCount               int `json:"active_count"`                  // 活跃数量
	WithSecretsCount          int `json:"with_secrets_count"`            // 含有Secrets的数量
	WithImagePullSecretsCount int `json:"with_image_pull_secrets_count"` // 含有ImagePullSecrets的数量
	AutoMountEnabledCount     int `json:"auto_mount_enabled_count"`      // 启用自动挂载的数量
}

// ServiceAccountTokenReq ServiceAccount Token请求
type ServiceAccountTokenReq struct {
	ClusterID         int    `json:"cluster_id" binding:"required" comment:"集群ID"`          // 集群ID，必填
	Namespace         string `json:"namespace" binding:"required" comment:"命名空间"`           // 命名空间，必填
	Name              string `json:"name" binding:"required" comment:"ServiceAccount名称"`    // ServiceAccount名称，必填
	ExpirationSeconds *int64 `json:"expiration_seconds" comment:"Token过期时间（秒），不设置则使用系统默认值"` // Token过期时间（秒）
}

// ServiceAccountTokenResp ServiceAccount Token响应
type ServiceAccountTokenResp struct {
	Token               string     `json:"token"`                          // Token
	ExpirationTimestamp *time.Time `json:"expiration_timestamp,omitempty"` // 过期时间
}

// ServiceAccountYamlReq ServiceAccount YAML请求
type ServiceAccountYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"` // ServiceAccount名称，必填
}

// ServiceAccountYamlResp ServiceAccount YAML响应
type ServiceAccountYamlResp struct {
	YAML string `json:"yaml"` // YAML内容
}

// ServiceAccountUpdateYamlReq ServiceAccount YAML更新请求
type ServiceAccountUpdateYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`       // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`        // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"` // ServiceAccount名称，必填
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`           // YAML内容，必填
}

// ====================== ServiceAccount内部转换用的结构体 ======================
