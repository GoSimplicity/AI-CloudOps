package dao

import "context"

type ProcessDAO interface {
	CreateProcess(ctx context.Context) error
	UpdateProcess(ctx context.Context) error
	DeleteProcess(ctx context.Context) error
	ListProcess(ctx context.Context) error
	GetProcess(ctx context.Context) error
}

type processDAO struct {
}

func NewProcessDAO() ProcessDAO {
	return &processDAO{}
}

// CreateProcess implements ProcessDAO.
func (p *processDAO) CreateProcess(ctx context.Context) error {
	panic("unimplemented")
}

// DeleteProcess implements ProcessDAO.
func (p *processDAO) DeleteProcess(ctx context.Context) error {
	panic("unimplemented")
}

// GetProcess implements ProcessDAO.
func (p *processDAO) GetProcess(ctx context.Context) error {
	panic("unimplemented")
}

// ListProcess implements ProcessDAO.
func (p *processDAO) ListProcess(ctx context.Context) error {
	panic("unimplemented")
}

// UpdateProcess implements ProcessDAO.
func (p *processDAO) UpdateProcess(ctx context.Context) error {
	panic("unimplemented")
}
