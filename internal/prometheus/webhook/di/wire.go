//go:build wireinject

package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/consumer"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/google/wire"
)

func InitWebServer() *Cmd {
	wire.Build(
		InitLogger,
		InitGinServer,
		InitMiddlewares,
		InitDB,
		InitWebHookCache,
		api.NewWebHookHandler,
		cache.NewWebhookCache,
		dao.NewWebhookDao,
		consumer.NewWebhookConsumer,
		content.NewWebhookContent,
		robot.NewWebhookRobot,
		wire.Struct(new(Cmd), "*"),
	)

	return new(Cmd)
}
