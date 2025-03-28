package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type ProcessService interface {
	CreateProcess(ctx context.Context, req *model.ProcessReq) error
	UpdateProcess(ctx context.Context, req *model.ProcessReq) error
	DeleteProcess(ctx context.Context, req model.DeleteProcessReqReq) error
	ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error)
	DetailProcess(ctx context.Context, req model.DetailProcessReqReq) (model.Process, error)
	PublishProcess(ctx context.Context, req model.PublishProcessReq) error
	CloneProcess(ctx context.Context, req model.CloneProcessReq) error
}

type processService struct {
	dao dao.ProcessDAO
	l   *zap.Logger
}

func NewProcessService(dao dao.ProcessDAO, l *zap.Logger) ProcessService {
	return &processService{
		dao: dao,
		l:   l,
	}
}

// CreateProcess implements ProcessService.
func (p *processService) CreateProcess(ctx context.Context, req *model.ProcessReq) error {
	process, err := utils.ConvertProcessReq(req)
	if err != nil {
		return err
	}
	return p.dao.CreateProcess(ctx, process)
}

// DeleteProcess implements ProcessService.
func (p *processService) DeleteProcess(ctx context.Context, req model.DeleteProcessReqReq) error {
	return p.dao.DeleteProcess(ctx, req.ID)
}

// DetailProcess implements ProcessService.
func (p *processService) DetailProcess(ctx context.Context, req model.DetailProcessReqReq) (model.Process, error) {
	return p.dao.GetProcess(ctx, req.ID)
}

// ListProcess implements ProcessService.
func (p *processService) ListProcess(ctx context.Context, req model.ListProcessReq) ([]model.Process, error) {
	return p.dao.ListProcess(ctx, req)
}

// UpdateProcess implements ProcessService.
func (p *processService) UpdateProcess(ctx context.Context, req *model.ProcessReq) error {
	process, err := utils.ConvertProcessReq(req)
	if err != nil {
		return err
	}
	return p.dao.UpdateProcess(ctx, process)
}

func (p *processService) PublishProcess(ctx context.Context, req model.PublishProcessReq) error {
	return p.dao.PublishProcess(ctx, req.ID)
}

func (p *processService) CloneProcess(ctx context.Context, req model.CloneProcessReq) error {
	process, err := p.dao.GetProcess(ctx, req.ID)
	if err != nil {
		return err
	}
	process.ID = 0
	process.Name = req.Name
	return p.dao.CreateProcess(ctx, &process)
}
