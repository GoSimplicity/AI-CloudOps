package onduty

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

type AlertManagerOnDutyDAO interface {
	GetAllMonitorOnDutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	SearchMonitorOnDutyGroupByName(ctx context.Context, name string) ([]*model.MonitorOnDutyGroup, error)
	CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error)
	CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error)
	GetMonitorOnDutyHistoryByGroupIdAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error)
	CreateMonitorOnDutyHistory(ctx context.Context, monitorOnDutyHistory *model.MonitorOnDutyHistory) error
	GetMonitorOnDutyHistoryByGroupIdAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error)
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

func (a *alertManagerOnDutyDAO) GetAllMonitorOnDutyGroup(ctx context.Context) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if err := a.db.WithContext(ctx).Preload("Members").Find(&groups).Error; err != nil {
		a.l.Error("获取所有 MonitorOnDutyGroup 失败", zap.Error(err))
		return nil, err
	}

	return groups, nil
}

func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if monitorOnDutyGroup == nil {
		a.l.Error("CreateMonitorOnDutyGroup 失败: group 为 nil")
		return fmt.Errorf("monitorOnDutyGroup 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(monitorOnDutyGroup).Error; err != nil {
		a.l.Error("创建 MonitorOnDutyGroup 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerOnDutyDAO) GetMonitorOnDutyGroupById(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	if id <= 0 {
		a.l.Error("GetMonitorOnDutyGroupById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var group model.MonitorOnDutyGroup
	if err := a.db.WithContext(ctx).Preload("Members").First(&group, id).Error; err != nil {
		a.l.Error("获取 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &group, nil
}

func (a *alertManagerOnDutyDAO) UpdateMonitorOnDutyGroup(ctx context.Context, monitorOnDutyGroup *model.MonitorOnDutyGroup) error {
	if monitorOnDutyGroup == nil {
		a.l.Error("UpdateMonitorOnDutyGroup 失败: group 为 nil")
		return fmt.Errorf("monitorOnDutyGroup 不能为空")
	}

	if monitorOnDutyGroup.ID == 0 {
		a.l.Error("UpdateMonitorOnDutyGroup 失败: ID 为 0", zap.Any("group", monitorOnDutyGroup))
		return fmt.Errorf("monitorOnDutyGroup 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", monitorOnDutyGroup.ID).
		Updates(monitorOnDutyGroup).Error; err != nil {
		a.l.Error("更新 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("id", monitorOnDutyGroup.ID))
		return err
	}

	return nil
}

func (a *alertManagerOnDutyDAO) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Error("DeleteMonitorOnDutyGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorOnDutyGroup{}, id)
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorOnDutyGroup 失败", zap.Error(err), zap.Int("ID", id))
		return err
	}

	return nil
}

func (a *alertManagerOnDutyDAO) SearchMonitorOnDutyGroupByName(ctx context.Context, name string) ([]*model.MonitorOnDutyGroup, error) {
	var groups []*model.MonitorOnDutyGroup

	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&groups).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorOnDutyGroup 失败", zap.Error(err))
		return nil, err
	}

	return groups, nil
}

func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyGroupChange(ctx context.Context, monitorOnDutyGroupChange *model.MonitorOnDutyChange) error {
	if monitorOnDutyGroupChange == nil {
		a.l.Error("CreateMonitorOnDutyGroupChange 失败: change 为 nil")
		return fmt.Errorf("monitorOnDutyGroupChange 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(monitorOnDutyGroupChange).Error; err != nil {
		a.l.Error("创建 MonitorOnDutyGroupChange 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerOnDutyDAO) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error) {
	if groupID <= 0 {
		a.l.Error("GetMonitorOnDutyChangesByGroupAndTimeRange 失败: 无效的 groupID", zap.Int("groupID", groupID))
		return nil, fmt.Errorf("无效的 groupID: %d", groupID)
	}

	var changes []*model.MonitorOnDutyChange
	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ?", groupID).
		Where("date BETWEEN ? AND ?", startTime, endTime).
		Find(&changes).Error; err != nil {
		a.l.Error("获取值班计划变更失败", zap.Error(err), zap.Int("groupID", groupID))
		return nil, err
	}

	return changes, nil
}

func (a *alertManagerOnDutyDAO) CheckMonitorOnDutyGroupExists(ctx context.Context, onDutyGroup *model.MonitorOnDutyGroup) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("id = ?", onDutyGroup.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIdAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error) {
	var historyList []*model.MonitorOnDutyHistory

	if err := a.db.WithContext(ctx).Where("on_duty_group_id = ? AND date_string >= ? AND date_string <= ?", groupID, startTime, endTime).Find(&historyList).Error; err != nil {
		a.l.Error("获取值班历史记录失败", zap.Error(err))
		return nil, err
	}

	return historyList, nil
}

func (a *alertManagerOnDutyDAO) CreateMonitorOnDutyHistory(ctx context.Context, monitorOnDutyHistory *model.MonitorOnDutyHistory) error {
	if err := a.db.WithContext(ctx).Create(monitorOnDutyHistory).Error; err != nil {
		a.l.Error("创建值班历史记录失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIdAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error) {
	var history *model.MonitorOnDutyHistory

	if err := a.db.WithContext(ctx).Where("on_duty_group_id = ? AND date_string = ?", groupID, day).First(&history).Error; err != nil {
		a.l.Error("获取值班历史记录失败", zap.Error(err))
		return nil, err
	}

	return history, nil
}

func (a *alertManagerOnDutyDAO) ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorOnDutyHistory{}).Where("on_duty_group_id = ? AND date_string = ?", groupID, day).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}
