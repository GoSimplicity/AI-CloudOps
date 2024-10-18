package send

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type AlertManagerSendDAO interface {
	GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error)
	GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error)
	SearchMonitorSendGroupByName(ctx context.Context, name string) ([]*model.MonitorSendGroup, error)
	GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error)
	GetMonitorSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
	CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
	CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
}

type alertManagerSendDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerSendDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerSendDAO {
	return &alertManagerSendDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

func (a *alertManagerSendDAO) GetMonitorSendGroupByPoolId(ctx context.Context, poolId int) ([]*model.MonitorSendGroup, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorSendGroupByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var sendGroups []*model.MonitorSendGroup
	if err := a.db.WithContext(ctx).
		Where("pool_id = ?", poolId).
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return sendGroups, nil
}

func (a *alertManagerSendDAO) GetMonitorSendGroupByOnDutyGroupId(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, error) {
	if onDutyGroupID <= 0 {
		a.l.Error("GetMonitorSendGroupByOnDutyGroupId 失败: 无效的 onDutyGroupID", zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, fmt.Errorf("无效的 onDutyGroupID: %d", onDutyGroupID)
	}

	var sendGroups []*model.MonitorSendGroup
	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ?", onDutyGroupID).
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, err
	}

	return sendGroups, nil
}

func (a *alertManagerSendDAO) SearchMonitorSendGroupByName(ctx context.Context, name string) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&sendGroups).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorSendGroup 失败", zap.Error(err))
		return nil, err
	}

	return sendGroups, nil
}

func (a *alertManagerSendDAO) GetMonitorSendGroupList(ctx context.Context) ([]*model.MonitorSendGroup, error) {
	var sendGroups []*model.MonitorSendGroup

	if err := a.db.WithContext(ctx).Find(&sendGroups).Error; err != nil {
		a.l.Error("获取所有 MonitorSendGroup 失败", zap.Error(err))
		return nil, err
	}

	return sendGroups, nil
}

func (a *alertManagerSendDAO) GetMonitorSendGroupById(ctx context.Context, id int) (*model.MonitorSendGroup, error) {
	if id <= 0 {
		a.l.Error("GetMonitorSendGroupById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var sendGroup model.MonitorSendGroup
	if err := a.db.WithContext(ctx).First(&sendGroup, id).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &sendGroup, nil
}

func (a *alertManagerSendDAO) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	if monitorSendGroup == nil {
		a.l.Error("CreateMonitorSendGroup 失败: sendGroup 为 nil")
		return fmt.Errorf("monitorSendGroup 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(monitorSendGroup).Error; err != nil {
		a.l.Error("创建 MonitorSendGroup 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerSendDAO) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	if monitorSendGroup == nil {
		a.l.Error("UpdateMonitorSendGroup 失败: sendGroup 为 nil")
		return fmt.Errorf("monitorSendGroup 不能为空")
	}

	if monitorSendGroup.ID == 0 {
		a.l.Error("UpdateMonitorSendGroup 失败: ID 为 0", zap.Any("sendGroup", monitorSendGroup))
		return fmt.Errorf("monitorSendGroup 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", monitorSendGroup.ID).
		Updates(monitorSendGroup).Error; err != nil {
		a.l.Error("更新 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", monitorSendGroup.ID))
		return err
	}

	return nil
}

func (a *alertManagerSendDAO) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Error("DeleteMonitorSendGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorSendGroup{}, id)
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorSendGroup 失败: %w", id, err)
	}

	return nil
}

func (a *alertManagerSendDAO) CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", sendGroup.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *alertManagerSendDAO) CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("name = ?", sendGroup.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
