package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
	var req model.TemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTemplate(ctx, req)
	})
}

func (h *TemplateHandler) UpdateTemplate(ctx *gin.Context) {
	var req model.TemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTemplate(ctx, req)
	})
}

func (h *TemplateHandler) DeleteTemplate(ctx *gin.Context) {
	var req model.DeleteTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteTemplate(ctx, req)
	})
}

func (h *TemplateHandler) ListTemplate(ctx *gin.Context) {
	var req model.ListTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListTemplate(ctx, req)
	})
}

func (h *TemplateHandler) DetailTemplate(ctx *gin.Context) {
	var req model.DetailTemplateReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailTemplate(ctx, req)
	})
}
