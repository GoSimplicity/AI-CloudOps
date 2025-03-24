package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type FormDesignService interface {
	CreateFormDesign(ctx context.Context, formDesignReq *model.FormDesignReq) error
	UpdateFormDesign(ctx context.Context, formDesign *model.FormDesignReq) error
	DeleteFormDesign(ctx context.Context, id int64) error
	PublishFormDesign(ctx context.Context, id int64) error
	CloneFormDesign(ctx context.Context, name string) error
	DetailFormDesign(ctx context.Context, id int64) (*model.FormDesign, error)
	ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error)
}

type formDesignService struct {
	dao dao.FormDesignDAO
	l   *zap.Logger
}

func NewFormDesignService(dao dao.FormDesignDAO, l *zap.Logger) FormDesignService {
	return &formDesignService{
		dao: dao,
		l:   l,
	}
}

// CreateFormDesign 创建表单设计
func (f *formDesignService) CreateFormDesign(ctx context.Context, formDesignReq *model.FormDesignReq) error {
	formDesign, err := utils.ConvertFormDesignReq(formDesignReq)
	if err != nil {
		f.l.Error("转换表单设计请求失败", zap.Error(err))
		return err
	}
	return f.dao.CreateFormDesign(ctx, formDesign)
}

// UpdateFormDesign 更新表单设计
func (f *formDesignService) UpdateFormDesign(ctx context.Context, formDesignReq *model.FormDesignReq) error {
	formDesign, err := utils.ConvertFormDesignReq(formDesignReq)
	if err != nil {
		f.l.Error("转换表单设计请求失败", zap.Error(err))
		return err
	}
	return f.dao.UpdateFormDesign(ctx, formDesign)
}

// DeleteFormDesign 删除表单设计
func (f *formDesignService) DeleteFormDesign(ctx context.Context, id int64) error {
	return f.dao.DeleteFormDesign(ctx, id)
}

// PublishFormDesign 发布表单设计
func (f *formDesignService) PublishFormDesign(ctx context.Context, id int64) error {
	return f.dao.PublishFormDesign(ctx, id)
}

// CloneFormDesign 克隆表单设计
func (f *formDesignService) CloneFormDesign(ctx context.Context, name string) error {
	return f.dao.CloneFormDesign(ctx, name)
}

// DetailFormDesign 获取表单设计详情
func (f *formDesignService) DetailFormDesign(ctx context.Context, id int64) (*model.FormDesign, error) {
	return f.dao.GetFormDesign(ctx, id)
}

// ListFormDesign 获取表单设计列表
func (f *formDesignService) ListFormDesign(ctx context.Context, req *model.ListFormDesignReq) ([]model.FormDesign, error) {
	return f.dao.ListFormDesign(ctx, req)
}
