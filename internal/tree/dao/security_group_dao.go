package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type TreeSecurityGroupDAO interface {
	CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error
	DeleteSecurityGroup(ctx context.Context, securityGroupID string) error
	GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error)
	ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (*model.ResourceSecurityGroupListResp, error)
}

type treeSecurityGroupDAO struct {
	db *gorm.DB
}

func NewTreeSecurityGroupDAO(db *gorm.DB) TreeSecurityGroupDAO {
	return &treeSecurityGroupDAO{
		db: db,
	}
}

// CreateSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) CreateSecurityGroup(ctx context.Context, securityGroup *model.ResourceSecurityGroup) error {
	panic("unimplemented")
}

// DeleteSecurityGroup implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) DeleteSecurityGroup(ctx context.Context, securityGroupID string) error {
	panic("unimplemented")
}

// GetSecurityGroupDetail implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) GetSecurityGroupDetail(ctx context.Context, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	panic("unimplemented")
}

// ListSecurityGroups implements TreeSecurityGroupDAO.
func (t *treeSecurityGroupDAO) ListSecurityGroups(ctx context.Context, req *model.ListSecurityGroupsReq) (*model.ResourceSecurityGroupListResp, error) {
	panic("unimplemented")
}
