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

	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerPoolService interface {
	GetMonitorAlertManagerPoolList(ctx context.Context, listReq *model.ListReq) (model.ListResp[*model.MonitorAlertManagerPool], error)
	GetMonitorAlertManagerPoolAll(ctx context.Context) (model.ListResp[*model.MonitorAlertManagerPool], error)
	CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error
	DeleteMonitorAlertManagerPool(ctx context.Context, id int) error
	GetMonitorAlertManagerPoolTotal(ctx context.Context) (int, error)
}

type alertManagerPoolService struct {
	dao     alert.AlertManagerPoolDAO
	sendDao alert.AlertManagerSendDAO
	cache   cache.MonitorCache
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAlertManagerPoolService(dao alert.AlertManagerPoolDAO, sendDao alert.AlertManagerSendDAO, cache cache.MonitorCache, l *zap.Logger, userDao userDao.UserDAO) AlertManagerPoolService {
	return &alertManagerPoolService{
		dao:     dao,
		sendDao: sendDao,
		userDao: userDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerPoolService) GetMonitorAlertManagerPoolList(ctx context.Context, listReq *model.ListReq) (model.ListResp[*model.MonitorAlertManagerPool], error) {
	var pools []*model.MonitorAlertManagerPool

	if listReq.Search != "" {
		pools, count, err := a.dao.SearchMonitorAlertManagerPoolByName(ctx, listReq.Search)
		if err != nil {
			a.l.Error("搜索告警事件失败", zap.String("search", listReq.Search), zap.Error(err))
			return model.ListResp[*model.MonitorAlertManagerPool]{}, err
		}
		return model.ListResp[*model.MonitorAlertManagerPool]{
			Items: pools,
			Total: count,
		}, nil
	}

	offset := (listReq.Page - 1) * listReq.Size
	limit := listReq.Size

	pools, count, err := a.dao.GetMonitorAlertManagerPoolList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取告警事件列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorAlertManagerPool]{}, err
	}

	for _, pool := range pools {
		user, err := a.userDao.GetUserByID(ctx, pool.UserID)
		if err != nil {
			a.l.Error("获取创建用户名失败", zap.Error(err))
		}
		if user.RealName == "" {
			pool.CreateUserName = user.Username
		} else {
			pool.CreateUserName = user.RealName
		}
	}

	return model.ListResp[*model.MonitorAlertManagerPool]{
		Items: pools,
		Total: count,
	}, nil
}

func (a *alertManagerPoolService) CreateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 检查 AlertManager Pool 是否已存在
	exists, err := a.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
	if err != nil {
		a.l.Error("创建 AlertManager 集群池失败：检查是否存在时出错", zap.Error(err))
		return err
	}
	if exists {
		return errors.New("AlertManager Pool 已存在")
	}

	// 检查 AlertManager IP 是否已存在
	if err := a.checkAlertIpExists(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("创建 AlertManager 集群池失败：检查 AlertManager IP 是否存在时出错", zap.Error(err))
		return err
	}

	// 创建 AlertManager 集群池
	if err := a.dao.CreateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("创建 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolService) UpdateMonitorAlertManagerPool(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	// 检查 ID 是否有效
	if monitorAlertManagerPool.ID <= 0 {
		return errors.New("无效的告警池ID")
	}

	// 先获取原有的告警池信息
	oldPool, err := a.dao.GetAlertPoolByID(ctx, monitorAlertManagerPool.ID)
	if err != nil {
		a.l.Error("更新 AlertManager 集群池失败：获取原有告警池信息出错", zap.Error(err))
		return err
	}

	// 如果名称发生变化,需要检查新名称是否已存在
	if oldPool.Name != monitorAlertManagerPool.Name {
		exists, err := a.dao.CheckMonitorAlertManagerPoolExists(ctx, monitorAlertManagerPool)
		if err != nil {
			a.l.Error("更新 AlertManager 集群池失败：检查 AlertManager Pool 是否存在时出错", zap.Error(err))
			return err
		}

		if exists {
			return errors.New("告警池名称已存在")
		}
	}

	// 检查 AlertManager IP 是否已被其他池使用
	if err := a.checkAlertIpExists(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("更新 AlertManager 集群池失败：检查 AlertManager IP 是否存在时出错", zap.Error(err))
		return err
	}

	// 更新 AlertManager 集群池
	if err := a.dao.UpdateMonitorAlertManagerPool(ctx, monitorAlertManagerPool); err != nil {
		a.l.Error("更新 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolService) DeleteMonitorAlertManagerPool(ctx context.Context, id int) error {
	// 检查 ID 是否有效
	if id <= 0 {
		return errors.New("无效的告警池ID")
	}

	// 检查 AlertManager 集群池是否有关联的发送组
	sendGroups, err := a.sendDao.GetMonitorSendGroupByPoolId(ctx, id)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		a.l.Error("删除 AlertManager 集群池失败：获取关联发送组时出错", zap.Error(err))
		return err
	}

	if len(sendGroups) > 0 {
		return errors.New("AlertManager 集群池存在关联发送组，无法删除")
	}

	// 删除 AlertManager 集群池
	if err := a.dao.DeleteMonitorAlertManagerPool(ctx, id); err != nil {
		a.l.Error("删除 AlertManager 集群池失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerPoolService) GetMonitorAlertManagerPoolTotal(ctx context.Context) (int, error) {
	total, err := a.dao.GetMonitorAlertManagerPoolTotal(ctx)
	if err != nil {
		a.l.Error("获取 AlertManager 集群池总数失败", zap.Error(err))
		return 0, err
	}

	return total, nil
}

func (a *alertManagerPoolService) checkAlertIpExists(ctx context.Context, monitorAlertManagerPool *model.MonitorAlertManagerPool) error {
	pools, _, err := a.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		a.l.Error("检查 AlertManager Pool 是否存在失败", zap.Error(err))
		return err
	}

	return pkg.CheckAlertIpExists(monitorAlertManagerPool, pools)
}

func (a *alertManagerPoolService) GetMonitorAlertManagerPoolAll(ctx context.Context) (model.ListResp[*model.MonitorAlertManagerPool], error) {
	pools, count, err := a.dao.GetAllAlertManagerPools(ctx)
	if err != nil {
		a.l.Error("获取所有告警池失败", zap.Error(err))
		return model.ListResp[*model.MonitorAlertManagerPool]{}, err
	}
	return model.ListResp[*model.MonitorAlertManagerPool]{
		Items: pools,
		Total: count,
	}, nil
}
