package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"go.uber.org/zap"
)

type RdsService interface {
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (*model.PageResp, error)
	GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error)
	CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error
}

type rdsService struct {
	logger *zap.Logger
	dao    *dao.RdsDAO
}



func NewRdsService(logger *zap.Logger, dao *dao.RdsDAO) RdsService {
	return &rdsService{
		logger: logger,
		dao:    dao,
	}
}

// CreateRdsResource implements RdsService.
func (r *rdsService) CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error {
	panic("unimplemented")
}

// GetRdsResourceById implements RdsService.
func (r *rdsService) GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error) {
	panic("unimplemented")
}

// ListRdsResources implements RdsService.
func (r *rdsService) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}