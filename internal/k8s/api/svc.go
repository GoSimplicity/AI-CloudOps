package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sSvcHandler struct {
	l          *zap.Logger
	svcService admin.SvcService
}

func NewK8sSvcHandler(l *zap.Logger, svcService admin.SvcService) *K8sSvcHandler {
	return &K8sSvcHandler{
		l:          l,
		svcService: svcService,
	}
}

// RegisterRouters 注册所有 Kubernetes 相关的路由
func (k *K8sSvcHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// Service 相关路由
	services := k8sGroup.Group("/services")
	{
		services.GET("/:id", k.GetServiceListByNamespace)       // 根据命名空间获取 Service 列表
		services.GET("/:id/:svcName/yaml", k.GetServiceYaml)    // 获取指定 Service 的 YAML 配置
		services.POST("/create", k.CreateService)               // 创建或更新 Service
		services.POST("/update", k.UpdateService)               // 更新指定 Name 的 Service
		services.DELETE("/batch_delete", k.BatchDeleteServices) // 批量删除 Service
	}
}

// GetServiceListByNamespace 根据命名空间获取 Service 列表
func (k *K8sSvcHandler) GetServiceListByNamespace(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.svcService.GetServicesByNamespace(ctx, id, namespace)
	})
}

// GetServiceYaml 获取 Service 的 YAML 配置
func (k *K8sSvcHandler) GetServiceYaml(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	svcName := ctx.Param("svcName")
	if svcName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'serviceName' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.svcService.GetServiceYaml(ctx, id, namespace, svcName)
	})
}

// CreateService 创建或更新 Service
func (k *K8sSvcHandler) CreateService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.CreateOrUpdateService(ctx, &req)
	})
}

// UpdateService 更新指定 Name 的 Service
func (k *K8sSvcHandler) UpdateService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.CreateOrUpdateService(ctx, &req)
	})
}

// BatchDeleteServices 批量删除 Service
func (k *K8sSvcHandler) BatchDeleteServices(ctx *gin.Context) {
	var req model.K8sServiceRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.svcService.DeleteService(ctx, req.ClusterId, req.Namespace, req.ServiceNames)
	})
}
