//go:build wireinject

package di

import (
	authHandler "github.com/GoSimplicity/CloudOps/internal/auth/api"
	authDAO "github.com/GoSimplicity/CloudOps/internal/auth/dao"
	authService "github.com/GoSimplicity/CloudOps/internal/auth/service"
	userHandler "github.com/GoSimplicity/CloudOps/internal/user/api"
	userDAO "github.com/GoSimplicity/CloudOps/internal/user/dao"
	userService "github.com/GoSimplicity/CloudOps/internal/user/service"
	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	_ "github.com/google/wire"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		InitMiddlewares,
		ijwt.NewJWTHandler,
		InitGinServer,
		InitLogger,
		InitRedis,
		InitDB,
		userDAO.NewUserDAO,
		userService.NewUserService,
		userHandler.NewUserHandler,
		authDAO.NewAuthDAO,
		authService.NewAuthService,
		authHandler.NewAuthHandler,
	)
	return gin.Default()
}
