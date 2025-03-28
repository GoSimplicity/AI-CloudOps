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

type ProjectService interface {
	CreateProjectOne(ctx context.Context, project *model.K8sProject) error
	GetprojectList(ctx context.Context) ([]model.K8sProject, error)
	GetprojectListByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error)
	DeleteProjectOne(ctx context.Context, id int64) error
	UpdateProjectOne(ctx context.Context, id int64, project *model.K8sProject) error
}
type projectService struct {
	client      client.K8sClient
	l           *zap.Logger
	dao         admin.ClusterDAO
	projectdao  user.ProjectDAO
	appdao      user.AppDAO
	instancedao user.InstanceDAO
}

func NewProjectService(dao admin.ClusterDAO, projectdao user.ProjectDAO, appdao user.AppDAO, instancedao user.InstanceDAO, client client.K8sClient, l *zap.Logger) ProjectService {
	return &projectService{
		dao:         dao,
		instancedao: instancedao,
		appdao:      appdao,
		projectdao:  projectdao,
		client:      client,
		l:           l,
	}
}

func (a *projectService) CreateProjectOne(ctx context.Context, project *model.K8sProject) error {
	// 0.先入数据库
	err := a.projectdao.CreateProjectOne(ctx, project)
	if err != nil {
		return fmt.Errorf("failed to create project in db: %w", err)
	}
	//1.开始创建K8SAPP
	for i := 0; i < len(project.K8sApps); i++ {
		for j := 0; j < len(project.K8sApps[i].K8sInstances); j++ {
			instance := project.K8sApps[i].K8sInstances[j]

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
		}
	}
	return nil
}
func (a *projectService) GetprojectList(ctx context.Context) ([]model.K8sProject, error) {
	projectList, err := a.projectdao.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project list: %w", err)
	}
	return projectList, nil
}

func (a *projectService) GetprojectListByIds(ctx context.Context, ids []int64) ([]model.K8sProject, error) {
	projectList, err := a.projectdao.GetByIds(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get project list by Ids: %w", err)
	}
	return projectList, nil
}
func (a *projectService) DeleteProjectOne(ctx context.Context, id int64) error {
	// 1.先通过id将k8sproject中的字段逻辑删除
	_, err := a.projectdao.DeleteProjectById(ctx, id)
	if err != nil {
		return fmt.Errorf("项目软删除失败: %w", err)
	}

	// 2.通过id到k8sApps中查找，并且也是将其软删除
	apps, err := a.appdao.GetAppsByProjectId(ctx, id)
	if err != nil {
		return fmt.Errorf("获取项目应用失败: %w", err)
	}
	// 3.通过k8sapps到k8sinstance中查找，并且将其软删除，同时需要记录其信息
	var allInstances []model.K8sInstance
	for _, app := range apps {
		// 软删除应用
		if _, err := a.appdao.DeleteAppById(ctx, int64(app.ID)); err != nil {
			a.l.Warn("应用软删除失败", zap.Int64("appId", int64(app.ID)))
		}
		// 3. 获取并软删除实例
		instances, err := a.instancedao.GetInstanceByApp(ctx, int64(app.ID))
		if err != nil {
			continue
		}
		instanceIDs := make([]int64, len(instances))
		for i, inst := range instances {
			instanceIDs[i] = int64(inst.ID)
		}
		if err := a.instancedao.DeleteInstanceByIds(ctx, instanceIDs); err != nil {
			a.l.Warn("实例软删除失败", zap.Int64s("instanceIds", instanceIDs))
		}
		allInstances = append(allInstances, instances...)
	}
	// 调用pkg中的删除Deployment和Service的删除instances
	for i := 0; i < len(allInstances); i++ {
		instance := allInstances[i]
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

func (a *projectService) UpdateProjectOne(ctx context.Context, id int64, project *model.K8sProject) error {
	// 1.更新Project
	if err := a.projectdao.UpdateProjectById(ctx, id, *project); err != nil {
		return fmt.Errorf("更新项目失败: %w", err)
	}
	for _, app := range project.K8sApps {
		for _, instance := range app.K8sInstances {
			// 转换实例为K8s资源
			deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
			if err != nil {
				return fmt.Errorf("failed to parse instance %d: %w", instance.ID, err)
			}

			// 获取集群信息
			cluster, err := a.dao.GetClusterByName(ctx, instance.Cluster)
			if err != nil {
				return fmt.Errorf("failed to get cluster for instance %d: %w", instance.ID, err)
			}

			// 删除Deployment
			depReq := model.K8sDeploymentRequest{
				ClusterId:       cluster.ID,
				Namespace:       instance.Namespace,
				DeploymentNames: []string{deployment.Name},
				DeploymentYaml:  &deployment,
			}
			if err := pkg.UpdateDeployment(ctx, &depReq, a.client, a.l); err != nil {
				return fmt.Errorf("failed to update deployment %s: %w", deployment.Name, err)
			}

			// 删除Service
			svcReq := model.K8sServiceRequest{
				ClusterId:    cluster.ID,
				Namespace:    instance.Namespace,
				ServiceNames: []string{service.Name},
				ServiceYaml:  &service,
			}
			if err := pkg.UpdateService(ctx, &svcReq, a.client, a.l); err != nil {
				return fmt.Errorf("failed to update service %s: %w", service.Name, err)
			}
		}
	}
	return nil
}
