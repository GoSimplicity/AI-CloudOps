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
	"fmt"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultTaskTimeout = 5 * time.Minute
)

type MonitorCache interface {
	MonitorCacheManager(ctx context.Context) error
}

type monitorCache struct {
	PrometheusMainConfig   PromConfigCache
	AlertManagerMainConfig AlertConfigCache
	AlertRuleConfig        RuleConfigCache
	AlertRecordConfig      RecordConfigCache
	l                      *zap.Logger
}

func NewMonitorCache(
	promConfig PromConfigCache,
	alertManagerConfig AlertConfigCache,
	alertRuleConfig RuleConfigCache,
	alertRecordConfig RecordConfigCache,
	l *zap.Logger,
) MonitorCache {
	return &monitorCache{
		PrometheusMainConfig:   promConfig,
		AlertManagerMainConfig: alertManagerConfig,
		AlertRuleConfig:        alertRuleConfig,
		AlertRecordConfig:      alertRecordConfig,
		l:                      l,
	}
}

// MonitorCacheManager 监控缓存管理入口
func (mc *monitorCache) MonitorCacheManager(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTaskTimeout)
	defer cancel()

	// 使用errgroup管理并发任务
	g, ctx := errgroup.WithContext(ctx)

	// 定义任务列表
	tasks := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Prometheus主配置", mc.PrometheusMainConfig.GeneratePrometheusMainConfig},
		{"AlertManager主配置", mc.AlertManagerMainConfig.GenerateAlertManagerMainConfig},
		{"告警规则配置", mc.AlertRuleConfig.GenerateAlertRuleConfigYaml},
		{"预聚合规则配置", mc.AlertRecordConfig.GenerateRecordRuleConfigYaml},
	}

	// 启动所有任务
	for _, task := range tasks {
		task := task // 避免闭包捕获问题
		g.Go(func() error {
			return mc.executeTask(ctx, task.name, task.fn)
		})
	}

	// 等待所有任务完成
	if err := g.Wait(); err != nil {
		mc.l.Error("监控缓存更新失败",
			zap.String("error", err.Error()),
			zap.Duration("timeout", DefaultTaskTimeout),
		)
		return fmt.Errorf("监控缓存更新失败: %w", err)
	}

	mc.l.Info("监控缓存配置更新成功完成")
	return nil
}

// executeTask 封装任务执行逻辑
func (mc *monitorCache) executeTask(ctx context.Context, taskName string, taskFn func(context.Context) error) error {
	startTime := time.Now()
	mc.l.Info("开始执行配置任务",
		zap.String("task", taskName),
		zap.Time("start_time", startTime),
	)

	taskCtx, cancel := context.WithTimeout(ctx, DefaultTaskTimeout)
	defer cancel()

	// 执行任务
	if err := taskFn(taskCtx); err != nil {
		mc.l.Error("配置任务执行失败",
			zap.String("task", taskName),
			zap.Error(err),
			zap.Duration("duration", time.Since(startTime)),
		)
		return fmt.Errorf("%s: %w", taskName, err)
	}

	mc.l.Info("配置任务完成",
		zap.String("task", taskName),
		zap.Duration("duration", time.Since(startTime)),
	)
	return nil
}
