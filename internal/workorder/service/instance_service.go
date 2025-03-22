package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type InstanceService interface {
	CreateInstance(ctx context.Context)
	UpdateInstance(ctx context.Context)
	DeleteInstance(ctx context.Context)
	ListInstance(ctx context.Context)
	DetailInstance(ctx context.Context)
}

type instanceService struct {
	dao dao.InstanceDAO
}

func NewInstanceService(dao dao.InstanceDAO) InstanceService {
	return &instanceService{
		dao: dao,
	}
}

// CreateInstance implements InstanceService.
func (i *instanceService) CreateInstance(ctx context.Context) {
	panic("unimplemented")
}

// DeleteInstance implements InstanceService.
func (i *instanceService) DeleteInstance(ctx context.Context) {
	panic("unimplemented")
}

// DetailInstance implements InstanceService.
func (i *instanceService) DetailInstance(ctx context.Context) {
	panic("unimplemented")
}

// ListInstance implements InstanceService.
func (i *instanceService) ListInstance(ctx context.Context) {
	panic("unimplemented")
}

// UpdateInstance implements InstanceService.
func (i *instanceService) UpdateInstance(ctx context.Context) {
	panic("unimplemented")
}
