package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeRdsHandler struct {
	rdsService service.TreeRdsService
}

func NewTreeRdsHandler(rdsService service.TreeRdsService) *TreeRdsHandler {
	return &TreeRdsHandler{
		rdsService: rdsService,
	}
}

func (h *TreeRdsHandler) RegisterRouters(server *gin.Engine) {
	rdsGroup := server.Group("/rds")
	{
		rdsGroup.POST("/list", h.ListRdsResources)
		rdsGroup.POST("/detail", h.GetRdsDetail)
		rdsGroup.POST("/create", h.CreateRdsResource)
		rdsGroup.POST("/start", h.StartRds)
		rdsGroup.POST("/stop", h.StopRds)
		rdsGroup.POST("/restart", h.RestartRds)
		rdsGroup.POST("/delete", h.DeleteRds)
	}
}

func (h *TreeRdsHandler) ListRdsResources(ctx *gin.Context) {

}

func (h *TreeRdsHandler) GetRdsDetail(ctx *gin.Context) {

}

func (h *TreeRdsHandler) CreateRdsResource(ctx *gin.Context) {

}

func (h *TreeRdsHandler) StartRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) StopRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) RestartRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) DeleteRds(ctx *gin.Context) {

}