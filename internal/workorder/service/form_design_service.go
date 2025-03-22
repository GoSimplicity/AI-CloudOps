package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context)
	UpdateFormDesign(ctx context.Context)
	DeleteFormDesign(ctx context.Context)
	ListFormDesign(ctx context.Context)
}

type formDesignService struct {
	dao dao.FormDesignDAO
}

func NewFormDesignService(dao dao.FormDesignDAO) FormDesignService {
	return &formDesignService{
		dao: dao,
	}
}

// CreateFormDesign implements FormDesignService.
func (f *formDesignService) CreateFormDesign(ctx context.Context) {
	panic("unimplemented")
}

// DeleteFormDesign implements FormDesignService.
func (f *formDesignService) DeleteFormDesign(ctx context.Context) {
	panic("unimplemented")
}

// ListFormDesign implements FormDesignService.
func (f *formDesignService) ListFormDesign(ctx context.Context) {
	panic("unimplemented")
}

// UpdateFormDesign implements FormDesignService.
func (f *formDesignService) UpdateFormDesign(ctx context.Context) {
	panic("unimplemented")
}
