package api

import (
	"strconv"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/tree/service"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TreeHandler struct {
	service service.TreeService
	l       *zap.Logger
}

func NewTreeHandler(service service.TreeService, l *zap.Logger) *TreeHandler {
	return &TreeHandler{
		service: service,
		l:       l,
	}
}

func (t *TreeHandler) RegisterRouters(server *gin.Engine) {
	treeGroup := server.Group("/api/tree")

	// 树节点相关路由
	treeGroup.GET("/listTreeNode", t.ListTreeNode)
	treeGroup.GET("/selectTreeNode", t.SelectTreeNode)
	treeGroup.GET("/getTopTreeNode", t.GetTopTreeNode)
	treeGroup.GET("/listLeafTreeNode", t.ListLeafTreeNodes)
	treeGroup.POST("/createTreeNode", t.CreateTreeNode)
	treeGroup.DELETE("/deleteTreeNode/:id", t.DeleteTreeNode)
	treeGroup.GET("/getChildrenTreeNode/:pid", t.GetChildrenTreeNode)
	treeGroup.POST("/updateTreeNode", t.UpdateTreeNode)

	// ECS, ELB, RDS 资源相关路由
	treeGroup.GET("/getEcsUnbindList", t.GetEcsUnbindList)
	treeGroup.GET("/getEcsList", t.GetEcsList)
	treeGroup.GET("/getElbUnbindList", t.GetElbUnbindList)
	treeGroup.GET("/getElbList", t.GetElbList)
	treeGroup.GET("/getRdsUnbindList", t.GetRdsUnbindList)
	treeGroup.GET("/getRdsList", t.GetRdsList)
	treeGroup.GET("/getAllResource", t.GetAllResource)

	// 资源绑定相关路由
	treeGroup.POST("/bindEcs", t.BindEcs)
	treeGroup.POST("/bindElb", t.BindElb)
	treeGroup.POST("/bindRds", t.BindRds)
	treeGroup.POST("/unBindEcs", t.UnBindEcs)
	treeGroup.POST("/unBindElb", t.UnBindElb)
	treeGroup.POST("/unBindRds", t.UnBindRds)
}

func (t *TreeHandler) ListTreeNode(ctx *gin.Context) {
	list, err := t.service.ListTreeNodes(ctx)

	if err != nil {
		t.l.Error("list tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) SelectTreeNode(ctx *gin.Context) {
	// 获取查询参数 "level" 和 "levelLt"，并设置默认值为 "0"
	levelStr := ctx.DefaultQuery("level", "0")
	levelLtStr := ctx.DefaultQuery("levelLt", "0")

	// 将字符串参数转换为整数，并处理转换错误
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		t.l.Warn("无效的 level 参数", zap.String("level", levelStr), zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "无效的 level 参数")
		return
	}

	levelLt, err := strconv.Atoi(levelLtStr)
	if err != nil {
		t.l.Warn("无效的 levelLt 参数", zap.String("levelLt", levelLtStr), zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "无效的 levelLt 参数")
		return
	}

	// 调用服务层方法获取过滤后的树节点
	nodes, err := t.service.SelectTreeNode(ctx, level, levelLt)
	if err != nil {
		t.l.Error("SelectTreeNode 调用失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	// 返回成功响应，包含过滤后的树节点
	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeHandler) GetTopTreeNode(ctx *gin.Context) {
	nodes, err := t.service.GetTopTreeNode(ctx)
	if err != nil {
		t.l.Error("get top tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

func (t *TreeHandler) ListLeafTreeNodes(ctx *gin.Context) {
	list, err := t.service.ListLeafTreeNodes(ctx)
	if err != nil {
		t.l.Error("get all tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) CreateTreeNode(ctx *gin.Context) {
	var req model.TreeNode

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.CreateTreeNode(ctx, &req); err != nil {
		t.l.Error("create tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) DeleteTreeNode(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.ErrorWithMessage(ctx, "id不能为空")
		return
	}

	nodeId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "id必须为整数")
		return
	}

	if err := t.service.DeleteTreeNode(ctx, nodeId); err != nil {
		t.l.Error("delete tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) GetChildrenTreeNode(ctx *gin.Context) {
	pid := ctx.Param("pid")
	if pid == "" {
		apiresponse.ErrorWithMessage(ctx, "pid不能为空")
		return
	}

	parentId, err := strconv.Atoi(pid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "pid必须为整数")
		return
	}

	list, err := t.service.GetChildrenTreeNodes(ctx, parentId)
	if err != nil {
		t.l.Error("get children tree nodes failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (t *TreeHandler) UpdateTreeNode(ctx *gin.Context) {
	var req model.TreeNode
	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.UpdateTreeNode(ctx, &req); err != nil {
		t.l.Error("update tree node failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) GetEcsUnbindList(ctx *gin.Context) {
	// TODO: Implement GetEcsUnbindList logic
}

func (t *TreeHandler) GetEcsList(ctx *gin.Context) {
	// TODO: Implement GetEcsList logic
}

func (t *TreeHandler) GetElbUnbindList(ctx *gin.Context) {
	// TODO: Implement GetElbUnbindList logic
}

func (t *TreeHandler) GetElbList(ctx *gin.Context) {
	// TODO: Implement GetElbList logic
}

func (t *TreeHandler) GetRdsUnbindList(ctx *gin.Context) {
	// TODO: Implement GetRdsUnbindList logic
}

func (t *TreeHandler) GetRdsList(ctx *gin.Context) {
	// TODO: Implement GetRdsList logic
}

func (t *TreeHandler) GetAllResource(ctx *gin.Context) {
	// TODO: Implement GetAllResource logic
}

func (t *TreeHandler) BindEcs(ctx *gin.Context) {
	// TODO: Implement BindEcs logic
}

func (t *TreeHandler) BindElb(ctx *gin.Context) {
	// TODO: Implement BindElb logic
}

func (t *TreeHandler) BindRds(ctx *gin.Context) {
	// TODO: Implement BindRds logic
}

func (t *TreeHandler) UnBindEcs(ctx *gin.Context) {
	// TODO: Implement UnBindEcs logic
}

func (t *TreeHandler) UnBindElb(ctx *gin.Context) {
	// TODO: Implement UnBindElb logic
}

func (t *TreeHandler) UnBindRds(ctx *gin.Context) {
	// TODO: Implement UnBindRds logic
}
