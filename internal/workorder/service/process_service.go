package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type ProcessService interface {
	CreateProcess(ctx context.Context)
	UpdateProcess(ctx context.Context)
	DeleteProcess(ctx context.Context)
	ListProcess(ctx context.Context)
	DetailProcess(ctx context.Context)
}

type processService struct {
	dao dao.ProcessDAO
}

func NewProcessService(dao dao.ProcessDAO) ProcessService {
	return &processService{
		dao: dao,
	}
}

// CreateProcess implements ProcessService.
func (p *processService) CreateProcess(ctx context.Context) {
	panic("unimplemented")
}

// DeleteProcess implements ProcessService.
func (p *processService) DeleteProcess(ctx context.Context) {
	panic("unimplemented")
}

// DetailProcess implements ProcessService.
func (p *processService) DetailProcess(ctx context.Context) {
	panic("unimplemented")
}

// ListProcess implements ProcessService.
func (p *processService) ListProcess(ctx context.Context) {
	panic("unimplemented")
}

// UpdateProcess implements ProcessService.
func (p *processService) UpdateProcess(ctx context.Context) {
	panic("unimplemented")
}
