package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"go.uber.org/zap"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req model.InstanceReq) error
	UpdateInstance(ctx context.Context, req model.InstanceReq) error
	DeleteInstance(ctx context.Context, req model.DeleteInstanceReq) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error)
	DetailInstance(ctx context.Context, id int64) (model.Instance, error)
	ApproveInstance(ctx context.Context, req model.InstanceFlowReq) error
	ActionInstance(ctx context.Context, req model.InstanceFlowReq) error
	CommentInstance(ctx context.Context, req model.InstanceCommentReq) error
}

type instanceService struct {
	dao dao.InstanceDAO
	l   *zap.Logger
}

func NewInstanceService(dao dao.InstanceDAO, l *zap.Logger) InstanceService {
	return &instanceService{
		dao: dao,
		l:   l,
	}
}

// CreateInstance implements InstanceService.
func (i *instanceService) CreateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		return err
	}
	return i.dao.CreateInstance(ctx, instance)
}

// DeleteInstance implements InstanceService.
func (i *instanceService) DeleteInstance(ctx context.Context, req model.DeleteInstanceReq) error {
	return i.dao.DeleteInstance(ctx, req.ID)
}

// DetailInstance implements InstanceService.
func (i *instanceService) DetailInstance(ctx context.Context, id int64) (model.Instance, error) {
	return i.dao.GetInstance(ctx, id)
}

// ListInstance implements InstanceService.
func (i *instanceService) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	return i.dao.ListInstance(ctx, req)
}

// UpdateInstance implements InstanceService.
func (i *instanceService) UpdateInstance(ctx context.Context, req model.InstanceReq) error {
	instance, err := utils.ConvertInstanceReq(&req)
	if err != nil {
		return err
	}
	return i.dao.UpdateInstance(ctx, instance)
}

func (i *instanceService) ApproveInstance(ctx context.Context, req model.InstanceFlowReq) error {
	// step1:给数据库中的InstanceFlow表添加一条记录
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceFlow(ctx, flowReq)
	if err != nil {
		return err
	}
	// step2:更新Instance表中的DueDate字段
	days := req.FormData.ApproveDays
	// 查找Instance
	instance, err := i.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		return err
	}
	// 转换InstanceReq=>Instance
	convertInstanceReq, err := utils.ConvertInstance(&instance)
	// 更改Instance的截至时间
	startDate, err := time.Parse("2006-01-02", convertInstanceReq.FormData.DateRange[0])
	if err != nil {
		i.l.Error("时间转换失败", zap.Error(err))
		return err
	}
	convertInstanceReq.DueDate = startDate.AddDate(0, 0, int(days))
	// 转换回去
	newinstance, err := utils.ConvertInstanceReq(convertInstanceReq)

	err = i.dao.UpdateInstance(ctx, newinstance)
	return nil
}

func (i *instanceService) ActionInstance(ctx context.Context, req model.InstanceFlowReq) error {
	// step1:给数据库中的InstanceFlow表添加一条记录
	flowReq, err := utils.ConvertInstanceFlowReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceFlow(ctx, flowReq)
	if err != nil {
		return err
	}
	return nil
}
func (i *instanceService) CommentInstance(ctx context.Context, req model.InstanceCommentReq) error {
	// step1:给数据库中的InstanceComment表添加一条记录
	commentReq, err := utils.ConvertInstanceCommentReq(&req)
	if err != nil {
		return err
	}
	err = i.dao.CreateInstanceComment(ctx, commentReq)
	if err != nil {
		return err
	}
	return nil
}
