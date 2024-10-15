// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/cache"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/consumer"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/content"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
)

// Injectors from wire.go:

func InitWebServer() *Cmd {
	logger := InitLogger()
	v := InitMiddlewares(logger)
	db := InitDB()
	webhookDao := dao.NewWebhookDao(logger, db)
	webHookHandler := api.NewWebHookHandler(logger, webhookDao)
	engine := InitGinServer(v, webHookHandler)
	webhookRobot := robot.NewWebhookRobot(logger)
	webhookCache := cache.NewWebhookCache(logger, webhookDao, webhookRobot)
	webhookContent := content.NewWebhookContent(logger, webhookDao, webhookRobot)
	webhookConsumer := consumer.NewWebhookConsumer(logger, webhookCache, webhookDao, webhookContent)
	v2 := InitWebHookCache(logger, webhookCache, webhookConsumer)
	cmd := &Cmd{
		Server: engine,
		Start:  v2,
	}
	return cmd
}
