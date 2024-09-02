package main

import (
	"github.com/GoSimplicity/CloudOps/config"
	"github.com/GoSimplicity/CloudOps/internal/di"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	Init()
}

func Init() {
	// 初始化配置
	config.InitViper()
	// 初始化 Web 服务器和其他组件
	server := di.InitWebServer()
	// 设置请求头打印路由
	server.GET("/headers", printHeaders)

	sp := viper.GetString("server.port")

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
