package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type VpcDAO interface {
	ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (*model.PageResp, error)
	GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error)
	CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error
	DeleteVpcResource(ctx context.Context, id int) error
}

type vpcDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewVpcDAO(logger *zap.Logger, db *gorm.DB) VpcDAO {
	return &vpcDAO{
		logger: logger,
		db:     db,
	}
}

// CreateVpcResource implements VpcDAO.
func (v *vpcDAO) CreateVpcResource(ctx context.Context, req *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteVpcResource implements VpcDAO.
func (v *vpcDAO) DeleteVpcResource(ctx context.Context, id int) error {
	panic("unimplemented")
}

// GetVpcResourceById implements VpcDAO.
func (v *vpcDAO) GetVpcResourceById(ctx context.Context, id int) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

// ListVpcResources implements VpcDAO.
func (v *vpcDAO) ListVpcResources(ctx context.Context, req *model.ListVpcResourcesReq) (*model.PageResp, error) {
	panic("unimplemented")
}