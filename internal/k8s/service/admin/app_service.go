package admin

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type AppService interface {
	CreateInstanceOne(ctx context.Context, instance *model.K8sInstanceRequest) error
}
type appService struct {
	dao               admin.ClusterDAO
	client            client.K8sClient
	l                 *zap.Logger
	deploymentService DeploymentService
	svcService        SvcService
}

func NewAppService(dao admin.ClusterDAO, client client.K8sClient, l *zap.Logger, deploymentService DeploymentService, svcService SvcService) AppService {
	return &appService{
		dao:               dao,
		client:            client,
		l:                 l,
		deploymentService: deploymentService,
		svcService:        svcService,
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
	err := a.deploymentService.CreateDeployment(ctx, deploymentRequest)
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
	err = a.svcService.CreateService(ctx, svcRequest)
	if err != nil {
		return fmt.Errorf("failed to create Service: %w", err)
	}
	return nil
}
