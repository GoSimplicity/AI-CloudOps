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

package user

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

// InstanceService 定义 Kubernetes 实例服务接口
type InstanceService interface {
	// 实例管理相关方法
	CreateInstance(ctx context.Context, req *model.CreateK8sInstanceRequest) error
	UpdateInstance(ctx context.Context, req *model.UpdateK8sInstanceRequest) error
	BatchDeleteInstance(ctx context.Context, instanceIDs []int64) error
	BatchRestartInstance(ctx context.Context, instanceIDs []int64) error
	GetInstanceByApp(ctx context.Context, appID int64) ([]model.K8sInstance, error)
	GetInstance(ctx context.Context, instanceID int64) (model.K8sInstance, error)
	GetInstanceList(ctx context.Context, req *model.GetK8sInstanceListRequest) ([]model.K8sInstance, error)
}

// instanceService 实现 InstanceService 接口
type instanceService struct {
	client      client.K8sClient
	logger      *zap.Logger
	clusterDAO  admin.ClusterDAO
	instanceDAO user.InstanceDAO
}

// NewInstanceService 创建新的实例服务
func NewInstanceService(clusterDAO admin.ClusterDAO, instanceDAO user.InstanceDAO, client client.K8sClient, logger *zap.Logger) InstanceService {
	return &instanceService{
		clusterDAO:  clusterDAO,
		client:      client,
		logger:      logger,
		instanceDAO: instanceDAO,
	}
}

// CreateInstance 创建 Kubernetes 实例
func (s *instanceService) CreateInstance(ctx context.Context, req *model.CreateK8sInstanceRequest) error {
	// 1. 先入数据库
	instance := &model.K8sInstance{}

	err := s.instanceDAO.CreateInstanceOne(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	// 2. 将instance转换成deployment和service内容
	deployment, service, err := pkg.ParseK8sInstance(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to parse instance: %w", err)
	}

	// 3. 通过clustername获取集群
	k8sCluster, err := s.clusterDAO.GetClusterByName(ctx, instance.Cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 4. 调用创建函数
	deploymentRequest := model.K8sDeploymentRequest{
		ClusterId:       k8sCluster.ID,
		Namespace:       instance.Namespace,
		DeploymentNames: []string{deployment.Name},
		DeploymentYaml:  &deployment,
	}

	// 5. 创建 Deployment
	err = pkg.CreateDeployment(ctx, &deploymentRequest, s.client, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// 6. 创建 Service
	serviceRequest := model.K8sServiceRequest{
		ClusterId:    k8sCluster.ID,
		Namespace:    instance.Namespace,
		ServiceNames: []string{service.Name},
		ServiceYaml:  &service,
	}

	err = pkg.CreateService(ctx, &serviceRequest, s.client, s.logger)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	return nil
}

// UpdateInstance 更新 Kubernetes 实例
func (s *instanceService) UpdateInstance(ctx context.Context, req *model.UpdateK8sInstanceRequest) error {
	// 更新实例信息
	instance := model.K8sInstance{
		Cluster:   req.Cluster,
		Image:     req.Image,
		Replicas:  req.Replicas,
		Namespace: req.Namespace,
		K8sAppID:  req.K8sAppID,
		ContainerCore: model.ContainerCore{
			CpuLimit:      req.ContainerCore.CpuLimit,
			MemoryLimit:   req.ContainerCore.MemoryLimit,
			VolumeJson:    req.ContainerCore.VolumeJson,
			PortJson:      req.ContainerCore.PortJson,
			Envs:          req.ContainerCore.Envs,
			Labels:        req.ContainerCore.Labels,
			Commands:      req.ContainerCore.Commands,
			Args:          req.ContainerCore.Args,
			CpuRequest:    req.ContainerCore.CpuRequest,
			MemoryRequest: req.ContainerCore.MemoryRequest,
		},
	}

	err := s.instanceDAO.UpdateInstanceById(ctx, int64(req.ID), instance)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	// 将实例转换成 Deployment 和 Service
	deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
	if err != nil {
		return fmt.Errorf("failed to parse k8s instance: %w", err)
	}

	// 4. 获取集群信息
	k8sCluster, err := s.clusterDAO.GetClusterByName(ctx, instance.Cluster)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 5. 更新 Deployment
	deploymentRequest := model.K8sDeploymentRequest{
		ClusterId:       k8sCluster.ID,
		Namespace:       instance.Namespace,
		DeploymentNames: []string{deployment.Name},
		DeploymentYaml:  &deployment,
	}

	if err := pkg.UpdateDeployment(ctx, &deploymentRequest, s.client, s.logger); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// 6. 更新 Service
	serviceRequest := model.K8sServiceRequest{
		ClusterId:    k8sCluster.ID,
		Namespace:    instance.Namespace,
		ServiceNames: []string{service.Name},
		ServiceYaml:  &service,
	}

	if err := pkg.UpdateService(ctx, &serviceRequest, s.client, s.logger); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}

	return nil
}

// BatchDeleteInstance 批量删除 Kubernetes 实例
func (s *instanceService) BatchDeleteInstance(ctx context.Context, instanceIDs []int64) error {
	// 1. 从 DB 中取出内容
	instances, err := s.instanceDAO.GetInstanceByIds(ctx, instanceIDs)
	if err != nil {
		return fmt.Errorf("failed to get instances: %w", err)
	}

	// 2. 从数据库中删除实例记录
	err = s.instanceDAO.DeleteInstanceByIds(ctx, instanceIDs)
	if err != nil {
		return fmt.Errorf("failed to delete instances from database: %w", err)
	}

	// 3. 从 Kubernetes 集群中删除实例
	for _, instance := range instances {
		// 将 instance 转换成 deployment 和 service
		deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
		if err != nil {
			return fmt.Errorf("failed to convert to deployment and service: %w", err)
		}

		// 获取集群信息
		k8sCluster, err := s.clusterDAO.GetClusterByName(ctx, instance.Cluster)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}

		// 删除 Deployment
		deploymentRequest := model.K8sDeploymentRequest{
			ClusterId:       k8sCluster.ID,
			Namespace:       instance.Namespace,
			DeploymentNames: []string{deployment.Name},
			DeploymentYaml:  &deployment,
		}

		err = pkg.DeleteDeployment(ctx, &deploymentRequest, s.client, s.logger)
		if err != nil {
			s.logger.Error("Failed to delete deployment", zap.Error(err))
		}

		// 删除 Service
		serviceRequest := model.K8sServiceRequest{
			ClusterId:    k8sCluster.ID,
			Namespace:    instance.Namespace,
			ServiceNames: []string{service.Name},
			ServiceYaml:  &service,
		}

		err = pkg.DeleteService(ctx, &serviceRequest, s.client, s.logger)
		if err != nil {
			s.logger.Error("Failed to delete service", zap.Error(err))
		}
	}

	return nil
}

// BatchRestartInstance 批量重启 Kubernetes 实例
func (s *instanceService) BatchRestartInstance(ctx context.Context, instanceIDs []int64) error {
	// 1. 从 DB 中取出内容
	instances, err := s.instanceDAO.GetInstanceByIds(ctx, instanceIDs)
	if err != nil {
		return fmt.Errorf("failed to get instances: %w", err)
	}

	var deploymentRequests []model.K8sDeploymentRequest

	for _, instance := range instances {
		// 将 instance 转换成 deployment
		deployment, _, err := pkg.ParseK8sInstance(ctx, &instance)
		if err != nil {
			return fmt.Errorf("failed to convert to deployment: %w", err)
		}

		// 获取集群信息
		k8sCluster, err := s.clusterDAO.GetClusterByName(ctx, instance.Cluster)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}

		deploymentRequest := model.K8sDeploymentRequest{
			ClusterId:       k8sCluster.ID,
			Namespace:       instance.Namespace,
			DeploymentNames: []string{deployment.Name},
			DeploymentYaml:  &deployment,
		}

		deploymentRequests = append(deploymentRequests, deploymentRequest)
	}

	err = pkg.BatchRestartK8sInstance(ctx, deploymentRequests, s.client, s.logger)
	if err != nil {
		return fmt.Errorf("failed to restart instances: %w", err)
	}

	return nil
}

// GetInstanceByApp 根据应用获取 Kubernetes 实例列表
func (s *instanceService) GetInstanceByApp(ctx context.Context, appID int64) ([]model.K8sInstance, error) {
	instances, err := s.instanceDAO.GetInstanceByApp(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances by app: %w", err)
	}

	return instances, nil
}

// GetInstance 获取单个 Kubernetes 实例
func (s *instanceService) GetInstance(ctx context.Context, instanceID int64) (model.K8sInstance, error) {
	instance, err := s.instanceDAO.GetInstanceById(ctx, instanceID)
	if err != nil {
		return model.K8sInstance{}, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// GetInstanceList 获取 Kubernetes 实例列表
func (s *instanceService) GetInstanceList(ctx context.Context, req *model.GetK8sInstanceListRequest) ([]model.K8sInstance, error) {
	// TODO: 实现基于请求参数的筛选逻辑
	allInstances, err := s.instanceDAO.GetInstanceAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	return allInstances, nil
}
