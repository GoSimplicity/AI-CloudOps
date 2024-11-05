package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sConfigMapHandler struct {
	configmapService admin.ConfigMapService
	l                *zap.Logger
}

func NewK8sConfigMapHandler(l *zap.Logger, configmapService admin.ConfigMapService) *K8sConfigMapHandler {
	return &K8sConfigMapHandler{
		l:                l,
		configmapService: configmapService,
	}
}

func (k *K8sConfigMapHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// ConfigMap 相关路由
	configMaps := k8sGroup.Group("/configmaps")
	{
		configMaps.GET("/", k.GetConfigMapListByNamespace)   // 根据命名空间获取 ConfigMap 列表
		configMaps.POST("/", k.CreateConfigMap)              // 创建新的 ConfigMap
		configMaps.PUT("/:name", k.UpdateConfigMap)          // 更新指定 Name 的 ConfigMap
		configMaps.PUT("/:name/data", k.UpdateConfigMapData) // 更新指定 Name 的 ConfigMap 数据
		configMaps.GET("/:name/yaml", k.GetConfigMapYaml)    // 获取指定 ConfigMap 的 YAML 配置
		configMaps.DELETE("/:name", k.DeleteConfigMap)       // 删除指定 Name 的 ConfigMap
		configMaps.DELETE("/", k.BatchDeleteConfigMaps)      // 批量删除 ConfigMap
	}
}

// GetConfigMapListByNamespace 根据命名空间获取 ConfigMap 列表
func (k *K8sConfigMapHandler) GetConfigMapListByNamespace(ctx *gin.Context) {
	namespace := ctx.Query("namespace")

	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	clusterName := ctx.Query("cluster_name")
	if clusterName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_name' 参数")
		return
	}

	configMaps, err := k.configmapService.GetConfigMapsByNamespace(ctx, clusterName, namespace)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, configMaps)
}

// CreateConfigMap 创建新的 ConfigMap
func (k *K8sConfigMapHandler) CreateConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.configmapService.CreateConfigMap(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateConfigMap 更新指定 Name 的 ConfigMap
func (k *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.configmapService.UpdateConfigMap(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateConfigMapData 更新指定 Name 的 ConfigMap 数据
func (k *K8sConfigMapHandler) UpdateConfigMapData(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.configmapService.UpdateConfigMapData(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetConfigMapYaml 获取 ConfigMap 的 YAML 配置
func (k *K8sConfigMapHandler) GetConfigMapYaml(ctx *gin.Context) {
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

	configMapName := ctx.Param("name")
	if configMapName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'configMapName' 参数")
		return
	}

	configMap, err := k.configmapService.GetConfigMapYaml(ctx, clusterName, namespace, configMapName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, configMap)
}

// DeleteConfigMap 删除指定 Name 的 ConfigMap
func (k *K8sConfigMapHandler) DeleteConfigMap(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.configmapService.DeleteConfigMap(ctx, req.ClusterName, req.Namespace, req.ConfigMapNames); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchDeleteConfigMaps 批量删除 ConfigMap
func (k *K8sConfigMapHandler) BatchDeleteConfigMaps(ctx *gin.Context) {
	var req model.K8sConfigMapRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.configmapService.DeleteConfigMap(ctx, req.ClusterName, req.Namespace, req.ConfigMapNames); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
