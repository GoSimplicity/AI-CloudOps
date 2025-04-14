package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ElbDAO interface {
	// ELB资源接口
	ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error)
	GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error)
	CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error
}

type elbDAO struct {
	db *gorm.DB
}



func NewElbDAO(db *gorm.DB) ElbDAO {
	return &elbDAO{
		db: db,
	}
}

// CreateElbResource implements ElbDAO.
func (e *elbDAO) CreateElbResource(ctx context.Context, params *model.ElbCreationParams) error {
	panic("unimplemented")
}

// GetElbResourceById implements ElbDAO.
func (e *elbDAO) GetElbResourceById(ctx context.Context, id int) (*model.ResourceELBResp, error) {
	panic("unimplemented")
}

// ListElbResources implements ElbDAO.
func (e *elbDAO) ListElbResources(ctx context.Context, req *model.ListElbResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}