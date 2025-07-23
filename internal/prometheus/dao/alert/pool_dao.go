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

type AlertManagerPoolDAO interface {
	GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
	GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error)
	CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error)
	CheckAlertIpExists(ctx context.Context, req *model.MonitorAlertManagerPool) error
}

type alertManagerPoolDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewAlertManagerPoolDAO(db *gorm.DB, l *zap.Logger) AlertManagerPoolDAO {
	return &alertManagerPoolDAO{
		db: db,
		l:  l,
	}
}

// CreateMonitorAlertManagerPool 创建 AlertManagerPool
func (a *alertManagerPoolDAO) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	monitorAlertManagerPool.CreatedAt = time.Now()
	monitorAlertManagerPool.UpdatedAt = time.Now()

	if err := a.db.WithContext(ctx).Create(monitorAlertManagerPool).Error; err != nil {
		a.l.Error("创建 MonitorAlertManagerPool 失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorAlertManagerPool 更新 AlertManagerPool
func (a *alertManagerPoolDAO) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	if monitorAlertManagerPool.ID == 0 {
		a.l.Error("UpdateMonitorAlertManagerPool 失败: ID 为 0", zap.Any("pool", monitorAlertManagerPool))
		return fmt.Errorf("monitorAlertManagerPool 的 ID 必须设置且非零")
	}

	monitorAlertManagerPool.UpdatedAt = time.Now()

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", monitorAlertManagerPool.ID).
		Updates(map[string]interface{}{
			"name":                    monitorAlertManagerPool.Name,
			"alert_manager_instances": monitorAlertManagerPool.AlertManagerInstances,
			"resolve_timeout":         monitorAlertManagerPool.ResolveTimeout,
			"group_wait":              monitorAlertManagerPool.GroupWait,
			"group_interval":          monitorAlertManagerPool.GroupInterval,
			"repeat_interval":         monitorAlertManagerPool.RepeatInterval,
			"group_by":                monitorAlertManagerPool.GroupBy,
			"receiver":                monitorAlertManagerPool.Receiver,
			"updated_at":              monitorAlertManagerPool.UpdatedAt,
		}).Error; err != nil {
		a.l.Error("更新 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", monitorAlertManagerPool.ID))
		return err
	}

	return nil
}

// DeleteMonitorAlertManagerPool 删除 AlertManagerPool
func (a *alertManagerPoolDAO) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Error("DeleteMonitorAlertManagerPool 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := a.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&model.MonitorAlertManagerPool{})

	if result.Error != nil {
		a.l.Error("删除 MonitorAlertManagerPool 失败", zap.Error(result.Error), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertManagerPool 失败: %w", id, result.Error)
	}

	if result.RowsAffected == 0 {
		a.l.Warn("尝试删除不存在的 MonitorAlertManagerPool", zap.Int("id", id))
		return fmt.Errorf("ID 为 %d 的 MonitorAlertManagerPool 不存在", id)
	}

	return nil
}

// GetAlertPoolByID 通过 ID 获取 AlertManagerPool
func (a *alertManagerPoolDAO) GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error) {
	if poolID <= 0 {
		a.l.Error("GetAlertPoolByID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var alertPool model.MonitorAlertManagerPool

	if err := a.db.WithContext(ctx).Where("id = ?", poolID).First(&alertPool).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未找到 ID 为 %d 的 AlertPool", poolID)
		}
		a.l.Error("获取 AlertPool 失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, err
	}

	return &alertPool, nil
}

// CheckMonitorAlertManagerPoolExists 检查 AlertManagerPool 是否存在
func (a *alertManagerPoolDAO) CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error) {
	if alertManagerPool.Name == "" {
		return false, fmt.Errorf("alertManagerPool 名称不能为空")
	}

	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("name = ?", alertManagerPool.Name).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorAlertManagerPool 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorAlertManagerPoolList 获取 AlertManagerPool 列表
func (a *alertManagerPoolDAO) GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) ([]*model.MonitorAlertManagerPool, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}

	if req.Size <= 0 {
		req.Size = 10
	}

	// 计算分页参数
	offset := (req.Page - 1) * req.Size
	limit := req.Size

	query := a.db.WithContext(ctx).Model(&model.MonitorAlertManagerPool{})

	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}

	var pools []*model.MonitorAlertManagerPool
	var count int64

	// 先获取总数
	if err := query.Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 再获取分页数据
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&pools).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 列表失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

func (a *alertManagerPoolDAO) CheckAlertIpExists(ctx context.Context, req *model.MonitorAlertManagerPool) error {
	var count int64

	for _, instance := range req.AlertManagerInstances {
		if err := a.db.WithContext(ctx).
			Model(&model.MonitorAlertManagerPool{}).
			Where("id != ? AND alert_manager_instances LIKE ?", req.ID, "%"+instance+"%").
			Count(&count).Error; err != nil {
			a.l.Error("检查 AlertManager Pool 是否存在失败", zap.Error(err))
			return err
		}
		if count > 0 {
			return fmt.Errorf("AlertManager实例 %s 已存在于其他Pool中", instance)
		}
	}

	return nil
}
