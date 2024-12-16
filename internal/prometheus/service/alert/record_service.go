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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	"go.uber.org/zap"
)

type AlertManagerRecordService interface {
	GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, id int) error
	BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, id int) error
	BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error
}

type alertManagerRecordService struct {
	dao     alert.AlertManagerRecordDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerRecordService(dao alert.AlertManagerRecordDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerRecordService {
	return &alertManagerRecordService{
		dao:     dao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerRecordService) GetMonitorRecordRuleList(ctx context.Context, searchName *string) ([]*model.MonitorRecordRule, error) {
	return pkg.HandleList(ctx, searchName,
		a.dao.SearchMonitorRecordRuleByName,
		a.dao.GetMonitorRecordRuleList)
}

func (a *alertManagerRecordService) CreateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 检查记录规则是否已存在
	exists, err := a.dao.CheckMonitorRecordRuleExists(ctx, monitorRecordRule)
	if err != nil {
		a.l.Error("创建记录规则失败：检查记录规则是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("记录规则已存在")
	}

	// 创建记录规则
	if err := a.dao.CreateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		a.l.Error("创建记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) UpdateMonitorRecordRule(ctx context.Context, monitorRecordRule *model.MonitorRecordRule) error {
	// 更新记录规则
	if err := a.dao.UpdateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		a.l.Error("更新记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) DeleteMonitorRecordRule(ctx context.Context, id int) error {
	// 删除记录规则
	if err := a.dao.DeleteMonitorRecordRule(ctx, id); err != nil {
		a.l.Error("删除记录规则失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) BatchDeleteMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := a.DeleteMonitorRecordRule(ctx, id); err != nil {
			// 记录错误但继续删除其他规则
			a.l.Error("批量删除记录规则失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除记录规则 ID %d 失败: %v", id, err)
		}
	}

	return nil
}

func (a *alertManagerRecordService) EnableSwitchMonitorRecordRule(ctx context.Context, id int) error {
	if err := a.dao.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
		a.l.Error("切换记录规则状态失败", zap.Error(err))
		return err
	}

	// 更新缓存
	if err := a.cache.MonitorCacheManager(ctx); err != nil {
		a.l.Error("更新缓存失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) BatchEnableSwitchMonitorRecordRule(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if err := a.EnableSwitchMonitorRecordRule(ctx, id); err != nil {
			a.l.Error("批量切换记录规则状态失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("切换记录规则 ID %d 状态失败: %v", id, err)
		}
	}
	return nil
}
