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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerOnDutyDAO interface {
	GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error)
	CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error)
	CheckMonitorOnDutyGroupExists(ctx context.Context, group *model.MonitorOnDutyGroup) (bool, error)
	GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error)
	CreateMonitorOnDutyHistory(ctx context.Context, history *model.MonitorOnDutyHistory) error
	GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error)
	ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error)
	GetMonitorOnDutyHistoryList(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) ([]*model.MonitorOnDutyHistory, int64, error)
	CreateMonitorOnDutyPlan(ctx context.Context, plan *model.MonitorOnDutyPlan) error
	GetMonitorOnDutyPlansByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyPlan, error)
	UpdateMonitorOnDutyPlan(ctx context.Context, plan *model.MonitorOnDutyPlan) error
	DeleteMonitorOnDutyPlan(ctx context.Context, id int) error
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

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	query := d.db.WithContext(ctx).Model(&model.MonitorOnDutyGroup{})

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.l.Error("获取值班组总数失败", zap.Error(err))
		return nil, 0, err
	}

	var groups []*model.MonitorOnDutyGroup
	offset := (req.Page - 1) * req.Size

	if err := query.Preload("Members").
		Order("id DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&groups).Error; err != nil {
		d.l.Error("获取值班组列表失败", zap.Error(err))
		return nil, 0, err
	}

	return groups, total, nil
}

func (d *alertManagerOnDutyDAO) CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	if err := d.db.WithContext(ctx).Create(group).Error; err != nil {
		d.l.Error("创建值班组失败", zap.Error(err))
		return err
	}

	if err := d.db.WithContext(ctx).Model(group).Association("Members").Replace(group.Members); err != nil {
		d.l.Error("创建值班组成员失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	var group model.MonitorOnDutyGroup
	err := d.db.WithContext(ctx).
		Preload("Members").
		Preload("DutyPlans").
		First(&group, id).Error

	if err != nil {
		return nil, err
	}

	return &group, nil
}

func (d *alertManagerOnDutyDAO) UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新基本信息
		if err := tx.Model(group).Updates(group).Error; err != nil {
			d.l.Error("更新值班组失败", zap.Error(err))
			return err
		}

		// 更新成员关联
		if err := tx.Model(group).Association("Members").Replace(group.Members); err != nil {
			d.l.Error("更新值班组成员失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (d *alertManagerOnDutyDAO) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清除成员关联
		group := &model.MonitorOnDutyGroup{Model: model.Model{ID: id}}
		if err := tx.Model(group).Association("Members").Clear(); err != nil {
			return err
		}

		// 删除相关的值班计划
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyPlan{}).Error; err != nil {
			return err
		}

		// 删除相关的值班历史记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyHistory{}).Error; err != nil {
			return err
		}

		// 删除相关的换班记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyChange{}).Error; err != nil {
			return err
		}

		// 删除值班组
		return tx.Delete(&model.MonitorOnDutyGroup{}, id).Error
	})
}

func (d *alertManagerOnDutyDAO) CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error {
	if err := d.db.WithContext(ctx).Create(change).Error; err != nil {
		d.l.Error("创建换班记录失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error) {
	var changes []*model.MonitorOnDutyChange
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date ASC").
		Find(&changes).Error

	return changes, err
}

func (d *alertManagerOnDutyDAO) CheckMonitorOnDutyGroupExists(ctx context.Context, group *model.MonitorOnDutyGroup) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("name = ?", group.Name).
		Count(&count).Error

	return count > 0, err
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error) {
	var histories []*model.MonitorOnDutyHistory
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date_string ASC").
		Find(&histories).Error

	return histories, err
}

func (d *alertManagerOnDutyDAO) CreateMonitorOnDutyHistory(ctx context.Context, history *model.MonitorOnDutyHistory) error {
	if err := d.db.WithContext(ctx).Create(history).Error; err != nil {
		d.l.Error("创建值班历史记录失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error) {
	var history model.MonitorOnDutyHistory
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		First(&history).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	return &history, nil
}

func (d *alertManagerOnDutyDAO) ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MonitorOnDutyHistory{}).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		Count(&count).Error

	return count > 0, err
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryList(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) ([]*model.MonitorOnDutyHistory, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.MonitorOnDutyHistory{}).
		Where("on_duty_group_id = ? AND date_string BETWEEN ? AND ?",
			req.OnDutyGroupID, req.StartDate, req.EndDate)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.l.Error("获取值班历史总数失败", zap.Error(err))
		return nil, 0, err
	}

	var histories []*model.MonitorOnDutyHistory
	if err := query.Order("date_string DESC").Find(&histories).Error; err != nil {
		d.l.Error("获取值班历史列表失败", zap.Error(err))
		return nil, 0, err
	}

	return histories, total, nil
}

func (d *alertManagerOnDutyDAO) CreateMonitorOnDutyPlan(ctx context.Context, plan *model.MonitorOnDutyPlan) error {
	if err := d.db.WithContext(ctx).Create(plan).Error; err != nil {
		d.l.Error("创建值班计划失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyPlansByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyPlan, error) {
	var plans []*model.MonitorOnDutyPlan
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date ASC").
		Find(&plans).Error

	return plans, err
}

func (d *alertManagerOnDutyDAO) UpdateMonitorOnDutyPlan(ctx context.Context, plan *model.MonitorOnDutyPlan) error {
	if err := d.db.WithContext(ctx).Model(plan).Updates(plan).Error; err != nil {
		d.l.Error("更新值班计划失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *alertManagerOnDutyDAO) DeleteMonitorOnDutyPlan(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.MonitorOnDutyPlan{}, id).Error; err != nil {
		d.l.Error("删除值班计划失败", zap.Error(err))
		return err
	}

	return nil
}
