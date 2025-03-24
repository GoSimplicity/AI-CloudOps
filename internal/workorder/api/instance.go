package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/gin-gonic/gin"
)

type InstanceHandler struct {
	service service.InstanceService
}

func NewInstanceHandler(service service.InstanceService) *InstanceHandler {
	return &InstanceHandler{
		service: service,
	}
}

func (h *InstanceHandler) RegisterRouters(server *gin.Engine) {
	instanceGroup := server.Group("/api/workorder/instance")
	{
		instanceGroup.POST("/create", h.CreateInstance)
		instanceGroup.POST("/approve", h.ApproveInstance)
		instanceGroup.POST("/action", h.ActionInstance)
		instanceGroup.POST("/comment", h.CommentInstance)
		instanceGroup.POST("/list", h.ListInstance)
		instanceGroup.POST("/detail", h.DetailInstance)
	}
}

func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {

}

func (h *InstanceHandler) ApproveInstance(ctx *gin.Context) {
}

func (h *InstanceHandler) ActionInstance(ctx *gin.Context) {
}

func (h *InstanceHandler) CommentInstance(ctx *gin.Context) {
}

func (h *InstanceHandler) ListInstance(ctx *gin.Context) {

}

func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {

}
