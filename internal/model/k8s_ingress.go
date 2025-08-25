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
type K8sIngressListRequest struct {
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
type K8sIngressCreateRequest struct {
	ClusterID        int                   `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace        string                `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name             string                `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	IngressClassName *string               `json:"ingress_class_name" comment:"Ingress类名"`       // Ingress类名
	Rules            []IngressRuleRequest  `json:"rules" binding:"required" comment:"Ingress规则"` // Ingress规则，必填
	TLS              []IngressTLSRequest   `json:"tls" comment:"TLS配置"`                          // TLS配置
	Labels           map[string]string     `json:"labels" comment:"标签"`                          // 标签
	Annotations      map[string]string     `json:"annotations" comment:"注解"`                     // 注解
	IngressYaml      *networkingv1.Ingress `json:"ingress_yaml" comment:"Ingress YAML对象"`        // Ingress YAML对象
}

// K8sIngressUpdateRequest 更新Ingress请求
type K8sIngressUpdateRequest struct {
	ClusterID        int                   `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace        string                `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name             string                `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	IngressClassName *string               `json:"ingress_class_name" comment:"Ingress类名"`       // Ingress类名
	Rules            []IngressRuleRequest  `json:"rules" comment:"Ingress规则"`                    // Ingress规则
	TLS              []IngressTLSRequest   `json:"tls" comment:"TLS配置"`                          // TLS配置
	Labels           map[string]string     `json:"labels" comment:"标签"`                          // 标签
	Annotations      map[string]string     `json:"annotations" comment:"注解"`                     // 注解
	IngressYaml      *networkingv1.Ingress `json:"ingress_yaml" comment:"Ingress YAML对象"`        // Ingress YAML对象
}

// K8sIngressDeleteRequest 删除Ingress请求
type K8sIngressDeleteRequest struct {
	ClusterID          int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace          string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name               string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	GracePeriodSeconds *int64 `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`     // 优雅删除时间
	Force              bool   `json:"force" comment:"是否强制删除"`                       // 是否强制删除
}

// K8sIngressBatchDeleteRequest 批量删除Ingress请求
type K8sIngressBatchDeleteRequest struct {
	ClusterID          int      `json:"cluster_id" binding:"required" comment:"集群ID"`   // 集群ID，必填
	Namespace          string   `json:"namespace" binding:"required" comment:"命名空间"`    // 命名空间，必填
	Names              []string `json:"names" binding:"required" comment:"Ingress名称列表"` // Ingress名称列表，必填
	GracePeriodSeconds *int64   `json:"grace_period_seconds" comment:"优雅删除时间（秒）"`       // 优雅删除时间
	Force              bool     `json:"force" comment:"是否强制删除"`                         // 是否强制删除
}

// K8sIngressEventRequest 获取Ingress事件请求
type K8sIngressEventRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	LimitDays int    `json:"limit_days" comment:"限制天数内的事件"`                // 限制天数内的事件
}

// K8sIngressTLSTestRequest Ingress TLS证书测试请求
type K8sIngressTLSTestRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
	Host      string `json:"host" binding:"required" comment:"测试主机名"`      // 测试主机名，必填
}

// K8sIngressBackendHealthRequest 检查Ingress后端健康状态请求
type K8sIngressBackendHealthRequest struct {
	ClusterID int    `json:"cluster_id" binding:"required" comment:"集群ID"` // 集群ID，必填
	Namespace string `json:"namespace" binding:"required" comment:"命名空间"`  // 命名空间，必填
	Name      string `json:"name" binding:"required" comment:"Ingress名称"`  // Ingress名称，必填
}
