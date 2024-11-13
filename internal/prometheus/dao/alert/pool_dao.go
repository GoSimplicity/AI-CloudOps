package alert

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

type AlertManagerPoolDAO interface {
	GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
	SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, error)
	GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error)
	CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error)
	GetMonitorAlertManagerPoolById(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error)
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

func (a *alertManagerPoolDAO) GetAllAlertManagerPools(ctx context.Context) ([]*model.MonitorAlertManagerPool, error) {
	var pools []*model.MonitorAlertManagerPool

	if err := a.db.WithContext(ctx).Find(&pools).Error; err != nil {
		a.l.Error("获取所有 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (a *alertManagerPoolDAO) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	if monitorAlertManagerPool == nil {
		a.l.Error("CreateMonitorAlertManagerPool 失败: pool 为 nil")
		return fmt.Errorf("monitorAlertManagerPool 不能为空")
	}

	if err := a.db.WithContext(ctx).Create(monitorAlertManagerPool).Error; err != nil {
		a.l.Error("创建 MonitorAlertManagerPool 失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolDAO) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	if monitorAlertManagerPool == nil {
		a.l.Error("UpdateMonitorAlertManagerPool 失败: pool 为 nil")
		return fmt.Errorf("monitorAlertManagerPool 不能为空")
	}

	if monitorAlertManagerPool.ID == 0 {
		a.l.Error("UpdateMonitorAlertManagerPool 失败: ID 为 0", zap.Any("pool", monitorAlertManagerPool))
		return fmt.Errorf("monitorAlertManagerPool 的 ID 必须设置且非零")
	}

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", monitorAlertManagerPool.ID).
		Updates(monitorAlertManagerPool).Error; err != nil {
		a.l.Error("更新 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", monitorAlertManagerPool.ID))
		return err
	}

	return nil
}

func (a *alertManagerPoolDAO) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	if id <= 0 {
		a.l.Error("DeleteMonitorAlertManagerPool 失败: 无效的 ID", zap.Int("id", id))
		return fmt.Errorf("无效的 ID: %d", id)
	}

	result := a.db.WithContext(ctx).Delete(&model.MonitorAlertManagerPool{}, id)
	if err := result.Error; err != nil {
		a.l.Error("删除 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除 ID 为 %d 的 MonitorAlertManagerPool 失败: %w", id, err)
	}

	return nil
}

func (a *alertManagerPoolDAO) SearchMonitorAlertManagerPoolByName(ctx context.Context, name string) ([]*model.MonitorAlertManagerPool, error) {
	var pools []*model.MonitorAlertManagerPool

	if err := a.db.WithContext(ctx).
		Where("LOWER(name) LIKE ?", "%"+strings.ToLower(name)+"%").
		Find(&pools).Error; err != nil {
		a.l.Error("通过名称搜索 MonitorAlertManagerPool 失败", zap.Error(err))
		return nil, err
	}

	return pools, nil
}

func (a *alertManagerPoolDAO) GetAlertPoolByID(ctx context.Context, poolID int) (*model.MonitorAlertManagerPool, error) {
	if poolID <= 0 {
		a.l.Error("GetAlertPoolByID 失败: 无效的 poolID", zap.Int("poolID", poolID))
		return nil, fmt.Errorf("无效的 poolID: %d", poolID)
	}

	var alertPool model.MonitorAlertManagerPool
	if err := a.db.WithContext(ctx).Where("id = ?", poolID).First(&alertPool).Error; err != nil {
		a.l.Error("获取 AlertPool 失败", zap.Error(err), zap.Int("poolID", poolID))
		return nil, err
	}

	return &alertPool, nil
}

func (a *alertManagerPoolDAO) CheckMonitorAlertManagerPoolExists(ctx context.Context, alertManagerPool *model.MonitorAlertManagerPool) (bool, error) {
	var count int64

	if err := a.db.WithContext(ctx).
		Model(&model.MonitorAlertManagerPool{}).
		Where("id = ?", alertManagerPool.ID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (a *alertManagerPoolDAO) GetMonitorAlertManagerPoolById(ctx context.Context, id int) (*model.MonitorAlertManagerPool, error) {
	if id <= 0 {
		a.l.Error("GetMonitorAlertManagerPoolById 失败: 无效的 ID", zap.Int("id", id))
		return nil, fmt.Errorf("无效的 ID: %d", id)
	}

	var alertManagerPool model.MonitorAlertManagerPool

	if err := a.db.WithContext(ctx).Where("id = ?", id).First(&alertManagerPool).Error; err != nil {
		a.l.Error("获取 MonitorAlertManagerPool 失败", zap.Error(err), zap.Int("id", id))
		return nil, err
	}

	return &alertManagerPool, nil
}
