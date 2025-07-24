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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerPoolService interface {
	GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) (model.ListResp[*model.MonitorAlertManagerPool], error)
	CreateMonitorAlertManagerPool(ctx context.Context, req *model.CreateMonitorAlertManagerPoolReq) error
	UpdateMonitorAlertManagerPool(ctx context.Context, req *model.UpdateMonitorAlertManagerPoolReq) error
	DeleteMonitorAlertManagerPool(ctx context.Context, req *model.DeleteMonitorAlertManagerPoolReq) error
	GetMonitorAlertManagerPool(ctx context.Context, req *model.GetMonitorAlertManagerPoolReq) (*model.MonitorAlertManagerPool, error)
}

type alertManagerPoolService struct {
	dao     alert.AlertManagerPoolDAO
	sendDao alert.AlertManagerSendDAO
	l       *zap.Logger
}

func NewAlertManagerPoolService(
	dao alert.AlertManagerPoolDAO,
	sendDao alert.AlertManagerSendDAO,
	l *zap.Logger,
) AlertManagerPoolService {
	return &alertManagerPoolService{
		dao:     dao,
		sendDao: sendDao,
		l:       l,
	}
}

func (a *alertManagerPoolService) GetMonitorAlertManagerPoolList(ctx context.Context, req *model.GetMonitorAlertManagerPoolListReq) (model.ListResp[*model.MonitorAlertManagerPool], error) {
	pools, count, err := a.dao.GetMonitorAlertManagerPoolList(ctx, req)
	if err != nil {
		a.l.Error("获取告警事件列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorAlertManagerPool]{}, err
	}

	return model.ListResp[*model.MonitorAlertManagerPool]{
		Items: pools,
		Total: count,
	}, nil
}

func (a *alertManagerPoolService) CreateMonitorAlertManagerPool(ctx context.Context, req *model.CreateMonitorAlertManagerPoolReq) error {
	pool := &model.MonitorAlertManagerPool{
		Name:                  req.Name,
		AlertManagerInstances: req.AlertManagerInstances,
		UserID:                req.UserID,
		ResolveTimeout:        req.ResolveTimeout,
		GroupWait:             req.GroupWait,
		GroupInterval:         req.GroupInterval,
		RepeatInterval:        req.RepeatInterval,
		GroupBy:               req.GroupBy,
		Receiver:              req.Receiver,
		CreateUserName:        req.CreateUserName,
	}

	// 检查 AlertManager Pool 是否已存在
	exists, err := a.dao.CheckMonitorAlertManagerPoolExists(ctx, pool)
	if err != nil {
		a.l.Error("创建 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager Pool 已存在")
	}

	// 检查 AlertManager IP 是否已存在
	if err := a.dao.CheckAlertIpExists(ctx, pool); err != nil {
		a.l.Error("创建 AlertManager 集群池失败：检查 AlertManager IP 是否存在时出错", zap.Error(err))
		return err
	}

	// 创建 AlertManager 集群池
	if err := a.dao.CreateMonitorAlertManagerPool(ctx, pool); err != nil {
		a.l.Error("创建 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolService) UpdateMonitorAlertManagerPool(ctx context.Context, req *model.UpdateMonitorAlertManagerPoolReq) error {
	// 检查 ID 是否有效
	if req.ID <= 0 {
		return errors.New("无效的告警池ID")
	}

	// 先获取原有的告警池信息
	oldPool, err := a.dao.GetAlertPoolByID(ctx, req.ID)
	if err != nil {
		a.l.Error("更新 AlertManager 集群池失败：获取原有告警池信息出错", zap.Error(err))
		return err
	}

	pool := &model.MonitorAlertManagerPool{
		Model:                 model.Model{ID: req.ID},
		Name:                  req.Name,
		AlertManagerInstances: req.AlertManagerInstances,
		ResolveTimeout:        req.ResolveTimeout,
		GroupWait:             req.GroupWait,
		GroupInterval:         req.GroupInterval,
		RepeatInterval:        req.RepeatInterval,
		GroupBy:               req.GroupBy,
		Receiver:              req.Receiver,
	}

	// 如果名称发生变化,需要检查新名称是否已存在
	if oldPool.Name != pool.Name {
		exists, err := a.dao.CheckMonitorAlertManagerPoolExists(ctx, pool)
		if err != nil {
			a.l.Error("更新 AlertManager 集群池失败：检查 AlertManager Pool 是否存在时出错", zap.Error(err))
			return err
		}

		if exists {
			return errors.New("告警池名称已存在")
		}
	}

	// 检查 AlertManager IP 是否已被其他池使用
	if err := a.dao.CheckAlertIpExists(ctx, pool); err != nil {
		a.l.Error("更新 AlertManager 集群池失败：检查 AlertManager IP 是否存在时出错", zap.Error(err))
		return err
	}

	// 更新 AlertManager 集群池
	if err := a.dao.UpdateMonitorAlertManagerPool(ctx, pool); err != nil {
		a.l.Error("更新 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolService) DeleteMonitorAlertManagerPool(ctx context.Context, req *model.DeleteMonitorAlertManagerPoolReq) error {
	if req.ID <= 0 {
		return errors.New("无效的告警池ID")
	}

	sendGroups, _, err := a.sendDao.GetMonitorSendGroupByPoolID(ctx, req.ID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除 AlertManager 集群池失败：获取关联发送组时出错", zap.Error(err))
		return err
	}

	if len(sendGroups) > 0 {
		return errors.New("AlertManager 集群池存在关联发送组，无法删除")
	}

	if err := a.dao.DeleteMonitorAlertManagerPool(ctx, req.ID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		a.l.Error("删除 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorAlertManagerPool 获取 AlertManager 实例池
func (a *alertManagerPoolService) GetMonitorAlertManagerPool(ctx context.Context, req *model.GetMonitorAlertManagerPoolReq) (*model.MonitorAlertManagerPool, error) {
	pool, err := a.dao.GetAlertPoolByID(ctx, req.ID)
	if err != nil {
		a.l.Error("获取 AlertManager 实例池失败", zap.Error(err))
		return nil, err
	}

	return pool, nil
}
