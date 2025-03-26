package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
		instanceGroup.POST("/delete", h.DeleteInstance)
	}
}

func (h *InstanceHandler) CreateInstance(ctx *gin.Context) {
	var req model.InstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateInstance(ctx, req)
	})
}

func (h *InstanceHandler) ApproveInstance(ctx *gin.Context) {
	var req model.InstanceFlowReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.ApproveInstance(ctx, req)
	})
}

func (h *InstanceHandler) ActionInstance(ctx *gin.Context) {
	var req model.InstanceFlowReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.ActionInstance(ctx, req)
	})
}

func (h *InstanceHandler) CommentInstance(ctx *gin.Context) {
	var req model.InstanceCommentReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CommentInstance(ctx, req)
	})
}

func (h *InstanceHandler) ListInstance(ctx *gin.Context) {
	var req model.ListInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListInstance(ctx, req)
	})
}

func (h *InstanceHandler) DetailInstance(ctx *gin.Context) {
	var req model.DetailInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailInstance(ctx, req.ID)
	})

}

func (h *InstanceHandler) DeleteInstance(ctx *gin.Context) {
	var req model.DeleteInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteInstance(ctx, req)
	})

}
