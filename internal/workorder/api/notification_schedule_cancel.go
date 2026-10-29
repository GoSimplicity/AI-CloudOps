package api

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type cancelScheduledNotificationRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// CancelScheduledNotification 使用鉴权中间件中的身份，绝不读取客户端操作者字段。
func (h *NotificationHandler) CancelScheduledNotification(ctx *gin.Context) {
	id, err := base.GetParamID(ctx)
	if err != nil || id <= 0 {
		base.ErrorWithMessage(ctx, "无效的事件ID")
		return
	}
	identity, ok := ctx.Get("user")
	user, valid := identity.(jwt.UserClaims)
	if !ok || !valid || user.Uid <= 0 {
		base.ForbiddenError(ctx, "缺少可信操作人")
		return
	}
	var req cancelScheduledNotificationRequest
	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		store, ok := h.service.(interface {
			CancelScheduledNotification(context.Context, uint64, string, int) error
		})
		if !ok {
			return nil, errors.New("排期取消服务不可用")
		}
		return nil, store.CancelScheduledNotification(ctx.Request.Context(), uint64(id), req.Reason, user.Uid)
	})
}
