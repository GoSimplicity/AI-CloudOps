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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configDao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type MonitorConfigService interface {
	GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) (model.ListResp[*model.MonitorConfig], error)
	GetMonitorConfigByID(ctx context.Context, req *model.GetMonitorConfigReq) (*model.MonitorConfig, error)
	CreateMonitorConfig(ctx context.Context, req *model.CreateMonitorConfigReq) error
	UpdateMonitorConfig(ctx context.Context, req *model.UpdateMonitorConfigReq) error
	DeleteMonitorConfig(ctx context.Context, req *model.DeleteMonitorConfigReq) error
}

type monitorConfigService struct {
	configDao configDao.MonitorConfigDAO
	l         *zap.Logger
}

func NewMonitorConfigService(l *zap.Logger, configDao configDao.MonitorConfigDAO) MonitorConfigService {
	return &monitorConfigService{
		configDao: configDao,
		l:         l,
	}
}

// GetMonitorConfigList 获取配置列表
func (s *monitorConfigService) GetMonitorConfigList(ctx context.Context, req *model.GetMonitorConfigListReq) (model.ListResp[*model.MonitorConfig], error) {
	list, total, err := s.configDao.GetMonitorConfigList(ctx, req)
	if err != nil {
		s.l.Error("获取监控配置列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorConfig]{}, err
	}

	return model.ListResp[*model.MonitorConfig]{
		Items: list,
		Total: total,
	}, nil
}

// GetMonitorConfigByID 通过ID获取配置
func (s *monitorConfigService) GetMonitorConfigByID(ctx context.Context, req *model.GetMonitorConfigReq) (*model.MonitorConfig, error) {
	config, err := s.configDao.GetMonitorConfigByID(ctx, req.ID)
	if err != nil {
		s.l.Error("获取监控配置失败", zap.Error(err))
		return nil, err
	}

	return config, nil
}

func (s *monitorConfigService) CreateMonitorConfig(ctx context.Context, req *model.CreateMonitorConfigReq) error {
	config := &model.MonitorConfig{
		Name:          req.Name,
		PoolID:        req.PoolID,
		InstanceIP:    req.InstanceIP,
		ConfigType:    req.ConfigType,
		ConfigContent: req.ConfigContent,
		Status:        req.Status,
	}

	err := s.configDao.CreateMonitorConfig(ctx, config)
	if err != nil {
		s.l.Error("创建监控配置失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *monitorConfigService) UpdateMonitorConfig(ctx context.Context, req *model.UpdateMonitorConfigReq) error {
	config := &model.MonitorConfig{
		Model:         model.Model{ID: req.ID},
		Name:          req.Name,
		PoolID:        req.PoolID,
		InstanceIP:    req.InstanceIP,
		ConfigType:    req.ConfigType,
		ConfigContent: req.ConfigContent,
		Status:        req.Status,
	}

	err := s.configDao.UpdateMonitorConfig(ctx, config)
	if err != nil {
		s.l.Error("更新监控配置失败", zap.Error(err))
		return err
	}

	return nil
}

func (s *monitorConfigService) DeleteMonitorConfig(ctx context.Context, req *model.DeleteMonitorConfigReq) error {
	err := s.configDao.DeleteMonitorConfig(ctx, req.ID)
	if err != nil {
		s.l.Error("删除监控配置失败", zap.Error(err))
		return err
	}

	return nil
}
