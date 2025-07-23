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
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AlertManagerRuleService interface {
	GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) (model.ListResp[*model.MonitorAlertRule], error)
	CreateMonitorAlertRule(ctx context.Context, req *model.CreateMonitorAlertRuleReq) error
	UpdateMonitorAlertRule(ctx context.Context, req *model.UpdateMonitorAlertRuleReq) error
	DeleteMonitorAlertRule(ctx context.Context, req *model.DeleteMonitorAlertRuleRequest) error
	PromqlExprCheck(ctx context.Context, req *model.PromqlAlertRuleExprCheckReq) (bool, error)
	GetMonitorAlertRule(ctx context.Context, req *model.GetMonitorAlertRuleReq) (*model.MonitorAlertRule, error)
}

type alertManagerRuleService struct {
	logger       *zap.Logger
	ruleDAO      alert.AlertManagerRuleDAO
	poolDAO      alert.AlertManagerPoolDAO
	sendGroupDAO alert.AlertManagerSendDAO
}

func NewAlertManagerRuleService(
	logger *zap.Logger,
	ruleDAO alert.AlertManagerRuleDAO,
	poolDAO alert.AlertManagerPoolDAO,
	sendGroupDAO alert.AlertManagerSendDAO,
) AlertManagerRuleService {
	return &alertManagerRuleService{
		logger:       logger,
		ruleDAO:      ruleDAO,
		poolDAO:      poolDAO,
		sendGroupDAO: sendGroupDAO,
	}
}

// GetMonitorAlertRuleList 获取告警规则列表
func (s *alertManagerRuleService) GetMonitorAlertRuleList(ctx context.Context, req *model.GetMonitorAlertRuleListReq) (model.ListResp[*model.MonitorAlertRule], error) {
	rules, count, err := s.ruleDAO.GetMonitorAlertRuleList(ctx, req)
	if err != nil {
		s.logger.Error("获取告警规则列表失败", zap.Error(err))
		return model.ListResp[*model.MonitorAlertRule]{}, err
	}

	// 补充额外信息
	for i := range rules {
		// 获取发送组名称
		sendGroup, err := s.sendGroupDAO.GetMonitorSendGroupById(ctx, rules[i].SendGroupID)
		if err == nil && sendGroup != nil {
			rules[i].SendGroupName = sendGroup.NameZh
		}

		// 获取Prometheus实例池名称
		pool, err := s.poolDAO.GetAlertPoolByID(ctx, rules[i].PoolID)
		if err == nil && pool != nil {
			rules[i].PoolName = pool.Name
		}
	}

	return model.ListResp[*model.MonitorAlertRule]{
		Items: rules,
		Total: count,
	}, nil
}

// CreateMonitorAlertRule 创建告警规则
func (s *alertManagerRuleService) CreateMonitorAlertRule(ctx context.Context, req *model.CreateMonitorAlertRuleReq) error {
	// 验证PromQL表达式
	_, err := s.PromqlExprCheck(ctx, &model.PromqlAlertRuleExprCheckReq{
		PromqlExpr: req.Expr,
	})

	if err != nil {
		return errors.New("PromQL表达式语法错误: " + err.Error())
	}

	// 检查Pool是否存在
	_, err = s.poolDAO.GetAlertPoolByID(ctx, req.PoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("指定的Prometheus实例池不存在")
		}
		return err
	}

	// 检查SendGroup是否存在
	_, err = s.sendGroupDAO.GetMonitorSendGroupById(ctx, req.SendGroupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("指定的发送组不存在")
		}
		return err
	}

	// 创建告警规则
	rule := &model.MonitorAlertRule{
		Name:           req.Name,
		UserID:         req.UserID,
		PoolID:         req.PoolID,
		SendGroupID:    req.SendGroupID,
		IpAddress:      req.IpAddress,
		Enable:         req.Enable,
		Expr:           req.Expr,
		Severity:       req.Severity,
		GrafanaLink:    req.GrafanaLink,
		ForTime:        req.ForTime,
		Labels:         req.Labels,
		Annotations:    req.Annotations,
		CreateUserName: req.CreateUserName,
	}

	err = s.ruleDAO.CreateMonitorAlertRule(ctx, rule)
	if err != nil {
		s.logger.Error("创建告警规则失败", zap.Error(err))
		return err
	}

	return nil
}

// UpdateMonitorAlertRule 更新告警规则
func (s *alertManagerRuleService) UpdateMonitorAlertRule(ctx context.Context, req *model.UpdateMonitorAlertRuleReq) error {
	// 验证PromQL表达式
	_, err := s.PromqlExprCheck(ctx, &model.PromqlAlertRuleExprCheckReq{
		PromqlExpr: req.Expr,
	})
	if err != nil {
		return err
	}

	// 检查规则是否存在
	_, err = s.ruleDAO.GetMonitorAlertRuleById(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("告警规则不存在")
		}
		return err
	}

	// 检查Pool是否存在
	_, err = s.poolDAO.GetAlertPoolByID(ctx, req.PoolID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("指定的Prometheus实例池不存在")
		}
		return err
	}

	// 检查SendGroup是否存在
	_, err = s.sendGroupDAO.GetMonitorSendGroupById(ctx, req.SendGroupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("指定的发送组不存在")
		}
		return err
	}

	// 更新规则
	err = s.ruleDAO.UpdateMonitorAlertRule(ctx, req)
	if err != nil {
		s.logger.Error("更新告警规则失败", zap.Error(err))
		return err
	}

	return nil
}

// DeleteMonitorAlertRule 删除告警规则
func (s *alertManagerRuleService) DeleteMonitorAlertRule(ctx context.Context, req *model.DeleteMonitorAlertRuleRequest) error {
	// 检查规则是否存在
	_, err := s.ruleDAO.GetMonitorAlertRuleById(ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("告警规则不存在")
		}
		return err
	}

	// 删除规则
	err = s.ruleDAO.DeleteMonitorAlertRule(ctx, req.ID)
	if err != nil {
		s.logger.Error("删除告警规则失败", zap.Error(err))
		return err
	}

	return nil
}

// PromqlExprCheck 检查PromQL表达式是否有效
func (s *alertManagerRuleService) PromqlExprCheck(ctx context.Context, req *model.PromqlAlertRuleExprCheckReq) (bool, error) {
	if req.PromqlExpr == "" {
		return false, errors.New("PromQL表达式不能为空")
	}

	// 尝试解析表达式
	_, err := parser.ParseExpr(req.PromqlExpr)
	if err != nil {
		s.logger.Error("PromQL表达式语法错误", zap.Error(err), zap.String("expr", req.PromqlExpr))
		return false, errors.New("PromQL表达式语法错误: " + err.Error())
	}

	return true, nil
}

// GetMonitorAlertRuleById 根据ID获取告警规则
func (s *alertManagerRuleService) GetMonitorAlertRule(ctx context.Context, req *model.GetMonitorAlertRuleReq) (*model.MonitorAlertRule, error) {
	rule, err := s.ruleDAO.GetMonitorAlertRuleById(ctx, req.ID)
	if err != nil {
		s.logger.Error("根据ID获取告警规则失败", zap.Int("id", req.ID), zap.Error(err))
		return nil, err
	}

	return rule, nil
}
