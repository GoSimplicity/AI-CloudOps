/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package alert

import (
	"context"
	"errors"
	"fmt"

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type AlertManagerRuleService interface {
	GetMonitorAlertRuleList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorAlertRule, error)
	PromqlExprCheck(ctx context.Context, expr string) (bool, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error
	DeleteMonitorAlertRule(ctx context.Context, id int) error
	BatchDeleteMonitorAlertRule(ctx context.Context, ids []int) error
	GetMonitorAlertRuleTotal(ctx context.Context) (int, error)
}

type alertManagerRuleService struct {
	dao     alert.AlertManagerRuleDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerRuleService(dao alert.AlertManagerRuleDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerRuleService {
	return &alertManagerRuleService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorAlertRuleList 获取告警规则列表
func (a *alertManagerRuleService) GetMonitorAlertRuleList(ctx context.Context, listReq *model.ListReq) ([]*model.MonitorAlertRule, error) {
	if listReq.Search != "" {
		rules, err := a.dao.SearchMonitorAlertRuleByName(ctx, listReq.Search)
		if err != nil {
			a.l.Error("搜索告警规则失败", zap.String("search", listReq.Search), zap.Error(err))
			return nil, err
		}
		return rules, nil
	}

	offset := (listReq.Page - 1) * listReq.Size
	limit := listReq.Size

	rules, err := a.dao.GetMonitorAlertRuleList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取告警规则列表失败", zap.Error(err))
		return nil, err
	}

	for _, rule := range rules {
		user, err := a.userDao.GetUserByID(ctx, rule.UserID)
		if err != nil {
			a.l.Error("获取创建用户名失败", zap.Error(err))
		}
		if user.RealName == "" {
			rule.CreateUserName = user.Username
		} else {
			rule.CreateUserName = user.RealName
		}
	}

	return rules, nil
}

// PromqlExprCheck 检查 PromQL 表达式是否有效
func (a *alertManagerRuleService) PromqlExprCheck(_ context.Context, expr string) (bool, error) {
	return pkg.PromqlExprCheck(expr)
}

// CreateMonitorAlertRule 创建告警规则
func (a *alertManagerRuleService) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则是否已存在
	exists, err := a.dao.CheckMonitorAlertRuleNameExists(ctx, monitorAlertRule)
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

	return nil
}

// UpdateMonitorAlertRule 更新告警规则
func (a *alertManagerRuleService) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	// 检查告警规则是否已存在
	exists, err := a.dao.CheckMonitorAlertRuleExists(ctx, monitorAlertRule)
	if err != nil {
		a.l.Error("更新告警规则失败：检查告警规则是否存在时出错", zap.Error(err))
		return err
	}

	if !exists {
		return errors.New("告警规则不存在")
	}

	// 更新告警规则
	if err := a.dao.UpdateMonitorAlertRule(ctx, monitorAlertRule); err != nil {
		a.l.Error("更新告警规则失败", zap.Error(err))
		return err
	}

	if ok, err := pkg.PromqlExprCheck(monitorAlertRule.Expr); err != nil || !ok {
		return errors.New("PromQL 表达式无效")
	}

	return nil
}

// EnableSwitchMonitorAlertRule 切换告警规则状态
func (a *alertManagerRuleService) EnableSwitchMonitorAlertRule(ctx context.Context, id int) error {
	if err := a.dao.EnableSwitchMonitorAlertRule(ctx, id); err != nil {
		a.l.Error("切换告警规则状态失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换告警规则状态
func (a *alertManagerRuleService) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ids []int) error {
	// 批量切换告警规则状态
	if err := a.dao.BatchEnableSwitchMonitorAlertRule(ctx, ids); err != nil {
		a.l.Error("批量切换告警规则状态失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除告警规则
func (a *alertManagerRuleService) DeleteMonitorAlertRule(ctx context.Context, id int) error {
	// 删除告警规则
	if err := a.dao.DeleteMonitorAlertRule(ctx, id); err != nil {
		a.l.Error("删除告警规则失败", zap.Error(err))
		return err
	}

	return nil
}

// BatchDeleteMonitorAlertRule 批量删除告警规则
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

// GetMonitorAlertRuleTotal 获取监控告警规则总数
func (a *alertManagerRuleService) GetMonitorAlertRuleTotal(ctx context.Context) (int, error) {
	return a.dao.GetMonitorAlertRuleTotal(ctx)
}
