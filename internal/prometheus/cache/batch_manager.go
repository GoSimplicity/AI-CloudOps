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

package cache

import (
	"context"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type ConfigData struct {
	Name       string
	PoolID     int
	ConfigType int8
	Content    string
}

type BatchConfigManager struct {
	configDAO configDao.MonitorConfigDAO
	logger    *zap.Logger
}

func NewBatchConfigManager(configDAO configDao.MonitorConfigDAO, logger *zap.Logger) *BatchConfigManager {
	return &BatchConfigManager{
		configDAO: configDAO,
		logger:    logger,
	}
}

// BatchSaveConfigs 批量保存配置到数据库
func (b *BatchConfigManager) BatchSaveConfigs(ctx context.Context, configMap map[string]ConfigData) error {
	if len(configMap) == 0 {
		return nil
	}

	// 收集所有实例IP
	instanceIPs := make([]string, 0, len(configMap))
	for ip := range configMap {
		instanceIPs = append(instanceIPs, ip)
	}

	// 确定配置类型（假设所有配置类型相同）
	var configType int8
	for _, cfg := range configMap {
		configType = cfg.ConfigType
		break
	}

	// 批量查询现有配置
	existingConfigs, err := b.configDAO.GetMonitorConfigsByInstances(ctx, instanceIPs, configType)
	if err != nil && !strings.Contains(err.Error(), "未找到对应的监控配置") {
		b.logger.Error(LogModuleMonitor+"批量查询监控配置失败",
			zap.Int8("config_type", configType),
			zap.Error(err))
		return err
	}

	// 构建IP到配置的映射
	existingMap := make(map[string]*model.MonitorConfig)
	for _, config := range existingConfigs {
		existingMap[config.InstanceIP] = config
	}

	// 准备要创建和更新的配置
	toCreate := make([]*model.MonitorConfig, 0)
	toUpdate := make([]*model.MonitorConfig, 0)
	now := time.Now().Unix()

	for ip, data := range configMap {
		// 验证YAML格式
		if err := validateYAMLConfig(data.Content); err != nil {
			b.logger.Error(LogModuleMonitor+"配置YAML格式验证失败",
				zap.String("instance_ip", ip),
				zap.Int8("config_type", data.ConfigType),
				zap.Error(err))
			continue
		}

		configHash := calculateConfigHash(data.Content)

		if existing, ok := existingMap[ip]; ok {
			// 配置内容未变化则跳过更新
			if existing.ConfigHash == configHash {
				b.logger.Debug(LogModuleMonitor+"配置内容未变化，跳过更新",
					zap.String("instance_ip", ip),
					zap.Int8("config_type", data.ConfigType))
				continue
			}

			// 更新现有配置
			existing.Name = data.Name
			existing.ConfigContent = data.Content
			existing.ConfigHash = configHash
			existing.Status = model.ConfigStatusActive
			existing.LastGeneratedTime = now
			toUpdate = append(toUpdate, existing)
		} else {
			// 创建新配置
			newConfig := &model.MonitorConfig{
				Name:              data.Name,
				PoolID:            data.PoolID,
				InstanceIP:        ip,
				ConfigType:        data.ConfigType,
				ConfigContent:     data.Content,
				ConfigHash:        configHash,
				Status:            model.ConfigStatusActive,
				LastGeneratedTime: now,
			}
			toCreate = append(toCreate, newConfig)
		}
	}

	// 批量创建新配置
	if len(toCreate) > 0 {
		if err := b.configDAO.BatchCreateMonitorConfigs(ctx, toCreate); err != nil {
			b.logger.Error(LogModuleMonitor+"批量创建监控配置失败",
				zap.Int("count", len(toCreate)),
				zap.Error(err))
			return err
		}
		b.logger.Info(LogModuleMonitor+"批量创建监控配置成功",
			zap.Int("count", len(toCreate)))
	}

	// 批量更新现有配置
	if len(toUpdate) > 0 {
		if err := b.configDAO.BatchUpdateMonitorConfigs(ctx, toUpdate); err != nil {
			b.logger.Error(LogModuleMonitor+"批量更新监控配置失败",
				zap.Int("count", len(toUpdate)),
				zap.Error(err))
			return err
		}
		b.logger.Info(LogModuleMonitor+"批量更新监控配置成功",
			zap.Int("count", len(toUpdate)))
	}

	return nil
}
