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
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
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
	cache   cache.MonitorCache
	l       *zap.Logger
}

func NewAlertManagerRecordService(dao alert.AlertManagerRecordDAO, poolDao scrape.ScrapePoolDAO, cache cache.MonitorCache, l *zap.Logger) AlertManagerRecordService {
	return &alertManagerRecordService{
		dao:     dao,
		poolDao: poolDao,
		l:       l,
		cache:   cache,
	}
}

func (a *alertManagerRecordService) GetMonitorRecordRuleList(ctx context.Context, req *model.GetMonitorRecordRuleListReq) (model.ListResp[*model.MonitorRecordRule], error) {
	if req.Search != "" {
		rules, count, err := a.dao.SearchMonitorRecordRuleByName(ctx, req.Search)
		if err != nil {
			a.l.Error("搜索记录规则失败", zap.String("search", req.Search), zap.Error(err))
			return model.ListResp[*model.MonitorRecordRule]{}, err
		}

		// 为每条规则添加用户名和池名称
		for _, rule := range rules {
			pool, err := a.poolDao.GetMonitorScrapePoolById(ctx, rule.PoolID)
			if err != nil {
				a.l.Error("获取Prometheus实例池失败", zap.Error(err))
			} else {
				rule.PoolName = pool.Name
			}
		}

		return model.ListResp[*model.MonitorRecordRule]{
			Total: count,
			Items: rules,
		}, nil
	}

	offset := (req.Page - 1) * req.Size
	limit := req.Size

	rules, count, err := a.dao.GetMonitorRecordRuleList(ctx, offset, limit)
	if err != nil {
		a.l.Error("获取记录规则列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorRecordRule]{}, err
	}

	// 为每条规则添加用户名和池名称
	for _, rule := range rules {
		pool, err := a.poolDao.GetMonitorScrapePoolById(ctx, rule.PoolID)
		if err != nil {
			a.l.Error("获取Prometheus实例池失败", zap.Error(err))
		} else {
			rule.PoolName = pool.Name
		}
	}

	return model.ListResp[*model.MonitorRecordRule]{
		Total: count,
		Items: rules,
	}, nil
}

func (a *alertManagerRecordService) CreateMonitorRecordRule(ctx context.Context, req *model.CreateMonitorRecordRuleReq) error {
	// 检查记录规则是否已存在
	monitorRecordRule := &model.MonitorRecordRule{
		Name:   req.Name,
		PoolID: req.PoolID,
		Expr:   req.Expr,
		UserID: req.UserID,
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
	}

	// 创建记录规则
	if err := a.dao.CreateMonitorRecordRule(ctx, monitorRecordRule); err != nil {
		a.l.Error("创建记录规则失败", zap.Error(err))
		return err
	}

	return nil
}

func (a *alertManagerRecordService) UpdateMonitorRecordRule(ctx context.Context, req *model.UpdateMonitorRecordRuleReq) error {
	// 检查记录规则是否已存在
	rule, err := a.dao.GetMonitorRecordRuleById(ctx, req.ID)
	if err != nil {
		a.l.Error("更新记录规则失败：获取记录规则时出错", zap.Error(err))
		return err
	}

	monitorRecordRule := &model.MonitorRecordRule{
		Name:   req.Name,
		PoolID: req.PoolID,
		Expr:   req.Expr,
	}
	monitorRecordRule.ID = req.ID

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
	rule, err := a.dao.GetMonitorRecordRuleById(ctx, req.ID)
	if err != nil {
		a.l.Error("获取记录规则失败", zap.Error(err))
		return nil, err
	}

	return rule, nil
}
