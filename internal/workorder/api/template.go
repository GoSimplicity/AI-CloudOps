package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	service service.TemplateService
}

func NewTemplateHandler(service service.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		service: service,
	}
}

func (h *TemplateHandler) RegisterRouters(server *gin.Engine) {
	templateGroup := server.Group("/api/workorder/template")
	{
		templateGroup.POST("/create", h.CreateTemplate)
		templateGroup.POST("/update", h.UpdateTemplate)
		templateGroup.POST("/delete", h.DeleteTemplate)
		templateGroup.POST("/list", h.ListTemplate)
		templateGroup.POST("/detail", h.DetailTemplate)
	}
}

func (h *TemplateHandler) CreateTemplate(ctx *gin.Context) {
}

func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
}

func (h *TemplateHandler) DeleteTemplate(ctx *gin.Context) {
}

func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
}

func (h *TemplateHandler) DetailTemplate(ctx *gin.Context) {
}
