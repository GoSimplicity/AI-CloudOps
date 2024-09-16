package di

import (
	authApi "github.com/GoSimplicity/CloudOps/internal/auth/api"
	treeApi "github.com/GoSimplicity/CloudOps/internal/tree/api"
	userApi "github.com/GoSimplicity/CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, userHdl *userApi.UserHandler, auth *authApi.AuthHandler, tree *treeApi.TreeHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	auth.RegisterRouters(server)
	tree.RegisterRouters(server)
	return server
}
