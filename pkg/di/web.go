package di

import (
	userApi "github.com/GoSimplicity/CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, userHdl *userApi.UserHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	return server
}
