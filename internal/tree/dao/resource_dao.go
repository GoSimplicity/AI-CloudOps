package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ResourceDAO interface {
	SyncResources(ctx context.Context, provider model.CloudProvider, region string) error
	DeleteResource(ctx context.Context, resourceType string, id int) error
	StartResource(ctx context.Context, resourceType string, id int) error
	StopResource(ctx context.Context, resourceType string, id int) error
	RestartResource(ctx context.Context, resourceType string, id int) error
	GetResourceById(ctx context.Context, resourceType string, id int) (*model.ResourceBase, error)
	SaveOrUpdateResource(ctx context.Context, resource interface{}) error
}

type resourceDAO struct {
	db *gorm.DB
}

func NewResourceDAO(db *gorm.DB) ResourceDAO {
	return &resourceDAO{
		db: db,
	}
}

// DeleteResource implements ResourceDAO.
func (r *resourceDAO) DeleteResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// RestartResource implements ResourceDAO.
func (r *resourceDAO) RestartResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// StartResource implements ResourceDAO.
func (r *resourceDAO) StartResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// StopResource implements ResourceDAO.
func (r *resourceDAO) StopResource(ctx context.Context, resourceType string, id int) error {
	panic("unimplemented")
}

// SyncResources implements ResourceDAO.
func (r *resourceDAO) SyncResources(ctx context.Context, provider model.CloudProvider, region string) error {
	panic("unimplemented")
}

// GetResourceById implements ResourceDAO.
func (r *resourceDAO) GetResourceById(ctx context.Context, resourceType string, id int) (*model.ResourceBase, error) {
	panic("unimplemented")
}

// SaveOrUpdateResource implements ResourceDAO.
func (r *resourceDAO) SaveOrUpdateResource(ctx context.Context, resource interface{}) error {
	panic("unimplemented")
}
