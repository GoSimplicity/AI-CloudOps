//go:build wireinject

package di

import (
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
	)
	return gin.Default()
}
