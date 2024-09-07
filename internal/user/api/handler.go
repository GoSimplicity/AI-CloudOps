package api

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/service"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type UserHandler struct {
	service service.UserService
	l       *zap.Logger
}

func NewUserHandler(service service.UserService, l *zap.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		l:       l,
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	userGroup := server.Group("/api/user")
	userGroup.POST("/create_user", u.CreateUser)
}

func (u *UserHandler) CreateUser(ctx *gin.Context) {
	var req model.User

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "绑定数据失败")
		return
	}

	if err := u.service.Create(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "创建用户失败")
		return
	}

	apiresponse.Success(ctx)
}
