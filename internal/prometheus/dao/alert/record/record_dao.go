package record

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
)

type AlertManagerRecordDAO interface {
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error)
	SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error)
	GetMonitorRecordRuleList(ctx context.Context) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error)
	UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, ruleID int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error
	CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
	CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
}

type alertManagerRecordDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerRecordDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerRecordDAO {
	return &alertManagerRecordDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

func (a *alertManagerRecordDAO) GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorRecordRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var recordRules []*model.MonitorRecordRule
	if err := a.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&recordRules).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return recordRules, nil
}

func (a *alertManagerRecordDAO) SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error) {
	if name == "" {
		return nil, fmt.Errorf("name 不能为空")
	}

	var recordRules []*model.MonitorRecordRule

	if err := a.db.WithContext(ctx).
		Where("name LIKE ?", "%"+name+"%").
		Find(&recordRules).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorRecordRule 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return recordRules, nil
}

func (a *alertManagerRecordDAO) GetMonitorRecordRuleList(ctx context.Context) ([]*model.MonitorRecordRule, error) {
	var recordRules []*model.MonitorRecordRule

	if err := a.db.WithContext(ctx).Find(&recordRules).Error; err != nil {
		a.l.Error("获取所有 MonitorRecordRule 失败", zap.Error(err))
		return nil, err
	}

	return recordRules, nil
}

func (a *alertManagerRecordDAO) CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	if recordRule == nil {
		a.l.Error("CreateMonitorRecordRule 失败: recordRule 为 nil")
		return fmt.Errorf("monitorRecordRule 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(recordRule).Error; err != nil {
		a.l.Error("创建 MonitorRecordRule 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordDAO) GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error) {
	if id <= 0 {
		a.l.Error("GetMonitorRecordRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var recordRule model.MonitorRecordRule
	if err := a.db.WithContext(ctx).First(&recordRule, id).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &recordRule, nil
}

func (a *alertManagerRecordDAO) UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	if recordRule == nil {
		a.l.Error("UpdateMonitorRecordRule 失败: recordRule 为 nil")
		return fmt.Errorf("monitorRecordRule 不能为空")
	}

	if recordRule.ID == 0 {
		a.l.Error("UpdateMonitorRecordRule 失败: ID 为 0", zap.Any("recordRule", recordRule))
		return fmt.Errorf("monitorRecordRule 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", recordRule.ID).
		Updates(recordRule).Error; err != nil {
		a.l.Error("更新 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", recordRule.ID))
		return err
	}

	return nil
}

func (a *alertManagerRecordDAO) DeleteMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("DeleteMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorRecordRule{}, ruleID)
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorRecordRule 失败: %w", ruleID, err)
	}

	return nil
}

func (a *alertManagerRecordDAO) EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("EnableSwitchMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	// 获取当前规则的状态
	var rule model.MonitorRecordRule
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", ruleID).
		First(&rule).Error; err != nil {
		a.l.Error("查询 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	// 切换状态，1->2 或 2->1·
	newEnable := 1
	if rule.Enable == 1 {
		newEnable = 2
	} else if rule.Enable == 2 {
		newEnable = 1
	}

	// 更新状态
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", ruleID).
		Update("enable", newEnable).Error; err != nil {
		a.l.Error("更新 MonitorRecordRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

func (a *alertManagerRecordDAO) CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", recordRule.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *alertManagerRecordDAO) CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("name = ?", recordRule.Name).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
