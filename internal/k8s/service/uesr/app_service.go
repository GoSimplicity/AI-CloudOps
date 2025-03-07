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

type AppService interface {
	// 实例
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstance) error
	UpdateInstanceOne(ctx context.Context, instance *model.K8sInstanceRequest) error
	BatchDeleteInstance(ctx context.Context, instance []*model.K8sInstanceRequest) error
	BatchRestartInstance(ctx context.Context, instance []*model.K8sInstanceRequest) error
	GetInstanceByApp(ctx context.Context, clusterId int, appName string) ([]model.K8sInstanceByApp, error)
	GetInstanceOne(ctx context.Context, clusterId int) ([]model.K8sInstance, error)
	GetInstanceAll(ctx context.Context, clusterId int) ([]model.K8sInstance, error)
	// 应用
	CreateAppOne(ctx context.Context, app *model.K8sApp) error
}
type appService struct {
	dao         admin.ClusterDAO
	appdao      uesr.AppDAO
	instancedao uesr.InstanceDAO
	client      client.K8sClient
	l           *zap.Logger
}

func NewAppService(dao admin.ClusterDAO, appdao uesr.AppDAO, instancedao uesr.InstanceDAO, client client.K8sClient, l *zap.Logger) AppService {
	return &appService{
		dao:         dao,
		appdao:      appdao,
		instancedao: instancedao,
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

func (a *appService) UpdateInstanceOne(ctx context.Context, instance *model.K8sInstanceRequest) error {
	// 1.更新deployment请求参数
	deploymentRequest := &model.K8sDeploymentRequest{
		ClusterId:       instance.ClusterId,
		Namespace:       instance.Namespace,
		DeploymentNames: instance.DeploymentNames,
		DeploymentYaml:  instance.DeploymentYaml,
	}
	err := pkg.UpdateDeployment(ctx, deploymentRequest, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to update Deployment: %w", err)
	}
	// 2.更新service请求参数
	svcRequest := &model.K8sServiceRequest{
		ClusterId:    instance.ClusterId,
		Namespace:    instance.Namespace,
		ServiceNames: instance.ServiceNames,
		ServiceYaml:  instance.ServiceYaml,
	}
	err = pkg.UpdateService(ctx, svcRequest, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to update Service: %w", err)
	}
	return nil
}

func (a *appService) BatchDeleteInstance(ctx context.Context, instance []*model.K8sInstanceRequest) error {
	// 1.遍历instance取出deploymentRequest和svcRequest
	var deploymentRequests []*model.K8sDeploymentRequest
	var svcRequests []*model.K8sServiceRequest
	for _, i := range instance {
		deploymentRequests = append(deploymentRequests, &model.K8sDeploymentRequest{
			ClusterId:       i.ClusterId,
			Namespace:       i.Namespace,
			DeploymentNames: i.DeploymentNames,
			DeploymentYaml:  i.DeploymentYaml,
		})
		svcRequests = append(svcRequests, &model.K8sServiceRequest{
			ClusterId:    i.ClusterId,
			Namespace:    i.Namespace,
			ServiceNames: i.ServiceNames,
			ServiceYaml:  i.ServiceYaml,
		})
	}
	// 2.调用deploymentService的BatchDeleteDeployment方法删除deployment
	err := pkg.BatchDeleteK8sInstance(ctx, deploymentRequests, svcRequests, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to delete Deployment: %w", err)
	}
	return nil
}
func (a *appService) BatchRestartInstance(ctx context.Context, instance []*model.K8sInstanceRequest) error {
	// 1.遍历instance取出deploymentRequest和svcRequest
	var deploymentRequests []*model.K8sDeploymentRequest
	for _, i := range instance {
		deploymentRequests = append(deploymentRequests, &model.K8sDeploymentRequest{
			ClusterId:       i.ClusterId,
			Namespace:       i.Namespace,
			DeploymentNames: i.DeploymentNames,
			DeploymentYaml:  i.DeploymentYaml,
		})
	}
	// 2.调用deploymentService的BatchDeleteDeployment方法删除deployment
	err := pkg.BatchRestartK8sInstance(ctx, deploymentRequests, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to delete Deployment: %w", err)
	}
	return nil
}

func (a *appService) GetInstanceByApp(ctx context.Context, clusterId int, appName string) ([]model.K8sInstanceByApp, error) {
	replies, err := pkg.GetDeploymentsByAppName(ctx, clusterId, appName, a.client, a.l)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 3.返回实例列表
	instances := make([]model.K8sInstanceByApp, len(replies))

	for i, reply := range replies {
		containers := make([]model.ContainerInfo, len(reply.Containers))
		for j, c := range reply.Containers {
			containers[j] = model.ContainerInfo{
				Name:  c.Name,
				Image: c.Image,
				Ports: c.Ports,
			}
		}

		instances[i] = model.K8sInstanceByApp{
			Name:       reply.Name,
			Status:     reply.Status,
			Replicas:   reply.Replicas,
			Containers: containers,
		}
	}
	return instances, nil
	//return nil, nil
}

func (a *appService) GetInstanceOne(ctx context.Context, clusterId int) ([]model.K8sInstance, error) {
	res, err := pkg.GetK8sInstanceOne(ctx, clusterId, a.client, a.l)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 3.返回实例列表
	return res, nil
	//return nil, nil
}
func (a *appService) GetInstanceAll(ctx context.Context, clusterId int) ([]model.K8sInstance, error) {
	res, err := pkg.GetK8sInstanceOne(ctx, clusterId, a.client, a.l)
	if err != nil {
		return nil, fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 3.返回实例列表
	return res, nil
	//return nil, nil
}
func (a *appService) CreateAppOne(ctx context.Context, app *model.K8sApp) error {
	// 0.先入数据库
	err0 := a.appdao.CreateAppOne(ctx, app)
	if err0 != nil {
		return fmt.Errorf("failed to create app in db: %w", err0)
	}
	// 1.解析获得deployment和service
	deployments, services, err := pkg.ParseK8sApp(ctx, app)
	if err != nil {
		return fmt.Errorf("failed to get Deployment: %w", err)
	}
	// 2.通过clustername获取集群
	k8scluster, err2 := a.dao.GetClusterByName(ctx, app.Cluster)
	if err2 != nil {
		return fmt.Errorf("failed to get Cluster: %w", err2)
	}
	// 3.封装deploymentRequest请求参数
	for i := 0; i < len(deployments); i++ {
		var deploymentRequest model.K8sDeploymentRequest
		deploymentRequest = model.K8sDeploymentRequest{
			ClusterId:       k8scluster.ID,
			Namespace:       app.Namespace,
			DeploymentNames: []string{deployments[i].Name},
			DeploymentYaml:  &deployments[i],
		}
		// 调用deploymentService的CreateDeployment方法创建deployment
		pkg.CreateDeployment(ctx, &deploymentRequest, a.client, a.l)
	}
	// 4.封装serviceRequest请求参数
	for i := 0; i < len(services); i++ {
		var serviceRequest model.K8sServiceRequest
		serviceRequest = model.K8sServiceRequest{
			ClusterId:    k8scluster.ID,
			Namespace:    app.Namespace,
			ServiceNames: []string{services[i].Name},
			ServiceYaml:  &services[i],
		}
		// 调用svcService的CreateService方法创建service
		pkg.CreateService(ctx, &serviceRequest, a.client, a.l)
	}
	//err = pkg.CreateK8sApp(ctx, k8scluster, deployments, services, a.client, a.l)
	//if err != nil {
	//	return fmt.Errorf("failed to create Deployment: %w", err)
	//}
	return nil
}
