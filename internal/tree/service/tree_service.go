package service

import (
	"context"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	"go.uber.org/zap"
)

type TreeService interface {
	// GetTreeList 获取树列表
	CreateResourceEcs(ctx context.Context, obj *model.ResourceEcs) error
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
