package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeElbHandler struct {
	elbService service.TreeElbService
}

func NewTreeElbHandler(elbService service.TreeElbService) *TreeElbHandler {
	return &TreeElbHandler{
		elbService: elbService,
	}
}

func (h *TreeElbHandler) RegisterRouters(server *gin.Engine) {
	elbGroup := server.Group("/elb")
	{
		elbGroup.POST("/list", h.ListElbResources)
		elbGroup.POST("/detail", h.GetElbDetail)
		elbGroup.POST("/create", h.CreateElbResource)
		elbGroup.POST("/delete", h.DeleteElb)
	}
}

func (h *TreeElbHandler) ListElbResources(ctx *gin.Context) {

}

func (h *TreeElbHandler) GetElbDetail(ctx *gin.Context) {

}

func (h *TreeElbHandler) CreateElbResource(ctx *gin.Context) {

}

func (h *TreeElbHandler) DeleteElb(ctx *gin.Context) {

}
