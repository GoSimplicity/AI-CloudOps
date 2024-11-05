package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sDeploymentHandler struct {
	l                 *zap.Logger
	deploymentService admin.DeploymentService
}

func NewK8sDeploymentHandler(l *zap.Logger, deploymentService admin.DeploymentService) *K8sDeploymentHandler {
	return &K8sDeploymentHandler{
		l:                 l,
		deploymentService: deploymentService,
	}
}

func (k *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// Deployment 相关路由
	deployments := k8sGroup.Group("/deployments")
	{
		deployments.GET("/", k.GetDeployListByNamespace) // 根据命名空间获取部署列表
		deployments.POST("/", k.CreateDeployment)        // 创建新的部署
		deployments.PUT("/:name", k.UpdateDeployment)    // 更新指定 deploymentName 的部署
		deployments.DELETE("/:name", k.DeleteDeployment) // 删除指定 deploymentName 的部署

		deployments.PUT("/:name/image", k.SetDeploymentContainerImage) // 设置部署中容器的镜像
		deployments.POST("/:name/scale", k.ScaleDeployment)            // 扩缩指定 ID 的部署
		deployments.POST("/restart", k.BatchRestartDeployments)        // 批量重启部署
		deployments.GET("/:name/yaml", k.GetDeployYaml)                // 获取指定部署的 YAML 配置
	}
}

// GetDeployListByNamespace 根据命名空间获取部署列表
func (k *K8sDeploymentHandler) GetDeployListByNamespace(ctx *gin.Context) {
	namesapce := ctx.Query("namespace")
	if namesapce == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	clusterName := ctx.Query("cluster_name")
	if clusterName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_name' 参数")
		return
	}

	deploys, err := k.deploymentService.GetDeploymentsByNamespace(ctx, clusterName, namesapce)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, deploys)
}

// CreateDeployment 创建新的部署
func (k *K8sDeploymentHandler) CreateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.CreateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateDeployment 更新指定 Name 的部署
func (k *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteDeployment 删除指定 Name 的部署
func (k *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.DeleteDeployment(ctx, req.ClusterName, req.Namespace, req.DeploymentNames[0]); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// SetDeploymentContainerImage 设置部署中容器的镜像
func (k *K8sDeploymentHandler) SetDeploymentContainerImage(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// ScaleDeployment 扩缩部署
func (k *K8sDeploymentHandler) ScaleDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchRestartDeployments 批量重启部署
func (k *K8sDeploymentHandler) BatchRestartDeployments(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.deploymentService.BatchRestartDeployments(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetDeployYaml 获取部署的 YAML 配置
func (k *K8sDeploymentHandler) GetDeployYaml(ctx *gin.Context) {
	// TODO: 实现获取部署的 YAML 配置的逻辑
}
