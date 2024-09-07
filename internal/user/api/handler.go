package api

import (
	"errors"

	. "github.com/GoSimplicity/CloudOps/pkg/ginp"

	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/GoSimplicity/CloudOps/internal/user/dto"
	"github.com/GoSimplicity/CloudOps/internal/user/service"
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
	userGroup := server.Group("/api/users")
	userGroup.POST("/signup", WrapBody(u.SignUp))
	userGroup.POST("/login", u.Login)
	userGroup.GET("/profile", u.Profile)

	userGroup.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "success"})
	})
}

func (u *UserHandler) SignUp(ctx *gin.Context, req dto.UserDTO) (Result, error) {
	err := u.service.SignUp(ctx, req)
	if err != nil {
		if errors.Is(err, constants.ErrCodeDuplicateUserNameOrMobile) {
			return Result{
				Code: constants.UserExistErrorCode,
				Msg:  constants.ErrorUserExist.Error(),
			}, nil
		}
		return Result{
			Code: constants.UserSignUpFailedErrorCode,
			Msg:  constants.ErrorUserSignUpFail.Error(),
		}, err
	}
	return Result{
		Code: constants.SuccessCode,
		Msg:  constants.SuccessMsg,
	}, nil
}

func (u *UserHandler) Login(ctx *gin.Context) {

}

func (u *UserHandler) Profile(ctx *gin.Context) {

}
