package user

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type AppService interface {
	CreateApp(ctx context.Context, req *model.CreateK8sAppRequest) error
	GetApp(ctx context.Context, id int64) (model.K8sApp, error)
	DeleteApp(ctx context.Context, id int64) error
	UpdateApp(ctx context.Context, req *model.UpdateK8sAppRequest) error
	GetAppList(ctx context.Context, req *model.GetK8sAppListRequest) ([]model.K8sApp, error)
	GetPodListByDeploy(ctx context.Context, id int64) ([]model.Resource, error)
}
type appService struct {
	dao         admin.ClusterDAO
	appdao      user.AppDAO
	client client.K8sClient
	l      *zap.Logger
}

func NewAppService(dao admin.ClusterDAO, appdao user.AppDAO, client client.K8sClient, l *zap.Logger) AppService {
	return &appService{
		dao:         dao,
		appdao:      appdao,
		client:      client,
		l:           l,
	}
}

// CreateApp implements AppService.
func (a *appService) CreateApp(ctx context.Context, req *model.CreateK8sAppRequest) error {
	panic("unimplemented")
}

// DeleteApp implements AppService.
func (a *appService) DeleteApp(ctx context.Context, id int64) error {
	panic("unimplemented")
}

// GetApp implements AppService.
func (a *appService) GetApp(ctx context.Context, id int64) (model.K8sApp, error) {
	panic("unimplemented")
}

// GetAppList implements AppService.
func (a *appService) GetAppList(ctx context.Context, req *model.GetK8sAppListRequest) ([]model.K8sApp, error) {
	panic("unimplemented")
}

// GetPodListByDeploy implements AppService.
func (a *appService) GetPodListByDeploy(ctx context.Context, id int64) ([]model.Resource, error) {
	panic("unimplemented")
}

// UpdateApp implements AppService.
func (a *appService) UpdateApp(ctx context.Context, req *model.UpdateK8sAppRequest) error {
	panic("unimplemented")
}
