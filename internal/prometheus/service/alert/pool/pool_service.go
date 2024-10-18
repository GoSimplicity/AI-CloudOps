package pool

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/pool"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/send"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerPoolService interface {
	GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
}

type alertManagerPoolService struct {
	dao     pool.AlertManagerPoolDAO
	sendDao send.AlertManagerSendDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerPoolService(dao pool.AlertManagerPoolDAO, sendDao send.AlertManagerSendDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerPoolService {
	return &alertManagerPoolService{
		dao:     dao,
		sendDao: sendDao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerPoolService) GetMonitorAlertManagerPoolList(ctx context.Context, searchName *string) ([]*model.MonitorAlertManagerPool, error) {
	return pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorAlertManagerPoolByName, // 搜索函数
		a.dao.GetAllAlertManagerPools)             // 获取所有函数
}

func (a *alertManagerPoolService) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 检查 AlertManager IP 是否已存在
	exists, err := a.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
	if err != nil {
		a.l.Error("创建 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 创建 AlertManager 集群池
	if err := a.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("创建 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	a.l.Info("创建 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

func (a *alertManagerPoolService) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	alerts, err := a.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		a.l.Error("更新 AlertManager 集群池失败：获取集群池时出错", zap.Error(err))
		return err
	}

	// 检查新的 AlertManager IP 是否已存在
	exists := pkg.CheckAlertIpExists(monitorAlertManagerPool, alerts)
	if exists {
		return errors.New("AlertManager 集群池 IP 已存在")
	}

	// 更新 AlertManager 集群池
	if err := a.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("更新 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	a.l.Info("更新 AlertManager 集群池成功", zap.Int("id", monitorAlertManagerPool.ID))
	return nil
}

func (a *alertManagerPoolService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	// 检查 AlertManager 集群池是否有关联的发送组
	sendGroups, err := a.sendDao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除 AlertManager 集群池失败：获取关联发送组时出错", zap.Error(err))
		return err
	}

	if len(sendGroups) > 0 {
		return errors.New("AlertManager 集群池存在关联发送组，无法删除")
	}

	// 删除 AlertManager 集群池
	if err := a.dao.DeleteMonitorAlertManagerPool(ctx, id); err != nil {
		a.l.Error("删除 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	a.l.Info("删除 AlertManager 集群池成功", zap.Int("id", id))
	return nil
}
