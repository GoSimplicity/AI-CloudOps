package dao

import "context"

type FormDesignDAO interface {
	CreateFormDesign(ctx context.Context) error
	UpdateFormDesign(ctx context.Context) error
	DeleteFormDesign(ctx context.Context) error
	ListFormDesign(ctx context.Context) error
	GetFormDesign(ctx context.Context) error
}

type formDesignDAO struct {
}

func NewFormDesignDAO() FormDesignDAO {
	return &formDesignDAO{}
}

// CreateFormDesign implements FormDesignDAO.
func (f *formDesignDAO) CreateFormDesign(ctx context.Context) error {
	panic("unimplemented")
}

// DeleteFormDesign implements FormDesignDAO.
func (f *formDesignDAO) DeleteFormDesign(ctx context.Context) error {
	panic("unimplemented")
}

// GetFormDesign implements FormDesignDAO.
func (f *formDesignDAO) GetFormDesign(ctx context.Context) error {
	panic("unimplemented")
}

// ListFormDesign implements FormDesignDAO.
func (f *formDesignDAO) ListFormDesign(ctx context.Context) error {
	panic("unimplemented")
}

// UpdateFormDesign implements FormDesignDAO.
func (f *formDesignDAO) UpdateFormDesign(ctx context.Context) error {
	panic("unimplemented")
}
