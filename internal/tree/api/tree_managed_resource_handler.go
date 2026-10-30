package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"strconv"
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
	group.GET("/operations/:operation_id", h.GetOperation)
	group.GET("/operations", h.ListOperations)
	group.POST("/operations/:operation_id/resolve", h.ReviewOperation)
	group.POST("/sync", h.Sync)
	group.POST("/history/:id/resolve", h.ResolveHistory)
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

func (h *ManagedCloudHandler) ResolveHistory(ctx *gin.Context) {
	user, ok := managedCloudUser(ctx)
	if !ok {
		return
	}
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的历史ID")
		return
	}
	var req model.ResolveCloudManagedHistoryReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.HistoryID = id
		req.OperatorID = user.Uid
		return nil, h.service.ResolveManagedHistory(ctx.Request.Context(), &req)
	})
}

func (h *ManagedCloudHandler) GetOperation(ctx *gin.Context) {
	user, ok := managedCloudUser(ctx)
	if !ok {
		return
	}
	owner := user.Uid
	if raw := ctx.Query("owner_id"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			base.ErrorWithMessage(ctx, "操作用户筛选无效")
			return
		}
		if parsed != user.Uid && !permission.IsSuperAdmin(ctx) {
			base.ForbiddenError(ctx, "无权读取其他用户操作")
			return
		}
		owner = parsed
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetManagedOperation(ctx.Request.Context(), ctx.Param("operation_id"), owner)
	})
}
func (h *ManagedCloudHandler) ListOperations(ctx *gin.Context) {
	user, ok := managedCloudUser(ctx)
	if !ok {
		return
	}
	var req model.ListCloudManagedOperationsReq
	if err := base.BindRequest(ctx, &req); err != nil {
		base.ErrorWithMessage(ctx, "操作列表筛选无效")
		return
	}
	super := permission.IsSuperAdmin(ctx)
	if (req.AllOwners || (req.OwnerID > 0 && req.OwnerID != user.Uid)) && !super {
		base.ForbiddenError(ctx, "无权读取其他用户操作")
		return
	}
	req.ViewerID = user.Uid
	req.TrustedSuper = super
	base.HandleRequest(ctx, nil, func() (interface{}, error) { return h.service.ListManagedOperations(ctx.Request.Context(), &req) })
}
func (h *ManagedCloudHandler) ReviewOperation(ctx *gin.Context) {
	user, ok := managedCloudUser(ctx)
	if !ok {
		return
	}
	var req model.ReviewCloudManagedOperationReq
	if err := base.BindRequest(ctx, &req); err != nil {
		base.ErrorWithMessage(ctx, "操作核对请求格式无效")
		return
	}
	req.OperatorID = user.Uid
	req.OperationID = ctx.Param("operation_id")
	req.TrustedSuper = permission.IsSuperAdmin(ctx)
	if req.OwnerID > 0 && req.OwnerID != user.Uid && !req.TrustedSuper {
		base.ForbiddenError(ctx, "无权核对其他用户操作")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) { return h.service.ReviewManagedOperation(ctx.Request.Context(), &req) })
}
