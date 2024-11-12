package rule

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

import (
	"context"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type AlertManagerRuleDAO interface {
	GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error)
	SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, error)
	GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error
	DeleteMonitorAlertRule(ctx context.Context, ruleID int) error
	GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, error)
	CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
	CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
}

type alertManagerRuleDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerRuleDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerRuleDAO {
	return &alertManagerRuleDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

func (a *alertManagerRuleDAO) GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorAlertRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var alertRules []*model.MonitorAlertRule
	if err := a.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&alertRules).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return alertRules, nil
}

func (a *alertManagerRuleDAO) SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, error) {
	var alertRules []*model.MonitorAlertRule

	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&alertRules).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertRule 失败", zap.Error(err))
		return nil, err
	}

	return alertRules, nil
}

func (a *alertManagerRuleDAO) GetMonitorAlertRuleList(ctx context.Context) ([]*model.MonitorAlertRule, error) {
	var alertRules []*model.MonitorAlertRule

	if err := a.db.WithContext(ctx).Find(&alertRules).Error; err != nil {
		a.l.Error("获取所有 MonitorAlertRule 失败", zap.Error(err))
		return nil, err
	}

	return alertRules, nil
}

func (a *alertManagerRuleDAO) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	if monitorAlertRule == nil {
		a.l.Error("CreateMonitorAlertRule 失败: alertRule 为 nil")
		return fmt.Errorf("monitorAlertRule 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(monitorAlertRule).Error; err != nil {
		a.l.Error("创建 MonitorAlertRule 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRuleDAO) GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertRule model.MonitorAlertRule
	if err := a.db.WithContext(ctx).First(&alertRule, id).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertRule, nil
}

func (a *alertManagerRuleDAO) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	if monitorAlertRule == nil {
		a.l.Error("UpdateMonitorAlertRule 失败: alertRule 为 nil")
		return fmt.Errorf("monitorAlertRule 不能为空")
	}

	if monitorAlertRule.ID == 0 {
		a.l.Error("UpdateMonitorAlertRule 失败: ID 为 0", zap.Any("alertRule", monitorAlertRule))
		return fmt.Errorf("monitorAlertRule 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", monitorAlertRule.ID).
		Updates(monitorAlertRule).Error; err != nil {
		a.l.Error("更新 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", monitorAlertRule.ID))
		return err
	}

	return nil
}

func (a *alertManagerRuleDAO) EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("EnableSwitchMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", ruleID).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		a.l.Error("更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

func (a *alertManagerRuleDAO) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		a.l.Error("BatchEnableSwitchMonitorAlertRule 失败: ruleIDs 为空")
		return fmt.Errorf("ruleIDs 不能为空")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id IN ?", ruleIDs).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		a.l.Error("批量更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Ints("ruleIDs", ruleIDs))
		return err
	}

	return nil
}

func (a *alertManagerRuleDAO) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorAlertRule{}, ruleID)
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorAlertRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertRule 失败: %w", ruleID, err)
	}

	return nil
}

func (a *alertManagerRuleDAO) GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, error) {
	if sendGroupId <= 0 {
		a.l.Error("GetAssociatedResourcesBySendGroupId 失败: 无效的 sendGroupId", zap.Int("sendGroupId", sendGroupId))
		return nil, fmt.Errorf("无效的 sendGroupId: %d", sendGroupId)
	}

	var scrapePools []*model.MonitorAlertRule

	if err := a.db.WithContext(ctx).
		Where("send_group_id = ?", sendGroupId).
		Find(&scrapePools).Error; err != nil {
		a.l.Error("获取关联资源失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, err
	}

	return scrapePools, nil
}

func (a *alertManagerRuleDAO) CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", alertRule.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *alertManagerRuleDAO) CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("name = ?", alertRule.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
