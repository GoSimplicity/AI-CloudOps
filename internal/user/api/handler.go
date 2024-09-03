package api

import (
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
	userGroup := server.Group("/api/user")
	userGroup.POST("/create_user")
}

func (u *UserHandler) CreateUser(ctx *gin.Context) {
	var req dto.UserDTO

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if err := u.service.Create(ctx, req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(200, gin.H{"message": "success"})
}
