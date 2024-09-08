package api

import (
	"errors"
	"net/http"

	"github.com/GoSimplicity/CloudOps/internal/constants"
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
	userGroup.POST("/signup", u.SignUp)
	userGroup.POST("/login", u.Login)
	userGroup.POST("/logout", u.Logout)

}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req model.User

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := u.service.SignUp(ctx, &req); err != nil {
		if errors.Is(err, constants.ErrorUserExist) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorUserExist.Error())
			return
		}

		u.l.Error("signup failed", zap.Error(err))

		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) Login(ctx *gin.Context) {
	var req model.User

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if _, err := u.service.Login(ctx, &req); err != nil {
		if errors.Is(err, constants.ErrorUserNotExist) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorUserNotExist.Error())
			return
		}

		if errors.Is(err, constants.ErrorPasswordIncorrect) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorPasswordIncorrect.Error())
			return
		}

		u.l.Error("login failed", zap.Error(err))

		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}
	// TODO: set token
	apiresponse.Success(ctx)
}

func (u *UserHandler) Logout(ctx *gin.Context) {

	apiresponse.Success(ctx)
}
