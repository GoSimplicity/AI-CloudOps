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
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AlertManagerOnDutyDAO 值班组数据访问接口
type AlertManagerOnDutyDAO interface {
	// 值班组管理
	GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error)
	CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error
	GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error)
	UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error
	DeleteMonitorOnDutyGroup(ctx context.Context, id int) error
	CheckMonitorOnDutyGroupExists(ctx context.Context, group *model.MonitorOnDutyGroup) (bool, error)

	// 换班记录管理
	CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error
	GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error)

	// 值班历史管理
	CreateMonitorOnDutyHistory(ctx context.Context, history *model.MonitorOnDutyHistory) error
	GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error)
	GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error)
	GetMonitorOnDutyHistoryList(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) ([]*model.MonitorOnDutyHistory, int64, error)
	ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error)
}

type onDutyDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewAlertManagerOnDutyDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerOnDutyDAO {
	return &onDutyDAO{
		db:     db,
		logger: l,
	}
}

// 值班组管理方法

func (d *onDutyDAO) GetMonitorOnDutyList(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) ([]*model.MonitorOnDutyGroup, int64, error) {
	query := d.buildGroupQuery(ctx, req)

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.logger.Error("获取值班组总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	var groups []*model.MonitorOnDutyGroup
	offset := d.calculateOffset(req.Page, req.Size)
	if err := query.Order("id DESC").
		Offset(offset).
		Limit(req.Size).
		Preload("Users").
		Find(&groups).Error; err != nil {
		d.logger.Error("获取值班组列表失败", zap.Error(err))
		return nil, 0, err
	}

	return groups, total, nil
}

func (d *onDutyDAO) CreateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			d.logger.Error("创建值班组失败", zap.Error(err))
			return err
		}

		if len(group.Users) > 0 {
			if err := tx.Model(group).Association("Users").Replace(group.Users); err != nil {
				d.logger.Error("创建值班组成员关联失败", zap.Error(err))
				return err
			}
		}

		return nil
	})
}

func (d *onDutyDAO) GetMonitorOnDutyGroupByID(ctx context.Context, id int) (*model.MonitorOnDutyGroup, error) {
	var group model.MonitorOnDutyGroup
	err := d.db.WithContext(ctx).
		Preload("Users").
		First(&group, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		d.logger.Error("获取值班组失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &group, nil
}

func (d *onDutyDAO) UpdateMonitorOnDutyGroup(ctx context.Context, group *model.MonitorOnDutyGroup) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新基本信息
		if err := tx.Model(group).Updates(map[string]interface{}{
			"name":        group.Name,
			"shift_days":  group.ShiftDays,
			"enable":      group.Enable,
			"description": group.Description,
		}).Error; err != nil {
			d.logger.Error("更新值班组失败", zap.Error(err))
			return err
		}

		// 更新多对多关系
		if err := tx.Model(group).Association("Users").Replace(group.Users); err != nil {
			d.logger.Error("更新值班组成员关联失败", zap.Error(err))
			return err
		}

		return nil
	})
}

func (d *onDutyDAO) DeleteMonitorOnDutyGroup(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取值班组
		var group model.MonitorOnDutyGroup
		if err := tx.First(&group, id).Error; err != nil {
			d.logger.Error("获取要删除的值班组失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 清除多对多关联
		if err := tx.Model(&group).Association("Users").Clear(); err != nil {
			d.logger.Error("清除值班组成员关联失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 删除相关的值班历史记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyHistory{}).Error; err != nil {
			d.logger.Error("删除值班历史记录失败", zap.Int("group_id", id), zap.Error(err))
			return err
		}

		// 删除相关的换班记录
		if err := tx.Where("on_duty_group_id = ?", id).Delete(&model.MonitorOnDutyChange{}).Error; err != nil {
			d.logger.Error("删除换班记录失败", zap.Int("group_id", id), zap.Error(err))
			return err
		}

		// 删除值班组
		return tx.Delete(&model.MonitorOnDutyGroup{}, id).Error
	})
}

func (d *onDutyDAO) CheckMonitorOnDutyGroupExists(ctx context.Context, group *model.MonitorOnDutyGroup) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MonitorOnDutyGroup{}).
		Where("name = ?", group.Name).
		Count(&count).Error

	if err != nil {
		d.logger.Error("检查值班组是否存在失败", zap.String("name", group.Name), zap.Error(err))
	}

	return count > 0, err
}

// 换班记录管理方法

func (d *onDutyDAO) CreateMonitorOnDutyGroupChange(ctx context.Context, change *model.MonitorOnDutyChange) error {
	if err := d.db.WithContext(ctx).Create(change).Error; err != nil {
		d.logger.Error("创建换班记录失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *onDutyDAO) GetMonitorOnDutyChangesByGroupAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyChange, error) {
	var changes []*model.MonitorOnDutyChange
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date ASC").
		Find(&changes).Error

	if err != nil {
		d.logger.Error("获取换班记录失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return changes, err
}

// 值班历史管理方法

func (d *onDutyDAO) CreateMonitorOnDutyHistory(ctx context.Context, history *model.MonitorOnDutyHistory) error {
	// 先检查是否已存在相同日期和组的记录
	exists, err := d.ExistsMonitorOnDutyHistory(ctx, history.OnDutyGroupID, history.DateString)
	if err != nil {
		return err
	}

	// 如果已存在，则更新而不是创建
	if exists {
		return d.db.WithContext(ctx).
			Model(&model.MonitorOnDutyHistory{}).
			Where("on_duty_group_id = ? AND date_string = ?", history.OnDutyGroupID, history.DateString).
			Updates(map[string]interface{}{
				"on_duty_user_id": history.OnDutyUserID,
				"origin_user_id":  history.OriginUserID,
			}).Error
	}

	// 不存在则创建新记录
	if err := d.db.WithContext(ctx).Create(history).Error; err != nil {
		d.logger.Error("创建值班历史记录失败", zap.Error(err))
		return err
	}

	return nil
}

func (d *onDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndDay(ctx context.Context, groupID int, day string) (*model.MonitorOnDutyHistory, error) {
	var history model.MonitorOnDutyHistory
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		First(&history).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		d.logger.Error("获取值班历史记录失败", zap.Int("group_id", groupID), zap.String("day", day), zap.Error(err))
		return nil, err
	}

	return &history, nil
}

func (d *onDutyDAO) GetMonitorOnDutyHistoryByGroupIDAndTimeRange(ctx context.Context, groupID int, startTime, endTime string) ([]*model.MonitorOnDutyHistory, error) {
	var histories []*model.MonitorOnDutyHistory
	err := d.db.WithContext(ctx).
		Where("on_duty_group_id = ? AND date_string BETWEEN ? AND ?", groupID, startTime, endTime).
		Order("date_string ASC").
		Find(&histories).Error

	if err != nil {
		d.logger.Error("获取值班历史记录失败", zap.Int("group_id", groupID), zap.Error(err))
		return nil, err
	}

	return histories, nil
}

func (d *onDutyDAO) ExistsMonitorOnDutyHistory(ctx context.Context, groupID int, day string) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).
		Model(&model.MonitorOnDutyHistory{}).
		Where("on_duty_group_id = ? AND date_string = ?", groupID, day).
		Count(&count).Error

	if err != nil {
		d.logger.Error("检查值班历史记录是否存在失败", zap.Int("group_id", groupID), zap.String("day", day), zap.Error(err))
	}

	return count > 0, err
}

func (d *onDutyDAO) GetMonitorOnDutyHistoryList(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) ([]*model.MonitorOnDutyHistory, int64, error) {
	query := d.buildHistoryQuery(ctx, req)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		d.logger.Error("获取值班历史总数失败", zap.Error(err))
		return nil, 0, err
	}

	var histories []*model.MonitorOnDutyHistory
	offset := d.calculateOffset(req.Page, req.Size)
	if err := query.Offset(offset).Limit(req.Size).Order("date_string DESC").Find(&histories).Error; err != nil {
		d.logger.Error("获取值班历史列表失败", zap.Error(err))
		return nil, 0, err
	}

	return histories, total, nil
}

// 私有辅助方法

func (d *onDutyDAO) buildGroupQuery(ctx context.Context, req *model.GetMonitorOnDutyGroupListReq) *gorm.DB {
	query := d.db.WithContext(ctx).Model(&model.MonitorOnDutyGroup{})

	// 应用过滤条件
	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	return query
}

func (d *onDutyDAO) buildHistoryQuery(ctx context.Context, req *model.GetMonitorOnDutyHistoryReq) *gorm.DB {
	query := d.db.WithContext(ctx).Model(&model.MonitorOnDutyHistory{}).
		Where("on_duty_group_id = ?", req.OnDutyGroupID)

	// 如果有指定起始和终止时间，则添加时间范围条件
	if req.StartDate != "" && req.EndDate != "" {
		query = query.Where("date_string BETWEEN ? AND ?", req.StartDate, req.EndDate)
	}

	// 处理搜索条件
	if req.Search != "" {
		query = query.Where("date_string LIKE ?", "%"+req.Search+"%")
	}

	return query
}

func (d *onDutyDAO) calculateOffset(page, size int) int {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	return (page - 1) * size
}
