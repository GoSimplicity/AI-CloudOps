package service

import (
	"context"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	"go.uber.org/zap"
)

type TreeService interface {
	CreateResourceEcs(ctx context.Context, obj *model.ResourceEcs) error
	DeleteResourceEcs(ctx context.Context, obj *model.ResourceEcs) error
	UpdateResourceEcs(ctx context.Context, obj *model.ResourceEcs) error
	GetAllResourceEcs(ctx context.Context, instanceID string) (*model.ResourceEcs, error)
	GetResourceEcsByID(ctx context.Context, id int) (*model.ResourceEcs, error)
}

type treeService struct {
	ecsDao ecs.TreeEcsDAO
	l      *zap.Logger
}

func NewTreeService(ecsDao ecs.TreeEcsDAO, l *zap.Logger) TreeService {
	return &treeService{
		ecsDao: ecsDao,
		l:      l,
	}
}

func (ts *treeService) CreateResourceEcs(ctx context.Context, obj *model.ResourceEcs) error {
	return ts.ecsDao.Create(ctx, obj)
}

func (ts *treeService) DeleteResourceEcs(ctx context.Context, obj *model.ResourceEcs) error {
	return ts.ecsDao.Delete(ctx, obj)
}

func (ts *treeService) UpdateResourceEcs(ctx context.Context, obj *model.ResourceEcs) error {
	return ts.ecsDao.Update(ctx, obj)
}

func (ts *treeService) GetAllResourceEcs(ctx context.Context, instanceID string) (*model.ResourceEcs, error) {
	return nil, nil
}

func (ts *treeService) GetResourceEcsByID(ctx context.Context, id int) (*model.ResourceEcs, error) {
	return nil, nil
}
