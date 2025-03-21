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
	"gorm.io/gorm"
)

type AppService interface {

	// 应用
	CreateAppOne(ctx context.Context, app *model.K8sApp) error
	GetAppOne(ctx context.Context, id int64) (model.K8sApp, error)
	DeleteAppOne(ctx context.Context, id int64) error
	UpdateAppOne(ctx context.Context, id int64, app model.K8sApp) error
	GetAppByIds(ctx context.Context, ids []int64) ([]model.K8sApp, error)
	GetPodListByDeploy(ctx context.Context, id int64) ([]model.Resource, error)
	// 项目

}
type appService struct {
	dao         admin.ClusterDAO
	appdao      user.AppDAO
	instancedao user.InstanceDAO

	client client.K8sClient
	l      *zap.Logger
}

func NewAppService(dao admin.ClusterDAO, appdao user.AppDAO, instancedao user.InstanceDAO, client client.K8sClient, l *zap.Logger) AppService {
	return &appService{
		dao:         dao,
		appdao:      appdao,
		instancedao: instancedao,
		client:      client,
		l:           l,
	}
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
