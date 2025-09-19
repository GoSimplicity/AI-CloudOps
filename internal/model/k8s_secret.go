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

// K8sSecretType Secret类型枚举
type K8sSecretType string

const (
	K8sSecretTypeOpaque              K8sSecretType = "Opaque"                              // 通用Secret
	K8sSecretTypeServiceAccountToken K8sSecretType = "kubernetes.io/service-account-token" // ServiceAccount令牌
	K8sSecretTypeDockercfg           K8sSecretType = "kubernetes.io/dockercfg"             // Docker配置
	K8sSecretTypeDockerConfigJson    K8sSecretType = "kubernetes.io/dockerconfigjson"      // Docker配置JSON
	K8sSecretTypeBasicAuth           K8sSecretType = "kubernetes.io/basic-auth"            // 基础认证
	K8sSecretTypeSSHAuth             K8sSecretType = "kubernetes.io/ssh-auth"              // SSH认证
	K8sSecretTypeTLS                 K8sSecretType = "kubernetes.io/tls"                   // TLS证书
	K8sSecretTypeBootstrapToken      K8sSecretType = "bootstrap.kubernetes.io/token"       // 引导令牌
)

// K8sSecret Kubernetes Secret
type K8sSecret struct {
	Name        string            `json:"name" binding:"required,min=1,max=200"`      // Secret名称
	Namespace   string            `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID   int               `json:"cluster_id" gorm:"index;not null"`           // 所属集群ID
	UID         string            `json:"uid" gorm:"size:100"`                        // Secret UID
	Type        K8sSecretType     `json:"type"`                                       // Secret类型
	Data        map[string][]byte `json:"data"`                                       // 加密数据
	StringData  map[string]string `json:"string_data"`                                // 明文数据
	Labels      map[string]string `json:"labels"`                                     // 标签
	Annotations map[string]string `json:"annotations"`                                // 注解
	Immutable   bool              `json:"immutable"`                                  // 是否不可变
	DataCount   int               `json:"data_count"`                                 // 数据条目数量
	Size        string            `json:"size"`                                       // 数据大小
	Age         string            `json:"age"`                                        // 存在时间
	CreatedAt   time.Time         `json:"created_at"`                                 // 创建时间
	UpdatedAt   time.Time         `json:"updated_at"`                                 // 更新时间
	RawSecret   *corev1.Secret    `json:"-"`                                          // 原始 Secret 对象，不序列化到 JSON
}

// GetSecretListReq 获取Secret列表请求
type GetSecretListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id" comment:"集群ID"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace" comment:"命名空间"`   // 命名空间
	Type      K8sSecretType     `json:"type" form:"type" comment:"Secret类型"`         // Secret类型
	Labels    map[string]string `json:"labels" form:"labels" comment:"标签"`           // 标签
}

// GetSecretDetailsReq 获取Secret详情请求
type GetSecretDetailsReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"Secret名称"`         // Secret名称
}

// GetSecretYamlReq 获取Secret YAML请求
type GetSecretYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"Secret名称"`         // Secret名称
}

// CreateSecretReq 创建Secret请求
type CreateSecretReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name        string            `json:"name" form:"name" binding:"required" comment:"Secret名称"`         // Secret名称
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Type        K8sSecretType     `json:"type" comment:"Secret类型"`                                        // Secret类型
	Data        map[string][]byte `json:"data" comment:"加密数据"`                                            // 加密数据
	StringData  map[string]string `json:"string_data" comment:"明文数据"`                                     // 明文数据
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
	Immutable   bool              `json:"immutable" comment:"是否不可变"`                                      // 是否不可变
}

// UpdateSecretReq 更新Secret请求
type UpdateSecretReq struct {
	ClusterID   int               `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Name        string            `json:"name" form:"name" binding:"required" comment:"Secret名称"`         // Secret名称
	Namespace   string            `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Data        map[string][]byte `json:"data" comment:"加密数据"`                                            // 加密数据
	StringData  map[string]string `json:"string_data" comment:"明文数据"`                                     // 明文数据
	Labels      map[string]string `json:"labels" comment:"标签"`                                            // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                                       // 注解
}

// CreateSecretByYamlReq 通过YAML创建Secret请求
type CreateSecretByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// UpdateSecretByYamlReq 通过YAML更新Secret请求
type UpdateSecretByYamlReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`                    // 命名空间
	Name      string `json:"name" form:"name" binding:"required" comment:"Secret名称"`         // Secret名称
	YAML      string `json:"yaml" binding:"required" comment:"YAML内容"`                       // YAML内容
}

// DeleteSecretReq 删除Secret请求
type DeleteSecretReq struct {
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID
	Namespace string `json:"namespace" form:"namespace" binding:"required" comment:"命名空间"`   // 命名空间
	Name      string `json:"name" binding:"required" comment:"Secret名称"`                     // Secret名称
}
