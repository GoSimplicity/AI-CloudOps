package uesr

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"log"
)

type AppService interface {
	// 实例
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstanceRequest) error
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
	dao    admin.ClusterDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewAppService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger) AppService {
	return &appService{
		dao:    dao,
		client: client,
		l:      l,
	}
}

func (a *appService) CreateInstanceOne(ctx context.Context, instance *model.K8sInstanceRequest) error {
	// 1.创建deployment请求参数
	deploymentRequest := &model.K8sDeploymentRequest{
		ClusterId:       instance.ClusterId,
		Namespace:       instance.Namespace,
		DeploymentNames: instance.DeploymentNames,
		DeploymentYaml:  instance.DeploymentYaml,
	}
	// 调用deploymentService的CreateDeployment方法创建deployment
	err := pkg.CreateDeployment(ctx, deploymentRequest, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to create Deployment: %w", err)
	}
	// 2.创建service请求参数
	svcRequest := &model.K8sServiceRequest{
		ClusterId:    instance.ClusterId,
		Namespace:    instance.Namespace,
		ServiceNames: instance.ServiceNames,
		ServiceYaml:  instance.ServiceYaml,
	}
	// 调用svcService的CreateService方法创建service
	err = pkg.CreateService(ctx, svcRequest, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}
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
	// 1.构建deployment和service
	if len(app.K8sInstances) == 0 {
		return fmt.Errorf("no instances provided")
	}
	var deployments []appsv1.Deployment
	var services []core.Service
	for _, instance := range app.K8sInstances {
		portJson, err2 := pkg.ParsePorts(instance.ContainerCore.PortJson)
		if portJson == nil {
			return fmt.Errorf("instance containerCore portJson is nil")
		}
		if err2 != nil {
			return fmt.Errorf("instance ContainerCore PortJson parse fail")
		}
		// step1:构建deployment
		deploy := map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      app.Name,
				"namespace": app.Namespace,
			},
			"spec": map[string]interface{}{
				"replicas": instance.Replicas,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": app.Name,
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": app.Name,
						},
					},
					"spec": map[string]interface{}{
						"containers": []map[string]interface{}{
							{
								"name":  instance.Name,
								"image": instance.Image,
								"ports": []map[string]interface{}{
									{
										"containerPort": portJson[0].Port,
										"protocol":      portJson[0].Protocol,
									},
								},
							},
						},
					},
				},
			},
		}

		deployBytes, err := json.Marshal(deploy) // 将 map 转换为 *appsv1.Deployment
		if err != nil {
			log.Fatalf("Error marshaling deploy: %v", err)
		}

		var deployment appsv1.Deployment
		err = json.Unmarshal(deployBytes, &deployment)
		if err != nil {
			log.Fatalf("Error unmarshaling deploy: %v", err)
		}

		deployments = append(deployments, deployment) // 将转换后的 deployment 添加到 deployments 切片中
		// step2:构建service
		service := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      app.Name,
				"namespace": app.Namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": app.Name,
				},
				"ports": []map[string]interface{}{
					{
						"port":       portJson[0].Port,
						"protocol":   portJson[0].Protocol,
						"targetPort": portJson[0].Port,
					},
				},
				"type": app.ServiceType,
			},
		}

		serviceBytes, err := json.Marshal(service) // 将 map 转换为 *core.Service
		if err != nil {
			log.Fatalf("Error marshaling service: %v", err)
		}
		var svc core.Service
		err = json.Unmarshal(serviceBytes, &svc)
		if err != nil {
			log.Fatalf("Error unmarshaling service: %v", err)
		}

		services = append(services, svc) // 将转换后的 service 添加到 services 切片中
	}

	// 2.调用deploymentService的CreateDeployment方法创建deployment
	k8scluster, err2 := a.dao.GetClusterByName(ctx, app.Cluster)
	if err2 != nil {
		return fmt.Errorf("failed to get Deployment: %w", err2)
	}
	err := pkg.CreateK8sApp(ctx, k8scluster, deployments, services, a.client, a.l)
	if err != nil {
		return fmt.Errorf("failed to create Deployment: %w", err)
	}
	return nil
}
