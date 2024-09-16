package api

import (
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

	// ecs 相关理由
	ecsGroup := treeGroup.Group("/ecs")
	ecsGroup.POST("/create", t.CreateResourceEcs)
	ecsGroup.POST("/delete", t.DeleteResourceEcs)
	ecsGroup.POST("/update", t.UpdateResourceEcs)
	ecsGroup.GET("/get", t.GetResourceEcs)

	// hello world
	treeGroup.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "hello world",
		})
	})
}

func (t *TreeHandler) CreateResourceEcs(ctx *gin.Context) {
	var req model.ResourceEcs

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := t.service.CreateResourceEcs(ctx, &req); err != nil {
		t.l.Error("create resource ecs failed", zap.Error(err))

		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (t *TreeHandler) DeleteResourceEcs(ctx *gin.Context) {

}

func (t *TreeHandler) UpdateResourceEcs(ctx *gin.Context) {

}

func (t *TreeHandler) GetResourceEcs(ctx *gin.Context) {

}
