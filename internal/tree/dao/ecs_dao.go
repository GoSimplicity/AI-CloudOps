package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type EcsDAO interface {
	ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error)
	GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error)
	CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error
}

type ecsDAO struct {
	db *gorm.DB
}


func NewEcsDAO(db *gorm.DB) EcsDAO {
	return &ecsDAO{
		db: db,
	}
}


// CreateEcsResource implements EcsDAO.
func (e *ecsDAO) CreateEcsResource(ctx context.Context, params *model.EcsCreationParams) error {
	if err := e.db.Create(params).Error; err != nil {
		return err
	}
	return nil
}

// GetEcsResourceById implements EcsDAO.
func (e *ecsDAO) GetEcsResourceById(ctx context.Context, id int) (*model.ResourceECSResp, error) {
	panic("unimplemented")
}

// ListEcsResources implements EcsDAO.
func (e *ecsDAO) ListEcsResources(ctx context.Context, req *model.ListEcsResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}
