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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IngressService Ingress业务服务接口
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

// CreateIngress 创建Ingress
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

	// 构建Ingress对象
	ingress, err := utils.BuildIngressFromSpec(req)
	if err != nil {
		s.logger.Error("CreateIngress: 构建Ingress对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Ingress对象失败: %w", err)
	}

	// 验证Ingress配置
	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("CreateIngress: Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	// 创建Ingress
	err = s.ingressManager.CreateIngress(ctx, req.ClusterID, req.Namespace, ingress)
	if err != nil {
		s.logger.Error("CreateIngress: 创建Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Ingress失败: %w", err)
	}

	s.logger.Info("CreateIngress: Ingress创建成功",
		zap.String("name", req.Name),
		zap.String("namespace", req.Namespace))

	return nil
}

// GetIngressList 获取Ingress列表
func (s *ingressService) GetIngressList(ctx context.Context, req *model.GetIngressListReq) (model.ListResp[*model.K8sIngress], error) {
	if req == nil {
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("获取Ingress列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("集群ID不能为空")
	}

	// 构建查询选项
	listOptions := utils.BuildIngressListOptions(req)

	k8sIngresses, err := s.ingressManager.GetIngressList(ctx, req.ClusterID, req.Namespace, listOptions)
	if err != nil {
		s.logger.Error("GetIngressList: 获取Ingress列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sIngress]{}, fmt.Errorf("获取Ingress列表失败: %w", err)
	}

	// 根据状态过滤
	var filteredIngresses []*model.K8sIngress
	if req.Status != "" {
		// 根据状态过滤
		for _, k8sIngress := range k8sIngresses {
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
			if strings.EqualFold(statusStr, req.Status) {
				filteredIngresses = append(filteredIngresses, k8sIngress)
			}
		}
	} else {
		filteredIngresses = k8sIngresses
	}

	// 分页处理
	page := req.Page
	size := req.Size
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10 // 默认每页显示10条
	}

	pagedItems, total := utils.PaginateK8sIngresses(filteredIngresses, page, size)

	return model.ListResp[*model.K8sIngress]{
		Total: total,
		Items: pagedItems,
	}, nil
}

// GetIngressDetails 获取Ingress详情
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
		s.logger.Error("GetIngressDetails: 获取Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Ingress失败: %w", err)
	}

	return utils.ConvertToK8sIngress(ingress, req.ClusterID), nil
}

// GetIngressYaml 获取Ingress YAML
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
		s.logger.Error("GetIngressYaml: 获取Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Ingress失败: %w", err)
	}

	// 转换为YAML
	yamlContent, err := utils.IngressToYAML(ingress)
	if err != nil {
		s.logger.Error("GetIngressYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("ingressName", ingress.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdateIngress 更新Ingress
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

	// 获取现有Ingress
	existingIngress, err := s.ingressManager.GetIngress(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdateIngress: 获取现有Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有Ingress失败: %w", err)
	}

	updatedIngress := existingIngress.DeepCopy()

	// 更新基本字段
	if req.Labels != nil {
		updatedIngress.Labels = req.Labels
	}
	if req.Annotations != nil {
		updatedIngress.Annotations = req.Annotations
	}
	if req.IngressClassName != nil {
		updatedIngress.Spec.IngressClassName = req.IngressClassName
	}

	// 验证更新后的Ingress配置
	if err := utils.ValidateIngress(updatedIngress); err != nil {
		s.logger.Error("UpdateIngress: Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	err = s.ingressManager.UpdateIngress(ctx, req.ClusterID, req.Namespace, updatedIngress)
	if err != nil {
		s.logger.Error("UpdateIngress: 更新Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Ingress失败: %w", err)
	}

	return nil
}

// DeleteIngress 删除Ingress
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
		s.logger.Error("DeleteIngress: 删除Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Ingress失败: %w", err)
	}

	s.logger.Info("DeleteIngress: Ingress删除成功",
		zap.String("name", req.Name),
		zap.String("namespace", req.Namespace))

	return nil
}

// CreateIngressByYaml 通过YAML创建Ingress
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

	// 解析YAML为Ingress对象
	ingress, err := utils.YAMLToIngress(req.YAML)
	if err != nil {
		s.logger.Error("CreateIngressByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Ingress配置
	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("CreateIngressByYaml: Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", ingress.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	// 创建Ingress
	err = s.ingressManager.CreateIngress(ctx, req.ClusterID, ingress.Namespace, ingress)
	if err != nil {
		s.logger.Error("CreateIngressByYaml: 创建Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", ingress.Namespace),
			zap.String("name", ingress.Name))
		return fmt.Errorf("创建Ingress失败: %w", err)
	}

	s.logger.Info("CreateIngressByYaml: Ingress创建成功",
		zap.String("name", ingress.Name),
		zap.String("namespace", ingress.Namespace))

	return nil
}

// UpdateIngressByYaml 通过YAML更新Ingress
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

	// 解析YAML为Ingress对象
	ingress, err := utils.YAMLToIngress(req.YAML)
	if err != nil {
		s.logger.Error("UpdateIngressByYaml: 解析YAML失败",
			zap.Error(err),
			zap.Int("cluster_id", req.ClusterID))
		return fmt.Errorf("解析YAML失败: %w", err)
	}

	// 验证Ingress配置
	if err := utils.ValidateIngress(ingress); err != nil {
		s.logger.Error("UpdateIngressByYaml: Ingress配置验证失败",
			zap.Error(err),
			zap.String("name", ingress.Name))
		return fmt.Errorf("Ingress配置验证失败: %w", err)
	}

	// 更新Ingress
	err = s.ingressManager.UpdateIngress(ctx, req.ClusterID, ingress.Namespace, ingress)
	if err != nil {
		s.logger.Error("UpdateIngressByYaml: 更新Ingress失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", ingress.Namespace),
			zap.String("name", ingress.Name))
		return fmt.Errorf("更新Ingress失败: %w", err)
	}

	s.logger.Info("UpdateIngressByYaml: Ingress更新成功",
		zap.String("name", ingress.Name),
		zap.String("namespace", ingress.Namespace))

	return nil
}
