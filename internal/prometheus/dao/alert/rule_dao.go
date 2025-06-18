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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerRuleDAO interface {
	GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, int64, error)
	SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, int64, error)
	GetMonitorAlertRuleList(ctx context.Context, offset, limit int) ([]*model.MonitorAlertRule, int64, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error
	BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error
	DeleteMonitorAlertRule(ctx context.Context, ruleID int) error
	GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, int64, error)
	CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
	CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error)
	GetMonitorAlertRuleTotal(ctx context.Context) (int, error)
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

// GetMonitorAlertRuleByPoolId 通过 poolId 获取 MonitorAlertRule
func (a *alertManagerRuleDAO) GetMonitorAlertRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorAlertRule, int64, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorAlertRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, 0, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var alertRules []*model.MonitorAlertRule
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("enable = ?", true).
		Where("pool_id = ? AND deleted_at = ?", poolId, 0).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ? AND deleted_at = ?", poolId, 0).
		Find(&alertRules).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// SearchMonitorAlertRuleByName 通过名称搜索 MonitorAlertRule
func (a *alertManagerRuleDAO) SearchMonitorAlertRuleByName(ctx context.Context, name string) ([]*model.MonitorAlertRule, int64, error) {
	var alertRules []*model.MonitorAlertRule
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("LOWER(name) LIKE ? AND deleted_at = ?", "%"+strings.ToLower(name)+"%", 0).
		Count(&count).Error; err != nil {
		a.l.Error("获取搜索结果总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ? AND deleted_at = ?", "%"+strings.ToLower(name)+"%", 0).
		Find(&alertRules).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertRule 失败", zap.Error(err))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// GetMonitorAlertRuleList 获取所有 MonitorAlertRule
func (a *alertManagerRuleDAO) GetMonitorAlertRuleList(ctx context.Context, offset, limit int) ([]*model.MonitorAlertRule, int64, error) {
	var alertRules []*model.MonitorAlertRule
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("deleted_at = ?", 0).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).Offset(offset).Limit(limit).Find(&alertRules).Error; err != nil {
		a.l.Error("获取所有 MonitorAlertRule 失败", zap.Error(err))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// CreateMonitorAlertRule 创建 MonitorAlertRule
func (a *alertManagerRuleDAO) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	monitorAlertRule.UpdatedAt = getTime()
	monitorAlertRule.CreatedAt = getTime()

	if err := a.db.WithContext(ctx).Create(monitorAlertRule).Error; err != nil {
		a.l.Error("创建 MonitorAlertRule 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertRuleById 通过 ID 获取 MonitorAlertRule
func (a *alertManagerRuleDAO) GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertRuleById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertRule model.MonitorAlertRule

	if err := a.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", id, 0).First(&alertRule).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertRule, nil
}

// UpdateMonitorAlertRule 更新 MonitorAlertRule
func (a *alertManagerRuleDAO) UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	if monitorAlertRule.ID <= 0 {
		a.l.Error("UpdateMonitorAlertRule 失败: ID 为 0", zap.Any("alertRule", monitorAlertRule))
		return fmt.Errorf("monitorAlertRule 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ? AND deleted_at = ?", monitorAlertRule.ID, 0).
		Updates(map[string]interface{}{
			"name":          monitorAlertRule.Name,
			"pool_id":       monitorAlertRule.PoolID,
			"send_group_id": monitorAlertRule.SendGroupID,
			"ip_address":    monitorAlertRule.IpAddress,
			"enable":        monitorAlertRule.Enable,
			"expr":          monitorAlertRule.Expr,
			"severity":      monitorAlertRule.Severity,
			"grafana_link":  monitorAlertRule.GrafanaLink,
			"for_time":      monitorAlertRule.ForTime,
			"labels":        monitorAlertRule.Labels,
			"annotations":   monitorAlertRule.Annotations,
			"updated_at":    getTime(),
		}).Error; err != nil {
		a.l.Error("更新 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", monitorAlertRule.ID))
		return err
	}

	return nil
}

// EnableSwitchMonitorAlertRule 切换 MonitorAlertRule 状态
func (a *alertManagerRuleDAO) EnableSwitchMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("EnableSwitchMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ? AND deleted_at = ?", ruleID, 0).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		a.l.Error("更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换 MonitorAlertRule 状态
func (a *alertManagerRuleDAO) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		a.l.Error("BatchEnableSwitchMonitorAlertRule 失败: ruleIDs 为空")
		return fmt.Errorf("ruleIDs 不能为空")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id IN ? AND deleted_at = ?", ruleIDs, 0).
		Update("enable", gorm.Expr("NOT enable")).Error; err != nil {
		a.l.Error("批量更新 MonitorAlertRule 状态失败", zap.Error(err), zap.Ints("ruleIDs", ruleIDs))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除 MonitorAlertRule
func (a *alertManagerRuleDAO) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	result := a.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("id = ? AND deleted_at = ?", ruleID, 0).Updates(map[string]interface{}{
		"deleted_at": getTime(),
	})
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorAlertRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertRule 失败: %w", ruleID, err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的 MonitorAlertRule", ruleID)
	}

	return nil
}

// GetAssociatedResourcesBySendGroupId 通过 sendGroupId 获取关联的 MonitorAlertRule
func (a *alertManagerRuleDAO) GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, int64, error) {
	if sendGroupId <= 0 {
		a.l.Error("GetAssociatedResourcesBySendGroupId 失败: 无效的 sendGroupId", zap.Int("sendGroupId", sendGroupId))
		return nil, 0, fmt.Errorf("无效的 sendGroupId: %d", sendGroupId)
	}

	var alertRules []*model.MonitorAlertRule
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("send_group_id = ? AND deleted_at = ?", sendGroupId, 0).
		Count(&count).Error; err != nil {
		a.l.Error("获取关联资源总数失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).
		Where("send_group_id = ? AND deleted_at = ?", sendGroupId, 0).
		Find(&alertRules).Error; err != nil {
		a.l.Error("获取关联资源失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// CheckMonitorAlertRuleExists 检查 MonitorAlertRule 是否存在
func (a *alertManagerRuleDAO) CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	if alertRule.ID <= 0 {
		a.l.Error("CheckMonitorAlertRuleExists 失败: 无效的 ID", zap.Int("id", alertRule.ID))
		return false, fmt.Errorf("无效的 ID: %d", alertRule.ID)
	}

	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ? AND deleted_at = ?", alertRule.ID, 0).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorAlertRule 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// CheckMonitorAlertRuleNameExists 检查 MonitorAlertRule 名称是否存在
func (a *alertManagerRuleDAO) CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("name = ? AND deleted_at = ?", alertRule.Name, 0).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorAlertRule 名称是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorAlertRuleTotal 获取监控告警事件总数
func (a *alertManagerRuleDAO) GetMonitorAlertRuleTotal(ctx context.Context) (int, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorAlertRule{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		a.l.Error("获取监控告警事件总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
