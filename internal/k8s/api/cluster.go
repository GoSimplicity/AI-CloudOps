package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

type K8sClusterHandler struct {
	clusterService admin.ClusterService
	l              *zap.Logger
}

func NewK8sClusterHandler(l *zap.Logger, clusterService admin.ClusterService) *K8sClusterHandler {
	return &K8sClusterHandler{
		l:              l,
		clusterService: clusterService,
	}
}

// RegisterRouters 注册集群相关的路由
func (k *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 集群相关路由
	clusters := k8sGroup.Group("/clusters")
	{
		clusters.GET("/list", k.GetAllClusters)                 // 获取集群列表
		clusters.GET("/:id", k.GetCluster)                      // 获取指定集群
		clusters.POST("/create", k.CreateCluster)               // 创建新的集群
		clusters.POST("/update", k.UpdateCluster)               // 更新指定 ID 的集群
		clusters.DELETE("/delete/:id", k.DeleteCluster)         // 删除指定 ID 的集群
		clusters.DELETE("/batch_delete", k.BatchDeleteClusters) // 批量删除集群
	}
}

// GetAllClusters 获取集群列表
func (k *K8sClusterHandler) GetAllClusters(ctx *gin.Context) {
	clusters, err := k.clusterService.ListAllClusters(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, clusters)
}

// GetCluster 获取指定 ID 的集群详情
func (k *K8sClusterHandler) GetCluster(ctx *gin.Context) {
	id := ctx.Param("id")

	if id == "" {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	cluster, err := k.clusterService.GetClusterByID(ctx, intId)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, cluster)
}

// CreateCluster 创建新的集群
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var cluster *model.K8sCluster

	if err := ctx.ShouldBind(&cluster); err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims) // 获取用户信息

	cluster.UserID = uc.Uid

	// 调用服务层方法创建集群
	if err := k.clusterService.CreateCluster(ctx, cluster); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateCluster 更新指定 ID 的集群
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var cluster *model.K8sCluster

	if err := ctx.ShouldBind(&cluster); err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if cluster.ID == 0 {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	// 调用服务层方法更新集群
	if err := k.clusterService.UpdateCluster(ctx, cluster); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteCluster 删除指定 ID 的集群
func (k *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	id := ctx.Param("id")

	if id == "" {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	intId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	// 调用服务层方法删除集群
	if err := k.clusterService.DeleteCluster(ctx, intId); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (k *K8sClusterHandler) BatchDeleteClusters(ctx *gin.Context) {
	var req model.BatchDeleteReq

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if len(req.IDs) == 0 {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	// 调用服务层方法批量删除集群
	if err := k.clusterService.BatchDeleteClusters(ctx, req.IDs); err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
