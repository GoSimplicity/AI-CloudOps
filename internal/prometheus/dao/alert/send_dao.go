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
	"time"

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
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	if req.Enable != nil {
		query = query.Where("enable = ?", *req.Enable)
	}

	if req.PoolID != nil {
		query = query.Where("pool_id = ?", *req.PoolID)
	}

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorSendGroup 总数失败", zap.Error(err))
		return nil, 0, err
	}

	offset := (req.Page - 1) * req.Size
	limit := req.Size

	// 获取数据列表
	if err := query.Offset(offset).Limit(limit).Find(&sendGroups).Error; err != nil {
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
	if err := a.db.WithContext(ctx).Create(monitorSendGroup).Error; err != nil {
		a.l.Error("创建 MonitorSendGroup 失败", zap.Error(err))
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

	result := a.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.MonitorSendGroup{})

	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorSendGroup 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorSendGroup 失败: %w", id, err)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到 ID 为 %d 的记录", id)
	}

	return nil
}

// CheckMonitorSendGroupExists 检查 MonitorSendGroup 是否存在
func (a *alertManagerSendDAO) CheckMonitorSendGroupExists(ctx context.Context, sendGroup *model.MonitorSendGroup) (bool, error) {
	if sendGroup.ID <= 0 {
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
	if sendGroup.Name == "" {
		return false, fmt.Errorf("名称为空")
	}

	var count int64
	if err := a.db.WithContext(ctx).
		Model(&model.MonitorSendGroup{}).
		Where("name = ?", sendGroup.Name).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorSendGroup 名称是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// UpdateMonitorSendGroup 更新 MonitorSendGroup
func (a *alertManagerSendDAO) UpdateMonitorSendGroup(ctx context.Context, monitorSendGroup *model.MonitorSendGroup) error {
	return a.db.WithContext(ctx).Model(monitorSendGroup).Updates(map[string]interface{}{
		"name":                    monitorSendGroup.Name,
		"name_zh":                 monitorSendGroup.NameZh,
		"enable":                  monitorSendGroup.Enable,
		"pool_id":                 monitorSendGroup.PoolID,
		"on_duty_group_id":        monitorSendGroup.OnDutyGroupID,
		"fei_shu_qun_robot_token": monitorSendGroup.FeiShuQunRobotToken,
		"repeat_interval":         monitorSendGroup.RepeatInterval,
		"send_resolved":           monitorSendGroup.SendResolved,
		"notify_methods":          monitorSendGroup.NotifyMethods,
		"need_upgrade":            monitorSendGroup.NeedUpgrade,
		"upgrade_minutes":         monitorSendGroup.UpgradeMinutes,
		"updated_at":              time.Now(),
	}).Error
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

	if err := a.db.WithContext(ctx).Find(&sendGroups).Error; err != nil {
		a.l.Error("获取所有发送组失败", zap.Error(err))
		return nil, 0, err
	}

	return sendGroups, count, nil
}
