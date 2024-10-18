package send

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/rule"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/send"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerSendService interface {
	GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
}

type alertManagerSendService struct {
	dao     send.AlertManagerSendDAO
	ruleDao rule.AlertManagerRuleDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerSendService(dao send.AlertManagerSendDAO, ruleDao rule.AlertManagerRuleDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerSendService {
	return &alertManagerSendService{
		dao:     dao,
		ruleDao: ruleDao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerSendService) GetMonitorSendGroupList(ctx context.Context, searchName *string) ([]*model.MonitorSendGroup, error) {
	return pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorSendGroupByName,
		a.dao.GetMonitorSendGroupList)
}

func (a *alertManagerSendService) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 检查发送组是否已存在
	exists, err := a.dao.CheckMonitorSendGroupExists(ctx, monitorSendGroup)
	if err != nil {
		a.l.Error("创建发送组失败：检查发送组是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("发送组已存在")
	}

	// 创建发送组
	if err := a.dao.CreateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		a.l.Error("创建发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerSendService) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 检查发送组名称是否重复
	exists, err := a.dao.CheckMonitorSendGroupNameExists(ctx, monitorSendGroup)
	if err != nil {
		a.l.Error("更新发送组失败：检查发送组名称时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("发送组名称已存在")
	}

	// 更新发送组
	if err := a.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		a.l.Error("更新发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerSendService) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	// 检查发送组是否有关联的资源
	associatedResources, err := a.ruleDao.GetAssociatedResourcesBySendGroupId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除发送组失败：获取关联资源时出错", zap.Error(err))
		return err
	}

	if len(associatedResources) > 0 {
		return errors.New("发送组存在关联资源，无法删除")
	}

	// 删除发送组
	if err := a.dao.DeleteMonitorSendGroup(ctx, id); err != nil {
		a.l.Error("删除发送组失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}
