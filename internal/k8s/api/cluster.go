package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.ListAllClusters(ctx)
	})
}

// GetCluster 获取指定 ID 的集群详情
func (k *K8sClusterHandler) GetCluster(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.clusterService.GetClusterByID(ctx, id)
	})
}

// CreateCluster 创建新的集群
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	uc := ctx.MustGet("user").(ijwt.UserClaims) // 获取用户信息

	req.UserID = uc.Uid

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.CreateCluster(ctx, &req)
	})
}

// UpdateCluster 更新指定 ID 的集群
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	var req model.K8sCluster

	if req.ID == 0 {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.UpdateCluster(ctx, &req)
	})
}

// DeleteCluster 删除指定 ID 的集群
func (k *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.clusterService.DeleteCluster(ctx, id)
	})
}

func (k *K8sClusterHandler) BatchDeleteClusters(ctx *gin.Context) {
	var req model.BatchDeleteReq

	if len(req.IDs) == 0 {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.clusterService.BatchDeleteClusters(ctx, req.IDs)
	})
}
