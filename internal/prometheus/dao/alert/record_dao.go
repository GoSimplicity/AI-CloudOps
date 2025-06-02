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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerRecordDAO interface {
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error)
	SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error)
	GetMonitorRecordRuleList(ctx context.Context, offset, limit int) ([]*model.MonitorRecordRule, error)
	CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error)
	UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, ruleID int) error
	EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error
	CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
	CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error)
	GetMonitorRecordRuleTotal(ctx context.Context) (int, error)
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

// GetMonitorRecordRuleByPoolId 通过 poolId 获取 MonitorRecordRule
func (a *alertManagerRecordDAO) GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorRecordRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var recordRules []*model.MonitorRecordRule

	if err := a.db.WithContext(ctx).
		Where("enable = ? AND deleted_at = ?", true, 0).
		Where("pool_id = ?", poolId).
		Find(&recordRules).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, err
	}

	return recordRules, nil
}

// SearchMonitorRecordRuleByName 通过名称搜索 MonitorRecordRule
func (a *alertManagerRecordDAO) SearchMonitorRecordRuleByName(ctx context.Context, name string) ([]*model.MonitorRecordRule, error) {
	if name == "" {
		return nil, fmt.Errorf("name 不能为空")
	}

	var recordRules []*model.MonitorRecordRule

	if err := a.db.WithContext(ctx).
		Where("name LIKE ? AND deleted_at = ?", "%"+name+"%", 0).
		Find(&recordRules).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorRecordRule 失败", zap.Error(err), zap.String("name", name))
		return nil, err
	}

	return recordRules, nil
}

// GetMonitorRecordRuleList 获取 MonitorRecordRule 列表
func (a *alertManagerRecordDAO) GetMonitorRecordRuleList(ctx context.Context, offset, limit int) ([]*model.MonitorRecordRule, error) {
	var recordRules []*model.MonitorRecordRule

	if offset < 0 || limit <= 0 {
		return nil, fmt.Errorf("无效的分页参数: offset=%d, limit=%d", offset, limit)
	}

	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).Offset(offset).Limit(limit).Find(&recordRules).Error; err != nil {
		a.l.Error("获取所有 MonitorRecordRule 失败", zap.Error(err))
		return nil, err
	}

	return recordRules, nil
}

// CreateMonitorRecordRule 创建 MonitorRecordRule
func (a *alertManagerRecordDAO) CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	recordRule.CreatedAt = getTime()
	recordRule.UpdatedAt = getTime()

	if err := a.db.WithContext(ctx).Create(recordRule).Error; err != nil {
		a.l.Error("创建 MonitorRecordRule 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorRecordRuleById 通过 ID 获取 MonitorRecordRule
func (a *alertManagerRecordDAO) GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error) {
	if id <= 0 {
		a.l.Error("GetMonitorRecordRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var recordRule model.MonitorRecordRule

	if err := a.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", id, 0).First(&recordRule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的 MonitorRecordRule", id)
		}
		a.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &recordRule, nil
}

// UpdateMonitorRecordRule 更新 MonitorRecordRule
func (a *alertManagerRecordDAO) UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
	if recordRule.ID == 0 {
		a.l.Error("UpdateMonitorRecordRule 失败: ID 为 0", zap.Any("recordRule", recordRule))
		return fmt.Errorf("monitorRecordRule 的 ID 必须设置且非零")
	}

	recordRule.UpdatedAt = getTime()

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ? AND deleted_at = ?", recordRule.ID, 0).
		Updates(map[string]interface{}{
			"name":        recordRule.Name,
			"pool_id":     recordRule.PoolID,
			"ip_address":  recordRule.IpAddress,
			"enable":      recordRule.Enable,
			"for_time":    recordRule.ForTime,
			"expr":        recordRule.Expr,
			"labels":      recordRule.Labels,
			"annotations": recordRule.Annotations,
			"updated_at":  getTime(),
		}).Error; err != nil {
		a.l.Error("更新 MonitorRecordRule 失败", zap.Error(err), zap.Int("id", recordRule.ID))
		return err
	}

	return nil
}

// DeleteMonitorRecordRule 删除 MonitorRecordRule
func (a *alertManagerRecordDAO) DeleteMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("DeleteMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ? AND deleted_at = ?", ruleID, 0).
		Updates(map[string]interface{}{
			"deleted_at": getTime(),
		})

	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorRecordRule 失败: %w", ruleID, err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorRecordRule", ruleID)
	}

	return nil
}

// EnableSwitchMonitorRecordRule 切换 MonitorRecordRule 状态
func (a *alertManagerRecordDAO) EnableSwitchMonitorRecordRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("EnableSwitchMonitorRecordRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	// 获取当前规则的状态
	var rule model.MonitorRecordRule

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ? AND deleted_at = ?", ruleID, 0).
		First(&rule).Error; err != nil {
		a.l.Error("查询 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	// 切换状态
	newEnable := !rule.Enable

	// 更新状态
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ? AND deleted_at = ?", ruleID, 0).
		Updates(map[string]interface{}{
			"enable":     newEnable,
			"updated_at": getTime(),
		}).Error; err != nil {
		a.l.Error("更新 MonitorRecordRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// CheckMonitorRecordRuleExists 检查 MonitorRecordRule 是否存在
func (a *alertManagerRecordDAO) CheckMonitorRecordRuleExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ? AND deleted_at = ?", recordRule.ID, 0).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorRecordRule 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// CheckMonitorRecordRuleNameExists 检查 MonitorRecordRule 名称是否存在
func (a *alertManagerRecordDAO) CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("name = ? AND deleted_at = ?", recordRule.Name, 0).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorRecordRule 名称是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorRecordRuleTotal 获取监控告警事件总数
func (a *alertManagerRecordDAO) GetMonitorRecordRuleTotal(ctx context.Context) (int, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorRecordRule{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		a.l.Error("获取监控告警事件总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
