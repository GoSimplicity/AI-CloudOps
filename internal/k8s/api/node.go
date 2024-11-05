package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNodeHandler struct {
	l           *zap.Logger
	nodeService admin.NodeService
}

func NewK8sNodeHandler(l *zap.Logger, nodeService admin.NodeService) *K8sNodeHandler {
	return &K8sNodeHandler{
		nodeService: nodeService,
		l:           l,
	}
}

func (k *K8sNodeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 节点相关路由
	nodes := k8sGroup.Group("/nodes")
	{
		nodes.GET("/", k.GetNodeList)                      // 获取节点列表
		nodes.GET("/:name", k.GetNodeDetail)               // 获取指定名称的节点详情
		nodes.POST("/labels/add", k.AddLabelNodes)         // 为节点添加标签
		nodes.DELETE("/labels/delete", k.DeleteLabelNodes) // 删除节点标签
	}
}

// GetNodeList 获取节点列表
func (k *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	nodes, err := k.nodeService.ListNodeByClusterId(ctx, clusterID)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

// GetNodeDetail 获取指定名称的节点详情
func (k *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	name := ctx.Param("name")
	if name == "" {
		apiresponse.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	node, err := k.nodeService.GetNodeByName(ctx, clusterID, name)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, node)
}

// AddLabelNodes 为节点添加标签
func (k *K8sNodeHandler) AddLabelNodes(ctx *gin.Context) {
	var label *model.LabelK8sNodesRequest

	err := ctx.ShouldBind(&label)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.nodeService.UpdateNodeLabel(ctx, label); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// DeleteLabelNodes 删除节点标签
func (k *K8sNodeHandler) DeleteLabelNodes(ctx *gin.Context) {
	var label *model.LabelK8sNodesRequest

	err := ctx.ShouldBind(&label)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.nodeService.UpdateNodeLabel(ctx, label); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}
