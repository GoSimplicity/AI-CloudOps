package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

func inboxUser(ctx *gin.Context) (int, bool) {
	value, ok := ctx.Get("user")
	user, valid := value.(jwt.UserClaims)
	if !ok || !valid || user.Uid <= 0 {
		base.ForbiddenError(ctx, "缺少可信用户身份")
		return 0, false
	}
	return user.Uid, true
}

func (h *NotificationHandler) ListInbox(ctx *gin.Context) {
	user, ok := inboxUser(ctx)
	if !ok {
		return
	}
	var req model.ListWorkorderInboxReq
	base.HandleRequest(ctx, &req, func() (interface{}, error) { return h.service.ListInbox(ctx.Request.Context(), user, &req) })
}
func (h *NotificationHandler) CountUnreadInbox(ctx *gin.Context) {
	user, ok := inboxUser(ctx)
	if !ok {
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		count, err := h.service.CountUnreadInbox(ctx.Request.Context(), user)
		return gin.H{"unread": count}, err
	})
}
func (h *NotificationHandler) ReadInbox(ctx *gin.Context) {
	user, ok := inboxUser(ctx)
	if !ok {
		return
	}
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的消息ID")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) { return nil, h.service.ReadInbox(ctx.Request.Context(), user, uint64(id)) })
}
func (h *NotificationHandler) ReadAllInbox(ctx *gin.Context) {
	user, ok := inboxUser(ctx)
	if !ok {
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) { return nil, h.service.ReadAllInbox(ctx.Request.Context(), user) })
}
func (h *NotificationHandler) DeleteInbox(ctx *gin.Context) {
	user, ok := inboxUser(ctx)
	if !ok {
		return
	}
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的消息ID")
		return
	}
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteInbox(ctx.Request.Context(), user, uint64(id))
	})
}
