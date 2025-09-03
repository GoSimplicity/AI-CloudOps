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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SvcService interface {
	GetServiceList(ctx context.Context, req *model.GetServiceListReq) (model.ListResp[*model.K8sService], error)
	GetServiceDetails(ctx context.Context, req *model.GetServiceDetailsReq) (*model.K8sService, error)
	GetServiceYaml(ctx context.Context, req *model.GetServiceYamlReq) (*model.K8sYaml, error)
	CreateService(ctx context.Context, req *model.CreateServiceReq) error
	UpdateService(ctx context.Context, req *model.UpdateServiceReq) error
	// YAML相关方法
	CreateServiceByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error
	UpdateServiceByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error
	DeleteService(ctx context.Context, req *model.DeleteServiceReq) error
	GetServiceEndpoints(ctx context.Context, req *model.GetServiceEndpointsReq) ([]*model.K8sServiceEndpoint, error)
	GetServiceEvents(ctx context.Context, req *model.GetServiceEventsReq) ([]*model.K8sServiceEvent, error)
}

type svcService struct {
	serviceManager manager.ServiceManager
	client         client.K8sClient
	logger         *zap.Logger
}

func NewSvcService(serviceManager manager.ServiceManager, client client.K8sClient, logger *zap.Logger) SvcService {
	return &svcService{
		serviceManager: serviceManager,
		client:         client,
		logger:         logger,
	}
}

// CreateService 创建Service
func (s *svcService) CreateService(ctx context.Context, req *model.CreateServiceReq) error {
	if req == nil {
		return fmt.Errorf("创建Service请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 从请求构建Service对象
	service, err := utils.BuildServiceFromRequest(req)
	if err != nil {
		s.logger.Error("CreateService: 构建Service对象失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("构建Service对象失败: %w", err)
	}

	// 验证Service配置
	if err := utils.ValidateService(service); err != nil {
		s.logger.Error("CreateService: Service配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Service配置验证失败: %w", err)
	}

	// 使用ServiceManager创建Service
	_, err = s.serviceManager.CreateService(ctx, req.ClusterID, service)
	if err != nil {
		s.logger.Error("CreateService: 创建Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("创建Service失败: %w", err)
	}

	s.logger.Info("CreateService: Service创建成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// DeleteService 删除Service
func (s *svcService) DeleteService(ctx context.Context, req *model.DeleteServiceReq) error {
	if req == nil {
		return fmt.Errorf("删除Service请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 使用ServiceManager删除Service
	err := s.serviceManager.DeleteService(ctx, req.ClusterID, req.Namespace, req.Name, metav1.DeleteOptions{})
	if err != nil {
		s.logger.Error("DeleteService: 删除Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("删除Service失败: %w", err)
	}

	s.logger.Info("DeleteService: Service删除成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// GetServiceDetails 获取Service详情
func (s *svcService) GetServiceDetails(ctx context.Context, req *model.GetServiceDetailsReq) (*model.K8sService, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Service详情请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	// 使用ServiceManager获取Service
	service, err := s.serviceManager.GetService(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetServiceDetails: 获取Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Service失败: %w", err)
	}

	// 获取Kubernetes客户端
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("GetServiceDetails: 获取Kubernetes客户端失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 构建详细Service信息
	k8sService, err := utils.BuildK8sService(ctx, req.ClusterID, *service, kubeClient)
	if err != nil {
		s.logger.Error("GetServiceDetails: 构建Service详细信息失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("构建Service详细信息失败: %w", err)
	}

	return k8sService, nil
}

// GetServiceEndpoints 获取Service端点
func (s *svcService) GetServiceEndpoints(ctx context.Context, req *model.GetServiceEndpointsReq) ([]*model.K8sServiceEndpoint, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Service端点请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	// 使用ServiceManager获取Service端点
	endpoints, err := s.serviceManager.GetServiceEndpoints(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetServiceEndpoints: 获取Service端点失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Service端点失败: %w", err)
	}

	// 转换为响应格式
	var serviceEndpoints []*model.K8sServiceEndpoint

	// 如果Endpoints为空或没有Subsets，返回空列表
	if endpoints == nil || len(endpoints.Subsets) == 0 {
		s.logger.Info("Service Endpoints为空，返回空列表",
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return serviceEndpoints, nil
	}

	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				endpoint := &model.K8sServiceEndpoint{
					IP:       address.IP,
					Port:     port.Port,
					Protocol: string(port.Protocol),
					Ready:    true,
				}
				serviceEndpoints = append(serviceEndpoints, endpoint)
			}
		}

		// 处理未就绪的地址
		for _, address := range subset.NotReadyAddresses {
			for _, port := range subset.Ports {
				endpoint := &model.K8sServiceEndpoint{
					IP:       address.IP,
					Port:     port.Port,
					Protocol: string(port.Protocol),
					Ready:    false,
				}
				serviceEndpoints = append(serviceEndpoints, endpoint)
			}
		}
	}

	return serviceEndpoints, nil
}

// GetServiceEvents 获取Service事件
func (s *svcService) GetServiceEvents(ctx context.Context, req *model.GetServiceEventsReq) ([]*model.K8sServiceEvent, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Service事件请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	// 获取Kubernetes客户端
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("GetServiceEvents: 获取Kubernetes客户端失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return nil, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 设置默认限制
	limit := 50
	if req.Limit > 0 {
		limit = req.Limit
	}

	// 使用utils获取Service事件
	events, err := utils.GetServiceEvents(ctx, kubeClient, req.Namespace, req.Name, limit)
	if err != nil {
		s.logger.Error("GetServiceEvents: 获取Service事件失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Service事件失败: %w", err)
	}

	// 根据事件类型过滤
	if req.EventType != "" {
		var filteredEvents []*model.K8sServiceEvent
		for _, event := range events {
			if event.Type == req.EventType {
				filteredEvents = append(filteredEvents, event)
			}
		}
		events = filteredEvents
	}

	return events, nil
}

// GetServiceList 获取Service列表
func (s *svcService) GetServiceList(ctx context.Context, req *model.GetServiceListReq) (model.ListResp[*model.K8sService], error) {
	if req == nil {
		return model.ListResp[*model.K8sService]{}, fmt.Errorf("获取Service列表请求不能为空")
	}

	if req.ClusterID <= 0 {
		return model.ListResp[*model.K8sService]{}, fmt.Errorf("集群ID不能为空")
	}

	// 使用ServiceManager获取Service列表
	serviceList, err := s.serviceManager.ListServices(ctx, req.ClusterID, req.Namespace)
	if err != nil {
		s.logger.Error("GetServiceList: 获取Service列表失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace))
		return model.ListResp[*model.K8sService]{}, fmt.Errorf("获取Service列表失败: %w", err)
	}

	services := serviceList.Items

	// 根据类型过滤
	if req.Type != "" {
		services = utils.FilterServicesByType(services, req.Type)
	}

	// 分页处理
	pagedServices, total := utils.BuildServiceListPagination(services, req.Page, req.Size)

	// 获取Kubernetes客户端
	kubeClient, err := s.client.GetKubeClient(req.ClusterID)
	if err != nil {
		s.logger.Error("GetServiceList: 获取Kubernetes客户端失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID))
		return model.ListResp[*model.K8sService]{}, fmt.Errorf("获取Kubernetes客户端失败: %w", err)
	}

	// 转换为响应格式
	var items []*model.K8sService
	for _, service := range pagedServices {
		k8sService, err := utils.BuildK8sService(ctx, req.ClusterID, service, kubeClient)
		if err != nil {
			s.logger.Warn("GetServiceList: 构建Service信息失败",
				zap.Error(err),
				zap.String("serviceName", service.Name))
			continue
		}
		items = append(items, k8sService)
	}

	return model.ListResp[*model.K8sService]{
		Total: total,
		Items: items,
	}, nil
}

// GetServiceYaml 获取Service YAML
func (s *svcService) GetServiceYaml(ctx context.Context, req *model.GetServiceYamlReq) (*model.K8sYaml, error) {
	if req == nil {
		return nil, fmt.Errorf("获取Service YAML请求不能为空")
	}

	if req.ClusterID <= 0 {
		return nil, fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return nil, fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return nil, fmt.Errorf("命名空间不能为空")
	}

	// 使用ServiceManager获取Service
	service, err := s.serviceManager.GetService(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("GetServiceYaml: 获取Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("获取Service失败: %w", err)
	}

	// 转换为YAML格式
	yamlContent, err := utils.ServiceToYAML(service)
	if err != nil {
		s.logger.Error("GetServiceYaml: 转换为YAML失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return nil, fmt.Errorf("转换为YAML失败: %w", err)
	}

	return &model.K8sYaml{
		YAML: yamlContent,
	}, nil
}

// UpdateService 更新Service
func (s *svcService) UpdateService(ctx context.Context, req *model.UpdateServiceReq) error {
	if req == nil {
		return fmt.Errorf("更新Service请求不能为空")
	}

	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}

	if req.Name == "" {
		return fmt.Errorf("Service名称不能为空")
	}

	if req.Namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}

	// 获取现有Service
	existingService, err := s.serviceManager.GetService(ctx, req.ClusterID, req.Namespace, req.Name)
	if err != nil {
		s.logger.Error("UpdateService: 获取现有Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("获取现有Service失败: %w", err)
	}

	updatedService := existingService.DeepCopy()

	// 如果提供了YAML，使用YAML内容
	if req.YAML != "" {
		yamlService, err := utils.YAMLToService(req.YAML)
		if err != nil {
			s.logger.Error("UpdateService: 解析YAML失败",
				zap.Error(err),
				zap.String("name", req.Name))
			return fmt.Errorf("解析YAML失败: %w", err)
		}
		updatedService.Spec = yamlService.Spec
		updatedService.Labels = yamlService.Labels
		updatedService.Annotations = yamlService.Annotations
	} else {
		// 更新基本字段
		if req.Type != "" {
			updatedService.Spec.Type = corev1.ServiceType(req.Type)
		}
		if req.Ports != nil {
			updatedService.Spec.Ports = utils.ConvertToCorePorts(req.Ports)
		}
		if req.Selector != nil {
			updatedService.Spec.Selector = req.Selector
		}
		if req.Labels != nil {
			updatedService.Labels = req.Labels
		}
		if req.Annotations != nil {
			updatedService.Annotations = req.Annotations
		}
	}

	// 验证更新后的Service配置
	if err := utils.ValidateService(updatedService); err != nil {
		s.logger.Error("UpdateService: Service配置验证失败",
			zap.Error(err),
			zap.String("name", req.Name))
		return fmt.Errorf("Service配置验证失败: %w", err)
	}

	// 使用ServiceManager更新Service
	_, err = s.serviceManager.UpdateService(ctx, req.ClusterID, updatedService)
	if err != nil {
		s.logger.Error("UpdateService: 更新Service失败",
			zap.Error(err),
			zap.Int("clusterID", req.ClusterID),
			zap.String("namespace", req.Namespace),
			zap.String("name", req.Name))
		return fmt.Errorf("更新Service失败: %w", err)
	}

	s.logger.Info("UpdateService: Service更新成功",
		zap.Int("clusterID", req.ClusterID),
		zap.String("namespace", req.Namespace),
		zap.String("name", req.Name))

	return nil
}

// CreateServiceByYaml 通过YAML创建Service
func (s *svcService) CreateServiceByYaml(ctx context.Context, req *model.CreateResourceByYamlReq) error {
	// TODO: 实现通过YAML创建Service的逻辑
	return fmt.Errorf("CreateServiceByYaml方法暂未实现")
}

// UpdateServiceByYaml 通过YAML更新Service
func (s *svcService) UpdateServiceByYaml(ctx context.Context, req *model.UpdateResourceByYamlReq) error {
	// TODO: 实现通过YAML更新Service的逻辑
	return fmt.Errorf("UpdateServiceByYaml方法暂未实现")
}
