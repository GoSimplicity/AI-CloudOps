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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/scrape"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerRecordService interface {
	GetMonitorRecordRuleList(ctx context.Context, req *model.GetMonitorRecordRuleListReq) (model.ListResp[*model.MonitorRecordRule], error)
	CreateMonitorRecordRule(ctx context.Context, req *model.CreateMonitorRecordRuleReq) error
	UpdateMonitorRecordRule(ctx context.Context, req *model.UpdateMonitorRecordRuleReq) error
	DeleteMonitorRecordRule(ctx context.Context, req *model.DeleteMonitorRecordRuleReq) error
	GetMonitorRecordRule(ctx context.Context, req *model.GetMonitorRecordRuleReq) (*model.MonitorRecordRule, error)
}

type alertManagerRecordService struct {
	dao     alert.AlertManagerRecordDAO
	poolDao scrape.ScrapePoolDAO
	l       *zap.Logger
}

func NewAlertManagerRecordService(dao alert.AlertManagerRecordDAO, poolDao scrape.ScrapePoolDAO, l *zap.Logger) AlertManagerRecordService {
	return &alertManagerRecordService{
		dao:     dao,
		poolDao: poolDao,
		l:       l,
	}
}

func (a *alertManagerRecordService) GetMonitorRecordRuleList(ctx context.Context, req *model.GetMonitorRecordRuleListReq) (model.ListResp[*model.MonitorRecordRule], error) {
	rules, count, err := a.dao.GetMonitorRecordRuleList(ctx, req)
	if err != nil {
		a.l.Error("获取记录规则列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorRecordRule]{}, err
	}

	// 为每条规则添加池名称
	for _, rule := range rules {
		if rule.PoolID > 0 {
			pool, err := a.poolDao.GetMonitorScrapePoolById(ctx, rule.PoolID)
			if err != nil {
				a.l.Error("获取Prometheus实例池失败", zap.Error(err), zap.Int("poolID", rule.PoolID))
			} else {
				rule.PoolName = pool.Name
			}
		}
	}

	return model.ListResp[*model.MonitorRecordRule]{
		Total: count,
		Items: rules,
	}, nil
}

func (a *alertManagerRecordService) CreateMonitorRecordRule(ctx context.Context, req *model.CreateMonitorRecordRuleReq) error {
	if req.Name == "" {
		return errors.New("记录规则名称不能为空")
	}

	if req.PoolID <= 0 {
		return errors.New("无效的实例池ID")
	}

	// 检查记录规则是否已存在
	monitorRecordRule := &model.MonitorRecordRule{
		Name:           req.Name,
		PoolID:         req.PoolID,
		Expr:           req.Expr,
		UserID:         req.UserID,
		CreateUserName: req.CreateUserName,
		IpAddress:      req.IpAddress,
		Enable:         req.Enable,
		ForTime:        req.ForTime,
		Labels:         req.Labels,
		Annotations:    req.Annotations,
	}

	exists, err := a.dao.CheckMonitorRecordRuleNameExists(ctx, monitorRecordRule)
	if err != nil {
		a.l.Error("创建记录规则失败：检查记录规则是否存在时出错", zap.Error(err))
		return err
	}

	if exists {
		return errors.New("记录规则已存在")
	}

	// 如果存在表达式，则检查 PromQL 语法是否合法
	if req.Expr != "" {
		if _, err := parser.ParseExpr(req.Expr); err != nil {
			a.l.Error("创建记录规则失败：PromQL 语法错误", zap.Error(err))
			return fmt.Errorf("PromQL 语法错误: %v", err)
		}
	} else {
		return errors.New("表达式不能为空")
	}

	// 创建记录规则
	if err := a.dao.CreateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		a.l.Error("创建记录规则失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) UpdateMonitorRecordRule(ctx context.Context, req *model.UpdateMonitorRecordRuleReq) error {
	if req.ID <= 0 {
		return errors.New("无效的记录规则ID")
	}

	if req.Name == "" {
		return errors.New("记录规则名称不能为空")
	}

	if req.PoolID <= 0 {
		return errors.New("无效的实例池ID")
	}

	// 检查记录规则是否已存在
	rule, err := a.dao.GetMonitorRecordRuleById(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("记录规则不存在，ID: %d", req.ID)
		}
		a.l.Error("更新记录规则失败：获取记录规则时出错", zap.Error(err))
		return err
	}

	monitorRecordRule := &model.MonitorRecordRule{
		Model:       model.Model{ID: req.ID},
		Name:        req.Name,
		PoolID:      req.PoolID,
		Expr:        req.Expr,
		IpAddress:   req.IpAddress,
		Enable:      req.Enable,
		ForTime:     req.ForTime,
		Labels:      req.Labels,
		Annotations: req.Annotations,
	}

	if rule.Name != req.Name {
		exists, err := a.dao.CheckMonitorRecordRuleNameExists(ctx, monitorRecordRule)
		if err != nil {
			a.l.Error("更新记录规则失败：检查记录规则名称是否存在时出错", zap.Error(err))
			return err
		}

		if exists {
			return errors.New("记录规则名称已存在")
		}
	}

	// 如果存在表达式，则检查 PromQL 语法是否合法
	if req.Expr != "" {
		if _, err := parser.ParseExpr(req.Expr); err != nil {
			a.l.Error("更新记录规则失败：PromQL 语法错误", zap.Error(err))
			return fmt.Errorf("PromQL 语法错误: %v", err)
		}
	} else {
		return errors.New("表达式不能为空")
	}

	// 更新记录规则
	if err := a.dao.UpdateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		a.l.Error("更新记录规则失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorRecordRule 删除记录规则
func (a *alertManagerRecordService) DeleteMonitorRecordRule(ctx context.Context, req *model.DeleteMonitorRecordRuleReq) error {
	if req.ID <= 0 {
		return errors.New("无效的记录规则ID")
	}

	// 检查记录规则是否存在
	_, err := a.dao.GetMonitorRecordRuleById(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.l.Error("删除记录规则失败：记录规则不存在", zap.Int("id", req.ID))
			return fmt.Errorf("记录规则不存在")
		}
		a.l.Error("删除记录规则失败：获取记录规则时出错", zap.Error(err))
		return err
	}

	// 删除记录规则
	if err := a.dao.DeleteMonitorRecordRule(ctx, req.ID); err != nil {
		a.l.Error("删除记录规则失败", zap.Error(err))
		return err
	}

	return nil
}

// GetMonitorRecordRule 获取记录规则
func (a *alertManagerRecordService) GetMonitorRecordRule(ctx context.Context, req *model.GetMonitorRecordRuleReq) (*model.MonitorRecordRule, error) {
	if req.ID <= 0 {
		return nil, errors.New("无效的记录规则ID")
	}

	rule, err := a.dao.GetMonitorRecordRuleById(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("记录规则不存在，ID: %d", req.ID)
		}
		a.l.Error("获取记录规则失败", zap.Error(err))
		return nil, err
	}

	// 获取池名称
	if rule.PoolID > 0 {
		pool, err := a.poolDao.GetMonitorScrapePoolById(ctx, rule.PoolID)
		if err != nil {
			a.l.Error("获取Prometheus实例池失败", zap.Error(err), zap.Int("poolID", rule.PoolID))
		} else {
			rule.PoolName = pool.Name
		}
	}

	return rule, nil
}
