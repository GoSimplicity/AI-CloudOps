package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type ElbService interface {
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error)
	GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error)
	CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error
}

type elbService struct {
	logger *zap.Logger
	dao    *dao.ElbDAO
}

func NewElbService(logger *zap.Logger, dao *dao.ElbDAO) ElbService {
	return &elbService{
		logger: logger,
		dao:    dao,
	}
}


// CreateElbResource implements ElbService.
func (e *elbService) CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error {
	panic("unimplemented")
}

// GetElbResourceById implements ElbService.
func (e *elbService) GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error) {
	panic("unimplemented")
}

// ListElbResources implements ElbService.
func (e *elbService) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
