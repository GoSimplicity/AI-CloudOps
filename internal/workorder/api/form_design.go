package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/gin-gonic/gin"
)

type FormDesignHandler struct {
	service service.FormDesignService
}

func NewFormDesignHandler(service service.FormDesignService) *FormDesignHandler {
	return &FormDesignHandler{
		service: service,
	}
}

func (h *FormDesignHandler) RegisterRouters(server *gin.Engine) {
	formDesignGroup := server.Group("/api/workorder/form_design")
	{
		formDesignGroup.POST("/create", h.CreateFormDesign)
		formDesignGroup.POST("/update", h.UpdateFormDesign)
		formDesignGroup.POST("/delete", h.DeleteFormDesign)
		formDesignGroup.POST("/list", h.ListFormDesign)
		formDesignGroup.POST("/detail", h.DetailFormDesign)
		formDesignGroup.POST("/publish", h.PublishFormDesign)
		formDesignGroup.POST("/clone", h.CloneFormDesign)
	}
}

func (h *FormDesignHandler) CreateFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) UpdateFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) DeleteFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) ListFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) DetailFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) PublishFormDesign(ctx *gin.Context) {
}

func (h *FormDesignHandler) CloneFormDesign(ctx *gin.Context) {
}
