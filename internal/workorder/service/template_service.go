package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context)
	UpdateTemplate(ctx context.Context)
	DeleteTemplate(ctx context.Context)
	ListTemplate(ctx context.Context)
	DetailTemplate(ctx context.Context)
}

type templateService struct {
	dao dao.TemplateDAO
}

func NewTemplateService(dao dao.TemplateDAO) TemplateService {
	return &templateService{
		dao: dao,
	}
}

// CreateTemplate implements TemplateService.
func (t *templateService) CreateTemplate(ctx context.Context) {
	panic("unimplemented")
}

// DeleteTemplate implements TemplateService.
func (t *templateService) DeleteTemplate(ctx context.Context) {
	panic("unimplemented")
}

// DetailTemplate implements TemplateService.
func (t *templateService) DetailTemplate(ctx context.Context) {
	panic("unimplemented")
}

// ListTemplate implements TemplateService.
func (t *templateService) ListTemplate(ctx context.Context) {
	panic("unimplemented")
}

// UpdateTemplate implements TemplateService.
func (t *templateService) UpdateTemplate(ctx context.Context) {
	panic("unimplemented")
}
