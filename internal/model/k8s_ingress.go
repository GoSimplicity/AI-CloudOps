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

	networkingv1 "k8s.io/api/networking/v1"
)

// K8sIngressStatus Ingress状态枚举
type K8sIngressStatus int8

const (
	K8sIngressStatusRunning K8sIngressStatus = iota + 1 // 运行中
	K8sIngressStatusPending                             // 等待中
	K8sIngressStatusFailed                              // 失败
)

// K8sIngress k8s ingress 实体
type K8sIngress struct {
	Model
	Name             string                `json:"name" binding:"required,min=1,max=200"`      // Ingress名称
	Namespace        string                `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID        int                   `json:"cluster_id"`                                 // 所属集群ID
	UID              string                `json:"uid"`                                        // Ingress UID
	IngressClassName *string               `json:"ingress_class_name"`                         // Ingress类名
	Rules            []IngressRule         `json:"rules"`                                      // Ingress规则
	TLS              []IngressTLS          `json:"tls"`                                        // TLS配置
	LoadBalancer     IngressLoadBalancer   `json:"load_balancer"`                              // 负载均衡器信息
	Labels           map[string]string     `json:"labels"`                                     // 标签
	Annotations      map[string]string     `json:"annotations"`                                // 注解
	CreatedAt        time.Time             `json:"created_at"`                                 // 创建时间
	Age              string                `json:"age"`                                        // 存在时间，前端计算使用
	Status           K8sIngressStatus      `json:"status" binding:"required"`                  // Ingress状态，前端计算使用
	Hosts            []string              `json:"hosts"`                                      // 主机列表，前端使用
	RawIngress       *networkingv1.Ingress `json:"-"`                                          // 原始Ingress对象
}

// IngressRule Ingress规则
type IngressRule struct {
	Host string               `json:"host"` // 主机名
	HTTP IngressHTTPRuleValue `json:"http"` // HTTP规则
}

// IngressHTTPRuleValue HTTP规则值
type IngressHTTPRuleValue struct {
	Paths []IngressHTTPIngressPath `json:"paths"` // 路径列表
}

// IngressHTTPIngressPath HTTP路径
type IngressHTTPIngressPath struct {
	Path     string                      `json:"path"`      // 路径
	PathType *networkingv1.PathType      `json:"path_type"` // 路径类型
	Backend  networkingv1.IngressBackend `json:"backend"`   // 后端服务
}

// IngressTLS TLS配置
type IngressTLS struct {
	Hosts      []string `json:"hosts"`       // 主机列表
	SecretName string   `json:"secret_name"` // Secret名称
}

// IngressLoadBalancer 负载均衡器
type IngressLoadBalancer struct {
	Ingress []IngressLoadBalancerIngress `json:"ingress"` // Ingress信息
}

// IngressLoadBalancerIngress 负载均衡器Ingress
type IngressLoadBalancerIngress struct {
	IP       string              `json:"ip"`       // IP地址
	Hostname string              `json:"hostname"` // 主机名
	Ports    []IngressPortStatus `json:"ports"`    // 端口状态
}

// IngressPortStatus 端口状态
type IngressPortStatus struct {
	Port     int32  `json:"port"`     // 端口号
	Protocol string `json:"protocol"` // 协议
	Error    string `json:"error"`    // 错误信息
}

// GetIngressListReq Ingress列表请求
type GetIngressListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace"`   // 命名空间
	Status    string            `json:"status" form:"status"`         // 状态过滤
	Labels    map[string]string `json:"labels" form:"labels"`         // 标签
}

// GetIngressDetailsReq 获取Ingress详情请求
type GetIngressDetailsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Ingress名称
}

// GetIngressYamlReq 获取Ingress YAML请求
type GetIngressYamlReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Ingress名称
}

// CreateIngressReq 创建Ingress请求
type CreateIngressReq struct {
	ClusterID        int               `json:"cluster_id" binding:"required"` // 集群ID
	Name             string            `json:"name" binding:"required"`       // Ingress名称
	Namespace        string            `json:"namespace" binding:"required"`  // 命名空间
	IngressClassName *string           `json:"ingress_class_name"`            // Ingress类名
	Rules            []IngressRule     `json:"rules"`                         // Ingress规则
	TLS              []IngressTLS      `json:"tls"`                           // TLS配置
	Labels           map[string]string `json:"labels"`                        // 标签
	Annotations      map[string]string `json:"annotations"`                   // 注解
}

// UpdateIngressReq 更新Ingress请求
type UpdateIngressReq struct {
	ClusterID        int               `json:"cluster_id"`         // 集群ID
	Name             string            `json:"name"`               // Ingress名称
	Namespace        string            `json:"namespace"`          // 命名空间
	IngressClassName *string           `json:"ingress_class_name"` // Ingress类名
	Rules            []IngressRule     `json:"rules"`              // Ingress规则
	TLS              []IngressTLS      `json:"tls"`                // TLS配置
	Labels           map[string]string `json:"labels"`             // 标签
	Annotations      map[string]string `json:"annotations"`        // 注解
}

// CreateIngressByYamlReq 通过YAML创建Ingress请求
type CreateIngressByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

// UpdateIngressByYamlReq 通过YAML更新Ingress请求
type UpdateIngressByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
	Name      string `json:"name" binding:"required"`       // Ingress名称
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

// DeleteIngressReq 删除Ingress请求
type DeleteIngressReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Ingress名称
}
