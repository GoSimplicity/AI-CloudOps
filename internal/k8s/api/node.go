package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sNodeHandler struct {
	logger      *zap.Logger
	nodeService admin.NodeService
}

func NewK8sNodeHandler(logger *zap.Logger, nodeService admin.NodeService) *K8sNodeHandler {
	return &K8sNodeHandler{
		nodeService: nodeService,
		logger:      logger,
	}
}

func (k *K8sNodeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	nodes := k8sGroup.Group("/nodes")
	{
		nodes.GET("/list/:id", k.GetNodeList)              // 获取节点列表
		nodes.GET("/:name", k.GetNodeDetail)               // 获取指定节点详情
		nodes.POST("/labels/add", k.AddLabelNodes)         // 添加节点标签
		nodes.DELETE("/labels/delete", k.DeleteLabelNodes) // 删除节点标签
	}
}

// GetNodeList 获取节点列表
func (k *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	clusterID, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeService.ListNodeByClusterName(ctx, clusterID)
	})
}

// GetNodeDetail 获取指定名称的节点详情
func (k *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
	name, err := apiresponse.GetParamName(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	id, err := apiresponse.GetQueryID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeService.GetNodeDetail(ctx, id, name)
	})
}

// AddLabelNodes 为节点添加标签
func (k *K8sNodeHandler) AddLabelNodes(ctx *gin.Context) {
	var req model.LabelK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.nodeService.AddOrUpdateNodeLabel(ctx, &req)
	})
}

// DeleteLabelNodes 删除节点标签
func (k *K8sNodeHandler) DeleteLabelNodes(ctx *gin.Context) {
	var req model.LabelK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.nodeService.AddOrUpdateNodeLabel(ctx, &req)
	})
}
