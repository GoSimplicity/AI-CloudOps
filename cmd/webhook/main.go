package main

import (
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/config"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	Init()
}

func Init() {
	// 初始化配置
	config.InitViper()

	sp := viper.GetString("webhook.port")
	server := gin.Default()
	server.GET("/headers", printHeaders)

	// 启动 Web 服务器
	if err := server.Run(":" + sp); err != nil {
		zap.L().Fatal("Failed to start web server", zap.Error(err))
	}

}

// printHeaders 打印请求头信息
func printHeaders(c *gin.Context) {
	headers := c.Request.Header
	for key, values := range headers {
		for _, value := range values {
			c.String(http.StatusOK, "%s: %s\n", key, value)
		}
	}
}
