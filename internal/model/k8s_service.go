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
	"k8s.io/apimachinery/pkg/util/intstr"
)

// K8sSvcStatus Service状态枚举
type K8sSvcStatus int8

const (
	K8sSvcStatusRunning K8sSvcStatus = iota + 1 // 运行中
	K8sSvcStatusStopped                         // 停止
	K8sSvcStatusError                           // 异常
)

// K8sService k8s service
type K8sService struct {
	Name           string               `json:"name" binding:"required,min=1,max=200"`      // Service名称
	Namespace      string               `json:"namespace" binding:"required,min=1,max=200"` // 所属命名空间
	ClusterID      int                  `json:"cluster_id"`                                 // 所属集群ID
	UID            string               `json:"uid"`                                        // Service UID
	Type           string               `json:"type"`                                       // Service类型
	ClusterIP      string               `json:"cluster_ip"`                                 // 集群内部IP
	ExternalIPs    []string             `json:"external_ips"`                               // 外部IP列表
	LoadBalancerIP string               `json:"load_balancer_ip"`                           // 负载均衡器IP
	Ports          []ServicePort        `json:"ports"`                                      // 端口配置
	Selector       map[string]string    `json:"selector"`                                   // Pod选择器
	Labels         map[string]string    `json:"labels"`                                     // 标签
	Annotations    map[string]string    `json:"annotations"`                                // 注解
	CreatedAt      time.Time            `json:"created_at"`                                 // 创建时间
	Age            string               `json:"age"`                                        // 存在时间，前端计算使用
	Status         K8sSvcStatus         `json:"status" binding:"required"`                  // Service状态，前端计算使用
	Endpoints      []K8sServiceEndpoint `json:"endpoints"`                                  // 服务端点，前端使用
}

// ServicePort 服务端口配置
type ServicePort struct {
	Name        string             `json:"name"`                   // 端口名称
	Protocol    corev1.Protocol    `json:"protocol"`               // 协议类型
	Port        int32              `json:"port"`                   // 服务端口
	TargetPort  intstr.IntOrString `json:"target_port"`            // 目标端口
	NodePort    int32              `json:"node_port,omitempty"`    // 节点端口（NodePort类型）
	AppProtocol *string            `json:"app_protocol,omitempty"` // 应用协议
}

// K8sServiceEndpoint k8s service端点信息
type K8sServiceEndpoint struct {
	IP       string `json:"ip"`       // 端点IP
	Port     int32  `json:"port"`     // 端点端口
	Protocol string `json:"protocol"` // 端口协议
	Ready    bool   `json:"ready"`    // 端点是否就绪
}

// ServiceEndpoint 服务端点详细信息
type ServiceEndpoint struct {
	Addresses  []string            `json:"addresses"`   // 端点地址列表
	Ports      []EndpointPort      `json:"ports"`       // 端点端口列表
	Ready      bool                `json:"ready"`       // 是否就绪
	Conditions []EndpointCondition `json:"conditions"`  // 端点条件
	TargetRef  *EndpointTargetRef  `json:"target_ref"`  // 目标引用
	Topology   map[string]string   `json:"topology"`    // 拓扑信息
	LastChange time.Time           `json:"last_change"` // 最后变更时间
}

// EndpointPort 端点端口信息
type EndpointPort struct {
	Name        string          `json:"name"`         // 端口名称
	Port        int32           `json:"port"`         // 端口号
	Protocol    corev1.Protocol `json:"protocol"`     // 协议
	AppProtocol *string         `json:"app_protocol"` // 应用协议
}

// EndpointCondition 端点条件
type EndpointCondition struct {
	Type               string    `json:"type"`                 // 条件类型
	Status             string    `json:"status"`               // 条件状态
	LastTransitionTime time.Time `json:"last_transition_time"` // 最后转换时间
	Reason             string    `json:"reason"`               // 原因
	Message            string    `json:"message"`              // 消息
}

// EndpointTargetRef 端点目标引用
type EndpointTargetRef struct {
	Kind            string `json:"kind"`             // 资源类型
	Namespace       string `json:"namespace"`        // 命名空间
	Name            string `json:"name"`             // 资源名称
	UID             string `json:"uid"`              // 资源UID
	APIVersion      string `json:"api_version"`      // API版本
	ResourceVersion string `json:"resource_version"` // 资源版本
}

type K8sYaml struct {
	YAML string `json:"yaml"`
}

// GetServiceListReq Service列表请求
type GetServiceListReq struct {
	ListReq
	ClusterID int               `json:"cluster_id" form:"cluster_id"` // 集群ID
	Namespace string            `json:"namespace" form:"namespace"`   // 命名空间
	Type      string            `json:"type" form:"type"`             // Service类型
	Labels    map[string]string `json:"labels" form:"labels"`         // 标签
}

// GetServiceDetailsReq 获取Service详情请求
type GetServiceDetailsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Service名称
}

// GetServiceYamlReq 获取Service YAML请求
type GetServiceYamlReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Service名称
}

// CreateServiceReq 创建Service请求
type CreateServiceReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required"` // 集群ID
	Name        string            `json:"name" binding:"required"`       // Service名称
	Namespace   string            `json:"namespace" binding:"required"`  // 命名空间
	Type        string            `json:"type" binding:"required"`       // Service类型
	Ports       []ServicePort     `json:"ports" binding:"required"`      // 端口配置
	Selector    map[string]string `json:"selector"`                      // Pod选择器
	Labels      map[string]string `json:"labels"`                        // 标签
	Annotations map[string]string `json:"annotations"`                   // 注解
	YAML        string            `json:"yaml"`                          // YAML内容
}

// UpdateServiceReq 更新Service请求
type UpdateServiceReq struct {
	ClusterID   int               `json:"cluster_id"`  // 集群ID
	Name        string            `json:"name"`        // Service名称
	Namespace   string            `json:"namespace"`   // 命名空间
	Type        string            `json:"type"`        // Service类型
	Ports       []ServicePort     `json:"ports"`       // 端口配置
	Selector    map[string]string `json:"selector"`    // Pod选择器
	Labels      map[string]string `json:"labels"`      // 标签
	Annotations map[string]string `json:"annotations"` // 注解
	YAML        string            `json:"yaml"`        // YAML内容
}

// DeleteServiceReq 删除Service请求
type DeleteServiceReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Service名称
}

// GetServiceEndpointsReq 获取Service端点请求
type GetServiceEndpointsReq struct {
	ClusterID int    `json:"cluster_id"` // 集群ID
	Namespace string `json:"namespace"`  // 命名空间
	Name      string `json:"name"`       // Service名称
}

type CreateServiceByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
}

type UpdateServiceByYamlReq struct {
	ClusterID int    `json:"cluster_id" binding:"required"` // 集群ID
	YAML      string `json:"yaml" binding:"required"`       // YAML内容
	Namespace string `json:"namespace" binding:"required"`  // 命名空间
	Name      string `json:"name" binding:"required"`       // Service名称
}
