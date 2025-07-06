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

package config

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type MonitorConfigDAO interface {
	GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error)
	GetMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error)
	CreateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	UpdateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	DeleteMonitorConfig(ctx context.Context, id int) error
	GetMonitorConfigByInstance(ctx context.Context, instanceIP string, configType int8) (*model.MonitorConfig, error)
}

type monitorConfigDAO struct {
	l  *zap.Logger
	db *gorm.DB
}

func NewMonitorConfigDAO(l *zap.Logger, db *gorm.DB) MonitorConfigDAO {
	return &monitorConfigDAO{
		l:  l,
		db: db,
	}
}

// GetMonitorConfigList 获取配置列表
func (d *monitorConfigDAO) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	var (
		total int64
		list  []*model.MonitorConfig
	)

	db := d.db.WithContext(ctx).Model(&model.MonitorConfig{})

	if req.PoolID != nil {
		db = db.Where("pool_id = ?", *req.PoolID)
	}

	if req.InstanceIP != "" {
		db = db.Where("instance_ip = ?", req.InstanceIP)
	}

	if req.ConfigType != nil {
		db = db.Where("config_type = ?", *req.ConfigType)
	}

	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}

	if err := db.Count(&total).Error; err != nil {
		d.l.Error("获取监控配置列表总数失败", zap.Error(err))
		return nil, 0, err
	}

	if total == 0 {
		return []*model.MonitorConfig{}, 0, nil
	}

	// 默认按创建时间排序
	db = db.Order("created_at desc")

	// 分页
	if req.Page > 0 && req.Size > 0 {
		offset := (req.Page - 1) * req.Size
		db = db.Offset(int(offset)).Limit(int(req.Size))
	}

	if err := db.Find(&list).Error; err != nil {
		d.l.Error("获取监控配置列表失败", zap.Error(err))
		return nil, 0, err
	}

	return list, total, nil
}

// GetMonitorConfigByID 通过ID获取配置
func (d *monitorConfigDAO) GetMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error) {
	var config model.MonitorConfig
	err := d.db.WithContext(ctx).Where("id = ?", id).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("监控配置不存在, ID: %d", id)
		}
		d.l.Error("获取监控配置失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	return &config, nil
}

// CreateMonitorConfig 创建监控配置
func (d *monitorConfigDAO) CreateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error {
	if err := d.db.WithContext(ctx).Create(config).Error; err != nil {
		d.l.Error("创建监控配置失败", zap.String("name", config.Name), zap.Error(err))
		return err
	}
	d.l.Info("创建监控配置成功", zap.String("name", config.Name), zap.Int("poolID", config.PoolID))
	return nil
}

// UpdateMonitorConfig 更新监控配置
func (d *monitorConfigDAO) UpdateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error {
	if err := d.db.WithContext(ctx).Model(&model.MonitorConfig{}).Where("id = ?", config.ID).Updates(map[string]interface{}{
		"name":                config.Name,
		"config_content":      config.ConfigContent,
		"config_hash":         config.ConfigHash,
		"status":              config.Status,
		"last_generated_time": config.LastGeneratedTime,
	}).Error; err != nil {
		d.l.Error("更新监控配置失败", zap.Int("id", config.ID), zap.Error(err))
		return err
	}
	d.l.Info("更新监控配置成功", zap.Int("id", config.ID), zap.String("name", config.Name))
	return nil
}

// DeleteMonitorConfig 删除监控配置
func (d *monitorConfigDAO) DeleteMonitorConfig(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Delete(&model.MonitorConfig{}, id).Error; err != nil {
		d.l.Error("删除监控配置失败", zap.Int("id", id), zap.Error(err))
		return err
	}
	d.l.Info("删除监控配置成功", zap.Int("id", id))
	return nil
}

func (d *monitorConfigDAO) GetMonitorConfigByInstance(ctx context.Context, instanceIP string, configType int8) (*model.MonitorConfig, error) {
	var config model.MonitorConfig
	err := d.db.WithContext(ctx).Where("instance_ip = ? AND config_type = ?", instanceIP, configType).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}
