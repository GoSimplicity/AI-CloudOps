package api

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/service"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{
		service: service,
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
	}

	if err := u.service.Create(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, "创建用户失败")
	}

	apiresponse.Success(ctx)
}
