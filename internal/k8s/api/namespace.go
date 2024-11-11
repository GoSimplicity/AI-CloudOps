package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNamespaceHandler struct {
	logger           *zap.Logger
	namespaceService admin.NamespaceService
}

func NewK8sNamespaceHandler(logger *zap.Logger, namespaceService admin.NamespaceService) *K8sNamespaceHandler {
	return &K8sNamespaceHandler{
		logger:           logger,
		namespaceService: namespaceService,
	}
}

func (k *K8sNamespaceHandler) RegisterRouters(router *gin.Engine) {
	k8sGroup := router.Group("/api/k8s")

	// 命名空间相关路由组
	namespaces := k8sGroup.Group("/namespaces")
	{
		namespaces.GET("/cascade", k.GetClusterNamespacesForCascade)   // 获取级联选择的命名空间列表
		namespaces.GET("/select/:id", k.GetClusterNamespacesForSelect) // 获取用于选择的命名空间列表
	}
}

// GetClusterNamespacesForCascade 获取级联选择的命名空间列表
func (k *K8sNamespaceHandler) GetClusterNamespacesForCascade(ctx *gin.Context) {
	namespaces, err := k.namespaceService.GetClusterNamespacesList(ctx)
	if err != nil {
		k.logger.Error("Failed to get cascade namespaces", zap.Error(err))
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}

// GetClusterNamespacesForSelect 获取用于选择的命名空间列表
func (k *K8sNamespaceHandler) GetClusterNamespacesForSelect(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	namespaces, err := k.namespaceService.GetClusterNamespacesById(ctx, id)
	if err != nil {
		k.logger.Error("Failed to get namespaces for select", zap.Strings("namespace", namespaces), zap.Error(err))
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}
