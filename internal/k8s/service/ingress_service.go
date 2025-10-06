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
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressService interface {
	CreateIngress(ctx context.Context, req *model.CreateIngressReq) error
	GetIngressList(ctx context.Context, req *model.GetIngressListReq) (model.ListResp[*model.K8sIngress], error)
	GetIngressDetails(ctx context.Context, req *model.GetIngressDetailsReq) (*model.K8sIngress, error)
	GetIngressYaml(ctx context.Context, req *model.GetIngressYamlReq) (*model.K8sYaml, error)
	UpdateIngress(ctx context.Context, req *model.UpdateIngressReq) error
	DeleteIngress(ctx context.Context, req *model.DeleteIngressReq) error
	CreateIngressByYaml(ctx context.Context, req *model.CreateIngressByYamlReq) error
	UpdateIngressByYaml(ctx context.Context, req *model.UpdateIngressByYamlReq) error
}

type ingressService struct {
	ingressManager manager.IngressManager
	logger         *zap.Logger
}

func NewIngressService(ingressManager manager.IngressManager, logger *zap.Logger) IngressService {
	return &ingressService{
		ingressManager: ingressManager,
		logger:         logger,
	}
}

func (s *ingressService) CreateIngress(ctx context.Context, req *model.CreateIngressReq) error {
	if req == nil {
		return fmt.Errorf("创建Ingress请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Ingress名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	ingress, err := utils.BuildIngressFromSpec(req)
	if err != nil {
		s.logger.Error("构建Ingress对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Ingress对象失败: %w", err)
	}

	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	err = s.ingressManager.CreateIngress(ctx, req.ClusterID, req.Namespace, ingress)
	if err != nil {
		s.logger.Error("创建Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Ingress失败: %w", err)
	}

	s.logger.Info("创建Ingress成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *ingressService) GetIngressList(ctx context.Context, req *model.GetIngressListReq) (model.ListResp[*model.K8sIngress], error) {
	if req == nil {
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("获取Ingress列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("集群ID不能为空")
	}

	listOptions := utils.BuildIngressListOptions(req)

	k8sIngresses, err := s.ingressManager.GetIngressList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("获取Ingress列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("获取Ingress列表失败: %w", err)
	}

	// 应用过滤条件
	var filteredIngresses []*model.K8sIngress
	for _, k8sIngress := range k8sIngresses {
		// 状态过滤
		if req.Status != "" {
			var statusStr string
			switch k8sIngress.Status {
			case model.K8sIngressStatusRunning:
				statusStr = "running"
			case model.K8sIngressStatusPending:
				statusStr = "pending"
			case model.K8sIngressStatusFailed:
				statusStr = "failed"
			default:
				statusStr = "unknown"
			}
			if !strings.EqualFold(statusStr, req.Status) {
				continue
			}
		}
		// 名称过滤（使用通用的Search字段，支持不区分大小写）
		if !utils.FilterByName(k8sIngress.Name, req.Search) {
			continue
		}
		filteredIngresses = append(filteredIngresses, k8sIngress)
	}

	// 按创建时间排序（最新的在前）
	utils.SortByCreationTime(filteredIngresses, func(ingress *model.K8sIngress) time.Time {
		return ingress.CreatedAt
	})

	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	pagedItems, total := utils.PaginateK8sIngresses(filteredIngresses, page, size)

	return model.ListResp[*model.K8sIngress]{
		Total: total,
		Items: pagedItems,
	}, nil
}

func (s *ingressService) GetIngressDetails(ctx context.Context, req *model.GetIngressDetailsReq) (*model.K8sIngress, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Ingress详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Ingress名称不能为空")
	}

	ingress, err := s.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Ingress失败: %w", err)
	}

	return utils.ConvertToK8sIngress(ingress, req.ClusterID), nil
}

func (s *ingressService) GetIngressYaml(ctx context.Context, req *model.GetIngressYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Ingress YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Ingress名称不能为空")
	}

	ingress, err := s.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Ingress失败: %w", err)
	}

	yamlContent, err := utils.IngressToYAML(ingress)
	if err != nil {
		s.logger.Error("转换为YAML失败",
			zap.Error(err),
			zap.String("ingressName", ingress.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

func (s *ingressService) UpdateIngress(ctx context.Context, req *model.UpdateIngressReq) error {
	if req == nil {
		return fmt.Errorf("更新Ingress请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Ingress名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	existingIngress, err := s.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("获取现有Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有Ingress失败: %w", err)
	}

	updatedIngress := existingIngress.DeepCopy()

	// 更新标签
	if len(req.Labels) > 0 {
		if updatedIngress.Labels == nil {
			updatedIngress.Labels = make(map[string]string)
		}
		for k, v := range req.Labels {
			updatedIngress.Labels[k] = v
		}
	}

	// 更新注解
	if len(req.Annotations) > 0 {
		if updatedIngress.Annotations == nil {
			updatedIngress.Annotations = make(map[string]string)
		}
		for k, v := range req.Annotations {
			updatedIngress.Annotations[k] = v
		}
	}

	// 更新 IngressClassName
	if req.IngressClassName != nil {
		updatedIngress.Spec.IngressClassName = req.IngressClassName
	}

	// 更新 Rules
	if len(req.Rules) > 0 {
		updatedIngress.Spec.Rules = make([]networkingv1.IngressRule, 0, len(req.Rules))
		for _, rule := range req.Rules {
			ingressRule := networkingv1.IngressRule{
				Host: rule.Host,
			}

			if len(rule.HTTP.Paths) > 0 {
				httpRule := &networkingv1.HTTPIngressRuleValue{
					Paths: make([]networkingv1.HTTPIngressPath, 0, len(rule.HTTP.Paths)),
				}

				for _, path := range rule.HTTP.Paths {
					httpPath := networkingv1.HTTPIngressPath{
						Path:    path.Path,
						Backend: path.Backend,
					}
					if path.PathType != nil {
						pathType := networkingv1.PathType(*path.PathType)
						httpPath.PathType = &pathType
					}
					httpRule.Paths = append(httpRule.Paths, httpPath)
				}
				ingressRule.HTTP = httpRule
			}

			updatedIngress.Spec.Rules = append(updatedIngress.Spec.Rules, ingressRule)
		}
	}

	if len(req.TLS) > 0 {
		updatedIngress.Spec.TLS = make([]networkingv1.IngressTLS, 0, len(req.TLS))
		for _, tls := range req.TLS {
			updatedIngress.Spec.TLS = append(updatedIngress.Spec.TLS, networkingv1.IngressTLS{
				Hosts:      tls.Hosts,
				SecretName: tls.SecretName,
			})
		}
	}

	if err := utils.ValidateIngress(updatedIngress); err != nil {
		s.logger.Error("Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	err = s.ingressManager.UpdateIngress(ctx, req.ClusterID, req.Namespace, updatedIngress)
	if err != nil {
		s.logger.Error("更新Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Ingress失败: %w", err)
	}

	s.logger.Info("更新Ingress成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *ingressService) DeleteIngress(ctx context.Context, req *model.DeleteIngressReq) error {
	if req == nil {
		return fmt.Errorf("删除Ingress请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Ingress名称不能为空")
	}

	deleteOptions := metav1.DeleteOptions{}

	err := s.ingressManager.DeleteIngress(ctx, req.ClusterID, req.Namespace, req.Name, deleteOptions)
	if err != nil {
		s.logger.Error("删除Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Ingress失败: %w", err)
	}

	s.logger.Info("删除Ingress成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

func (s *ingressService) CreateIngressByYaml(ctx context.Context, req *model.CreateIngressByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML创建Ingress请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML创建Ingress",
		zap.Int("clusterID", req.ClusterID))

	ingress, err := utils.YAMLToIngress(req.YAML)
	if err != nil {
		s.logger.Error("解析YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.Error(err))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 如果YAML中没有指定namespace，使用default
	if ingress.Namespace == "" {
		ingress.Namespace = "default"
	}

	// YAML中必须包含name信息
	if ingress.Name == "" {
		s.logger.Error("YAML中必须指定name",
			zap.Int("clusterID", req.ClusterID))
		return fmt.Errorf("YAML中必须指定name")
	}

	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("Ingress配置验证失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", ingress.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	err = s.ingressManager.CreateIngress(ctx, req.ClusterID, ingress.Namespace, ingress)
	if err != nil {
		s.logger.Error("创建Ingress失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", ingress.Namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return fmt.Errorf("创建Ingress失败: %w", err)
	}

	s.logger.Info("创建Ingress成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", ingress.Namespace),
		zap.String("name", ingress.Name))

	return nil
}

func (s *ingressService) UpdateIngressByYaml(ctx context.Context, req *model.UpdateIngressByYamlReq) error {
	if req == nil {
		return fmt.Errorf("通过YAML更新Ingress请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.YAML == "" {
		return fmt.Errorf("YAML内容不能为空")
	}

	s.logger.Info("开始通过YAML更新Ingress",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	ingress, err := utils.YAMLToIngress(req.YAML)
	if err != nil {
		s.logger.Error("解析YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name),
			zap.Error(err))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 确保YAML中的namespace和name与请求参数一致
	if ingress.Namespace != "" && ingress.Namespace != req.Namespace {
		s.logger.Error("YAML中的namespace与请求参数不一致",
			zap.Int("clusterID", req.ClusterID),
			zap.String("yamlNamespace", ingress.Namespace),
			zap.String("reqNamespace", req.Namespace))
		return fmt.Errorf("YAML中的namespace (%s) 与请求参数不一致 (%s)", ingress.Namespace, req.Namespace)
	}

	if ingress.Name != "" && ingress.Name != req.Name {
		s.logger.Error("YAML中的name与请求参数不一致",
			zap.Int("clusterID", req.ClusterID),
			zap.String("yamlName", ingress.Name),
			zap.String("reqName", req.Name))
		return fmt.Errorf("YAML中的name (%s) 与请求参数不一致 (%s)", ingress.Name, req.Name)
	}

	// 如果YAML中没有指定，使用请求参数
	if ingress.Namespace == "" {
		ingress.Namespace = req.Namespace
	}

	if ingress.Name == "" {
		ingress.Name = req.Name
	}

	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("Ingress配置验证失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("name", ingress.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	err = s.ingressManager.UpdateIngress(ctx, req.ClusterID, ingress.Namespace, ingress)
	if err != nil {
		s.logger.Error("更新Ingress失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", ingress.Namespace),
			zap.String("name", ingress.Name),
			zap.Error(err))
		return fmt.Errorf("更新Ingress失败: %w", err)
	}

	s.logger.Info("更新Ingress成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", ingress.Namespace),
		zap.String("name", ingress.Name))

	return nil
}
