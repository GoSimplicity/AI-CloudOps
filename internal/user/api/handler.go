package api

import (
	"net/http"

	"github.com/GoSimplicity/CloudOps/internal/user/dto"
	"github.com/GoSimplicity/CloudOps/internal/user/service"
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
	userGroup := server.Group("/api/users")
	userGroup.POST("/signup", u.SignUp)
	userGroup.POST("/login", u.Login)
	userGroup.GET("/profile", u.Profile)

	userGroup.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{"message": "success"})
	})
}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req dto.UserDTO

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	res, err := u.service.SignUp(ctx, req)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
	}

	ctx.JSON(http.StatusOK, res)
}

func (u *UserHandler) Login(ctx *gin.Context) {

}

func (u *UserHandler) Profile(ctx *gin.Context) {

}
