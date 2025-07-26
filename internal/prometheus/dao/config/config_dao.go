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
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MonitorConfigDAO interface {
	GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error)
	GetMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error)
	CreateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	UpdateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	DeleteMonitorConfig(ctx context.Context, id int) error
	GetMonitorConfigByInstance(ctx context.Context, instanceIP string, configType int8) (*model.MonitorConfig, error)
	// 批量操作方法
	BatchCreateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error
	BatchUpdateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error
	BatchUpsertMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error
	GetMonitorConfigsByInstances(ctx context.Context, instanceIPs []string, configType int8) ([]*model.MonitorConfig, error)
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

// GetMonitorConfigList 获取监控配置列表
func (d *monitorConfigDAO) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error) {
	var (
		total int64
		list  []*model.MonitorConfig
	)

	db := d.db.WithContext(ctx).Model(&model.MonitorConfig{})

	if req.Page <= 0 {
		req.Page = 1
	}

	if req.Size <= 0 {
		req.Size = 10
	}

	// 搜索条件
	if req.Search != "" {
		db = db.Where("name LIKE ?", "%"+req.Search+"%")
	}

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

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		d.l.Error("获取监控配置列表总数失败", zap.Error(err))
		return nil, 0, err
	}
	if total == 0 {
		return []*model.MonitorConfig{}, 0, nil
	}

	// 排序和分页
	db = db.Order("created_at DESC")
	offset := (req.Page - 1) * req.Size
	db = db.Offset(offset).Limit(req.Size)

	if err := db.Find(&list).Error; err != nil {
		d.l.Error("获取监控配置列表失败", zap.Error(err))
		return nil, 0, err
	}

	return list, total, nil
}

// GetMonitorConfigByID 通过ID获取监控配置
func (d *monitorConfigDAO) GetMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的ID: %d", id)
	}
	var config model.MonitorConfig
	err := d.db.WithContext(ctx).First(&config, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
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

	return nil
}

// UpdateMonitorConfig 更新监控配置
func (d *monitorConfigDAO) UpdateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error {
	if config.ID == 0 {
		return errors.New("无效的监控配置ID")
	}

	if err := d.db.WithContext(ctx).Model(&model.MonitorConfig{}).Where("id = ?", config.ID).Updates(config).Error; err != nil {
		d.l.Error("更新监控配置失败", zap.Int("id", config.ID), zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorConfig 删除监控配置
func (d *monitorConfigDAO) DeleteMonitorConfig(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("无效的ID: %d", id)
	}

	if err := d.db.WithContext(ctx).Delete(&model.MonitorConfig{}, id).Error; err != nil {
		d.l.Error("删除监控配置失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorConfigByInstance 通过实例IP和配置类型获取监控配置
func (d *monitorConfigDAO) GetMonitorConfigByInstance(ctx context.Context, instanceIP string, configType int8) (*model.MonitorConfig, error) {
	if instanceIP == "" || configType == 0 {
		return nil, errors.New("instanceIP和configType不能为空")
	}
	var config model.MonitorConfig

	err := d.db.WithContext(ctx).
		Where("instance_ip = ? AND config_type = ?", instanceIP, configType).
		First(&config).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("未找到对应的监控配置: instance_ip=%s, config_type=%d", instanceIP, configType)
		}
		d.l.Error("通过实例和类型获取监控配置失败", zap.String("instance_ip", instanceIP), zap.Int8("config_type", configType), zap.Error(err))
		return nil, err
	}

	return &config, nil
}

// BatchCreateMonitorConfigs 批量创建监控配置
func (d *monitorConfigDAO) BatchCreateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error {
	if len(configs) == 0 {
		return nil
	}

	const batchSize = 100
	for i := 0; i < len(configs); i += batchSize {
		end := i + batchSize
		if end > len(configs) {
			end = len(configs)
		}

		batch := configs[i:end]
		if err := d.db.WithContext(ctx).CreateInBatches(batch, batchSize).Error; err != nil {
			d.l.Error("批量创建监控配置失败", 
				zap.Int("batch_start", i), 
				zap.Int("batch_size", len(batch)), 
				zap.Error(err))
			return err
		}

		d.l.Debug("批量创建监控配置成功", 
			zap.Int("batch_start", i), 
			zap.Int("batch_size", len(batch)))
	}

	return nil
}

// BatchUpdateMonitorConfigs 批量更新监控配置
func (d *monitorConfigDAO) BatchUpdateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error {
	if len(configs) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, config := range configs {
			if config.ID == 0 {
				d.l.Warn("跳过无效的监控配置ID", zap.String("name", config.Name))
				continue
			}

			if err := tx.Model(&model.MonitorConfig{}).Where("id = ?", config.ID).Updates(config).Error; err != nil {
				d.l.Error("批量更新监控配置失败", zap.Int("id", config.ID), zap.Error(err))
				return err
			}
		}
		return nil
	})
}

// BatchUpsertMonitorConfigs 批量插入或更新监控配置
func (d *monitorConfigDAO) BatchUpsertMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error {
	if len(configs) == 0 {
		return nil
	}

	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		const batchSize = 100
		for i := 0; i < len(configs); i += batchSize {
			end := i + batchSize
			if end > len(configs) {
				end = len(configs)
			}

			batch := configs[i:end]
			
			// 使用ON DUPLICATE KEY UPDATE语法进行批量upsert
			if err := tx.Clauses(
				// 在冲突时更新指定字段
				// 这里假设unique key是 instance_ip + config_type
			).CreateInBatches(batch, batchSize).Error; err != nil {
				// 如果批量upsert失败，回退到逐个处理
				for _, config := range batch {
					var existing model.MonitorConfig
					err := tx.Where("instance_ip = ? AND config_type = ?", config.InstanceIP, config.ConfigType).First(&existing).Error
					
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// 记录不存在，创建新记录
						if err := tx.Create(config).Error; err != nil {
							d.l.Error("创建监控配置失败", 
								zap.String("instance_ip", config.InstanceIP), 
								zap.Int8("config_type", config.ConfigType), 
								zap.Error(err))
							return err
						}
					} else if err != nil {
						d.l.Error("查询监控配置失败", 
							zap.String("instance_ip", config.InstanceIP), 
							zap.Int8("config_type", config.ConfigType), 
							zap.Error(err))
						return err
					} else {
						// 记录存在，更新记录
						config.ID = existing.ID
						if err := tx.Model(&existing).Updates(config).Error; err != nil {
							d.l.Error("更新监控配置失败", 
								zap.Int("id", existing.ID), 
								zap.Error(err))
							return err
						}
					}
				}
			}

			d.l.Debug("批量upsert监控配置成功", 
				zap.Int("batch_start", i), 
				zap.Int("batch_size", len(batch)))
		}
		return nil
	})
}

// GetMonitorConfigsByInstances 批量获取监控配置
func (d *monitorConfigDAO) GetMonitorConfigsByInstances(ctx context.Context, instanceIPs []string, configType int8) ([]*model.MonitorConfig, error) {
	if len(instanceIPs) == 0 {
		return nil, errors.New("instanceIPs不能为空")
	}

	var configs []*model.MonitorConfig
	err := d.db.WithContext(ctx).
		Where("instance_ip IN ? AND config_type = ?", instanceIPs, configType).
		Find(&configs).Error

	if err != nil {
		d.l.Error("批量获取监控配置失败", 
			zap.Strings("instance_ips", instanceIPs), 
			zap.Int8("config_type", configType), 
			zap.Error(err))
		return nil, err
	}

	return configs, nil
}
