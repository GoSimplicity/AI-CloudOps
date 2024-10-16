package di

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/consumer"
	"go.uber.org/zap"
)

func InitWebHookCache(logger *zap.Logger, webHookCache cache.WebhookCache, webHookConsumer consumer.WebhookConsumer) func() {
	return func() {
		// 执行初始刷新 WebHookCache
		go func() {
			ctx := context.Background() // 使用持久上下文

			logger.Info("开始初始刷新 WebHook Cache")
			if err := webHookCache.RenewAllCaches(ctx); err != nil {
				logger.Error("WebHook Cache 刷新失败", zap.Error(err))
			} else {
				logger.Info("WebHook Cache 刷新成功")
			}
		}()

		// 执行初始刷新 WebHookConsumer
		go func() {
			ctx := context.Background() // 使用持久上下文

			logger.Info("开始初始刷新 WebHook Consumer")
			if err := webHookConsumer.AlertReceiveConsumerManager(ctx); err != nil {
				logger.Error("WebHook Consumer 刷新失败", zap.Error(err))
			} else {
				logger.Info("WebHook Consumer 刷新成功")
			}
		}()
	}
}
