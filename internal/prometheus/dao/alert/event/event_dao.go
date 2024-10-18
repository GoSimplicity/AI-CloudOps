package event

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerEventDAO interface {
	GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error)
	GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error)
	EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error
	GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error)
	UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error
}

type alertManagerEventDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerEventDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerEventDAO {
	return &alertManagerEventDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

func (a *alertManagerEventDAO) GetMonitorAlertEventById(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertEventById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := a.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

func (a *alertManagerEventDAO) SearchMonitorAlertEventByName(ctx context.Context, name string) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).
		Where("alert_name LIKE ?", "%"+name+"%").
		Find(&alertEvents).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertEvent 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return alertEvents, nil
}

func (a *alertManagerEventDAO) GetMonitorAlertEventList(ctx context.Context) ([]*model.MonitorAlertEvent, error) {
	var alertEvents []*model.MonitorAlertEvent

	if err := a.db.WithContext(ctx).Find(&alertEvents).Error; err != nil {
		a.l.Error("获取 MonitorAlertEvent 列表失败", zap.Error(err))
		return nil, err
	}

	return alertEvents, nil
}

func (a *alertManagerEventDAO) EventAlertClaim(ctx context.Context, event *model.MonitorAlertEvent) error {
	if event == nil {
		a.l.Error("EventAlertClaim 失败: event 为 nil")
		return fmt.Errorf("event 不能为空")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertEvent{}).
		Where("id = ?", event.ID).
		Updates(event).Error; err != nil {
		a.l.Error("EventAlertClaim 更新失败", zap.Error(err), zap.Int("id", event.ID))
		return err
	}

	return nil
}

func (a *alertManagerEventDAO) GetAlertEventByID(ctx context.Context, id int) (*model.MonitorAlertEvent, error) {
	if id <= 0 {
		a.l.Error("GetAlertEventByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertEvent model.MonitorAlertEvent
	if err := a.db.WithContext(ctx).First(&alertEvent, id).Error; err != nil {
		a.l.Error("获取 AlertEvent 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertEvent, nil
}

func (a *alertManagerEventDAO) UpdateAlertEvent(ctx context.Context, alertEvent *model.MonitorAlertEvent) error {
	if alertEvent == nil {
		a.l.Error("UpdateAlertEvent 失败: alertEvent 为 nil")
		return fmt.Errorf("alertEvent 不能为空")
	}

	if err := a.db.WithContext(ctx).Save(alertEvent).Error; err != nil {
		a.l.Error("更新 AlertEvent 失败", zap.Error(err), zap.Int("id", alertEvent.ID))
		return err
	}

	return nil
}
