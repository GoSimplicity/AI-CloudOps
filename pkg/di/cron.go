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

package di

import (
	"context"
	"time"

	cn "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// InitAndRefreshK8sClient 初始化并启动定时刷新任务
// 返回 cron 调度器实例以便调用者可以在需要时停止它
func InitAndRefreshK8sClient(K8sClient client.K8sClient, logger *zap.Logger, PromCache cache.MonitorCache, manager cn.CronManager) *cron.Cron {
	stdLogger := zap.NewStdLog(logger) // 将 zap 日志转换为标准库日志

	// 启用秒级调度，并集成日志记录和恢复中间件
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cron.VerbosePrintfLogger(stdLogger)),
		cron.WithChain(
			cron.Recover(cron.VerbosePrintfLogger(stdLogger)),
		),
	)

	// 执行初始化任务
	initTasks := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Kubernetes 客户端", K8sClient.RefreshClients},
		{"Prometheus 缓存", PromCache.MonitorCacheManager},
	}

	for _, task := range initTasks {
		go func(t struct {
			name string
			fn   func(context.Context) error
		}) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			logger.Info("开始初始刷新 " + t.name)
			if err := t.fn(ctx); err != nil {
				logger.Error("初始刷新"+t.name+"失败", zap.Error(err))
			} else {
				logger.Info("成功完成初始刷新 " + t.name)
			}
		}(task)
	}

	// 启动值班历史记录填充任务
	go func() {
		ctx := context.Background()
		if err := manager.StartOnDutyHistoryManager(ctx); err != nil {
			logger.Error("启动值班历史记录填充任务失败", zap.Error(err))
		} else {
			logger.Info("成功启动值班历史记录填充任务")
		}
	}()

	// 配置定时任务
	cronConfigs := []struct {
		name      string
		cronExpr  string
		configKey string
		fn        func(context.Context) error
	}{
		{
			name:      "Kubernetes 客户端",
			cronExpr:  viper.GetString("k8s.refresh_cron"),
			configKey: "k8s.refresh_cron",
			fn:        K8sClient.RefreshClients,
		},
		{
			name:      "Prometheus 缓存",
			cronExpr:  viper.GetString("prometheus.refresh_cron"),
			configKey: "prometheus.refresh_cron",
			fn:        PromCache.MonitorCacheManager,
		},
		{
			name:      "主机状态检查",
			cronExpr:  viper.GetString("tree.check_status_cron"),
			configKey: "tree.check_status_cron",
			fn:        manager.StartCheckHostStatusManager,
		},
	}

	for _, config := range cronConfigs {
		if config.cronExpr == "" {
			logger.Warn("未配置"+config.name+"刷新 cron 表达式", zap.String("configKey", config.configKey))
			continue
		}

		_, err := c.AddFunc(config.cronExpr, func(name string, fn func(context.Context) error) func() {
			return func() {
				taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				logger.Info("开始定时刷新 " + name)
				if err := fn(taskCtx); err != nil {
					logger.Error("定时刷新"+name+"失败", zap.Error(err))
				} else {
					logger.Info("成功刷新 " + name)
				}
			}
		}(config.name, config.fn))

		if err != nil {
			logger.Error("添加"+config.name+"定时刷新任务失败", zap.Error(err))
		}
	}

	return c
}
