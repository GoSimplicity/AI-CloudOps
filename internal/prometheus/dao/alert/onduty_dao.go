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
	GetMonitorOnDutyGroupFuturePlan(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyPlan, error)
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
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	// 构建查询
	query := d.db.WithContext(ctx).Model(&model.MonitorOnDutyGroup{})

	// 应用过滤条件
	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.l.Error("获取值班组总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 计算分页偏移量
	offset := (req.Page - 1) * req.Size

	var groups []*model.MonitorOnDutyGroup
	if err := query.Order("id DESC").
		Offset(offset).
		Limit(req.Size).
		Preload("Users").
		Find(&groups).Error; err != nil {
		d.l.Error("获取值班组列表失败", zap.Error(err))
		return nil, 0, err
	}

	return groups, total, nil
}

func (d *alertManagerOnDutyDAO) CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建值班组
		if err := tx.Create(group).Error; err != nil {
			d.l.Error("创建值班组失败", zap.Error(err))
			return err
		}

		// 处理值班组成员关联
		if len(group.Users) > 0 {
			if err := tx.Model(group).Association("Users").Replace(group.Users); err != nil {
				d.l.Error("创建值班组成员关联失败", zap.Error(err))
				return err
			}
		}

		return nil
	})
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	var group model.MonitorOnDutyGroup
	err := d.db.WithContext(ctx).
		Preload("Users").
		Where("id = ?", id).
		First(&group).Error

	if err != nil {
		d.l.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &group, nil
}

func (d *alertManagerOnDutyDAO) UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新基本信息
		if err := tx.Model(group).Updates(map[string]interface{}{
			"name":        group.Name,
			"shift_days":  group.ShiftDays,
			"enable":      group.Enable,
			"description": group.Description,
		}).Error; err != nil {
			d.l.Error("更新值班组失败", zap.Error(err))
			return err
		}

		// 更新多对多关系
		if err := tx.Model(group).Association("Users").Replace(group.Users); err != nil {
			d.l.Error("更新值班组成员关联失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (d *alertManagerOnDutyDAO) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取值班组
		var group model.MonitorOnDutyGroup
		if err := tx.First(&group, id).Error; err != nil {
			d.l.Error("获取要删除的值班组失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 清除多对多关联
		if err := tx.Model(&group).Association("Users").Clear(); err != nil {
			d.l.Error("清除值班组成员关联失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 删除相关的值班计划
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyPlan{}).Error; err != nil {
			d.l.Error("删除值班计划失败", zap.Int("group_id", id), zap.Error(err))
			return err
		}

		// 删除相关的值班历史记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyHistory{}).Error; err != nil {
			d.l.Error("删除值班历史记录失败", zap.Int("group_id", id), zap.Error(err))
			return err
		}

		// 删除相关的换班记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyChange{}).Error; err != nil {
			d.l.Error("删除换班记录失败", zap.Int("group_id", id), zap.Error(err))
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

	if err != nil {
		d.l.Error("获取换班记录失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return changes, err
}

func (d *alertManagerOnDutyDAO) CheckMonitorOnDutyGroupExists(ctx context.Context, group *model.MonitorOnDutyGroup) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("name = ?", group.Name).
		Count(&count).Error

	if err != nil {
		d.l.Error("检查值班组是否存在失败", zap.String("name", group.Name), zap.Error(err))
	}

	return count > 0, err
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error) {
	var histories []*model.MonitorOnDutyHistory
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date_string ASC").
		Find(&histories).Error

	if err != nil {
		d.l.Error("获取值班历史记录失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return histories, nil
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
		d.l.Error("获取值班历史记录失败", zap.Int("group_id", groupID), zap.String("day", day), zap.Error(err))
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

	if err != nil {
		d.l.Error("检查值班历史记录是否存在失败", zap.Int("group_id", groupID), zap.String("day", day), zap.Error(err))
	}

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

	if err != nil {
		d.l.Error("获取值班计划失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return plans, nil
}

func (d *alertManagerOnDutyDAO) GetMonitorOnDutyGroupFuturePlan(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyPlan, error) {
	var plans []*model.MonitorOnDutyPlan
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date BETWEEN ? AND ? AND status = ?",
			groupID, startTime, endTime, 3). // status=3 表示未开始的计划
		Order("date ASC").
		Find(&plans).Error

	if err != nil {
		d.l.Error("获取未来值班计划失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return plans, nil
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
