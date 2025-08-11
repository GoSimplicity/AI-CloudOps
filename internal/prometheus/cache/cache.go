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

	"github.com/spf13/viper"
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
	PrometheusMainConfig   PrometheusConfigCache
	AlertManagerMainConfig AlertManagerConfigCache
	AlertRuleConfig        AlertRuleConfigCache
	AlertRecordConfig      RecordRuleConfigCache
	l                      *zap.Logger
}

func NewMonitorCache(
	promConfig PrometheusConfigCache,
	alertManagerConfig AlertManagerConfigCache,
	alertRuleConfig AlertRuleConfigCache,
	alertRecordConfig RecordRuleConfigCache,
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

	g, ctx := errgroup.WithContext(ctx)

	// 任务定义与执行优化
	type taskDef struct {
		name string
		fn   func(context.Context) error
	}
	enableAlert := viper.GetInt("prometheus.enable_alert") == 1
	enableRecord := viper.GetInt("prometheus.enable_record") == 1

	tasks := make([]taskDef, 0, 4)
	// 主配置（Prometheus、AlertManager）始终可执行，因为其生成与开关无关
	tasks = append(tasks, taskDef{"Prometheus主配置", mc.PrometheusMainConfig.GenerateMainConfig})
	tasks = append(tasks, taskDef{"AlertManager主配置", mc.AlertManagerMainConfig.GenerateMainConfig})
	if enableAlert {
		tasks = append(tasks, taskDef{"告警规则配置", mc.AlertRuleConfig.GenerateMainConfig})
	} else {
		mc.l.Info("跳过告警规则配置生成：prometheus.enable_alert=0")
	}
	if enableRecord {
		tasks = append(tasks, taskDef{"预聚合规则配置", mc.AlertRecordConfig.GenerateMainConfig})
	} else {
		mc.l.Info("跳过预聚合规则配置生成：prometheus.enable_record=0")
	}

	for i := range tasks {
		task := tasks[i] // 避免闭包变量问题
		g.Go(func() error {
			return mc.executeTask(ctx, task.name, task.fn)
		})
	}

	if err := g.Wait(); err != nil {
		mc.l.Error("监控缓存更新失败",
			zap.Error(err),
			zap.Duration("timeout", DefaultTaskTimeout),
		)
		return fmt.Errorf("监控缓存更新失败: %w", err)
	}

	mc.l.Info("监控缓存配置全部更新成功")
	return nil
}

// executeTask 封装任务执行逻辑
func (mc *monitorCache) executeTask(ctx context.Context, taskName string, taskFn func(context.Context) error) error {
	startTime := time.Now()
	mc.l.Info("开始执行配置任务",
		zap.String("task", taskName),
		zap.Time("start_time", startTime),
	)

	if err := taskFn(ctx); err != nil {
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
