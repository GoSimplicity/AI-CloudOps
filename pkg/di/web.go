package di

import (
	authApi "github.com/GoSimplicity/AI-CloudOps/internal/auth/api"
	k8sApi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	prometheusApi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	treeApi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	userApi "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, userHdl *userApi.UserHandler, authHdl *authApi.AuthHandler, treeHdl *treeApi.TreeHandler, k8sHdl *k8sApi.K8sHandler, promHdl *prometheusApi.PrometheusHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	authHdl.RegisterRouters(server)
	treeHdl.RegisterRouters(server)
	k8sHdl.RegisterRouters(server)
	promHdl.RegisterRouters(server)

	return server
}
