package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
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
}

func (h *ProcessHandler) UpdateProcess(ctx *gin.Context) {
}

func (h *ProcessHandler) DeleteProcess(ctx *gin.Context) {
}

func (h *ProcessHandler) ListProcess(ctx *gin.Context) {
}

func (h *ProcessHandler) DetailProcess(ctx *gin.Context) {
}

func (h *ProcessHandler) PublishProcess(ctx *gin.Context) {
}

func (h *ProcessHandler) CloneProcess(ctx *gin.Context) {
}
