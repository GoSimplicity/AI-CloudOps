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

// K8sServiceEntity Kubernetes Service数据库实体
type K8sServiceEntity struct {
	Model
	Name              string            `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:Service名称"`   // Service名称
	Namespace         string            `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int               `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string            `json:"uid" gorm:"size:100;comment:Service UID"`                                   // Service UID
	Type              string            `json:"type" gorm:"size:50;comment:Service类型"`                                     // Service类型
	ClusterIP         string            `json:"cluster_ip" gorm:"size:50;comment:集群内部IP"`                                  // 集群内部IP
	ExternalIPs       []string          `json:"external_ips" gorm:"type:text;serializer:json;comment:外部IP列表"`              // 外部IP列表
	LoadBalancerIP    string            `json:"load_balancer_ip" gorm:"size:50;comment:负载均衡器IP"`                           // 负载均衡器IP
	Ports             []ServicePort     `json:"ports" gorm:"type:text;serializer:json;comment:端口配置"`                       // 端口配置
	Selector          map[string]string `json:"selector" gorm:"type:text;serializer:json;comment:Pod选择器"`                  // Pod选择器
	Labels            map[string]string `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time         `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string            `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	Status            string            `json:"status" gorm:"-"`                                                           // Service状态，前端计算使用
	Endpoints         []ServiceEndpoint `json:"endpoints" gorm:"-"`                                                        // 服务端点，前端使用
}

func (k *K8sServiceEntity) TableName() string {
	return "cl_k8s_services"
}

// ServiceEndpoint 服务端点信息
type ServiceEndpoint struct {
	IP       string `json:"ip" comment:"端点IP"`       // 端点IP
	Port     int32  `json:"port" comment:"端点端口"`     // 端点端口
	Protocol string `json:"protocol" comment:"端口协议"` // 端口协议
	Ready    bool   `json:"ready" comment:"端点是否就绪"`  // 端点是否就绪
}

// K8sServiceListRequest Service列表查询请求
type K8sServiceListReq struct {
	ClusterID     int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace     string `json:"namespace" form:"namespace" comment:"命名空间"`                      // 命名空间
	LabelSelector string `json:"label_selector" form:"label_selector" comment:"标签选择器"`           // 标签选择器
	FieldSelector string `json:"field_selector" form:"field_selector" comment:"字段选择器"`           // 字段选择器
	Type          string `json:"type" form:"type" comment:"Service类型过滤"`                         // Service类型过滤
	Status        string `json:"status" form:"status" comment:"状态过滤"`                            // 状态过滤
	Page          int    `json:"page" form:"page" comment:"页码"`                                  // 页码
	PageSize      int    `json:"page_size" form:"page_size" comment:"每页大小"`                      // 每页大小
}

// K8sServiceCreateRequest 创建Service请求
type K8sServiceCreateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	Type        string            `json:"type" comment:"Service类型"`                     // Service类型
	Selector    map[string]string `json:"selector" comment:"Pod选择器"`                    // Pod选择器
	Ports       []ServicePort     `json:"ports" binding:"required" comment:"服务端口"`      // 服务端口，必填
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
	ServiceYaml *corev1.Service   `json:"service_yaml" comment:"Service YAML对象"`        // Service YAML对象
}

// K8sServiceUpdateRequest 更新Service请求
type K8sServiceUpdateReq struct {
	ClusterID   int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace   string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name        string            `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	Type        string            `json:"type" comment:"Service类型"`                     // Service类型
	Selector    map[string]string `json:"selector" comment:"Pod选择器"`                    // Pod选择器
	Ports       []ServicePort     `json:"ports" comment:"服务端口"`                         // 服务端口
	Labels      map[string]string `json:"labels" comment:"标签"`                          // 标签
	Annotations map[string]string `json:"annotations" comment:"注解"`                     // 注解
	ServiceYaml *corev1.Service   `json:"service_yaml" comment:"Service YAML对象"`        // Service YAML对象
}

// K8sServiceDeleteRequest 删除Service请求
type K8sServiceDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sServiceBatchDeleteRequest 批量删除Service请求
type K8sServiceBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"Service名称列表"` // Service名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`       // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                         // 是否强制删除
}

// K8sServiceEndpointsRequest 获取Service端点请求
type K8sServiceEndpointsReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
}

// K8sServiceEventRequest 获取Service事件请求
type K8sServiceEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sServicePortForwardRequest Service端口转发请求
type K8sServicePortForwardReq struct {
	ClusterID int               `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string            `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string            `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	Ports     []PortForwardPort `json:"ports" binding:"required" comment:"端口转发配置"`    // 端口转发配置，必填
}

// K8sServiceDNSTestRequest Service DNS解析测试请求
type K8sServiceDNSTestReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Service名称"`  // Service名称，必填
	TestPod   string `json:"test_pod" comment:"测试用Pod名称"`                  // 测试用Pod名称，可选
}

// ====================== Service响应实体 ======================

// ServiceEntity Service响应实体
type ServiceEntity struct {
	Name                     string                  `json:"name"`                        // Service名称
	Namespace                string                  `json:"namespace"`                   // 命名空间
	UID                      string                  `json:"uid"`                         // Service UID
	Labels                   map[string]string       `json:"labels"`                      // 标签
	Annotations              map[string]string       `json:"annotations"`                 // 注解
	Type                     string                  `json:"type"`                        // Service类型
	ClusterIP                string                  `json:"cluster_ip"`                  // 集群内部IP
	ClusterIPs               []string                `json:"cluster_ips"`                 // 集群IP列表
	ExternalIPs              []string                `json:"external_ips"`                // 外部IP列表
	LoadBalancerIP           string                  `json:"load_balancer_ip"`            // 负载均衡器IP
	LoadBalancerSourceRanges []string                `json:"load_balancer_source_ranges"` // 负载均衡器源IP范围
	ExternalName             string                  `json:"external_name"`               // 外部名称(ExternalName类型)
	SessionAffinity          string                  `json:"session_affinity"`            // 会话亲和性
	Ports                    []ServicePortEntity     `json:"ports"`                       // 端口配置
	Selector                 map[string]string       `json:"selector"`                    // Pod选择器
	Endpoints                []ServiceEndpointEntity `json:"endpoints"`                   // 服务端点
	Status                   string                  `json:"status"`                      // Service状态
	Age                      string                  `json:"age"`                         // 存在时间
	CreatedAt                string                  `json:"created_at"`                  // 创建时间
}

// ServicePortEntity 服务端口实体
type ServicePortEntity struct {
	Name        string `json:"name"`         // 端口名称
	Protocol    string `json:"protocol"`     // 协议
	Port        int32  `json:"port"`         // 服务端口
	TargetPort  string `json:"target_port"`  // 目标端口
	NodePort    int32  `json:"node_port"`    // 节点端口(NodePort类型)
	AppProtocol string `json:"app_protocol"` // 应用协议
}

// ServiceEndpointEntity 服务端点实体
type ServiceEndpointEntity struct {
	IP          string                           `json:"ip"`          // 端点IP
	Hostname    string                           `json:"hostname"`    // 主机名
	NodeName    string                           `json:"node_name"`   // 节点名称
	Ports       []ServiceEndpointPortEntity      `json:"ports"`       // 端口信息
	Conditions  []ServiceEndpointConditionEntity `json:"conditions"`  // 端点条件
	Ready       bool                             `json:"ready"`       // 是否就绪
	Serving     bool                             `json:"serving"`     // 是否服务中
	Terminating bool                             `json:"terminating"` // 是否终止中
}

// ServiceEndpointPortEntity 端点端口信息
type ServiceEndpointPortEntity struct {
	Name        string `json:"name"`         // 端口名称
	Port        int32  `json:"port"`         // 端口号
	Protocol    string `json:"protocol"`     // 协议
	AppProtocol string `json:"app_protocol"` // 应用协议
}

// ServiceEndpointConditionEntity 端点条件
type ServiceEndpointConditionEntity struct {
	Type               string `json:"type"`                 // 条件类型
	Status             string `json:"status"`               // 条件状态
	LastTransitionTime string `json:"last_transition_time"` // 最后转换时间
}

// ServiceListResponse Service列表响应
type ServiceListResponse struct {
	Items      []ServiceEntity `json:"items"`       // Service列表
	TotalCount int             `json:"total_count"` // 总数
}

// ServiceDetailResponse Service详情响应
type ServiceDetailResponse struct {
	Service   ServiceEntity           `json:"service"`   // Service信息
	YAML      string                  `json:"yaml"`      // YAML内容
	Events    []ServiceEventEntity    `json:"events"`    // 事件列表
	Endpoints []ServiceEndpointEntity `json:"endpoints"` // 详细端点信息
	Pods      []PodEntity             `json:"pods"`      // 关联Pod列表
}

// ServiceEventEntity Service事件实体
type ServiceEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// ServicePortForwardResponse 端口转发响应
type ServicePortForwardResponse struct {
	LocalPort  int    `json:"local_port"`  // 本地端口
	RemotePort int    `json:"remote_port"` // 远程端口
	Status     string `json:"status"`      // 转发状态
	Message    string `json:"message"`     // 状态消息
}

// ServiceDNSTestResponse DNS解析测试响应
type ServiceDNSTestResponse struct {
	ServiceName string   `json:"service_name"` // Service名称
	Namespace   string   `json:"namespace"`    // 命名空间
	DNSNames    []string `json:"dns_names"`    // DNS名称列表
	ResolvedIPs []string `json:"resolved_ips"` // 解析到的IP列表
	Status      string   `json:"status"`       // 测试状态
	Message     string   `json:"message"`      // 测试消息
}
