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
	"sort"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MonitorConfigDAO interface {
	GetGeneratedConfigPoolCandidates(context.Context, GeneratedConfigPoolQuery) ([]*model.MonitorConfig, int64, error)
	GetGeneratedConfigNamespacePage(context.Context, GeneratedConfigKeysQuery) ([]*model.MonitorConfig, int64, error)
	UpdateGeneratedMonitorConfigs(context.Context, []GeneratedConfigUpdate) error
	DeleteGeneratedMonitorConfig(context.Context, GeneratedConfigOwner) error
	GetGeneratedMonitorConfigByID(context.Context, int) (*model.MonitorConfig, error)
	GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) ([]*model.MonitorConfig, int64, error)
	GetMonitorConfigByID(ctx context.Context, id int) (*model.MonitorConfig, error)
	CreateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	UpdateMonitorConfig(ctx context.Context, config *model.MonitorConfig) error
	DeleteMonitorConfig(ctx context.Context, id int) error
	GetMonitorConfigByInstance(ctx context.Context, instanceIP string, configType int8) (*model.MonitorConfig, error)
	// 批量操作方法
	BatchCreateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error
	BatchUpdateMonitorConfigs(ctx context.Context, configs []*model.MonitorConfig) error
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

func (d *monitorConfigDAO) CreateMonitorConfig(ctx context.Context, row *model.MonitorConfig) error {
	return d.BatchCreateMonitorConfigs(ctx, []*model.MonitorConfig{row})
}

func (d *monitorConfigDAO) UpdateMonitorConfig(ctx context.Context, row *model.MonitorConfig) error {
	return d.BatchUpdateMonitorConfigs(ctx, []*model.MonitorConfig{row})
}

func (d *monitorConfigDAO) DeleteMonitorConfig(ctx context.Context, id int) error {
	if id <= 0 {
		return ErrInvalidConfigWrite
	}
	var observed model.MonitorConfig
	if err := d.db.WithContext(ctx).First(&observed, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return configWriteError(ctx, err)
	}
	err := d.withConfigNamespaces(ctx, []configNamespaceKey{configKey(&observed)}, func(tx *gorm.DB) error {
		current, err := currentConfigByID(tx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if GeneratedConfigOwnerOf(current) != GeneratedConfigOwnerOf(&observed) {
			return ErrConfigNamespaceConflict
		}
		rows, err := currentConfigNamespace(tx, configKey(current))
		if err != nil {
			return err
		}
		if !namespaceContains(rows, id) {
			return ErrConfigNamespaceConflict
		}
		// Explicit ID deletion may repair a historical duplicate; never delete peers.
		result := tx.Where("id = ?", id).Delete(&model.MonitorConfig{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrConfigNamespaceConflict
		}
		return nil
	})
	return configWriteError(ctx, err)
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

func (d *monitorConfigDAO) BatchCreateMonitorConfigs(ctx context.Context, input []*model.MonitorConfig) error {
	if len(input) == 0 {
		return nil
	}
	copies := make([]*model.MonitorConfig, len(input))
	keys := make([]configNamespaceKey, 0, len(input))
	seen := map[configNamespaceKey]bool{}
	for i, row := range input {
		if row == nil || !validConfigNamespace(configKey(row)) {
			return ErrInvalidConfigWrite
		}
		key := configKey(row)
		if seen[key] {
			return ErrConfigNamespaceConflict
		}
		seen[key] = true
		keys = append(keys, key)
		copy := *row
		copies[i] = &copy
	}
	err := d.withConfigNamespaces(ctx, keys, func(tx *gorm.DB) error {
		for _, key := range keys {
			rows, err := currentConfigNamespace(tx, key)
			if err != nil {
				return err
			}
			if len(rows) != 0 {
				return ErrConfigNamespaceConflict
			}
		}
		return tx.CreateInBatches(copies, 100).Error
	})
	if err != nil {
		return configWriteError(ctx, err)
	}
	for i := range input {
		*input[i] = *copies[i]
	}
	return nil
}

func (d *monitorConfigDAO) BatchUpdateMonitorConfigs(ctx context.Context, input []*model.MonitorConfig) error {
	if len(input) == 0 {
		return nil
	}
	copies := make([]*model.MonitorConfig, len(input))
	observed := make([]model.MonitorConfig, len(input))
	keys := make([]configNamespaceKey, 0, 2*len(input))
	seenIDs := map[int]bool{}
	for i, row := range input {
		if !validConfigPatch(row) {
			return ErrInvalidConfigWrite
		}
		if seenIDs[row.ID] {
			return ErrInvalidConfigWrite
		}
		seenIDs[row.ID] = true
		copy := *row
		copies[i] = &copy
	}
	for i, row := range copies {
		if err := d.db.WithContext(ctx).First(&observed[i], row.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrConfigNamespaceConflict
			}
			return configWriteError(ctx, err)
		}
		source, target := configKey(&observed[i]), configPatchKey(&observed[i], row)
		if !validConfigNamespace(source) || !validConfigNamespace(target) {
			return ErrInvalidConfigWrite
		}
		keys = append(keys, source, target)
	}
	err := d.withConfigNamespaces(ctx, keys, func(tx *gorm.DB) error {
		// Data rows are locked by ID after all namespace guards, never vice versa.
		indices := make([]int, len(copies))
		for i := range indices {
			indices[i] = i
		}
		sort.Slice(indices, func(i, j int) bool { return copies[indices[i]].ID < copies[indices[j]].ID })
		current := make([]*model.MonitorConfig, len(copies))
		for _, i := range indices {
			row, err := currentConfigByID(tx, copies[i].ID)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrConfigNamespaceConflict
			}
			if err != nil {
				return err
			}
			if GeneratedConfigOwnerOf(row) != GeneratedConfigOwnerOf(&observed[i]) {
				return ErrConfigNamespaceConflict
			}
			current[i] = row
		}
		targets := map[configNamespaceKey]bool{}
		for i, patch := range copies {
			source, target := configKey(current[i]), configPatchKey(current[i], patch)
			rows, err := currentConfigNamespace(tx, target)
			if err != nil {
				return err
			}
			if source == target {
				if !namespaceContains(rows, patch.ID) {
					return ErrConfigNamespaceConflict
				}
			} else {
				if len(rows) != 0 || targets[target] {
					return ErrConfigNamespaceConflict
				}
			}
			targets[target] = true
		}
		for _, patch := range copies {
			if err := tx.Model(&model.MonitorConfig{}).Where("id = ?", patch.ID).Updates(patch).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return configWriteError(ctx, err)
	}
	for i := range input {
		*input[i] = *copies[i]
	}
	return nil
}

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
