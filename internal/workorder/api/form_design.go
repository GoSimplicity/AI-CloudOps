package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
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
	var req model.FormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateFormDesign(ctx, &req)
	})
}

func (h *FormDesignHandler) UpdateFormDesign(ctx *gin.Context) {
	var req model.FormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateFormDesign(ctx, &req)
	})
}

func (h *FormDesignHandler) DeleteFormDesign(ctx *gin.Context) {
	var req model.DetailFormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteFormDesign(ctx, req.ID)
	})

}

func (h *FormDesignHandler) ListFormDesign(ctx *gin.Context) {
	var req model.ListFormDesignReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.ListFormDesign(ctx, &req)
	})
}

func (h *FormDesignHandler) DetailFormDesign(ctx *gin.Context) {
	var req model.DetailFormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.DetailFormDesign(ctx, req.ID)
	})
}

func (h *FormDesignHandler) PublishFormDesign(ctx *gin.Context) {
	var req model.PublishFormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.PublishFormDesign(ctx, req.ID)
	})
}

func (h *FormDesignHandler) CloneFormDesign(ctx *gin.Context) {
	var req model.CloneFormDesignReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CloneFormDesign(ctx, req.Name)
	})
}
