package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesign *model.FormDesignReq) (model.FormDesignReq, error)
	UpdateFormDesign(ctx context.Context, formDesign *model.FormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int64) error
	//ListFormDesign(ctx context.Context)
	PublishFormDesign(ctx context.Context, id int64) error
	CloneFormDesign(ctx context.Context, id int64, name string) (*model.FormDesignReq, error)
	DetailFormDesign(ctx context.Context, id int64) (*model.FormDesignReq, error)
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesignReq, error)
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
func (f *formDesignService) CreateFormDesign(ctx context.Context, formDesignReq *model.FormDesignReq) (model.FormDesignReq, error) {

	req, err := convertReq(formDesignReq)
	if err != nil {
		return model.FormDesignReq{}, err
	}
	err = f.dao.CreateFormDesign(ctx, req)
	if err != nil {
		return model.FormDesignReq{}, err
	}
	origin, err := convertOrigin(req)
	if err != nil {
		return model.FormDesignReq{}, err
	}
	return *origin, nil
}

// UpdateFormDesign implements FormDesignService.
func (f *formDesignService) UpdateFormDesign(ctx context.Context, formDesignReq *model.FormDesignReq) error {
	req, err := convertReq(formDesignReq)
	if err != nil {
		return err
	}
	return f.dao.UpdateFormDesign(ctx, req)
}

// DeleteFormDesign implements FormDesignService.
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int64) error {
	return f.dao.DeleteFormDesign(ctx, id)
}

// PublishFormDesign implements FormDesignService.
func (f *formDesignService) PublishFormDesign(ctx context.Context, id int64) error {
	return f.dao.PublishFormDesign(ctx, id)
}

// CloneFormDesign implements FormDesignService.
func (f *formDesignService) CloneFormDesign(ctx context.Context, id int64, name string) (*model.FormDesignReq, error) {
	design, err := f.dao.CloneFormDesign(ctx, id, name)
	if err != nil {
		return nil, err
	}
	origin, err := convertOrigin(design)
	if err != nil {
		return nil, err
	}
	return origin, nil
}

// ListFormDesign implements FormDesignService.
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesignReq, error) {
	design, err := f.dao.ListFormDesign(ctx, req)
	if err != nil {
		return nil, err
	}
	var origin []model.FormDesignReq
	for i := range design {
		tmp, err := convertOrigin(&design[i])
		if err != nil {
			return nil, err
		}
		origin = append(origin, *tmp)
	}
	return origin, nil
}

// DetailFormDesign implements FormDesignService.
func (f *formDesignService) DetailFormDesign(ctx context.Context, id int64) (*model.FormDesignReq, error) {
	design, err := f.dao.GetFormDesign(ctx, id)
	if err != nil {
		return nil, err
	}
	origin, err := convertOrigin(design)
	if err != nil {
		return nil, err
	}
	return origin, nil
}

func convertReq(formDesign *model.FormDesignReq) (*model.FormDesign, error) {
	formDesignMarshal, err := json.Marshal(formDesign.Schema)
	if err != nil {
		return nil, fmt.Errorf("序列化表单 Schema 失败: %v", err)
	}
	return &model.FormDesign{
		ID:          formDesign.ID,
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      string(formDesignMarshal),
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		CreatorID:   formDesign.CreatorID,
		CreatorName: formDesign.CreatorName,
	}, nil
}
func convertOrigin(formDesign *model.FormDesign) (*model.FormDesignReq, error) {
	var p model.Schema
	err := json.Unmarshal([]byte(formDesign.Schema), &p)

	if err != nil {
		return nil, fmt.Errorf("序列化表单 Schema 失败: %v", err)
	}
	return &model.FormDesignReq{
		ID:          formDesign.ID,
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      p,
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		CreatorID:   formDesign.CreatorID,
		CreatorName: formDesign.CreatorName,
	}, nil
}
