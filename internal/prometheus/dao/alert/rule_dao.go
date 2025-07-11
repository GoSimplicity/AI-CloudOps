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
	GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) ([]*model.MonitorAlertRule, int64, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	GetMonitorAlertRuleById(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	UpdateMonitorAlertRule(ctx context.Context, req *model.UpdateMonitorAlertRuleReq) error
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
		Where("pool_id = ?", poolId).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
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
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Count(&count).Error; err != nil {
		a.l.Error("获取搜索结果总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取数据列表
	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&alertRules).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertRule 失败", zap.Error(err))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// GetMonitorAlertRuleList 获取所有 MonitorAlertRule
func (a *alertManagerRuleDAO) GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) ([]*model.MonitorAlertRule, int64, error) {
	var alertRules []*model.MonitorAlertRule
	var count int64

	query := a.db.WithContext(ctx).Model(&model.MonitorAlertRule{})

	// 添加筛选条件
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	if req.Severity != "" {
		query = query.Where("severity = ?", req.Severity)
	}

	if req.PoolID != nil {
		query = query.Where("pool_id = ?", *req.PoolID)
	}

	if req.SendGroupID != nil {
		query = query.Where("send_group_id = ?", *req.SendGroupID)
	}

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取数据列表
	if err := query.Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&alertRules).Error; err != nil {
		a.l.Error("获取所有 MonitorAlertRule 失败", zap.Error(err))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// CreateMonitorAlertRule 创建 MonitorAlertRule
func (a *alertManagerRuleDAO) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
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

	if err := a.db.WithContext(ctx).Where("id = ?", id).First(&alertRule).Error; err != nil {
		a.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertRule, nil
}

// UpdateMonitorAlertRule 更新 MonitorAlertRule
func (a *alertManagerRuleDAO) UpdateMonitorAlertRule(ctx context.Context, req *model.UpdateMonitorAlertRuleReq) error {
	if req.ID <= 0 {
		a.l.Error("UpdateMonitorAlertRule 失败: ID 为 0", zap.Any("req", req))
		return fmt.Errorf("MonitorAlertRule 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", req.ID).
		Updates(map[string]interface{}{
			"name":          req.Name,
			"pool_id":       req.PoolID,
			"send_group_id": req.SendGroupID,
			"ip_address":    req.IpAddress,
			"enable":        req.Enable,
			"expr":          req.Expr,
			"severity":      req.Severity,
			"grafana_link":  req.GrafanaLink,
			"for_time":      req.ForTime,
			"labels":        req.Labels,
			"annotations":   req.Annotations,
		}).Error; err != nil {
		a.l.Error("更新 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", req.ID))
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

	// 先获取当前状态
	var rule model.MonitorAlertRule
	if err := a.db.WithContext(ctx).Select("enable").Where("id = ?", ruleID).First(&rule).Error; err != nil {
		a.l.Error("获取告警规则状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	// 切换状态
	var newStatus int8
	if rule.Enable == 1 {
		newStatus = 2 // 禁用
	} else {
		newStatus = 1 // 启用
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("id = ?", ruleID).
		Update("enable", newStatus).Error; err != nil {
		a.l.Error("切换告警规则状态失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// BatchEnableSwitchMonitorAlertRule 批量切换 MonitorAlertRule 状态
func (a *alertManagerRuleDAO) BatchEnableSwitchMonitorAlertRule(ctx context.Context, ruleIDs []int) error {
	if len(ruleIDs) == 0 {
		return nil
	}

	// 查询当前记录状态
	var rules []*model.MonitorAlertRule
	if err := a.db.WithContext(ctx).
		Select("id", "enable").
		Where("id IN ?", ruleIDs).
		Find(&rules).Error; err != nil {
		a.l.Error("获取告警规则状态失败", zap.Error(err))
		return err
	}

	// 根据当前状态分组
	enableIDs := make([]int, 0)
	disableIDs := make([]int, 0)

	for _, rule := range rules {
		if rule.Enable == 1 {
			disableIDs = append(disableIDs, rule.ID) // 当前启用的，改为禁用
		} else {
			enableIDs = append(enableIDs, rule.ID) // 当前禁用的，改为启用
		}
	}

	// 批量更新启用状态
	if len(enableIDs) > 0 {
		if err := a.db.WithContext(ctx).
			Model(&model.MonitorAlertRule{}).
			Where("id IN ?", enableIDs).
			Update("enable", 1).Error; err != nil {
			a.l.Error("批量启用告警规则失败", zap.Error(err))
			return err
		}
	}

	// 批量更新禁用状态
	if len(disableIDs) > 0 {
		if err := a.db.WithContext(ctx).
			Model(&model.MonitorAlertRule{}).
			Where("id IN ?", disableIDs).
			Update("enable", 2).Error; err != nil {
			a.l.Error("批量禁用告警规则失败", zap.Error(err))
			return err
		}
	}

	return nil
}

// DeleteMonitorAlertRule 删除 MonitorAlertRule
func (a *alertManagerRuleDAO) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		a.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	if err := a.db.WithContext(ctx).
		Where("id = ?", ruleID).
		Delete(&model.MonitorAlertRule{}).Error; err != nil {
		a.l.Error("删除告警规则失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// GetAssociatedResourcesBySendGroupId 获取与发送组关联的告警规则
func (a *alertManagerRuleDAO) GetAssociatedResourcesBySendGroupId(ctx context.Context, sendGroupId int) ([]*model.MonitorAlertRule, int64, error) {
	if sendGroupId <= 0 {
		a.l.Error("GetAssociatedResourcesBySendGroupId 失败: 无效的 sendGroupId", zap.Int("sendGroupId", sendGroupId))
		return nil, 0, fmt.Errorf("无效的 sendGroupId: %d", sendGroupId)
	}

	var rules []*model.MonitorAlertRule
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("send_group_id = ?", sendGroupId).
		Count(&count).Error; err != nil {
		a.l.Error("获取关联告警规则总数失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Where("send_group_id = ?", sendGroupId).
		Find(&rules).Error; err != nil {
		a.l.Error("获取关联告警规则失败", zap.Error(err), zap.Int("sendGroupId", sendGroupId))
		return nil, 0, err
	}

	return rules, count, nil
}

// CheckMonitorAlertRuleExists 检查 MonitorAlertRule 是否存在
func (a *alertManagerRuleDAO) CheckMonitorAlertRuleExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("name = ? AND pool_id = ? AND id != ?", alertRule.Name, alertRule.PoolID, alertRule.ID).
		Count(&count).Error; err != nil {
		a.l.Error("检查告警规则是否存在失败", zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

// CheckMonitorAlertRuleNameExists 检查 MonitorAlertRule 名称是否存在
func (a *alertManagerRuleDAO) CheckMonitorAlertRuleNameExists(ctx context.Context, alertRule *model.MonitorAlertRule) (bool, error) {
	var count int64
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("name = ?", alertRule.Name).
		Where("id != ?", alertRule.ID).
		Count(&count).Error; err != nil {
		a.l.Error("检查告警规则名称是否存在失败", zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

// GetMonitorAlertRuleTotal 获取 MonitorAlertRule 总数
func (a *alertManagerRuleDAO) GetMonitorAlertRuleTotal(ctx context.Context) (int, error) {
	var count int64
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Count(&count).Error; err != nil {
		a.l.Error("获取告警规则总数失败", zap.Error(err))
		return 0, err
	}
	return int(count), nil
}
