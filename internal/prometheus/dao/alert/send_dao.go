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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerSendDAO interface {
	GetMonitorSendGroupByPoolID(ctx context.Context, poolID int) ([]*model.MonitorSendGroup, int64, error)
	GetMonitorSendGroupByOnDutyGroupID(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, int64, error)
	GetMonitorSendGroupList(ctx context.Context, req *model.GetMonitorSendGroupListReq) ([]*model.MonitorSendGroup, int64, error)
	GetMonitorSendGroupByID(ctx context.Context, id int) (*model.MonitorSendGroup, error)
	CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error
	DeleteMonitorSendGroup(ctx context.Context, id int) error
	CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
	CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error)
	GetMonitorSendGroups(ctx context.Context) ([]*model.MonitorSendGroup, int64, error)
}

type alertManagerSendDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAlertManagerSendDAO(db *gorm.DB, l *zap.Logger) AlertManagerSendDAO {
	return &alertManagerSendDAO{
		db: db,
		l:  l,
	}
}

// GetMonitorSendGroupByPoolID 通过 poolID 获取 MonitorSendGroup
func (a *alertManagerSendDAO) GetMonitorSendGroupByPoolID(ctx context.Context, poolID int) ([]*model.MonitorSendGroup, int64, error) {
	if poolID <= 0 {
		a.l.Error("GetMonitorSendGroupByPoolID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, 0, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var sendGroups []*model.MonitorSendGroup
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("pool_id = ?", poolID).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 总数失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Where("pool_id = ?", poolID).
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, 0, err
	}

	return sendGroups, count, nil
}

// GetMonitorSendGroupByOnDutyGroupID 通过 onDutyGroupID 获取 MonitorSendGroup
func (a *alertManagerSendDAO) GetMonitorSendGroupByOnDutyGroupID(ctx context.Context, onDutyGroupID int) ([]*model.MonitorSendGroup, int64, error) {
	if onDutyGroupID <= 0 {
		a.l.Error("GetMonitorSendGroupByOnDutyGroupID 失败: 无效的 onDutyGroupID", zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, 0, fmt.Errorf("无效的 onDutyGroupID: %d", onDutyGroupID)
	}

	var sendGroups []*model.MonitorSendGroup
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("on_duty_group_id = ?", onDutyGroupID).
		Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 总数失败", zap.Error(err), zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Where("on_duty_group_id = ?", onDutyGroupID).
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("onDutyGroupID", onDutyGroupID))
		return nil, 0, err
	}

	return sendGroups, count, nil
}

// GetMonitorSendGroupList 获取所有 MonitorSendGroup
func (a *alertManagerSendDAO) GetMonitorSendGroupList(ctx context.Context, req *model.GetMonitorSendGroupListReq) ([]*model.MonitorSendGroup, int64, error) {
	var sendGroups []*model.MonitorSendGroup
	var count int64

	query := a.db.WithContext(ctx).Model(&model.MonitorSendGroup{})

	// 添加筛选条件
	if req.Search != "" {
		query = query.Where("name LIKE ? OR name_zh LIKE ?", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	if req.PoolID != nil {
		query = query.Where("pool_id = ?", *req.PoolID)
	}

	if req.OnDutyGroupID != nil {
		query = query.Where("on_duty_group_id = ?", *req.OnDutyGroupID)
	}

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.Size
	limit := req.Size

	// 获取数据列表，同时预加载关联的用户数据
	if err := query.Preload("StaticReceiveUsers").
		Preload("FirstUpgradeUsers").
		Preload("SecondUpgradeUsers").
		Offset(offset).
		Limit(limit).
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取所有 MonitorSendGroup 失败", zap.Error(err))
		return nil, 0, err
	}

	return sendGroups, count, nil
}

// GetMonitorSendGroupByID 通过 ID 获取 MonitorSendGroup
func (a *alertManagerSendDAO) GetMonitorSendGroupByID(ctx context.Context, id int) (*model.MonitorSendGroup, error) {
	if id <= 0 {
		a.l.Error("GetMonitorSendGroupByID 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var sendGroup model.MonitorSendGroup

	if err := a.db.WithContext(ctx).
		Where("id = ?", id).
		Preload("StaticReceiveUsers").
		Preload("FirstUpgradeUsers").
		Preload("SecondUpgradeUsers").
		First(&sendGroup).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的记录", id)
		}
		a.l.Error("获取 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &sendGroup, nil
}

// CreateMonitorSendGroup 创建 MonitorSendGroup
func (a *alertManagerSendDAO) CreateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	// 开启事务
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		a.l.Error("开启事务失败", zap.Error(tx.Error))
		return tx.Error
	}

	// 创建发送组基本信息
	if err := tx.Create(monitorSendGroup).Error; err != nil {
		tx.Rollback()
		a.l.Error("创建 MonitorSendGroup 失败", zap.Error(err))
		return err
	}

	// 处理静态接收用户关联
	if len(monitorSendGroup.StaticReceiveUsers) > 0 {
		if err := tx.Model(monitorSendGroup).Association("StaticReceiveUsers").Replace(monitorSendGroup.StaticReceiveUsers); err != nil {
			tx.Rollback()
			a.l.Error("关联静态接收用户失败", zap.Error(err))
			return err
		}
	}

	// 处理第一级升级用户关联
	if len(monitorSendGroup.FirstUpgradeUsers) > 0 {
		if err := tx.Model(monitorSendGroup).Association("FirstUpgradeUsers").Replace(monitorSendGroup.FirstUpgradeUsers); err != nil {
			tx.Rollback()
			a.l.Error("关联第一级升级用户失败", zap.Error(err))
			return err
		}
	}

	// 处理第二级升级用户关联
	if len(monitorSendGroup.SecondUpgradeUsers) > 0 {
		if err := tx.Model(monitorSendGroup).Association("SecondUpgradeUsers").Replace(monitorSendGroup.SecondUpgradeUsers); err != nil {
			tx.Rollback()
			a.l.Error("关联第二级升级用户失败", zap.Error(err))
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		a.l.Error("提交事务失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorSendGroup 删除 MonitorSendGroup
func (a *alertManagerSendDAO) DeleteMonitorSendGroup(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Error("DeleteMonitorSendGroup 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	// 开启事务
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		a.l.Error("开启事务失败", zap.Error(tx.Error))
		return tx.Error
	}

	// 查找要删除的发送组
	sendGroup := &model.MonitorSendGroup{}
	if err := tx.First(sendGroup, id).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("未找到 ID 为 %d 的记录", id)
		}
		a.l.Error("查找 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 清除关联的静态接收用户
	if err := tx.Model(sendGroup).Association("StaticReceiveUsers").Clear(); err != nil {
		tx.Rollback()
		a.l.Error("清除静态接收用户关联失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 清除关联的第一级升级用户
	if err := tx.Model(sendGroup).Association("FirstUpgradeUsers").Clear(); err != nil {
		tx.Rollback()
		a.l.Error("清除第一级升级用户关联失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 清除关联的第二级升级用户
	if err := tx.Model(sendGroup).Association("SecondUpgradeUsers").Clear(); err != nil {
		tx.Rollback()
		a.l.Error("清除第二级升级用户关联失败", zap.Error(err), zap.Int("id", id))
		return err
	}

	// 删除发送组
	if err := tx.Delete(&model.MonitorSendGroup{}, id).Error; err != nil {
		tx.Rollback()
		a.l.Error("删除 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorSendGroup 失败: %w", id, err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		a.l.Error("提交事务失败", zap.Error(err))
		return err
	}

	return nil
}

// CheckMonitorSendGroupExists 检查 MonitorSendGroup 是否存在
func (a *alertManagerSendDAO) CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	if sendGroup == nil || sendGroup.ID <= 0 {
		return false, fmt.Errorf("无效的 sendGroup 或 ID")
	}

	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("id = ?", sendGroup.ID).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorSendGroup 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// CheckMonitorSendGroupNameExists 检查 MonitorSendGroup 名称是否存在
func (a *alertManagerSendDAO) CheckMonitorSendGroupNameExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	if sendGroup == nil || sendGroup.Name == "" {
		return false, fmt.Errorf("无效的 sendGroup 或名称为空")
	}

	var count int64
	query := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("name = ?", sendGroup.Name)

	// 如果是更新操作，排除自身
	if sendGroup.ID > 0 {
		query = query.Where("id != ?", sendGroup.ID)
	}

	if err := query.Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorSendGroup 名称是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}
// UpdateMonitorSendGroup 更新 MonitorSendGroup
func (a *alertManagerSendDAO) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	if monitorSendGroup == nil || monitorSendGroup.ID <= 0 {
		return fmt.Errorf("无效的 monitorSendGroup 或 ID")
	}

	// 开启事务
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		a.l.Error("开启事务失败", zap.Error(tx.Error))
		return tx.Error
	}

	// 更新发送组基本信息
	if err := tx.Model(monitorSendGroup).Updates(monitorSendGroup).Error; err != nil {
		tx.Rollback()
		a.l.Error("更新 MonitorSendGroup 失败", zap.Error(err))
		return err
	}

	// 处理静态接收用户关联
	if err := tx.Model(monitorSendGroup).Association("StaticReceiveUsers").Replace(monitorSendGroup.StaticReceiveUsers); err != nil {
		tx.Rollback()
		a.l.Error("更新静态接收用户关联失败", zap.Error(err))
		return err
	}

	// 处理第一级升级用户关联
	if err := tx.Model(monitorSendGroup).Association("FirstUpgradeUsers").Replace(monitorSendGroup.FirstUpgradeUsers); err != nil {
		tx.Rollback()
		a.l.Error("更新第一级升级用户关联失败", zap.Error(err))
		return err
	}

	// 处理第二级升级用户关联
	if err := tx.Model(monitorSendGroup).Association("SecondUpgradeUsers").Replace(monitorSendGroup.SecondUpgradeUsers); err != nil {
		tx.Rollback()
		a.l.Error("更新第二级升级用户关联失败", zap.Error(err))
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		a.l.Error("提交事务失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorSendGroups 获取所有发送组
func (a *alertManagerSendDAO) GetMonitorSendGroups(ctx context.Context) ([]*model.MonitorSendGroup, int64, error) {
	var sendGroups []*model.MonitorSendGroup
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Count(&count).Error; err != nil {
		a.l.Error("获取发送组总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Preload("StaticReceiveUsers").
		Preload("FirstUpgradeUsers").
		Preload("SecondUpgradeUsers").
		Find(&sendGroups).Error; err != nil {
		a.l.Error("获取所有发送组失败", zap.Error(err))
		return nil, 0, err
	}

	return sendGroups, count, nil
}
