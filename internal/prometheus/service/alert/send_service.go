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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
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
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerSendService(dao alert.AlertManagerSendDAO, ruleDao alert.AlertManagerRuleDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerSendService {
	return &alertManagerSendService{
		dao:     dao,
		ruleDao: ruleDao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

// GetMonitorSendGroupList 获取发送组列表
func (a *alertManagerSendService) GetMonitorSendGroupList(ctx context.Context, req *model.GetMonitorSendGroupListReq) (model.ListResp[*model.MonitorSendGroup], error) {
	if req.Search != "" {
		groups, total, err := a.dao.SearchMonitorSendGroupByName(ctx, req.Search)
		if err != nil {
			a.l.Error("搜索发送组失败", zap.String("search", req.Search), zap.Error(err))
			return model.ListResp[*model.MonitorSendGroup]{}, err
		}
		return model.ListResp[*model.MonitorSendGroup]{
			Total: total,
			Items: groups,
		}, nil
	}

	offset := (req.Page - 1) * req.Size
	limit := req.Size

	groups, total, err := a.dao.GetMonitorSendGroupList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取发送组列表失败", zap.Error(err))
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

	// 合并并去重用户名
	usernameSet := make(map[string]struct{})
	addToSet := func(names []string) {
		for _, name := range names {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				usernameSet[trimmed] = struct{}{}
			}
		}
	}

	addToSet(monitorSendGroup.StaticReceiveUserNames)
	addToSet(monitorSendGroup.FirstUserNames)
	addToSet(monitorSendGroup.SecondUserNames)

	usernames := make([]string, 0, len(usernameSet))
	for name := range usernameSet {
		usernames = append(usernames, name)
	}

	// 批量获取用户
	userMap := make(map[string]*model.User)
	if len(usernames) > 0 {
		users, err := a.userDao.GetUserByUsernames(ctx, usernames)
		if err != nil {
			a.l.Error("批量获取用户失败",
				zap.Strings("usernames", usernames),
				zap.Error(err))
			return fmt.Errorf("用户数据获取失败")
		}
		for _, u := range users {
			userMap[u.Username] = u
		}

		// 检查无效用户名
		var missingUsers []string
		checkMissing := func(names []string) {
			for _, name := range names {
				if name == "" {
					continue
				}
				if _, ok := userMap[name]; !ok {
					missingUsers = append(missingUsers, name)
				}
			}
		}

		checkMissing(monitorSendGroup.StaticReceiveUserNames)
		checkMissing(monitorSendGroup.FirstUserNames)
		checkMissing(monitorSendGroup.SecondUserNames)
		if len(missingUsers) > 0 {
			return fmt.Errorf("以下用户不存在: %s", strings.Join(missingUsers, ", "))
		}
	}

	// 按输入顺序映射用户
	mapUsers := func(names []string) []*model.User {
		result := make([]*model.User, 0, len(names))
		for _, name := range names {
			if user := userMap[name]; user != nil {
				result = append(result, user)
			}
		}
		return result
	}

	monitorSendGroup.StaticReceiveUsers = mapUsers(monitorSendGroup.StaticReceiveUserNames)
	monitorSendGroup.FirstUpgradeUsers = mapUsers(monitorSendGroup.FirstUserNames)
	monitorSendGroup.SecondUpgradeUsers = mapUsers(monitorSendGroup.SecondUserNames)

	// 更新发送组
	if err := a.dao.UpdateMonitorSendGroup(ctx, monitorSendGroup); err != nil {
		a.l.Error("更新发送组失败", zap.Error(err))
		return fmt.Errorf("更新失败，请稍后重试")
	}

	return nil
}

// DeleteMonitorSendGroup 删除发送组
func (a *alertManagerSendService) DeleteMonitorSendGroup(ctx context.Context, req *model.DeleteMonitorSendGroupReq) error {
	// 检查发送组是否有关联的资源
	_, total, err := a.ruleDao.GetAssociatedResourcesBySendGroupId(ctx, req.ID)
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

	return nil
}

// GetMonitorSendGroup 获取发送组详情
func (a *alertManagerSendService) GetMonitorSendGroup(ctx context.Context, req *model.GetMonitorSendGroupReq) (*model.MonitorSendGroup, error) {
	return a.dao.GetMonitorSendGroupById(ctx, req.ID)
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
