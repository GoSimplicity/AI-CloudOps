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
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerSendService interface {
	GetMonitorSendGroupList(ctx context.Context, req *model.GetMonitorSendGroupListReq) (model.ListResp[*model.MonitorSendGroup], error)
	CreateMonitorSendGroup(ctx context.Context, req *model.CreateMonitorSendGroupReq) error
	UpdateMonitorSendGroup(ctx context.Context, req *model.UpdateMonitorSendGroupReq) error
	DeleteMonitorSendGroup(ctx context.Context, req *model.DeleteMonitorSendGroupReq) error
	GetMonitorSendGroup(ctx context.Context, req *model.GetMonitorSendGroupReq) (*model.MonitorSendGroup, error)
	GetMonitorSendGroupAll(ctx context.Context) (model.ListResp[*model.MonitorSendGroup], error)
}

type alertManagerSendService struct {
	dao     alert.AlertManagerSendDAO
	ruleDao alert.AlertManagerRuleDAO
	userDao userDao.UserDAO
	cache   cache.MonitorCache
	l       *zap.Logger
}

func NewAlertManagerSendService(dao alert.AlertManagerSendDAO, ruleDao alert.AlertManagerRuleDAO, l *zap.Logger, userDao userDao.UserDAO, cache cache.MonitorCache) AlertManagerSendService {
	return &alertManagerSendService{
		dao:     dao,
		ruleDao: ruleDao,
		userDao: userDao,
		cache:   cache,
		l:       l,
	}
}

// GetMonitorSendGroupList 获取发送组列表
func (a *alertManagerSendService) GetMonitorSendGroupList(ctx context.Context, req *model.GetMonitorSendGroupListReq) (model.ListResp[*model.MonitorSendGroup], error) {
	groups, total, err := a.dao.GetMonitorSendGroupList(ctx, req)
	if err != nil {
		a.l.Error("搜索发送组失败", zap.String("search", req.Search), zap.Error(err))
		return model.ListResp[*model.MonitorSendGroup]{}, err
	}

	return model.ListResp[*model.MonitorSendGroup]{
		Total: total,
		Items: groups,
	}, nil
}

// CreateMonitorSendGroup 创建发送组
func (a *alertManagerSendService) CreateMonitorSendGroup(ctx context.Context, req *model.CreateMonitorSendGroupReq) error {
	monitorSendGroup := &model.MonitorSendGroup{
		Name:                req.Name,
		NameZh:              req.NameZh,
		Enable:              req.Enable,
		UserID:              req.UserID,
		PoolID:              req.PoolID,
		OnDutyGroupID:       req.OnDutyGroupID,
		StaticReceiveUsers:  req.StaticReceiveUsers,
		FeiShuQunRobotToken: req.FeiShuQunRobotToken,
		RepeatInterval:      req.RepeatInterval,
		SendResolved:        req.SendResolved,
		NotifyMethods:       req.NotifyMethods,
		NeedUpgrade:         req.NeedUpgrade,
		FirstUpgradeUsers:   req.FirstUpgradeUsers,
		UpgradeMinutes:      req.UpgradeMinutes,
		SecondUpgradeUsers:  req.SecondUpgradeUsers,
		CreateUserName:      req.CreateUserName,
	}

	// 检查发送组是否已存在
	exists, err := a.dao.CheckMonitorSendGroupNameExists(ctx, monitorSendGroup)
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

	go func() {
		if err := a.cache.MonitorCacheManager(context.Background()); err != nil {
			a.l.Error("创建发送组后刷新缓存失败", zap.Error(err))
		}
	}()

	return nil
}

// UpdateMonitorSendGroup 更新发送组
func (a *alertManagerSendService) UpdateMonitorSendGroup(ctx context.Context, req *model.UpdateMonitorSendGroupReq) error {
	monitorSendGroup := &model.MonitorSendGroup{
		Model: model.Model{
			ID: req.ID,
		},
		Name:                req.Name,
		NameZh:              req.NameZh,
		Enable:              req.Enable,
		PoolID:              req.PoolID,
		OnDutyGroupID:       req.OnDutyGroupID,
		StaticReceiveUsers:  req.StaticReceiveUsers,
		FeiShuQunRobotToken: req.FeiShuQunRobotToken,
		RepeatInterval:      req.RepeatInterval,
		SendResolved:        req.SendResolved,
		NotifyMethods:       req.NotifyMethods,
		NeedUpgrade:         req.NeedUpgrade,
		FirstUpgradeUsers:   req.FirstUpgradeUsers,
		UpgradeMinutes:      req.UpgradeMinutes,
		SecondUpgradeUsers:  req.SecondUpgradeUsers,
	}

	// 检查发送组是否存在
	exists, err := a.dao.CheckMonitorSendGroupExists(ctx, monitorSendGroup)
	if err != nil {
		a.l.Error("检查发送组存在失败", zap.Int("id", monitorSendGroup.ID), zap.Error(err))
		return fmt.Errorf("系统错误，请稍后重试")
	}

	if !exists {
		return fmt.Errorf("发送组不存在或已被删除")
	}

	// 检查名称是否已被其他发送组使用
	exists, err = a.dao.CheckMonitorSendGroupNameExists(ctx, monitorSendGroup)
	if err != nil {
		a.l.Error("更新发送组失败：检查发送组名称是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("发送组名称已被使用")
	}

	// 更新发送组
	if err := a.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		a.l.Error("更新发送组失败", zap.Error(err))
		return fmt.Errorf("更新失败，请稍后重试")
	}

	go func() {
		if err := a.cache.MonitorCacheManager(context.Background()); err != nil {
			a.l.Error("更新发送组后刷新缓存失败", zap.Error(err))
		}
	}()

	return nil
}

// DeleteMonitorSendGroup 删除发送组
func (a *alertManagerSendService) DeleteMonitorSendGroup(ctx context.Context, req *model.DeleteMonitorSendGroupReq) error {
	// 检查发送组是否有关联的资源
	_, total, err := a.ruleDao.GetAssociatedResourcesBySendGroupID(ctx, req.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除发送组失败：获取关联资源时出错", zap.Error(err))
		return err
	}

	if total > 0 {
		return errors.New("发送组存在关联资源，无法删除")
	}

	// 删除发送组
	if err := a.dao.DeleteMonitorSendGroup(ctx, req.ID); err != nil {
		a.l.Error("删除发送组失败", zap.Error(err))
		return err
	}

	go func() {
		if err := a.cache.MonitorCacheManager(context.Background()); err != nil {
			a.l.Error("删除发送组后刷新缓存失败", zap.Error(err))
		}
	}()

	return nil
}

// GetMonitorSendGroup 获取发送组详情
func (a *alertManagerSendService) GetMonitorSendGroup(ctx context.Context, req *model.GetMonitorSendGroupReq) (*model.MonitorSendGroup, error) {
	group, err := a.dao.GetMonitorSendGroupByID(ctx, req.ID)
	if err != nil {
		a.l.Error("获取发送组详情失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	// 处理用户名列表
	if group.StaticReceiveUsers != nil {
		group.StaticReceiveUserNames = make([]string, 0, len(group.StaticReceiveUsers))
		for _, user := range group.StaticReceiveUsers {
			group.StaticReceiveUserNames = append(group.StaticReceiveUserNames, user.Username)
		}
	}

	if group.FirstUpgradeUsers != nil {
		group.FirstUserNames = make([]string, 0, len(group.FirstUpgradeUsers))
		for _, user := range group.FirstUpgradeUsers {
			group.FirstUserNames = append(group.FirstUserNames, user.Username)
		}
	}

	if group.SecondUpgradeUsers != nil {
		group.SecondUserNames = make([]string, 0, len(group.SecondUpgradeUsers))
		for _, user := range group.SecondUpgradeUsers {
			group.SecondUserNames = append(group.SecondUserNames, user.Username)
		}
	}

	return group, nil
}

// GetMonitorSendGroupAll 获取所有发送组
func (a *alertManagerSendService) GetMonitorSendGroupAll(ctx context.Context) (model.ListResp[*model.MonitorSendGroup], error) {
	groups, count, err := a.dao.GetMonitorSendGroups(ctx)
	if err != nil {
		a.l.Error("获取所有发送组失败", zap.Error(err))
		return model.ListResp[*model.MonitorSendGroup]{}, err
	}
	return model.ListResp[*model.MonitorSendGroup]{
		Items: groups,
		Total: count,
	}, nil
}
