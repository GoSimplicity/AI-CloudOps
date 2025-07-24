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
	GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, int64, error)
	GetMonitorRecordRuleList(ctx context.Context, req *model.GetMonitorRecordRuleListReq) ([]*model.MonitorRecordRule, int64, error)
	CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	GetMonitorRecordRuleById(ctx context.Context, id int) (*model.MonitorRecordRule, error)
	UpdateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error
	DeleteMonitorRecordRule(ctx context.Context, ruleID int) error
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

// GetMonitorRecordRuleByPoolId 通过 poolId 获取 MonitorRecordRule
func (a *alertManagerRecordDAO) GetMonitorRecordRuleByPoolId(ctx context.Context, poolId int) ([]*model.MonitorRecordRule, int64, error) {
	if poolId <= 0 {
		a.l.Error("GetMonitorRecordRuleByPoolId 失败: 无效的 poolId", zap.Int("poolId", poolId))
		return nil, 0, fmt.Errorf("无效的 poolId: %d", poolId)
	}

	var recordRules []*model.MonitorRecordRule
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("pool_id = ?", poolId).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 总数失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Where("enable = ?", true).
		Where("pool_id = ?", poolId).
		Find(&recordRules).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 失败", zap.Error(err), zap.Int("poolId", poolId))
		return nil, 0, err
	}

	return recordRules, count, nil
}

// GetMonitorRecordRuleList 获取 MonitorRecordRule 列表
func (a *alertManagerRecordDAO) GetMonitorRecordRuleList(ctx context.Context, req *model.GetMonitorRecordRuleListReq) ([]*model.MonitorRecordRule, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}

	if req.Size <= 0 {
		req.Size = 10
	}

	// 计算分页参数
	offset := (req.Page - 1) * req.Size
	limit := req.Size

	query := a.db.WithContext(ctx).Model(&model.MonitorRecordRule{})

	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.PoolID != nil {
		query = query.Where("pool_id = ?", *req.PoolID)
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	var recordRules []*model.MonitorRecordRule
	var count int64

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 再获取分页数据
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&recordRules).Error; err != nil {
		a.l.Error("获取 MonitorRecordRule 列表失败", zap.Error(err))
		return nil, 0, err
	}

	return recordRules, count, nil
}

// CreateMonitorRecordRule 创建 MonitorRecordRule
func (a *alertManagerRecordDAO) CreateMonitorRecordRule(ctx context.Context, recordRule *model.MonitorRecordRule) error {
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

	if err := a.db.WithContext(ctx).Where("id = ?", id).First(&recordRule).Error; err != nil {
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

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("id = ?", recordRule.ID).
		Updates(recordRule).Error; err != nil {
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

	// 执行删除操作
	if err := a.db.WithContext(ctx).
		Where("id = ?", ruleID).
		Delete(&model.MonitorRecordRule{}).Error; err != nil {
		a.l.Error("删除 MonitorRecordRule 失败", zap.Error(err), zap.Int("ruleID", ruleID))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorRecordRule 失败: %w", ruleID, err)
	}

	return nil
}

// CheckMonitorRecordRuleNameExists 检查 MonitorRecordRule 名称是否存在
func (a *alertManagerRecordDAO) CheckMonitorRecordRuleNameExists(ctx context.Context, recordRule *model.MonitorRecordRule) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorRecordRule{}).
		Where("name = ?", recordRule.Name).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorRecordRule 名称是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
