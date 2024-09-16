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
	// Casbin 检查权限的中间件

	treeGroup.POST("/resource/ecs", t.CreateResourceEcs)
	// hello world
	treeGroup.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "hello world",
		})
	})
}

func (t *TreeHandler) CreateResourceEcs(ctx *gin.Context) {

}
