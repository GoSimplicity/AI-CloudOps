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

func (k *K8sClusterHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 集群相关路由
	clusters := k8sGroup.Group("/clusters")
	{
		clusters.GET("/", k.GetClusterList)                // 获取集群列表
		clusters.GET("/select", k.GetClusterListForSelect) // 获取用于选择的集群列表
		clusters.POST("/", k.CreateCluster)                // 创建新的集群
		clusters.PUT("/:id", k.UpdateCluster)              // 更新指定 ID 的集群
		clusters.DELETE("/:id", k.DeleteCluster)           // 删除指定 ID 的集群
	}
}

// GetClusterList 获取集群列表
func (k *K8sClusterHandler) GetClusterList(ctx *gin.Context) {
	clusters, err := k.clusterService.ListAllClusters(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, clusters)
}

// GetClusterListForSelect 获取用于选择的集群列表
func (k *K8sClusterHandler) GetClusterListForSelect(ctx *gin.Context) {
	// TODO: 实现获取用于选择的集群列表的逻辑
}

// CreateCluster 创建新的集群
func (k *K8sClusterHandler) CreateCluster(ctx *gin.Context) {
	var cluster *model.K8sCluster

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	err := ctx.ShouldBind(&cluster)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	cluster.UserID = uc.Uid

	err = k.clusterService.CreateCluster(ctx, cluster)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateCluster 更新指定 ID 的集群
func (k *K8sClusterHandler) UpdateCluster(ctx *gin.Context) {
	// TODO: 实现更新集群的逻辑
}

// DeleteCluster 删除指定 ID 的集群
func (k *K8sClusterHandler) DeleteCluster(ctx *gin.Context) {
	// TODO: 实现删除集群的逻辑
}
