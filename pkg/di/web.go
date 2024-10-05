package di

import (
	k8sApi "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	notAuthHandler "github.com/GoSimplicity/AI-CloudOps/internal/not_auth/api"
	prometheusApi "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/api"
	authApi "github.com/GoSimplicity/AI-CloudOps/internal/system/api"
	treeApi "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	userApi "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	"github.com/gin-gonic/gin"
)

// InitGinServer 初始化web服务
func InitGinServer(m []gin.HandlerFunc, userHdl *userApi.UserHandler, authHdl *authApi.AuthHandler, treeHdl *treeApi.TreeHandler, k8sHdl *k8sApi.K8sHandler, promHdl *prometheusApi.PrometheusHandler, notAuthHdl *notAuthHandler.NotAuthHandler) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	authHdl.RegisterRouters(server)
	treeHdl.RegisterRouters(server)
	k8sHdl.RegisterRouters(server)
	promHdl.RegisterRouters(server)
	notAuthHdl.RegisterRouters(server)

	return server
}
