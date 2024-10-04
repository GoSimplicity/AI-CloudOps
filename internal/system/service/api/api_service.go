package api

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao/role"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type ApiService interface {
	GetApiList(ctx context.Context, uid int) ([]*model.Api, error)
	GetApiListAll(ctx context.Context) ([]*model.Api, error)
	DeleteApi(ctx context.Context, apiID string) error
	CreateApi(ctx context.Context, api *model.Api) error
	UpdateApi(ctx context.Context, api *model.Api) error
}

type apiService struct {
	apiDao  api.ApiDAO
	roleDao role.RoleDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewApiService(apiDao api.ApiDAO, roleDao role.RoleDAO, l *zap.Logger, userDao userDao.UserDAO) ApiService {
	return &apiService{
		apiDao:  apiDao,
		roleDao: roleDao,
		l:       l,
		userDao: userDao,
	}
}

func (a *apiService) GetApiList(ctx context.Context, uid int) ([]*model.Api, error) {
	user, err := a.userDao.GetUserByID(ctx, uid)
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	apis := make([]*model.Api, 0)

	for _, role := range user.Roles {
		roleApis, err := a.roleDao.GetApisByRoleID(ctx, role.ID)
		if err != nil {
			a.l.Error("GetApisByRoleID failed", zap.Error(err))
			return nil, err
		}

		apis = append(apis, roleApis...)
	}

	return apis, nil
}

func (a *apiService) GetApiListAll(ctx context.Context) ([]*model.Api, error) {
	return a.apiDao.GetAllApis(ctx)
}

func (a *apiService) DeleteApi(ctx context.Context, apiID string) error {
	return a.apiDao.DeleteApi(ctx, apiID)
}

func (a *apiService) CreateApi(ctx context.Context, api *model.Api) error {
	return a.apiDao.CreateApi(ctx, api)
}

func (a *apiService) UpdateApi(ctx context.Context, api *model.Api) error {
	return a.apiDao.UpdateApi(ctx, api)
}
