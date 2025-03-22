package dao

import "context"

type InstanceDAO interface {
	CreateInstance(ctx context.Context) error
	UpdateInstance(ctx context.Context) error
	DeleteInstance(ctx context.Context) error
	ListInstance(ctx context.Context) error
	GetInstance(ctx context.Context) error
}

type instanceDAO struct {
}

func NewInstanceDAO() InstanceDAO {
	return &instanceDAO{}
}

// CreateInstance implements InstanceDAO.
func (i *instanceDAO) CreateInstance(ctx context.Context) error {
	panic("unimplemented")
}

// DeleteInstance implements InstanceDAO.
func (i *instanceDAO) DeleteInstance(ctx context.Context) error {
	panic("unimplemented")
}

// GetInstance implements InstanceDAO.
func (i *instanceDAO) GetInstance(ctx context.Context) error {
	panic("unimplemented")
}

// ListInstance implements InstanceDAO.
func (i *instanceDAO) ListInstance(ctx context.Context) error {
	panic("unimplemented")
}

// UpdateInstance implements InstanceDAO.
func (i *instanceDAO) UpdateInstance(ctx context.Context) error {
	panic("unimplemented")
}
