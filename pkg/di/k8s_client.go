package di

import (
	"context"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// InitAndRefreshK8sClient 初始化并启动定时刷新 Kubernetes 客户端
// 返回 cron 调度器实例以便调用者可以在需要时停止它
func InitAndRefreshK8sClient(K8sClient client.K8sClient, logger *zap.Logger) *cron.Cron {
	stdLogger := zap.NewStdLog(logger)

	// 启用秒级调度，并集成日志记录和恢复中间件
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cron.VerbosePrintfLogger(stdLogger)), // 集成日志记录
		cron.WithChain(
			cron.Recover(cron.VerbosePrintfLogger(stdLogger)), // 添加恢复中间件，防止任务崩溃调度器
		),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // 确保超时或操作完成后释放资源

	// 初始刷新
	if err := K8sClient.RefreshClients(ctx); err != nil {
		logger.Error("InitAndRefreshK8sClient: 初始刷新 Kubernetes 客户端失败", zap.Error(err))
		return nil
	}

	// 添加 cron job，每5秒执行一次
	_, err := c.AddFunc("@every 5s", func() {
		taskCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := K8sClient.RefreshClients(taskCtx); err != nil {
			logger.Error("InitAndRefreshK8sClient: 定时刷新 Kubernetes 客户端失败", zap.Error(err))
		} else {
			logger.Info("InitAndRefreshK8sClient: 成功刷新 Kubernetes 客户端")
		}
	})

	if err != nil {
		logger.Error("InitAndRefreshK8sClient: 添加 cron job 失败", zap.Error(err))
		return nil
	}

	return c
}
