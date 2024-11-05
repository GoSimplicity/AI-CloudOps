package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNamespaceHandler struct {
	l                *zap.Logger
	namespaceService admin.NamespaceService
}

func NewK8sNamespaceHandler(l *zap.Logger, namespaceService admin.NamespaceService) *K8sNamespaceHandler {
	return &K8sNamespaceHandler{
		l:                l,
		namespaceService: namespaceService,
	}
}

func (k *K8sNamespaceHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 命名空间相关路由
	namespaces := k8sGroup.Group("/namespaces")
	{
		namespaces.GET("/cascade", k.GetClusterNamespacesForCascade) // 获取级联选择的命名空间列表
		namespaces.GET("/select", k.GetClusterNamespacesForSelect)   // 获取用于选择的命名空间列表
	}
}

// GetClusterNamespacesForCascade 获取级联选择的命名空间列表
func (k *K8sNamespaceHandler) GetClusterNamespacesForCascade(ctx *gin.Context) {
	namespaces, err := k.namespaceService.GetClusterNamespacesList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}

// GetClusterNamespacesForSelect 获取用于选择的命名空间列表
func (k *K8sNamespaceHandler) GetClusterNamespacesForSelect(ctx *gin.Context) {
	namespace := ctx.Query("namespace")

	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	namespaces, err := k.namespaceService.GetClusterNamespacesByName(ctx, namespace)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}
