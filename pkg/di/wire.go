//go:build wireinject

package di

import (
	authHandler "github.com/GoSimplicity/CloudOps/internal/auth/api"
	apiDao "github.com/GoSimplicity/CloudOps/internal/auth/dao/api"
	authDao "github.com/GoSimplicity/CloudOps/internal/auth/dao/casbin"
	menuDao "github.com/GoSimplicity/CloudOps/internal/auth/dao/menu"
	roleDao "github.com/GoSimplicity/CloudOps/internal/auth/dao/role"
	apiService "github.com/GoSimplicity/CloudOps/internal/auth/service/api"
	menuService "github.com/GoSimplicity/CloudOps/internal/auth/service/menu"
	roleService "github.com/GoSimplicity/CloudOps/internal/auth/service/role"
	treeHandler "github.com/GoSimplicity/CloudOps/internal/tree/api"
	ecsDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/ecs"
	elbDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/elb"
	rdsDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/rds"
	nodeDao "github.com/GoSimplicity/CloudOps/internal/tree/dao/tree_node"
	treeService "github.com/GoSimplicity/CloudOps/internal/tree/service"
	userHandler "github.com/GoSimplicity/CloudOps/internal/user/api"
	userDao "github.com/GoSimplicity/CloudOps/internal/user/dao"
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
		userHandler.NewUserHandler,
		authHandler.NewAuthHandler,
		treeHandler.NewTreeHandler,
		userService.NewUserService,
		treeService.NewTreeService,
		apiService.NewApiService,
		roleService.NewRoleService,
		menuService.NewMenuService,
		userDao.NewUserDAO,
		apiDao.NewApiDAO,
		roleDao.NewRoleDAO,
		menuDao.NewMenuDAO,
		authDao.NewCasbinDAO,
		ecsDao.NewTreeEcsDAO,
		rdsDao.NewTreeRdsDAO,
		elbDao.NewTreeElbDAO,
		nodeDao.NewTreeNodeDAO,
	)
	return gin.Default()
}
