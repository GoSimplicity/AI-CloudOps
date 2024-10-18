package rule

import (
	"context"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/rule"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
)

type AlertManagerRuleService interface {
	GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error)
	PromqlExprCheck(ctx context.Context, expr string) (bool, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error
	DeleteMonitorAlertRule(ctx context.Context, id int) error
	BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error
}

type alertManagerRuleService struct {
	dao     rule.AlertManagerRuleDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerRuleService(dao rule.AlertManagerRuleDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerRuleService {
	return &alertManagerRuleService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerRuleService) GetMonitorAlertRuleList(ctx context.Context, searchName *string) ([]*model.MonitorAlertRule, error) {
	return pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorAlertRuleByName,
		a.dao.GetMonitorAlertRuleList)
}

func (a *alertManagerRuleService) PromqlExprCheck(_ context.Context, expr string) (bool, error) {
	return pkg.PromqlExprCheck(expr)
}

func (a *alertManagerRuleService) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则是否已存在
	exists, err := a.dao.CheckMonitorAlertRuleExists(ctx, monitorAlertRule)
	if err != nil {
		a.l.Error("创建告警规则失败：检查告警规则是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("告警规则已存在")
	}

	// 创建告警规则
	if err := a.dao.CreateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		a.l.Error("创建告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleService) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则名称是否重复
	exists, err := a.dao.CheckMonitorAlertRuleNameExists(ctx, monitorAlertRule)
	if err != nil {
		a.l.Error("更新告警规则失败：检查告警规则名称时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("告警规则名称已存在")
	}

	// 更新告警规则
	if err := a.dao.UpdateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		a.l.Error("更新告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleService) EnableSwitchMonitorAlertRule(ctx context.Context, id int) error {
	if err := a.dao.EnableSwitchMonitorAlertRule(ctx, id); err != nil {
		a.l.Error("切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleService) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error {
	// 批量切换告警规则状态
	if err := a.dao.BatchEnableSwitchMonitorAlertRule(ctx, ids); err != nil {
		a.l.Error("批量切换告警规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleService) DeleteMonitorAlertRule(ctx context.Context, id int) error {
	// 删除告警规则
	if err := a.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
		a.l.Error("删除告警规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleService) BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := a.DeleteMonitorAlertRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			a.l.Error("批量删除告警规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除告警规则 ID %d 失败: %v", id, err)
		}
	}

	return nil
}
