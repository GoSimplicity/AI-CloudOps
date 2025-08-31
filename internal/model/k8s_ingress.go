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

// ====================== 内部结构体 ======================

// IngressRule Ingress规则(内部使用)
type IngressRule struct {
	Host string               `json:"host"`
	HTTP IngressHTTPRuleValue `json:"http"`
}

// IngressHTTPRuleValue HTTP规则值(内部使用)
type IngressHTTPRuleValue struct {
	Paths []IngressHTTPIngressPath `json:"paths"`
}

// IngressHTTPIngressPath HTTP路径(内部使用)
type IngressHTTPIngressPath struct {
	Path     string         `json:"path"`
	PathType string         `json:"path_type"`
	Backend  IngressBackend `json:"backend"`
}

// IngressBackend 后端(内部使用)
type IngressBackend struct {
	Service  IngressServiceBackendPort `json:"service"`
	Resource IngressResourceRef        `json:"resource"`
}

// IngressServiceBackendPort 服务后端端口(内部使用)
type IngressServiceBackendPort struct {
	Name string                        `json:"name"`
	Port IngressServiceBackendPortSpec `json:"port" gorm:"type:text;serializer:json"`
}

// IngressServiceBackendPortSpec 服务后端端口规格(内部使用)
type IngressServiceBackendPortSpec struct {
	Name   string `json:"name"`
	Number int32  `json:"number"`
}

// IngressResourceRef 资源引用(内部使用)
type IngressResourceRef struct {
	APIGroup string `json:"api_group"`
	Kind     string `json:"kind"`
	Name     string `json:"name"`
}

// IngressTLS TLS配置(内部使用)
type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secret_name"`
}

// IngressLoadBalancer 负载均衡器(内部使用)
type IngressLoadBalancer struct {
	Ingress []IngressLoadBalancerIngress `json:"ingress"`
}

// IngressLoadBalancerIngress 负载均衡器Ingress(内部使用)
type IngressLoadBalancerIngress struct {
	IP       string              `json:"ip"`
	Hostname string              `json:"hostname"`
	Ports    []IngressPortStatus `json:"ports"`
}

// IngressPortStatus 端口状态(内部使用)
type IngressPortStatus struct {
	Port     int32  `json:"port"`
	Protocol string `json:"protocol"`
	Error    string `json:"error"`
}

// K8sIngressEntity Kubernetes Ingress数据库实体
type K8sIngressEntity struct {
	Model
	Name              string              `json:"name" binding:"required,min=1,max=200" gorm:"size:200;comment:Ingress名称"`   // Ingress名称
	Namespace         string              `json:"namespace" binding:"required,min=1,max=200" gorm:"size:200;comment:所属命名空间"` // 所属命名空间
	ClusterID         int                 `json:"cluster_id" gorm:"index;not null;comment:所属集群ID"`                           // 所属集群ID
	UID               string              `json:"uid" gorm:"size:100;comment:Ingress UID"`                                   // Ingress UID
	IngressClassName  string              `json:"ingress_class_name" gorm:"size:200;comment:Ingress类名"`                      // Ingress类名
	Rules             []IngressRule       `json:"rules" gorm:"type:text;serializer:json;comment:Ingress规则"`                  // Ingress规则
	TLS               []IngressTLS        `json:"tls" gorm:"type:text;serializer:json;comment:TLS配置"`                        // TLS配置
	LoadBalancer      IngressLoadBalancer `json:"load_balancer" gorm:"type:text;serializer:json;comment:负载均衡器信息"`            // 负载均衡器信息
	Labels            map[string]string   `json:"labels" gorm:"type:text;serializer:json;comment:标签"`                        // 标签
	Annotations       map[string]string   `json:"annotations" gorm:"type:text;serializer:json;comment:注解"`                   // 注解
	CreationTimestamp time.Time           `json:"creation_timestamp" gorm:"comment:Kubernetes创建时间"`                          // Kubernetes创建时间
	Age               string              `json:"age" gorm:"-"`                                                              // 存在时间，前端计算使用
	Status            string              `json:"status" gorm:"-"`                                                           // Ingress状态，前端计算使用
	Hosts             []string            `json:"hosts" gorm:"-"`                                                            // 主机列表，前端使用
}

func (k *K8sIngressEntity) TableName() string {
	return "cl_k8s_ingresses"
}

// K8sIngressListRequest Ingress列表查询请求
type K8sIngressListReq struct {
	ClusterID        int    `json:"cluster_id" form:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace        string `json:"namespace" form:"namespace" comment:"命名空间"`                        // 命名空间
	LabelSelector    string `json:"label_selector" form:"label_selector" comment:"标签选择器"`             // 标签选择器
	FieldSelector    string `json:"field_selector" form:"field_selector" comment:"字段选择器"`             // 字段选择器
	IngressClassName string `json:"ingress_class_name" form:"ingress_class_name" comment:"Ingress类名"` // Ingress类名过滤
	Host             string `json:"host" form:"host" comment:"主机名过滤"`                                 // 主机名过滤
	Status           string `json:"status" form:"status" comment:"状态过滤"`                              // 状态过滤
	Page             int    `json:"page" form:"page" comment:"页码"`                                    // 页码
	PageSize         int    `json:"page_size" form:"page_size" comment:"每页大小"`                        // 每页大小
}

// K8sIngressCreateRequest 创建Ingress请求
type K8sIngressCreateReq struct {
	ClusterID        int                   `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace        string                `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name             string                `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	IngressClassName *string               `json:"ingress_class_name" comment:"Ingress类名"`       // Ingress类名
	Rules            []IngressRuleReq      `json:"rules" binding:"required" comment:"Ingress规则"` // Ingress规则，必填
	TLS              []IngressTLSReq       `json:"tls" comment:"TLS配置"`                          // TLS配置
	Labels           map[string]string     `json:"labels" comment:"标签"`                          // 标签
	Annotations      map[string]string     `json:"annotations" comment:"注解"`                     // 注解
	IngressYaml      *networkingv1.Ingress `json:"ingress_yaml" comment:"Ingress YAML对象"`        // Ingress YAML对象
}

// K8sIngressUpdateRequest 更新Ingress请求
type K8sIngressUpdateReq struct {
	ClusterID        int                   `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace        string                `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name             string                `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	IngressClassName *string               `json:"ingress_class_name" comment:"Ingress类名"`       // Ingress类名
	Rules            []IngressRuleReq      `json:"rules" comment:"Ingress规则"`                    // Ingress规则
	TLS              []IngressTLSReq       `json:"tls" comment:"TLS配置"`                          // TLS配置
	Labels           map[string]string     `json:"labels" comment:"标签"`                          // 标签
	Annotations      map[string]string     `json:"annotations" comment:"注解"`                     // 注解
	IngressYaml      *networkingv1.Ingress `json:"ingress_yaml" comment:"Ingress YAML对象"`        // Ingress YAML对象
}

// K8sIngressDeleteRequest 删除Ingress请求
type K8sIngressDeleteReq struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sIngressBatchDeleteRequest 批量删除Ingress请求
type K8sIngressBatchDeleteReq struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"Ingress名称列表"` // Ingress名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`       // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                         // 是否强制删除
}

// K8sIngressEventRequest 获取Ingress事件请求
type K8sIngressEventReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sIngressTLSTestRequest Ingress TLS证书测试请求
type K8sIngressTLSTestReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	Host      string `json:"host" binding:"required" comment:"测试主机名"`      // 测试主机名，必填
}

// K8sIngressBackendHealthRequest 检查Ingress后端健康状态请求
type K8sIngressBackendHealthReq struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
}

// ====================== Ingress响应实体 ======================

// IngressEntity Ingress响应实体
type IngressEntity struct {
	Name             string                    `json:"name"`               // Ingress名称
	Namespace        string                    `json:"namespace"`          // 命名空间
	UID              string                    `json:"uid"`                // Ingress UID
	Labels           map[string]string         `json:"labels"`             // 标签
	Annotations      map[string]string         `json:"annotations"`        // 注解
	IngressClassName string                    `json:"ingress_class_name"` // Ingress类名
	Rules            []IngressRuleEntity       `json:"rules"`              // Ingress规则
	TLS              []IngressTLSEntity        `json:"tls"`                // TLS配置
	LoadBalancer     IngressLoadBalancerEntity `json:"load_balancer"`      // 负载均衡器信息
	Status           string                    `json:"status"`             // Ingress状态
	Hosts            []string                  `json:"hosts"`              // 主机列表
	Endpoints        []IngressEndpointEntity   `json:"endpoints"`          // 端点列表
	Age              string                    `json:"age"`                // 存在时间
	CreatedAt        string                    `json:"created_at"`         // 创建时间
}

// IngressRuleEntity Ingress规则实体
type IngressRuleEntity struct {
	Host string                     `json:"host"` // 主机名
	HTTP IngressHTTPRuleValueEntity `json:"http"` // HTTP规则
}

// IngressHTTPRuleValueEntity HTTP规则值实体
type IngressHTTPRuleValueEntity struct {
	Paths []IngressHTTPIngressPathEntity `json:"paths"` // 路径列表
}

// IngressHTTPIngressPathEntity HTTP路径实体
type IngressHTTPIngressPathEntity struct {
	Path     string                      `json:"path"`      // 路径
	PathType string                      `json:"path_type"` // 路径类型
	Backend  IngressIngressBackendEntity `json:"backend"`   // 后端服务
}

// IngressIngressBackendEntity Ingress后端实体
type IngressIngressBackendEntity struct {
	Service  IngressServiceBackendPortEntity `json:"service"`  // 服务后端
	Resource IngressResourceRefEntity        `json:"resource"` // 资源后端
}

// IngressServiceBackendPortEntity 服务后端端口实体
type IngressServiceBackendPortEntity struct {
	Name string                                `json:"name"` // 服务名称
	Port IngressServiceBackendPortNumberEntity `json:"port"` // 端口信息
}

// IngressServiceBackendPortNumberEntity 服务后端端口号实体
type IngressServiceBackendPortNumberEntity struct {
	Name   string `json:"name"`   // 端口名称
	Number int32  `json:"number"` // 端口号
}

// IngressResourceRefEntity 资源引用实体
type IngressResourceRefEntity struct {
	APIGroup string `json:"api_group"` // API组
	Kind     string `json:"kind"`      // 资源类型
	Name     string `json:"name"`      // 资源名称
}

// IngressTLSEntity TLS配置实体
type IngressTLSEntity struct {
	Hosts      []string `json:"hosts"`       // 主机列表
	SecretName string   `json:"secret_name"` // Secret名称
}

// IngressLoadBalancerEntity 负载均衡器实体
type IngressLoadBalancerEntity struct {
	Ingress []IngressLoadBalancerIngressEntity `json:"ingress"` // Ingress信息
}

// IngressLoadBalancerIngressEntity 负载均衡器Ingress实体
type IngressLoadBalancerIngressEntity struct {
	IP       string                    `json:"ip"`       // IP地址
	Hostname string                    `json:"hostname"` // 主机名
	Ports    []IngressPortStatusEntity `json:"ports"`    // 端口状态
}

// IngressPortStatusEntity 端口状态实体
type IngressPortStatusEntity struct {
	Port     int32  `json:"port"`     // 端口号
	Protocol string `json:"protocol"` // 协议
	Error    string `json:"error"`    // 错误信息
}

// IngressEndpointEntity Ingress端点实体
type IngressEndpointEntity struct {
	Host        string `json:"host"`         // 主机名
	Path        string `json:"path"`         // 路径
	ServiceName string `json:"service_name"` // 服务名称
	ServicePort string `json:"service_port"` // 服务端口
	Ready       bool   `json:"ready"`        // 是否就绪
	Available   bool   `json:"available"`    // 是否可用
}

// IngressListResponse Ingress列表响应
type IngressListResponse struct {
	Items      []IngressEntity `json:"items"`       // Ingress列表
	TotalCount int             `json:"total_count"` // 总数
}

// IngressDetailResponse Ingress详情响应
type IngressDetailResponse struct {
	Ingress       IngressEntity              `json:"ingress"`        // Ingress信息
	YAML          string                     `json:"yaml"`           // YAML内容
	Events        []IngressEventEntity       `json:"events"`         // 事件列表
	BackendHealth IngressBackendHealthEntity `json:"backend_health"` // 后端健康状态
	TLSStatus     IngressTLSStatusEntity     `json:"tls_status"`     // TLS状态
}

// IngressEventEntity Ingress事件实体
type IngressEventEntity struct {
	Type      string `json:"type"`       // 事件类型
	Reason    string `json:"reason"`     // 原因
	Message   string `json:"message"`    // 消息
	Source    string `json:"source"`     // 来源
	FirstTime string `json:"first_time"` // 首次时间
	LastTime  string `json:"last_time"`  // 最后时间
	Count     int32  `json:"count"`      // 次数
}

// IngressBackendHealthEntity Ingress后端健康状态实体
type IngressBackendHealthEntity struct {
	TotalBackends   int                          `json:"total_backends"`   // 总后端数
	HealthyBackends int                          `json:"healthy_backends"` // 健康后端数
	Backends        []IngressBackendStatusEntity `json:"backends"`         // 后端状态列表
}

// IngressBackendStatusEntity Ingress后端状态实体
type IngressBackendStatusEntity struct {
	ServiceName string `json:"service_name"` // 服务名称
	ServicePort string `json:"service_port"` // 服务端口
	Host        string `json:"host"`         // 主机名
	Path        string `json:"path"`         // 路径
	Status      string `json:"status"`       // 状态
	Message     string `json:"message"`      // 消息
	Healthy     bool   `json:"healthy"`      // 是否健康
}

// IngressTLSStatusEntity Ingress TLS状态实体
type IngressTLSStatusEntity struct {
	TotalHosts      int                     `json:"total_hosts"`      // 总主机数
	SecuredHosts    int                     `json:"secured_hosts"`    // 安全主机数
	CertificateInfo []IngressCertInfoEntity `json:"certificate_info"` // 证书信息
}

// IngressCertInfoEntity Ingress证书信息实体
type IngressCertInfoEntity struct {
	SecretName   string   `json:"secret_name"`    // Secret名称
	Hosts        []string `json:"hosts"`          // 主机列表
	Issuer       string   `json:"issuer"`         // 签发者
	Subject      string   `json:"subject"`        // 主题
	NotBefore    string   `json:"not_before"`     // 生效时间
	NotAfter     string   `json:"not_after"`      // 过期时间
	IsValid      bool     `json:"is_valid"`       // 是否有效
	DaysToExpiry int      `json:"days_to_expiry"` // 到期天数
}

// IngressTLSTestResponse Ingress TLS测试响应
type IngressTLSTestResponse struct {
	Host         string                `json:"host"`          // 主机名
	Port         int                   `json:"port"`          // 端口
	Connected    bool                  `json:"connected"`     // 是否连接成功
	TLSVersion   string                `json:"tls_version"`   // TLS版本
	CipherSuite  string                `json:"cipher_suite"`  // 加密套件
	Certificate  IngressCertInfoEntity `json:"certificate"`   // 证书信息
	ResponseTime int                   `json:"response_time"` // 响应时间(ms)
	Error        string                `json:"error"`         // 错误信息
}

// ====================== 请求结构体 ======================

// IngressRuleReq Ingress规则请求
type IngressRuleReq struct {
	Host string                  `json:"host" comment:"主机名"`
	HTTP IngressHTTPRuleValueReq `json:"http" comment:"HTTP规则"`
}

// IngressHTTPRuleValueReq HTTP规则值请求
type IngressHTTPRuleValueReq struct {
	Paths []IngressHTTPIngressPathReq `json:"paths" comment:"路径列表"`
}

// IngressHTTPIngressPathReq HTTP路径请求
type IngressHTTPIngressPathReq struct {
	Path     string            `json:"path" comment:"路径"`
	PathType string            `json:"path_type" comment:"路径类型"`
	Backend  IngressBackendReq `json:"backend" comment:"后端服务"`
}

// IngressBackendReq 后端请求
type IngressBackendReq struct {
	Service  IngressServiceBackendPortReq `json:"service" comment:"服务后端"`
	Resource IngressResourceRefReq        `json:"resource" comment:"资源后端"`
}

// IngressServiceBackendPortReq 服务后端端口请求
type IngressServiceBackendPortReq struct {
	Name string                           `json:"name" comment:"服务名称"`
	Port IngressServiceBackendPortSpecReq `json:"port" comment:"端口信息"`
}

// IngressServiceBackendPortSpecReq 服务后端端口规格请求
type IngressServiceBackendPortSpecReq struct {
	Name   string `json:"name" comment:"端口名称"`
	Number int32  `json:"number" comment:"端口号"`
}

// IngressResourceRefReq 资源引用请求
type IngressResourceRefReq struct {
	APIGroup string `json:"api_group" comment:"API组"`
	Kind     string `json:"kind" comment:"资源类型"`
	Name     string `json:"name" comment:"资源名称"`
}

// IngressTLSReq TLS配置请求
type IngressTLSReq struct {
	Hosts      []string `json:"hosts" comment:"主机列表"`
	SecretName string   `json:"secret_name" comment:"Secret名称"`
}

// K8sTLSTestResult Ingress TLS证书测试结果
type K8sTLSTestResult struct {
	Host             string `json:"host"`               // 测试主机名
	Port             int    `json:"port"`               // 测试端口
	Valid            bool   `json:"valid"`              // 证书是否有效
	CertIssuer       string `json:"cert_issuer"`        // 证书颁发者
	CertSubject      string `json:"cert_subject"`       // 证书主体
	CertExpiry       string `json:"cert_expiry"`        // 证书过期时间
	CertDNSNames     string `json:"cert_dns_names"`     // 证书DNS名称
	CertSerialNumber string `json:"cert_serial_number"` // 证书序列号
	ErrorMessage     string `json:"error_message"`      // 错误信息
	TestTime         string `json:"test_time"`          // 测试时间
}

// K8sBackendHealth Ingress后端健康状态
type K8sBackendHealth struct {
	ServiceName  string `json:"service_name"`  // 服务名称
	ServicePort  int    `json:"service_port"`  // 服务端口
	PodName      string `json:"pod_name"`      // Pod名称
	PodIP        string `json:"pod_ip"`        // Pod IP
	Ready        bool   `json:"ready"`         // 是否就绪
	Status       string `json:"status"`        // 状态描述
	CheckTime    string `json:"check_time"`    // 检查时间
	ErrorMessage string `json:"error_message"` // 错误信息
}
