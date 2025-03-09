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

package uesr

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/uesr"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
)

type InstanceService interface {
	// 实例
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error
	UpdateInstanceOne(ctx context.Context, id int64, instance model.K8sInstance) error
	BatchDeleteInstance(ctx context.Context, ids []int64) error
	BatchRestartInstance(ctx context.Context, ids []int64) error
	GetInstanceByApp(ctx context.Context, appId int64) ([]model.K8sInstance, error)
	GetInstanceOne(ctx context.Context, instanceId int64) (model.K8sInstance, error)
	GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error)
}
type instanceService struct {
	client      client.K8sClient
	l           *zap.Logger
	dao         admin.ClusterDAO
	instancedao uesr.InstanceDAO
}

func NewInstanceService(dao admin.ClusterDAO, instancedao uesr.InstanceDAO, client client.K8sClient, l *zap.Logger) InstanceService {
	return &instanceService{
		dao:         dao,
		client:      client,
		l:           l,
		instancedao: instancedao,
	}
}

func (a *instanceService) CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error {
	//0 先入数据库
	err := a.instancedao.CreateInstanceOne(ctx, instance) // 单侧先删除外键：ALTER TABLE k8s_instances DROP FOREIGN KEY fk_k8s_apps_k8s_instances;
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}
	// 将instance转换成deployment和service内容
	deployment, service, err := pkg.ParseK8sInstance(ctx, instance)
	// 2.通过clustername获取集群
	k8scluster, err2 := a.dao.GetClusterByName(ctx, instance.Cluster)
	if err2 != nil {
		return fmt.Errorf("failed to get Cluster: %w", err2)
	}
	// 调用创建函数
	deploymentRequest := model.K8sDeploymentRequest{
		ClusterId:       k8scluster.ID,
		Namespace:       instance.Namespace,
		DeploymentNames: []string{deployment.Name},
		DeploymentYaml:  &deployment,
	}
	// 调用deploymentService的CreateDeployment方法创建deployment
	pkg.CreateDeployment(ctx, &deploymentRequest, a.client, a.l)
	//
	serviceRequest := model.K8sServiceRequest{
		ClusterId:    k8scluster.ID,
		Namespace:    instance.Namespace,
		ServiceNames: []string{service.Name},
		ServiceYaml:  &service,
	}
	// 调用svcService的CreateService方法创建service
	pkg.CreateService(ctx, &serviceRequest, a.client, a.l)
	return nil
}

func (a *instanceService) UpdateInstanceOne(ctx context.Context, id int64, instance model.K8sInstance) error {
	// 1.从DB中取出具体的内容，然后更新=>[DB层面]
	_, err := a.instancedao.GetInstanceById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}
	// 2.将instance转换成deployment和service内容
	err = a.instancedao.UpdateInstanceById(ctx, id, instance)
	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}
	// 3. 将instance转换成deployment和service内容
	deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
	if err != nil {
		return fmt.Errorf("failed to parse k8s instance: %w", err)
	}

	// 4. 获取集群信息
	k8scluster, err := a.dao.GetClusterByName(ctx, instance.Cluster)
	if err != nil {
		return fmt.Errorf("failed to get Cluster: %w", err)
	}

	// 5. 更新Deployment
	deploymentRequest := model.K8sDeploymentRequest{
		ClusterId:       k8scluster.ID,
		Namespace:       instance.Namespace,
		DeploymentNames: []string{deployment.Name},
		DeploymentYaml:  &deployment,
	}
	if err := pkg.UpdateDeployment(ctx, &deploymentRequest, a.client, a.l); err != nil {
		return fmt.Errorf("failed to update deployment: %w", err)
	}

	// 6. 更新Service
	serviceRequest := model.K8sServiceRequest{
		ClusterId:    k8scluster.ID,
		Namespace:    instance.Namespace,
		ServiceNames: []string{service.Name},
		ServiceYaml:  &service,
	}
	if err := pkg.UpdateService(ctx, &serviceRequest, a.client, a.l); err != nil {
		return fmt.Errorf("failed to update service: %w", err)
	}
	return nil
}

func (a *instanceService) BatchDeleteInstance(ctx context.Context, ids []int64) error {
	// 1.从DB中取出内容
	instances, err := a.instancedao.GetInstanceByIds(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 2.然后删除对应的instances信息
	err = a.instancedao.DeleteInstanceByIds(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to delete Deployment: %w", err)
	}
	// 3.接着需要删除对应的实例
	for i := 0; i < len(instances); i++ {
		instance := instances[i]
		// 将instance转换成deployment和service内容
		deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
		if err != nil {
			return fmt.Errorf("failed to 转换 deployment, service: %w", err)
		}
		// 2.通过clustername获取集群
		k8scluster, err2 := a.dao.GetClusterByName(ctx, instance.Cluster)
		if err2 != nil {
			return fmt.Errorf("failed to get Cluster: %w", err2)
		}
		// 调用deploymentService的DeleteDeployment方法删除deployment
		deploymentRequest := model.K8sDeploymentRequest{
			ClusterId:       k8scluster.ID,
			Namespace:       instance.Namespace,
			DeploymentNames: []string{deployment.Name},
			DeploymentYaml:  &deployment,
		}
		pkg.DeleteDeployment(ctx, &deploymentRequest, a.client, a.l)
		//	调用svcService的DeleteService方法删除service
		serviceRequest := model.K8sServiceRequest{
			ClusterId:    k8scluster.ID,
			Namespace:    instance.Namespace,
			ServiceNames: []string{service.Name},
			ServiceYaml:  &service,
		}
		pkg.DeleteService(ctx, &serviceRequest, a.client, a.l)
	}
	return nil
}
func (a *instanceService) BatchRestartInstance(ctx context.Context, ids []int64) error {
	// 1.从DB中取出内容
	instances, err := a.instancedao.GetInstanceByIds(ctx, ids)
	if err != nil {
		return fmt.Errorf("failed to get Deployment: %w", err)
	}
	var deploymentRequests []model.K8sDeploymentRequest
	for i := 0; i < len(instances); i++ {
		instance := instances[i]

		// 将instance转换成deployment和service内容
		deployment, _, err := pkg.ParseK8sInstance(ctx, &instance)
		if err != nil {
			return fmt.Errorf("failed to 转换 deployment, service: %w", err)
		}

		// 2.通过clustername获取集群
		k8scluster, err2 := a.dao.GetClusterByName(ctx, instance.Cluster)
		if err2 != nil {
			return fmt.Errorf("failed to get Cluster: %w", err2)
		}
		deploymentRequest := model.K8sDeploymentRequest{
			ClusterId:       k8scluster.ID,
			Namespace:       instance.Namespace,
			DeploymentNames: []string{deployment.Name},
			DeploymentYaml:  &deployment,
		}
		deploymentRequests = append(deploymentRequests, deploymentRequest)

	}
	pkg.BatchRestartK8sInstance(ctx, deploymentRequests, a.client, a.l)
	return nil
}

func (a *instanceService) GetInstanceByApp(ctx context.Context, appId int64) ([]model.K8sInstance, error) {
	// 1.根据appId获取实例列表
	instances, err := a.instancedao.GetInstanceByApp(ctx, appId)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	return instances, nil
}

func (a *instanceService) GetInstanceOne(ctx context.Context, instanceId int64) (model.K8sInstance, error) {
	// 1.根据instanceId获取实例
	instance, err := a.instancedao.GetInstanceById(ctx, instanceId)
	if err != nil {
		return model.K8sInstance{}, fmt.Errorf("failed to get Deployment: %w", err)
	}
	return instance, nil
}
func (a *instanceService) GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error) {
	allinstances, err := a.instancedao.GetInstanceAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 3.返回实例列表
	return allinstances, nil
}
