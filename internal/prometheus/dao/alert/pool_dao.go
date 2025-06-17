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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerPoolDAO interface {
	GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, int64, error)
	GetMonitorAlertManagerPoolList(ctx context.Context, offset, limit int) ([]*model.MonitorAlertManagerPool, int64, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
	GetMonitorAlertManagerPoolTotal(ctx context.Context) (int, error)
	SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, int64, error)
	GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error)
	CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error)
}

type alertManagerPoolDAO struct {
	db      *gorm.DB
	l       *zap.Logger
	userDao userDao.UserDAO
}

func NewAlertManagerPoolDAO(db *gorm.DB, l *zap.Logger, userDao userDao.UserDAO) AlertManagerPoolDAO {
	return &alertManagerPoolDAO{
		db:      db,
		l:       l,
		userDao: userDao,
	}
}

// GetAllAlertManagerPools 获取所有 AlertManagerPool
func (a *alertManagerPoolDAO) GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, int64, error) {
	var pools []*model.MonitorAlertManagerPool
	var count int64

	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).Find(&pools).Count(&count).Error; err != nil {
		a.l.Error("获取所有 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// CreateMonitorAlertManagerPool 创建 AlertManagerPool
func (a *alertManagerPoolDAO) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	monitorAlertManagerPool.CreatedAt = getTime()
	monitorAlertManagerPool.UpdatedAt = getTime()

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

	monitorAlertManagerPool.UpdatedAt = getTime()

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ? AND deleted_at = ?", monitorAlertManagerPool.ID, 0).
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

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ? AND deleted_at = ?", id, 0).
		Update("deleted_at", getTime()).
		Error; err != nil {
		a.l.Error("删除 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertManagerPool 失败: %w", id, err)
	}

	return nil
}

// SearchMonitorAlertManagerPoolByName 通过名称搜索 AlertManagerPool
func (a *alertManagerPoolDAO) SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, int64, error) {
	if name == "" {
		return nil, 0, fmt.Errorf("搜索名称不能为空")
	}

	var pools []*model.MonitorAlertManagerPool
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("LOWER(name) LIKE ? AND deleted_at = ?", "%"+strings.ToLower(name)+"%", 0).
		Count(&count).Error; err != nil {
		a.l.Error("获取搜索结果总数失败", zap.Error(err))
		return nil, 0, err
	}

	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ? AND deleted_at = ?", "%"+strings.ToLower(name)+"%", 0).
		Find(&pools).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// GetAlertPoolByID 通过 ID 获取 AlertManagerPool
func (a *alertManagerPoolDAO) GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error) {
	if poolID <= 0 {
		a.l.Error("GetAlertPoolByID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var alertPool model.MonitorAlertManagerPool

	if err := a.db.WithContext(ctx).Where("id = ? AND deleted_at = ?", poolID, 0).First(&alertPool).Error; err != nil {
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
		Where("name = ? AND deleted_at = ?", alertManagerPool.Name, 0).
		Count(&count).Error; err != nil {
		a.l.Error("检查 MonitorAlertManagerPool 是否存在失败", zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetMonitorAlertManagerPoolList 获取 AlertManagerPool 列表
func (a *alertManagerPoolDAO) GetMonitorAlertManagerPoolList(ctx context.Context, offset int, limit int) ([]*model.MonitorAlertManagerPool, int64, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, fmt.Errorf("无效的分页参数: offset=%d, limit=%d", offset, limit)
	}

	var pools []*model.MonitorAlertManagerPool
	var count int64

	// 先获取总数
	if err := a.db.WithContext(ctx).Model(&model.MonitorAlertManagerPool{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 再获取分页数据
	if err := a.db.WithContext(ctx).Where("deleted_at = ?", 0).Order("created_at DESC").Offset(offset).Limit(limit).Find(&pools).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 列表失败", zap.Error(err))
		return nil, 0, err
	}

	return pools, count, nil
}

// GetMonitorAlertManagerPoolTotal 获取 AlertManager 集群池总数
func (a *alertManagerPoolDAO) GetMonitorAlertManagerPoolTotal(ctx context.Context) (int, error) {
	var count int64

	if err := a.db.WithContext(ctx).Model(&model.MonitorAlertManagerPool{}).Where("deleted_at = ?", 0).Count(&count).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 总数失败", zap.Error(err))
		return 0, err
	}

	return int(count), nil
}
