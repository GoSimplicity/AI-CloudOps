package dao

import "context"

type TemplateDAO interface {
	CreateTemplate(ctx context.Context) error
	UpdateTemplate(ctx context.Context) error
	DeleteTemplate(ctx context.Context) error
	ListTemplate(ctx context.Context) error
	GetTemplate(ctx context.Context) error
}

type templateDAO struct {
}

func NewTemplateDAO() TemplateDAO {
	return &templateDAO{}
}

// CreateTemplate implements TemplateDAO.
func (t *templateDAO) CreateTemplate(ctx context.Context) error {
	panic("unimplemented")
}

// DeleteTemplate implements TemplateDAO.
func (t *templateDAO) DeleteTemplate(ctx context.Context) error {
	panic("unimplemented")
}

// GetTemplate implements TemplateDAO.
func (t *templateDAO) GetTemplate(ctx context.Context) error {
	panic("unimplemented")
}

// ListTemplate implements TemplateDAO.
func (t *templateDAO) ListTemplate(ctx context.Context) error {
	panic("unimplemented")
}

// UpdateTemplate implements TemplateDAO.
func (t *templateDAO) UpdateTemplate(ctx context.Context) error {
	panic("unimplemented")
}
