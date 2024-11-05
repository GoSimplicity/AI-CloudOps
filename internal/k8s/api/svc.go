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
		services.GET("/", k.GetServiceListByNamespace) // 根据命名空间获取 Service 列表
		services.GET("/:name/yaml", k.GetServiceYaml)  // 获取指定 Service 的 YAML 配置
		services.POST("/", k.CreateOrUpdateService)    // 创建或更新 Service
		services.PUT("/:name", k.UpdateService)        // 更新指定 Name 的 Service
		services.DELETE("/:name", k.DeleteService)     // 删除指定 Name 的 Service
		services.DELETE("/", k.BatchDeleteServices)    // 批量删除 Service
	}
}

// GetServiceListByNamespace 根据命名空间获取 Service 列表
func (k *K8sSvcHandler) GetServiceListByNamespace(ctx *gin.Context) {
	clusterName := ctx.Query("cluster_name")
	if clusterName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	services, err := k.svcService.GetServicesByNamespace(ctx, clusterName, namespace)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, services)
}

// GetServiceYaml 获取 Service 的 YAML 配置
func (k *K8sSvcHandler) GetServiceYaml(ctx *gin.Context) {
	clusterName := ctx.Query("cluster_name")
	if clusterName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	serviceName := ctx.Param("name")
	if serviceName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'serviceName' 参数")
		return
	}

	service, err := k.svcService.GetServiceYaml(ctx, clusterName, namespace, serviceName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, service)
}

// CreateOrUpdateService 创建或更新 Service
func (k *K8sSvcHandler) CreateOrUpdateService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.svcService.CreateOrUpdateService(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateService 更新指定 Name 的 Service
func (k *K8sSvcHandler) UpdateService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.svcService.UpdateService(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteService 删除指定 Name 的 Service
func (k *K8sSvcHandler) DeleteService(ctx *gin.Context) {
	var req model.K8sServiceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.svcService.DeleteService(ctx, req.ClusterName, req.Namespace, req.ServiceNames); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchDeleteServices 批量删除 Service
func (k *K8sSvcHandler) BatchDeleteServices(ctx *gin.Context) {
	var req model.K8sServiceRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.svcService.DeleteService(ctx, req.ClusterName, req.Namespace, req.ServiceNames); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
