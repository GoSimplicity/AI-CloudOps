package di

import "github.com/gin-gonic/gin"

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	return server
}
