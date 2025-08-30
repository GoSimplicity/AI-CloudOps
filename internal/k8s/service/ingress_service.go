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

package service

import (
	"context"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressService interface {
	GetIngressList(ctx context.Context, req *model.K8sIngressListReq) ([]*model.K8sIngressEntity, error)
	GetIngressesByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sIngressEntity, error)
	GetIngress(ctx context.Context, clusterID int, namespace, name string) (*model.K8sIngressEntity, error)
	GetIngressYaml(ctx context.Context, clusterID int, namespace, name string) (string, error)
	CreateIngress(ctx context.Context, req *model.K8sIngressCreateReq) error
	UpdateIngress(ctx context.Context, req *model.K8sIngressUpdateReq) error
	DeleteIngress(ctx context.Context, req *model.K8sIngressDeleteReq) error
	BatchDeleteIngresses(ctx context.Context, req *model.K8sIngressBatchDeleteReq) error
	GetIngressEvents(ctx context.Context, req *model.K8sIngressEventReq) ([]*model.K8sEvent, error)
	TestIngressTLS(ctx context.Context, req *model.K8sIngressTLSTestReq) (*model.K8sTLSTestResult, error)
	CheckIngressBackendHealth(ctx context.Context, req *model.K8sIngressBackendHealthReq) ([]*model.K8sBackendHealth, error)
}

type ingressService struct {
	dao            dao.ClusterDAO         // 保持对DAO的依赖
	client         client.K8sClient       // 保持向后兼容
	ingressManager manager.IngressManager // 新的依赖注入
	logger         *zap.Logger
}

// NewIngressService 创建新的 IngressService 实例
func NewIngressService(dao dao.ClusterDAO, client client.K8sClient, ingressManager manager.IngressManager, logger *zap.Logger) IngressService {
	return &ingressService{
		dao:            dao,
		client:         client,
		ingressManager: ingressManager,
		logger:         logger,
	}
}

// GetIngressList 获取Ingress列表
func (i *ingressService) GetIngressList(ctx context.Context, req *model.K8sIngressListReq) ([]*model.K8sIngressEntity, error) {
	// 构建查询选项
	listOptions := metav1.ListOptions{}
	if req.LabelSelector != "" {
		listOptions.LabelSelector = req.LabelSelector
	}
	if req.FieldSelector != "" {
		listOptions.FieldSelector = req.FieldSelector
	}

	// 使用 IngressManager 获取列表
	ingresses, err := i.ingressManager.GetIngressList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		i.logger.Error("获取Ingress列表失败",
			zap.String("namespace", req.Namespace),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceList, "获取Ingress列表失败")
	}

	entities := make([]*model.K8sIngressEntity, 0, len(ingresses.Items))
	for _, ingress := range ingresses.Items {
		entity := i.convertIngressToEntity(&ingress, req.ClusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetIngressesByNamespace 根据命名空间获取Ingress列表
func (i *ingressService) GetIngressesByNamespace(ctx context.Context, clusterID int, namespace string) ([]*model.K8sIngressEntity, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, constants.ErrorK8sClientNotReady
	}

	ingresses, err := kubeClient.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		i.logger.Error("获取Ingress列表失败",
			zap.String("namespace", namespace),
			zap.Error(err))
		return nil, err
	}

	entities := make([]*model.K8sIngressEntity, 0, len(ingresses.Items))
	for _, ingress := range ingresses.Items {
		entity := i.convertIngressToEntity(&ingress, clusterID)
		entities = append(entities, entity)
	}

	return entities, nil
}

// GetIngress 获取单个Ingress详情
func (i *ingressService) GetIngress(ctx context.Context, clusterID int, namespace, name string) (*model.K8sIngressEntity, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取Ingress详情失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return nil, pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Ingress详情失败")
	}

	return i.convertIngressToEntity(ingress, clusterID), nil
}

// GetIngressYaml 获取Ingress的YAML
func (i *ingressService) GetIngressYaml(ctx context.Context, clusterID int, namespace, name string) (string, error) {
	kubeClient, err := i.client.GetKubeClient(clusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	ingress, err := kubeClient.NetworkingV1().Ingresses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		i.logger.Error("获取Ingress失败",
			zap.String("namespace", namespace),
			zap.String("name", name),
			zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceGet, "获取Ingress失败")
	}

	yamlData, err := yaml.Marshal(ingress)
	if err != nil {
		i.logger.Error("序列化Ingress为YAML失败", zap.Error(err))
		return "", pkg.NewBusinessError(constants.ErrK8sResourceOperation, "序列化Ingress为YAML失败")
	}

	return string(yamlData), nil
}

// CreateIngress 创建Ingress
func (i *ingressService) CreateIngress(ctx context.Context, req *model.K8sIngressCreateReq) error {
	kubeClient, err := i.client.GetKubeClient(req.ClusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.IngressYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "Ingress YAML不能为空")
	}

	_, err = kubeClient.NetworkingV1().Ingresses(req.Namespace).Create(ctx, req.IngressYaml, metav1.CreateOptions{})
	if err != nil {
		i.logger.Error("创建Ingress失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceCreate, "创建Ingress失败")
	}

	i.logger.Info("成功创建Ingress",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// UpdateIngress 更新Ingress
func (i *ingressService) UpdateIngress(ctx context.Context, req *model.K8sIngressUpdateReq) error {
	kubeClient, err := i.client.GetKubeClient(req.ClusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	if req.IngressYaml == nil {
		return pkg.NewBusinessError(constants.ErrInvalidParam, "Ingress YAML不能为空")
	}

	_, err = kubeClient.NetworkingV1().Ingresses(req.Namespace).Update(ctx, req.IngressYaml, metav1.UpdateOptions{})
	if err != nil {
		i.logger.Error("更新Ingress失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceUpdate, "更新Ingress失败")
	}

	i.logger.Info("成功更新Ingress",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// DeleteIngress 删除Ingress
func (i *ingressService) DeleteIngress(ctx context.Context, req *model.K8sIngressDeleteReq) error {
	kubeClient, err := i.client.GetKubeClient(req.ClusterID)
	if err != nil {
		i.logger.Error("获取Kubernetes客户端失败", zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sClientInit, "无法连接到Kubernetes集群")
	}

	err = kubeClient.NetworkingV1().Ingresses(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{})
	if err != nil {
		i.logger.Error("删除Ingress失败",
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return pkg.NewBusinessError(constants.ErrK8sResourceDelete, "删除Ingress失败")
	}

	i.logger.Info("成功删除Ingress",
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))
	return nil
}

// BatchDeleteIngresses 批量删除Ingress
func (i *ingressService) BatchDeleteIngresses(ctx context.Context, req *model.K8sIngressBatchDeleteReq) error {
	// TODO: 实现批量删除功能
	return pkg.NewBusinessError(constants.ErrNotImplemented, "批量删除Ingress功能尚未实现")
}

// GetIngressEvents 获取Ingress事件
func (i *ingressService) GetIngressEvents(ctx context.Context, req *model.K8sIngressEventReq) ([]*model.K8sEvent, error) {
	// TODO: 实现获取事件功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "获取Ingress事件功能尚未实现")
}

// TestIngressTLS 测试Ingress TLS配置
func (i *ingressService) TestIngressTLS(ctx context.Context, req *model.K8sIngressTLSTestReq) (*model.K8sTLSTestResult, error) {
	// TODO: 实现TLS测试功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "Ingress TLS测试功能尚未实现")
}

// CheckIngressBackendHealth 检查Ingress后端健康状态
func (i *ingressService) CheckIngressBackendHealth(ctx context.Context, req *model.K8sIngressBackendHealthReq) ([]*model.K8sBackendHealth, error) {
	// TODO: 实现后端健康检查功能
	return nil, pkg.NewBusinessError(constants.ErrNotImplemented, "Ingress后端健康检查功能尚未实现")
}

// convertIngressToEntity 将Kubernetes Ingress转换为实体模型
func (i *ingressService) convertIngressToEntity(ingress *networkingv1.Ingress, clusterID int) *model.K8sIngressEntity {
	// 提取主机列表
	hosts := make([]string, 0)
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
	}

	// 计算年龄
	age := pkg.GetAge(ingress.CreationTimestamp.Time)

	// 确定状态
	status := "Ready"
	if len(ingress.Status.LoadBalancer.Ingress) == 0 {
		status = "Pending"
	}

	// 获取Ingress类名
	ingressClassName := ""
	if ingress.Spec.IngressClassName != nil {
		ingressClassName = *ingress.Spec.IngressClassName
	}

	// 转换规则（简化处理）
	rules := make([]model.IngressRule, 0, len(ingress.Spec.Rules))
	for _, rule := range ingress.Spec.Rules {
		ingressRule := model.IngressRule{
			Host: rule.Host,
		}

		if rule.HTTP != nil {
			paths := make([]model.IngressHTTPIngressPath, 0, len(rule.HTTP.Paths))
			for _, path := range rule.HTTP.Paths {
				ingressPath := model.IngressHTTPIngressPath{
					Path: path.Path,
				}
				if path.PathType != nil {
					ingressPath.PathType = string(*path.PathType)
				}
				paths = append(paths, ingressPath)
			}
			ingressRule.HTTP = model.IngressHTTPRuleValue{
				Paths: paths,
			}
		}

		rules = append(rules, ingressRule)
	}

	// 转换TLS配置（简化处理）
	tls := make([]model.IngressTLS, 0, len(ingress.Spec.TLS))
	for _, tlsConfig := range ingress.Spec.TLS {
		ingressTLS := model.IngressTLS{
			Hosts:      tlsConfig.Hosts,
			SecretName: tlsConfig.SecretName,
		}
		tls = append(tls, ingressTLS)
	}

	// 负载均衡器信息（简化处理）
	loadBalancer := model.IngressLoadBalancer{}

	return &model.K8sIngressEntity{
		Name:              ingress.Name,
		Namespace:         ingress.Namespace,
		ClusterID:         clusterID,
		UID:               string(ingress.UID),
		IngressClassName:  ingressClassName,
		Rules:             rules,
		TLS:               tls,
		LoadBalancer:      loadBalancer,
		Labels:            ingress.Labels,
		Annotations:       ingress.Annotations,
		CreationTimestamp: ingress.CreationTimestamp.Time,
		Age:               age,
		Status:            status,
		Hosts:             hosts,
	}
}
