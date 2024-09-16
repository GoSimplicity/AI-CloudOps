package api

import (
	"github.com/GoSimplicity/CloudOps/internal/tree/service"
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
	treeGroup.GET("/getAllTreeNode", t.GetAllTreeNode)
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
	// TODO: Implement ListTreeNode logic
}

func (t *TreeHandler) SelectTreeNode(ctx *gin.Context) {
	// TODO: Implement SelectTreeNode logic
}

func (t *TreeHandler) GetTopTreeNode(ctx *gin.Context) {
	// TODO: Implement GetTopTreeNode logic
}

func (t *TreeHandler) GetAllTreeNode(ctx *gin.Context) {
	// TODO: Implement GetAllTreeNode logic
}

func (t *TreeHandler) CreateTreeNode(ctx *gin.Context) {
	// TODO: Implement CreateTreeNode logic
}

func (t *TreeHandler) DeleteTreeNode(ctx *gin.Context) {
	// TODO: Implement DeleteTreeNode logic
}

func (t *TreeHandler) GetChildrenTreeNode(ctx *gin.Context) {
	// TODO: Implement GetChildrenTreeNode logic
}

func (t *TreeHandler) UpdateTreeNode(ctx *gin.Context) {
	// TODO: Implement UpdateTreeNode logic
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
