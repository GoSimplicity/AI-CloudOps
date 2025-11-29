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
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerRuleDAO interface {
	GetMonitorAlertRuleByPoolID(ctx context.Context, poolID int) ([]*model.MonitorAlertRule, int64, error)
	GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) ([]*model.MonitorAlertRule, int64, error)
	CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error
	GetMonitorAlertRuleByID(ctx context.Context, id int) (*model.MonitorAlertRule, error)
	UpdateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.UpdateMonitorAlertRuleReq) error
	DeleteMonitorAlertRule(ctx context.Context, ruleID int) error
	GetAssociatedResourcesBySendGroupID(ctx context.Context, sendGroupID int) ([]*model.MonitorAlertRule, int64, error)
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

// GetMonitorAlertRuleByPoolID 通过 poolID 获取 MonitorAlertRule
func (d *alertManagerRuleDAO) GetMonitorAlertRuleByPoolID(ctx context.Context, poolID int) ([]*model.MonitorAlertRule, int64, error) {
	if poolID <= 0 {
		d.l.Error("GetMonitorAlertRuleByPoolID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, 0, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var alertRules []*model.MonitorAlertRule
	var count int64

	// 先获取总数
	if err := d.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("enable = ?", true).
		Where("pool_id = ?", poolID).
		Count(&count).Error; err != nil {
		d.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, 0, err
	}

	// 获取数据列表
	if err := d.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolID).
		Find(&alertRules).Error; err != nil {
		d.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// GetMonitorAlertRuleList 获取所有 MonitorAlertRule
func (d *alertManagerRuleDAO) GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) ([]*model.MonitorAlertRule, int64, error) {
	var alertRules []*model.MonitorAlertRule
	var count int64

	query := d.db.WithContext(ctx).Model(&model.MonitorAlertRule{})

	// 添加筛选条件
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	if req.Severity != nil {
		query = query.Where("severity = ?", req.Severity)
	}

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		d.l.Error("获取 MonitorAlertRule 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 获取数据列表
	if err := query.Offset((req.Page - 1) * req.Size).Limit(req.Size).Find(&alertRules).Error; err != nil {
		d.l.Error("获取所有 MonitorAlertRule 失败", zap.Error(err))
		return nil, 0, err
	}

	return alertRules, count, nil
}

// CreateMonitorAlertRule 创建 MonitorAlertRule
func (d *alertManagerRuleDAO) CreateMonitorAlertRule(ctx context.Context, monitorAlertRule *model.MonitorAlertRule) error {
	if err := d.db.WithContext(ctx).Create(monitorAlertRule).Error; err != nil {
		d.l.Error("创建 MonitorAlertRule 失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertRuleByID 通过 ID 获取 MonitorAlertRule
func (d *alertManagerRuleDAO) GetMonitorAlertRuleByID(ctx context.Context, id int) (*model.MonitorAlertRule, error) {
	if id <= 0 {
		d.l.Error("GetMonitorAlertRuleByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertRule model.MonitorAlertRule

	if err := d.db.WithContext(ctx).Where("id = ?", id).First(&alertRule).Error; err != nil {
		d.l.Error("获取 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertRule, nil
}

// UpdateMonitorAlertRule 更新 MonitorAlertRule
func (d *alertManagerRuleDAO) UpdateMonitorAlertRule(ctx context.Context, req *model.UpdateMonitorAlertRuleReq) error {
	if req.ID <= 0 {
		d.l.Error("UpdateMonitorAlertRule 失败: ID 为 0", zap.Any("req", req))
		return fmt.Errorf("MonitorAlertRule 的 ID 必须设置且非零")
	}

	if err := d.db.WithContext(ctx).
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
		d.l.Error("更新 MonitorAlertRule 失败", zap.Error(err), zap.Int("id", req.ID))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除 MonitorAlertRule
func (d *alertManagerRuleDAO) DeleteMonitorAlertRule(ctx context.Context, ruleID int) error {
	if ruleID <= 0 {
		d.l.Error("DeleteMonitorAlertRule 失败: 无效的 ruleID", zap.Int("ruleID", ruleID))
		return fmt.Errorf("无效的 ruleID: %d", ruleID)
	}

	if err := d.db.WithContext(ctx).
		Where("id = ?", ruleID).
		Delete(&model.MonitorAlertRule{}).Error; err != nil {
		d.l.Error("删除告警规则失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return err
	}

	return nil
}

// GetAssociatedResourcesBySendGroupID 获取与发送组关联的告警规则
func (d *alertManagerRuleDAO) GetAssociatedResourcesBySendGroupID(ctx context.Context, sendGroupID int) ([]*model.MonitorAlertRule, int64, error) {
	if sendGroupID <= 0 {
		d.l.Error("GetAssociatedResourcesBySendGroupID 失败: 无效的 sendGroupID", zap.Int("sendGroupID", sendGroupID))
		return nil, 0, fmt.Errorf("无效的 sendGroupID: %d", sendGroupID)
	}

	var rules []*model.MonitorAlertRule
	var count int64

	if err := d.db.WithContext(ctx).
		Model(&model.MonitorAlertRule{}).
		Where("send_group_id = ?", sendGroupID).
		Count(&count).Error; err != nil {
		d.l.Error("获取关联的告警规则总数失败", zap.Error(err), zap.Int("sendGroupID", sendGroupID))
		return nil, 0, err
	}

	if err := d.db.WithContext(ctx).
		Where("send_group_id = ?", sendGroupID).
		Find(&rules).Error; err != nil {
		d.l.Error("获取关联的告警规则失败", zap.Error(err), zap.Int("sendGroupID", sendGroupID))
		return nil, 0, err
	}

	return rules, count, nil
}
