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
	corev1 "k8s.io/api/core/v1"
)

// GetServiceAccountListReq 获取ServiceAccount列表请求
type GetServiceAccountListReq struct {
	ListReq
	ClusterID int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" form:"namespace" comment:"命名空间"`
}

// GetServiceAccountDetailsReq 获取ServiceAccount详情请求
type GetServiceAccountDetailsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"`
}

// GetServiceAccountYamlReq 获取ServiceAccount YAML请求
type GetServiceAccountYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"`
}

// CreateServiceAccountReq 创建ServiceAccount请求
type CreateServiceAccountReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`
	Labels                       map[string]string `json:"labels" comment:"标签"`
	Annotations                  map[string]string `json:"annotations" comment:"注解"`
	AutomountServiceAccountToken *bool             `json:"automount_service_account_token" comment:"是否自动挂载服务账户令牌"`
	ImagePullSecrets             []string          `json:"image_pull_secrets" comment:"镜像拉取密钥列表"`
	Secrets                      []string          `json:"secrets" comment:"关联的Secret列表"`
}

// CreateServiceAccountByYamlReq 通过YAML创建ServiceAccount请求
type CreateServiceAccountByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// UpdateServiceAccountReq 更新ServiceAccount请求
type UpdateServiceAccountReq struct {
	ClusterID                    int               `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace                    string            `json:"namespace" binding:"required" comment:"命名空间"`
	Name                         string            `json:"name" binding:"required" comment:"ServiceAccount名称"`
	Labels                       map[string]string `json:"labels" comment:"标签"`
	Annotations                  map[string]string `json:"annotations" comment:"注解"`
	AutomountServiceAccountToken *bool             `json:"automount_service_account_token" comment:"是否自动挂载服务账户令牌"`
	ImagePullSecrets             []string          `json:"image_pull_secrets" comment:"镜像拉取密钥列表"`
	Secrets                      []string          `json:"secrets" comment:"关联的Secret列表"`
}

// UpdateServiceAccountByYamlReq 通过YAML更新ServiceAccount请求
type UpdateServiceAccountByYamlReq struct {
	ClusterID   int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace   string `json:"namespace" binding:"required" comment:"命名空间"`
	Name        string `json:"name" binding:"required" comment:"ServiceAccount名称"`
	YamlContent string `json:"yaml_content" binding:"required" comment:"YAML内容"`
}

// DeleteServiceAccountReq 删除ServiceAccount请求
type DeleteServiceAccountReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"`
}

// GetServiceAccountTokenReq 获取ServiceAccount Token请求
type GetServiceAccountTokenReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`
	Name      string `json:"name" binding:"required" comment:"ServiceAccount名称"`
}

// CreateServiceAccountTokenReq 创建ServiceAccount Token请求
type CreateServiceAccountTokenReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"`
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`
	ServiceAccountName string `json:"service_account_name" binding:"required" comment:"ServiceAccount名称"`
	ExpirationSeconds  *int64 `json:"expiration_seconds" comment:"令牌过期时间（秒），不传或0为3600，允许600~3600"`
}

// K8sServiceAccount ServiceAccount主model
type K8sServiceAccount struct {
	Name                         string                 `json:"name"`
	Namespace                    string                 `json:"namespace"`
	ClusterID                    int                    `json:"cluster_id"`
	UID                          string                 `json:"uid"`
	CreatedAt                    string                 `json:"created_at"`
	Labels                       map[string]string      `json:"labels"`
	Annotations                  map[string]string      `json:"annotations"`
	AutomountServiceAccountToken *bool                  `json:"automount_service_account_token"`
	ImagePullSecrets             []string               `json:"image_pull_secrets"`
	Secrets                      []string               `json:"secrets"`
	ResourceVersion              string                 `json:"resource_version"`
	Age                          string                 `json:"age"`
	RawServiceAccount            *corev1.ServiceAccount `json:"-"` // 原始ServiceAccount对象，不序列化
}

// ServiceAccountTokenInfo ServiceAccount Token信息响应
type ServiceAccountTokenInfo struct {
	Token             string `json:"token"`
	ExpirationSeconds *int64 `json:"expiration_seconds"`
	CreatedAt         string `json:"created_at"`
	ExpirationTime    string `json:"expiration_time"`
}

// POST /token（TokenRequest）的有效期约束（秒）。不传或传 0 使用默认值；
// 下限为 K8s TokenRequest 允许的最小值 600，上限 3600，越界返回参数错误且不调用 K8s。
const (
	ServiceAccountTokenDefaultExpirationSeconds int64 = 3600
	ServiceAccountTokenMinExpirationSeconds     int64 = 600
	ServiceAccountTokenMaxExpirationSeconds     int64 = 3600
)

// K8sServiceAccountTokenRoute ServiceAccount token 接口的完整路由（权限码对应的 path）。
// GET 读取 SA 名下的 token Secret，POST 通过 TokenRequest 签发新 token；两者都只允许按该路由 + 方法显式授权，
// /api/k8s/*、/* 及带 :param 的通配授权都不生效。
const K8sServiceAccountTokenRoute = "/api/k8s/serviceaccount/:cluster_id/:namespace/:name/token"

const (
	// K8sServiceAccountTokenPermissionName GET token 在接口目录（cl_system_apis.name）中的权限名
	K8sServiceAccountTokenPermissionName = "k8s:serviceaccount:token"
	// K8sServiceAccountTokenCreatePermissionName POST token（TokenRequest）在接口目录中的权限名；
	// cl_system_apis.name 有唯一索引，GET/POST 不能共用同一个名字
	K8sServiceAccountTokenCreatePermissionName = "k8s:serviceaccount:token:create"
)

const (
	ServiceAccountTokenRevealAuditAction    = "k8s_serviceaccount_token_reveal"
	ServiceAccountTokenCreateAuditAction    = "k8s_serviceaccount_token_create"
	ServiceAccountTokenRevealAuditOperation = "TOKEN_REVEAL"
	ServiceAccountTokenCreateAuditOperation = "TOKEN_CREATE"
	ServiceAccountTokenAuditTargetType      = "k8s_serviceaccount_token"
)

// ServiceAccountTokenAuditRecord token 接口审计内容，写入 AuditLog.RequestBody。只含元数据，绝不含 token。
type ServiceAccountTokenAuditRecord struct {
	Action            string `json:"action"`
	UserID            int    `json:"user_id"`
	Username          string `json:"username"`
	ClusterID         int    `json:"cluster_id"`
	Namespace         string `json:"namespace"`
	ServiceAccount    string `json:"service_account"`
	ServiceAccountUID string `json:"service_account_uid,omitempty"`
	SecretName        string `json:"secret_name,omitempty"`        // GET：返回的 token 所在 Secret 名称
	ExpirationSeconds *int64 `json:"expiration_seconds,omitempty"` // POST：生效的有效期（越界被拒时为请求值）
	Success           bool   `json:"success"`
	IssuedAt          string `json:"issued_at"` // RFC3339 UTC
}
