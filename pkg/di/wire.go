//go:build wireinject

package di

import (
	authHandler "github.com/GoSimplicity/CloudOps/internal/auth/api"
	"github.com/GoSimplicity/CloudOps/internal/auth/dao/auth"
	authDAO "github.com/GoSimplicity/CloudOps/internal/auth/dao/casbin"
	authService "github.com/GoSimplicity/CloudOps/internal/auth/service"
	treeHandler "github.com/GoSimplicity/CloudOps/internal/tree/api"
	ecsDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	elbDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/elb"
	rdsDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/rds"
	nodeDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/tree_node"
	treeService "github.com/GoSimplicity/CloudOps/internal/tree/service"
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
		InitCasbin,
		userDAO.NewUserDAO,
		userService.NewUserService,
		userHandler.NewUserHandler,
		auth.NewAuthDAO,
		authDAO.NewCasbinDAO,
		authService.NewAuthService,
		authHandler.NewAuthHandler,
		ecsDao.NewTreeEcsDAO,
		rdsDao.NewTreeRdsDAO,
		elbDao.NewTreeElbDAO,
		nodeDao.NewTreeNodeDAO,
		treeService.NewTreeService,
		treeHandler.NewTreeHandler,
	)
	return gin.Default()
}
