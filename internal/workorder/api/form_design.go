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

	var formDesignReq model.FormDesignReq
	utils.HandleRequest(ctx, &formDesignReq, func() (interface{}, error) {
		return h.service.CreateFormDesign(ctx, &formDesignReq)
	})
}

func (h *FormDesignHandler) UpdateFormDesign(ctx *gin.Context) {

	var formDesignreq model.FormDesignReq
	utils.HandleRequest(ctx, &formDesignreq, func() (interface{}, error) {
		return nil, h.service.UpdateFormDesign(ctx, &formDesignreq)
	})
}

func (h *FormDesignHandler) DeleteFormDesign(ctx *gin.Context) {

	var request struct {
		ID int64 `json:"id"`
	}
	utils.HandleRequest(ctx, &request, func() (interface{}, error) {
		return h.service.DeleteFormDesign(ctx, request.ID), nil
	})

}

func (h *FormDesignHandler) ListFormDesign(ctx *gin.Context) {

	var request model.ListFormDesignReq
	utils.HandleRequest(ctx, &request, func() (interface{}, error) {
		return h.service.ListFormDesign(ctx, &request)
	})
}

func (h *FormDesignHandler) DetailFormDesign(ctx *gin.Context) {

	var request struct {
		ID int64 `json:"id"`
	}
	utils.HandleRequest(ctx, &request, func() (interface{}, error) {
		return h.service.DetailFormDesign(ctx, request.ID)
	})
}

func (h *FormDesignHandler) PublishFormDesign(ctx *gin.Context) {

	var request struct {
		ID int64 `json:"id"`
	}
	utils.HandleRequest(ctx, &request, func() (interface{}, error) {
		return nil, h.service.PublishFormDesign(ctx, request.ID)
	})
}

func (h *FormDesignHandler) CloneFormDesign(ctx *gin.Context) {

	var request struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	utils.HandleRequest(ctx, &request, func() (interface{}, error) {
		return h.service.CloneFormDesign(ctx, request.ID, request.Name)
	})
}
