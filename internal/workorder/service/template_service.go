package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type TemplateService interface {
	CreateTemplate(ctx context.Context, req model.TemplateReq) error
	UpdateTemplate(ctx context.Context, req model.TemplateReq) error
	DeleteTemplate(ctx context.Context, req model.DeleteTemplateReq) error
	ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error)
	DetailTemplate(ctx context.Context, req model.DetailTemplateReq) (*model.Template, error)
}

type templateService struct {
	dao dao.TemplateDAO
	l   *zap.Logger
}

func NewTemplateService(dao dao.TemplateDAO, l *zap.Logger) TemplateService {
	return &templateService{
		dao: dao,
		l:   l,
	}
}

// CreateTemplate implements TemplateService.
func (t *templateService) CreateTemplate(ctx context.Context, req model.TemplateReq) error {
	template, err := utils.ConvertTemplateReq(&req)
	if err != nil {
		return err
	}
	return t.dao.CreateTemplate(ctx, template)
}

// DeleteTemplate implements TemplateService.
func (t *templateService) DeleteTemplate(ctx context.Context, req model.DeleteTemplateReq) error {
	return t.dao.DeleteTemplate(ctx, req.ID)
}

// DetailTemplate implements TemplateService.
func (t *templateService) DetailTemplate(ctx context.Context, req model.DetailTemplateReq) (*model.Template, error) {
	template, err := t.dao.GetTemplate(ctx, req.ID)
	if err != nil {
		t.l.Error("获取模板失败", zap.Error(err))
		return nil, err
	}
	return &template, nil
}

// ListTemplate implements TemplateService.
func (t *templateService) ListTemplate(ctx context.Context, req model.ListTemplateReq) ([]model.Template, error) {
	templates, err := t.dao.ListTemplate(ctx, req)
	if err != nil {
		t.l.Error("获取模板列表失败", zap.Error(err))
		return nil, err
	}
	return templates, nil
}

// UpdateTemplate implements TemplateService.
func (t *templateService) UpdateTemplate(ctx context.Context, req model.TemplateReq) error {
	template, err := utils.ConvertTemplateReq(&req)
	if err != nil {
		return err
	}
	return t.dao.UpdateTemplate(ctx, template)
}
