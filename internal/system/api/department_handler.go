package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct{ service service.DepartmentService }

func NewDepartmentHandler(s service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{s}
}
func (h *DepartmentHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/api/department")
	// Organization assignments are administrative state, never self-service profile data.
	group.Use(func(c *gin.Context) {
		claims, err := userutils.ExtractClaims(c)
		if err != nil || claims.Username != "admin" {
			base.ForbiddenError(c, "仅管理员可管理部门及用户归属")
			c.Abort()
			return
		}
		c.Next()
	})
	group.GET("/list", h.List)
	group.GET("/:id/detail", h.Get)
	group.POST("/create", h.Create)
	group.PUT("/:id/update", h.Update)
	group.DELETE("/:id/delete", h.Delete)
	group.PUT("/user/:id", h.AssignUser)
}
func (h *DepartmentHandler) List(c *gin.Context) {
	var r model.ListReq
	base.HandleRequest(c, &r, func() (any, error) { return h.service.List(c.Request.Context(), &r) })
}
func (h *DepartmentHandler) Get(c *gin.Context) {
	id, e := base.GetParamID(c)
	if e != nil {
		base.BadRequestError(c, "部门ID不合法")
		return
	}
	base.HandleRequest(c, nil, func() (any, error) { return h.service.Get(c.Request.Context(), id) })
}
func (h *DepartmentHandler) Create(c *gin.Context) {
	var r model.DepartmentWriteReq
	base.HandleRequest(c, &r, func() (any, error) { return h.service.Write(c.Request.Context(), 0, &r) })
}
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, e := base.GetParamID(c)
	if e != nil {
		base.BadRequestError(c, "部门ID不合法")
		return
	}
	var r model.DepartmentWriteReq
	base.HandleRequest(c, &r, func() (any, error) { return h.service.Write(c.Request.Context(), id, &r) })
}
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, e := base.GetParamID(c)
	if e != nil {
		base.BadRequestError(c, "部门ID不合法")
		return
	}
	base.HandleRequest(c, nil, func() (any, error) { return nil, h.service.Delete(c.Request.Context(), id) })
}
func (h *DepartmentHandler) AssignUser(c *gin.Context) {
	id, e := base.GetParamID(c)
	if e != nil {
		base.BadRequestError(c, "用户ID不合法")
		return
	}
	var r model.AssignUserDepartmentReq
	base.HandleRequest(c, &r, func() (any, error) { return nil, h.service.AssignUser(c.Request.Context(), id, *r.DepartmentID) })
}
