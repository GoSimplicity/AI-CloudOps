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
func InitGinServer(
	m []gin.HandlerFunc,
	userHdl *userApi.UserHandler,
	authHdl *authApi.AuthHandler,
	treeHdl *treeApi.TreeHandler,
	k8sClusterHdl *k8sApi.K8sClusterHandler,
	k8sConfigMapHdl *k8sApi.K8sConfigMapHandler,
	k8sDeploymentHdl *k8sApi.K8sDeploymentHandler,
	k8sNamespaceHdl *k8sApi.K8sNamespaceHandler,
	k8sNodeHdl *k8sApi.K8sNodeHandler,
	k8sPodHdl *k8sApi.K8sPodHandler,
	k8sSvcHdl *k8sApi.K8sSvcHandler,
	k8sTaintHdl *k8sApi.K8sTaintHandler,
	k8sYamlTaskHdl *k8sApi.K8sYamlTaskHandler,
	k8sYamlTemplateHdl *k8sApi.K8sYamlTemplateHandler,
	k8sAppHdl *k8sApi.K8sAppHandler,
	promHdl *prometheusApi.PrometheusHandler,
	notAuthHdl *notAuthHandler.NotAuthHandler,
) *gin.Engine {
	server := gin.Default()
	server.Use(m...)
	userHdl.RegisterRoutes(server)
	authHdl.RegisterRouters(server)
	treeHdl.RegisterRouters(server)
	notAuthHdl.RegisterRouters(server)
	promHdl.RegisterRouters(server)
	k8sClusterHdl.RegisterRouters(server)
	k8sAppHdl.RegisterRouters(server)
	k8sConfigMapHdl.RegisterRouters(server)
	k8sDeploymentHdl.RegisterRouters(server)
	k8sNamespaceHdl.RegisterRouters(server)
	k8sNodeHdl.RegisterRouters(server)
	k8sPodHdl.RegisterRouters(server)
	k8sSvcHdl.RegisterRouters(server)
	k8sTaintHdl.RegisterRouters(server)
	k8sYamlTaskHdl.RegisterRouters(server)
	k8sYamlTemplateHdl.RegisterRouters(server)

	return server
}
