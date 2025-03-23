package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/gin-gonic/gin"
	"net/http"
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
	var formDesignreq model.FormDesignReq
	if err := ctx.ShouldBindJSON(&formDesignreq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	formDesign, err := h.service.CreateFormDesign(ctx, &formDesignreq)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id": formDesign.ID, // 假设 formDesign 结构体中有 ID 字段
		},
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) UpdateFormDesign(ctx *gin.Context) {
	var formDesignreq model.FormDesignReq
	if err := ctx.ShouldBindJSON(&formDesignreq); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateFormDesign(ctx, &formDesignreq); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	response := gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) DeleteFormDesign(ctx *gin.Context) {
	// 定义一个结构体来接收请求中的 id
	var request struct {
		ID int64 `json:"id"`
	}
	// 绑定请求中的 JSON 数据到 request 结构体
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用服务层的删除方法
	if err := h.service.DeleteFormDesign(ctx, request.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 构造成功响应
	response := gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) ListFormDesign(ctx *gin.Context) {
	// 定义结构体来接收请求参数
	var request model.ListFormDesignReq
	// 绑定请求中的 JSON 数据到 request 结构体
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 调用服务层的方法来获取表单设计列表
	formDesigns, err := h.service.ListFormDesign(ctx, &request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 构造成功响应
	response := gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"list":      formDesigns,
			"total":     len(formDesigns),
			"page":      request.Page,
			"page_size": request.PageSize,
		},
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) DetailFormDesign(ctx *gin.Context) {
	// 定义结构体来接收请求中的 id
	var request struct {
		ID int64 `json:"id"`
	}
	// 绑定请求中的 JSON 数据到 request 结构体
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用服务层的获取表单详情方法
	formDesignReq, err := h.service.DetailFormDesign(ctx, request.ID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 构造成功响应
	response := gin.H{
		"code":    0,
		"message": "success",
		"data":    formDesignReq,
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) PublishFormDesign(ctx *gin.Context) {
	// Define a struct to hold the request data
	var request struct {
		ID int64 `json:"id"`
	}
	// Bind the JSON data from the request to the struct
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Call the service layer to publish the form design
	if err := h.service.PublishFormDesign(ctx, request.ID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Construct a success response
	response := gin.H{
		"code":    0,
		"message": "success",
		"data":    nil,
	}
	ctx.JSON(http.StatusOK, response)
}

func (h *FormDesignHandler) CloneFormDesign(ctx *gin.Context) {
	// 定义结构体来接收请求数据
	var request struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	// 绑定请求中的 JSON 数据到 request 结构体
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 调用服务层的克隆方法
	clonedFormDesign, err := h.service.CloneFormDesign(ctx, request.ID, request.Name)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 构造成功响应
	response := gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"id": clonedFormDesign.ID,
		},
	}
	ctx.JSON(http.StatusOK, response)
}
