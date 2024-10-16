package di

import (
	webhookApi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, webHookHdl *webhookApi.WebHookHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	webHookHdl.RegisterRouters(server)
	return server
}
