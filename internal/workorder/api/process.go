package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ProcessHandler struct {
	service service.ProcessService
}

func NewProcessHandler(service service.ProcessService) *ProcessHandler {
	return &ProcessHandler{
		service: service,
	}
}

func (h *ProcessHandler) RegisterRouters(server *gin.Engine) {
	processGroup := server.Group("/api/workorder/process")
	{
		processGroup.POST("/create", h.CreateProcess)
		processGroup.POST("/update", h.UpdateProcess)
		processGroup.POST("/delete", h.DeleteProcess)
		processGroup.POST("/list", h.ListProcess)
		processGroup.POST("/detail", h.DetailProcess)
		processGroup.POST("/publish", h.PublishProcess)
		processGroup.POST("/clone", h.CloneProcess)
	}
}

func (h *ProcessHandler) CreateProcess(ctx *gin.Context) {
	var req model.ProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateProcess(ctx, &req)
	})
}

func (h *ProcessHandler) UpdateProcess(ctx *gin.Context) {
	var req model.ProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateProcess(ctx, &req)
	})

}

func (h *ProcessHandler) DeleteProcess(ctx *gin.Context) {
	var id model.DeleteProcessReqReq
	utils.HandleRequest(ctx, &id, func() (interface{}, error) {
		return nil, h.service.DeleteProcess(ctx, id)
	})

}

func (h *ProcessHandler) ListProcess(ctx *gin.Context) {
	var req model.ListProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListProcess(ctx, req)
	})
}

func (h *ProcessHandler) DetailProcess(ctx *gin.Context) {
	var req model.DetailProcessReqReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailProcess(ctx, req)
	})
}

func (h *ProcessHandler) PublishProcess(ctx *gin.Context) {
	var req model.PublishProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.PublishProcess(ctx, req)
	})
}

func (h *ProcessHandler) CloneProcess(ctx *gin.Context) {
	var req model.CloneProcessReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CloneProcess(ctx, req)
	})

}
