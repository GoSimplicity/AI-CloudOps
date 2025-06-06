package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeEcsHandler struct {
	ecsService service.TreeEcsService
}

func NewTreeEcsHandler(ecsService service.TreeEcsService) *TreeEcsHandler {
	return &TreeEcsHandler{
		ecsService: ecsService,
	}
}

func (h *TreeEcsHandler) RegisterRouters(server *gin.Engine) {
	ecsGroup := server.Group("/ecs")
	{
		ecsGroup.POST("/list", h.ListEcsResources)
		ecsGroup.POST("/instance_options", h.ListInstanceOptions)
		ecsGroup.POST("/detail", h.GetEcsDetail)
		ecsGroup.POST("/create", h.CreateEcsResource)
		ecsGroup.DELETE("/delete", h.DeleteEcs)
		ecsGroup.POST("/start", h.StartEcs)
		ecsGroup.POST("/stop", h.StopEcs)
		ecsGroup.POST("/restart", h.RestartEcs)
	}
}

func (h *TreeEcsHandler) ListEcsResources(c *gin.Context) {

}

func (h *TreeEcsHandler) ListInstanceOptions(c *gin.Context) {

}

func (h *TreeEcsHandler) GetEcsDetail(c *gin.Context) {

}

func (h *TreeEcsHandler) CreateEcsResource(c *gin.Context) {

}

func (h *TreeEcsHandler) DeleteEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) StartEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) StopEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) RestartEcs(c *gin.Context) {

}
