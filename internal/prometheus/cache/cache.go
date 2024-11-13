package cache

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

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MonitorCache interface {
	// MonitorCacheManager 更新缓存
	MonitorCacheManager(ctx context.Context) error
}

type monitorCache struct {
	PrometheusMainConfig  PromConfigCache
	AlertMangerMainConfig AlertConfigCache
	AlertRuleConfig       RuleConfigCache
	AlertRecordConfig     RecordConfigCache
	l                     *zap.Logger
}

func NewMonitorCache(PrometheusMainConfig PromConfigCache, AlertMangerMainConfig AlertConfigCache, AlertRuleConfig RuleConfigCache, AlertRecordConfig RecordConfigCache, l *zap.Logger) MonitorCache {
	return &monitorCache{
		PrometheusMainConfig:  PrometheusMainConfig,
		AlertMangerMainConfig: AlertMangerMainConfig,
		AlertRuleConfig:       AlertRuleConfig,
		AlertRecordConfig:     AlertRecordConfig,
		l:                     l,
	}
}

// MonitorCacheManager 定期更新缓存并监听退出信号
func (mc *monitorCache) MonitorCacheManager(ctx context.Context) error {
	mc.l.Info("开始更新所有监控缓存配置")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(4)

	// 创建一个通道来收集错误
	errChan := make(chan error, 4)

	// 定义一个辅助函数来执行任务
	executeTask := func(taskName string, taskFunc func(context.Context) error) {
		defer wg.Done()
		mc.l.Info(fmt.Sprintf("开始执行任务: %s", taskName))
		if err := taskFunc(ctx); err != nil {
			mc.l.Error(fmt.Sprintf("任务 %s 失败", taskName), zap.Error(err))
			errChan <- fmt.Errorf("%s: %w", taskName, err)
			return
		}
		mc.l.Info(fmt.Sprintf("任务 %s 成功完成", taskName))
	}

	// 并发执行各个配置生成任务
	go executeTask("生成 Prometheus 主配置", mc.PrometheusMainConfig.GeneratePrometheusMainConfig)
	go executeTask("生成 AlertManager 主配置", mc.AlertMangerMainConfig.GenerateAlertManagerMainConfig)
	go executeTask("生成 Prometheus 告警规则配置", mc.AlertRuleConfig.GenerateAlertRuleConfigYaml)
	go executeTask("生成 Prometheus 预聚合规则配置", mc.AlertRecordConfig.GenerateRecordRuleConfigYaml)

	wg.Wait()
	close(errChan)

	// 收集所有错误
	var aggregatedErrors []error
	for err := range errChan {
		aggregatedErrors = append(aggregatedErrors, err)
	}

	if len(aggregatedErrors) > 0 {
		mc.l.Warn("部分任务执行失败，详情请查看日志")
		return fmt.Errorf("部分任务执行失败: %v", aggregatedErrors)
	}

	mc.l.Info("所有监控缓存配置更新完成")
	return nil
}
