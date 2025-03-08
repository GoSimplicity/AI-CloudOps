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
	"gorm.io/gorm"
)

type AppService interface {
	// 实例
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error
	UpdateInstanceOne(ctx context.Context, id int64, instance model.K8sInstance) error
	BatchDeleteInstance(ctx context.Context, ids []int64) error
	BatchRestartInstance(ctx context.Context, ids []int64) error
	GetInstanceByApp(ctx context.Context, appId int64) ([]model.K8sInstance, error)
	GetInstanceOne(ctx context.Context, instanceId int64) (model.K8sInstance, error)
	GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error)
	// 应用
	CreateAppOne(ctx context.Context, app *model.K8sApp) error
	GetAppOne(ctx context.Context, id int64) (model.K8sApp, error)
	DeleteAppOne(ctx context.Context, id int64) error
	UpdateAppOne(ctx context.Context, id int64, app model.K8sApp) error
	GetAppByIds(ctx context.Context, ids []int64) ([]model.K8sApp, error)
	GetPodListByDeploy(ctx context.Context, id int64) ([]model.Resource, error)
	// 项目
	CreateProjectOne(ctx context.Context, project *model.K8sProject) error
}
type appService struct {
	dao         admin.ClusterDAO
	appdao      uesr.AppDAO
	instancedao uesr.InstanceDAO
	projectdao  uesr.ProjectDAO
	client      client.K8sClient
	l           *zap.Logger
}

func NewAppService(dao admin.ClusterDAO, appdao uesr.AppDAO, instancedao uesr.InstanceDAO, projectdao uesr.ProjectDAO, client client.K8sClient, l *zap.Logger) AppService {
	return &appService{
		dao:         dao,
		appdao:      appdao,
		instancedao: instancedao,
		projectdao:  projectdao,
		client:      client,
		l:           l,
	}
}

func (a *appService) CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error {
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

func (a *appService) UpdateInstanceOne(ctx context.Context, id int64, instance model.K8sInstance) error {
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

func (a *appService) BatchDeleteInstance(ctx context.Context, ids []int64) error {
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
func (a *appService) BatchRestartInstance(ctx context.Context, ids []int64) error {
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

func (a *appService) GetInstanceByApp(ctx context.Context, appId int64) ([]model.K8sInstance, error) {
	// 1.根据appId获取实例列表
	instances, err := a.instancedao.GetInstanceByApp(ctx, appId)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	return instances, nil
}

func (a *appService) GetInstanceOne(ctx context.Context, instanceId int64) (model.K8sInstance, error) {
	// 1.根据instanceId获取实例
	instance, err := a.instancedao.GetInstanceById(ctx, instanceId)
	if err != nil {
		return model.K8sInstance{}, fmt.Errorf("failed to get Deployment: %w", err)
	}
	return instance, nil
}
func (a *appService) GetInstanceAll(ctx context.Context) ([]model.K8sInstance, error) {
	allinstances, err := a.instancedao.GetInstanceAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 3.返回实例列表
	return allinstances, nil
}
func (a *appService) CreateAppOne(ctx context.Context, app *model.K8sApp) error {
	// 0.先入数据库
	err0 := a.appdao.CreateAppOne(ctx, app)
	if err0 != nil {
		return fmt.Errorf("failed to create app in db: %w", err0)
	}
	// 1.创建实例
	for i := 0; i < len(app.K8sInstances); i++ {
		instance := app.K8sInstances[i]
		// 将instance转换成deployment和service内容
		deployment, service, err := pkg.ParseK8sInstance(ctx, &instance)
		if err != nil {
			return fmt.Errorf("failed to 转换 deployment, service: %w", err0)
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
	return nil
}
func (a *appService) DeleteAppOne(ctx context.Context, id int64) error {
	// 0.操作数据库
	app, err := a.appdao.DeleteAppById(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete app in db: %w", err)
	}

	// 1. 查找关联的instances
	instances, err := a.instancedao.GetInstanceByApp(ctx, int64(app.ID))
	for _, instance := range instances {
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
		if err := pkg.DeleteDeployment(ctx, &depReq, a.client, a.l); err != nil {
			return fmt.Errorf("failed to delete deployment %s: %w", deployment.Name, err)
		}

		// 删除Service
		svcReq := model.K8sServiceRequest{
			ClusterId:    cluster.ID,
			Namespace:    instance.Namespace,
			ServiceNames: []string{service.Name},
			ServiceYaml:  &service,
		}
		if err := pkg.DeleteService(ctx, &svcReq, a.client, a.l); err != nil {
			return fmt.Errorf("failed to delete service %s: %w", service.Name, err)
		}
	}
	return nil
}
func (a *appService) GetAppOne(ctx context.Context, id int64) (model.K8sApp, error) {
	app, err := a.appdao.GetAppById(ctx, id)
	if err != nil {
		return model.K8sApp{}, fmt.Errorf("failed to get app in db: %w", err)
	}
	return app, nil
}
func (a *appService) UpdateAppOne(ctx context.Context, id int64, app model.K8sApp) error {
	// 更新数据库
	err := a.appdao.UpdateAppById(ctx, id, app)
	if err != nil {
		return fmt.Errorf("failed to update app in db: %w", err)
	}
	// 开始实例更新
	// TODO:存在DB-instances不统一问题
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
	return nil
}

func (a *appService) GetAppByIds(ctx context.Context, ids []int64) ([]model.K8sApp, error) {
	var apps []model.K8sApp
	for _, id := range ids {
		app, err := a.appdao.GetAppById(ctx, id)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				a.l.Warn("部分应用不存在", zap.Int64("appId", id))
				continue
			}
			a.l.Error("批量查询应用失败", zap.Int64("appId", id), zap.Error(err))
			return nil, fmt.Errorf("failed to get app %d: %w", id, err)
		}
		apps = append(apps, app)
	}

	if len(apps) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return apps, nil
}
func (a *appService) GetPodListByDeploy(ctx context.Context, id int64) ([]model.Resource, error) {
	// 1. 根据 id 获取应用信息
	app, err := a.appdao.GetAppById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get app by id: %w", err)
	}

	// 2. 通过 clustername 获取集群
	k8scluster, err := a.dao.GetClusterByName(ctx, app.Cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	kubeClient, err := pkg.GetKubeClient(k8scluster.ID, a.client, a.l)
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes client: %w", err)
	}
	resources, err := pkg.GetPodResources(ctx, kubeClient, app.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}
	return resources, nil
}
func (a *appService) CreateProjectOne(ctx context.Context, project *model.K8sProject) error {
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
