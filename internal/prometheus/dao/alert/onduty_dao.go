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

type AlertManagerOnDutyDAO interface {
	GetAllMonitorOnDutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error)
	GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyChange *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error)
	CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error)
	GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error)
	CreateMonitorOnDutyHistory(ctx context.Context, monitorOnDutyHistory *model.MonitorOnDutyHistory) error
	GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error)
	ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error)
}

type alertManagerOnDutyDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerOnDutyDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerOnDutyDAO {
	return &alertManagerOnDutyDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

// GetAllMonitorOnDutyGroup 获取所有值班组信息
func (a *alertManagerOnDutyDAO) GetAllMonitorOnDutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if err := a.db.WithContext(ctx).
		Preload("Members").
		Find(&groups).Error; err != nil {
		a.l.Error("获取所有值班组失败", zap.Error(err))
		return nil, fmt.Errorf("获取值班组失败: %w", err)
	}

	return groups, nil
}

// CreateMonitorOnDutyGroup 创建新的值班组
func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if err := a.db.WithContext(ctx).Create(monitorOnDutyGroup).Error; err != nil {
		a.l.Error("创建值班组失败", zap.Error(err))
		return fmt.Errorf("创建值班组失败: %w", err)
	}

	return nil
}

// GetMonitorOnDutyGroupByID 根据ID获取值班组信息
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的值班组ID: %d", id)
	}

	var group model.MonitorOnDutyGroup

	if err := a.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("Members").
		First(&group).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到ID为%d的值班组", id)
		}
		a.l.Error("获取值班组失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取值班组失败: %w", err)
	}

	return &group, nil
}

// UpdateMonitorOnDutyGroup 更新值班组信息
func (a *alertManagerOnDutyDAO) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if monitorOnDutyGroup.ID <= 0 {
		return fmt.Errorf("无效的值班组ID: %d", monitorOnDutyGroup.ID)
	}

	// 使用单个事务处理所有更新操作
	return a.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先获取原有的值班组信息,确保记录存在
		var existingGroup model.MonitorOnDutyGroup
		if err := tx.Where("id = ?", monitorOnDutyGroup.ID).First(&existingGroup).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("未找到ID为%d的值班组", monitorOnDutyGroup.ID)
			}
			a.l.Error("获取原有值班组信息失败", zap.Error(err), zap.Int("id", monitorOnDutyGroup.ID))
			return fmt.Errorf("获取原有值班组信息失败: %w", err)
		}

		if err := tx.Model(&existingGroup).Updates(map[string]interface{}{
			"name":                          monitorOnDutyGroup.Name,
			"shift_days":                    monitorOnDutyGroup.ShiftDays,
			"yesterday_normal_duty_user_id": monitorOnDutyGroup.YesterdayNormalDutyUserID,
		}).Error; err != nil {
			a.l.Error("更新值班组基本信息失败", zap.Error(err), zap.Int("id", monitorOnDutyGroup.ID))
			return fmt.Errorf("更新值班组基本信息失败: %w", err)
		}

		// 更新成员关联
		if err := tx.Model(&existingGroup).Association("Members").Replace(monitorOnDutyGroup.Members); err != nil {
			a.l.Error("更新成员关联失败", zap.Error(err), zap.Int("id", monitorOnDutyGroup.ID))
			return fmt.Errorf("更新成员关联失败: %w", err)
		}

		return nil
	})
}

// DeleteMonitorOnDutyGroup 删除值班组
func (a *alertManagerOnDutyDAO) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("无效的值班组ID: %d", id)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorOnDutyGroup{}, id)
	if err := result.Error; err != nil {
		a.l.Error("删除值班组失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除值班组失败: %w", err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到ID为%d的值班组", id)
	}

	return nil
}

// CreateMonitorOnDutyGroupChange 创建值班组变更记录
func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error {
	if err := a.db.WithContext(ctx).Create(monitorOnDutyGroupChange).Error; err != nil {
		a.l.Error("创建值班组变更记录失败", zap.Error(err))
		return fmt.Errorf("创建值班组变更记录失败: %w", err)
	}

	return nil
}

// GetMonitorOnDutyChangesByGroupAndTimeRange 获取指定时间范围内的值班组变更记录
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("无效的值班组ID: %d", groupID)
	}

	if startTime == "" || endTime == "" {
		return nil, fmt.Errorf("开始时间和结束时间不能为空")
	}

	var changes []*model.MonitorOnDutyChange

	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date BETWEEN ? AND ?", groupID, startTime, endTime).
		Find(&changes).Error; err != nil {
		a.l.Error("获取值班组变更记录失败", zap.Error(err), zap.Int("groupID", groupID))
		return nil, fmt.Errorf("获取值班组变更记录失败: %w", err)
	}

	return changes, nil
}

// CheckMonitorOnDutyGroupExists 检查值班组是否存在
func (a *alertManagerOnDutyDAO) CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("name = ?", onDutyGroup.Name).
		Count(&count).Error; err != nil {
		a.l.Error("检查值班组存在性失败", zap.Error(err), zap.String("name", onDutyGroup.Name))
		return false, fmt.Errorf("检查值班组存在性失败: %w", err)
	}

	return count > 0, nil
}

// GetMonitorOnDutyHistoryByGroupIDAndTimeRange 获取指定时间范围内的值班历史记录
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("无效的值班组ID: %d", groupID)
	}

	if startTime == "" || endTime == "" {
		return nil, fmt.Errorf("开始时间和结束时间不能为空")
	}

	var historyList []*model.MonitorOnDutyHistory

	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string BETWEEN ? AND ?", groupID, startTime, endTime).
		Find(&historyList).Error; err != nil {
		a.l.Error("获取值班历史记录失败", zap.Error(err), zap.Int("groupID", groupID))
		return nil, fmt.Errorf("获取值班历史记录失败: %w", err)
	}

	return historyList, nil
}

// CreateMonitorOnDutyHistory 创建值班历史记录
func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyHistory(ctx context.Context, monitorOnDutyHistory *model.MonitorOnDutyHistory) error {
	if err := a.db.WithContext(ctx).Create(monitorOnDutyHistory).Error; err != nil {
		a.l.Error("创建值班历史记录失败", zap.Error(err))
		return fmt.Errorf("创建值班历史记录失败: %w", err)
	}

	return nil
}

// GetMonitorOnDutyHistoryByGroupIDAndDay 获取指定日期的值班历史记录
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error) {
	if groupID <= 0 {
		return nil, fmt.Errorf("无效的值班组ID: %d", groupID)
	}

	if day == "" {
		return nil, fmt.Errorf("日期不能为空")
	}

	var history model.MonitorOnDutyHistory

	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		First(&history).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		a.l.Error("获取值班历史记录失败", zap.Error(err), zap.Int("groupID", groupID), zap.String("day", day))
		return nil, fmt.Errorf("获取值班历史记录失败: %w", err)
	}

	return &history, nil
}

// ExistsMonitorOnDutyHistory 检查指定日期的值班历史记录是否存在
func (a *alertManagerOnDutyDAO) ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error) {
	if groupID <= 0 {
		return false, fmt.Errorf("无效的值班组ID: %d", groupID)
	}

	if day == "" {
		return false, fmt.Errorf("日期不能为空")
	}

	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorOnDutyHistory{}).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		Count(&count).Error; err != nil {
		a.l.Error("检查值班历史记录存在性失败", zap.Error(err), zap.Int("groupID", groupID), zap.String("day", day))
		return false, fmt.Errorf("检查值班历史记录存在性失败: %w", err)
	}

	return count > 0, nil
}

// GetMonitorOnDutyList 获取值班组列表
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	if req.Page < 0 {
		req.Page = 1
	}

	if req.Size <= 0 {
		req.Size = 10
	}

	offset := (req.Page - 1) * req.Size

	var groups []*model.MonitorOnDutyGroup
	var count int64
	if err := a.db.WithContext(ctx).
		Preload("Members").
		Offset(offset).
		Limit(req.Size).
		Find(&groups).Error; err != nil {
		a.l.Error("获取值班组列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取值班组列表失败: %w", err)
	}

	if err := a.db.WithContext(ctx).Model(&model.MonitorOnDutyGroup{}).Count(&count).Error; err != nil {
		a.l.Error("获取值班组列表总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取值班组列表总数失败: %w", err)
	}

	return groups, count, nil
}

// GetMonitorOnDutyTotal 获取监控告警事件总数
func (a *alertManagerOnDutyDAO) GetMonitorOnDutyTotal(ctx context.Context) (int, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorOnDutyGroup{}).Count(&count).Error; err != nil {
		a.l.Error("获取监控告警事件总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
