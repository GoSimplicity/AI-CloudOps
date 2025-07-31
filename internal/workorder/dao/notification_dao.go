package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type NotificationDAO interface {
	CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error
	UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error
	DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error
	ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (model.ListResp[*model.WorkorderNotification], error)
	DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error)
	GetNotificationByID(ctx context.Context, id int) (*model.WorkorderNotification, error)
	AddSendLog(ctx context.Context, log *model.WorkorderNotificationLog) error
	GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (model.ListResp[*model.WorkorderNotificationLog], error)
	IncrementSentCount(ctx context.Context, id int) error
}

type notificationDAO struct {
	db *gorm.DB
}

// AddSendLog implements NotificationDAO.
func (n *notificationDAO) AddSendLog(ctx context.Context, log *model.WorkorderNotificationLog) error {
	panic("unimplemented")
}

// CreateNotification implements NotificationDAO.
func (n *notificationDAO) CreateNotification(ctx context.Context, req *model.CreateWorkorderNotificationReq) error {
	panic("unimplemented")
}

// DeleteNotification implements NotificationDAO.
func (n *notificationDAO) DeleteNotification(ctx context.Context, req *model.DeleteWorkorderNotificationReq) error {
	panic("unimplemented")
}

// DetailNotification implements NotificationDAO.
func (n *notificationDAO) DetailNotification(ctx context.Context, req *model.DetailWorkorderNotificationReq) (*model.WorkorderNotification, error) {
	panic("unimplemented")
}

// GetNotificationByID implements NotificationDAO.
func (n *notificationDAO) GetNotificationByID(ctx context.Context, id int) (*model.WorkorderNotification, error) {
	panic("unimplemented")
}

// GetSendLogs implements NotificationDAO.
func (n *notificationDAO) GetSendLogs(ctx context.Context, req *model.ListWorkorderNotificationLogReq) (model.ListResp[*model.WorkorderNotificationLog], error) {
	panic("unimplemented")
}

// IncrementSentCount implements NotificationDAO.
func (n *notificationDAO) IncrementSentCount(ctx context.Context, id int) error {
	panic("unimplemented")
}

// ListNotification implements NotificationDAO.
func (n *notificationDAO) ListNotification(ctx context.Context, req *model.ListWorkorderNotificationReq) (model.ListResp[*model.WorkorderNotification], error) {
	panic("unimplemented")
}

// UpdateNotification implements NotificationDAO.
func (n *notificationDAO) UpdateNotification(ctx context.Context, req *model.UpdateWorkorderNotificationReq) error {
	panic("unimplemented")
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &notificationDAO{db: db}
}
