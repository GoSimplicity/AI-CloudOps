package di

import (
	authApi "github.com/GoSimplicity/CloudOps/internal/auth/api"
	userApi "github.com/GoSimplicity/CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, userHdl *userApi.UserHandler, auth *authApi.AuthHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	auth.RegisterRouters(server)
	return server
}
