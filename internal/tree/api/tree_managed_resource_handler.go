package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type ManagedCloudHandler struct{ service service.ManagedCloudService }

func NewManagedCloudHandler(s service.ManagedCloudService) *ManagedCloudHandler {
	return &ManagedCloudHandler{service: s}
}
func (h *ManagedCloudHandler) RegisterRouters(server *gin.Engine) {
	group := server.Group("/api/tree/cloud/managed")
	group.GET("/list", h.List)
	group.GET("/detail/:id", h.Detail)
	group.GET("/history", h.History)
	group.POST("/sync", h.Sync)
}

func (h *ManagedCloudHandler) Detail(ctx *gin.Context) {
	if _, ok := managedCloudUser(ctx); !ok {
		return
	}
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}
	var req model.ListCloudManagedResourcesReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) { return h.service.GetManaged(ctx.Request.Context(), &req, id) })
}
func managedCloudUser(ctx *gin.Context) (jwt.UserClaims, bool) {
	value, ok := ctx.Get("user")
	user, valid := value.(jwt.UserClaims)
	if !ok || !valid || user.Uid <= 0 {
		base.ForbiddenError(ctx, "缺少可信用户身份")
		return jwt.UserClaims{}, false
	}
	return user, true
}
func (h *ManagedCloudHandler) List(ctx *gin.Context) {
	if _, ok := managedCloudUser(ctx); !ok {
		return
	}
	var req model.ListCloudManagedResourcesReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) { return h.service.ListManaged(ctx.Request.Context(), &req) })
}
func (h *ManagedCloudHandler) History(ctx *gin.Context) {
	if _, ok := managedCloudUser(ctx); !ok {
		return
	}
	var req model.ListCloudManagedResourcesReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) { return h.service.ListManagedHistory(ctx.Request.Context(), &req) })
}
func (h *ManagedCloudHandler) Sync(ctx *gin.Context) {
	user, ok := managedCloudUser(ctx)
	if !ok {
		return
	}
	var req model.SyncCloudManagedResourcesReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.OperatorID = user.Uid
		return h.service.SyncManaged(ctx.Request.Context(), &req)
	})
}
