package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type RdsDAO interface {
	// RDS资源接口
	ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (*model.PageResp, error)
	GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error)
	CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error
}

type rdsDAO struct {
	db *gorm.DB
}


func NewRdsDAO(db *gorm.DB) RdsDAO {
	return &rdsDAO{
		db: db,
	}
}

// CreateRdsResource implements RdsDAO.
func (r *rdsDAO) CreateRdsResource(ctx context.Context, params *model.RdsCreationParams) error {
	panic("unimplemented")
}

// GetRdsResourceById implements RdsDAO.
func (r *rdsDAO) GetRdsResourceById(ctx context.Context, id int) (*model.ResourceRDSResp, error) {
	panic("unimplemented")
}

// ListRdsResources implements RdsDAO.
func (r *rdsDAO) ListRdsResources(ctx context.Context, req *model.ListRdsResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
