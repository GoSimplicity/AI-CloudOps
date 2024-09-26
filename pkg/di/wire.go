//go:build wireinject

package di

import (
	authHandler "github.com/GoSimplicity/AI-CloudOps/internal/auth/api"
	apiDao "github.com/GoSimplicity/AI-CloudOps/internal/auth/dao/api"
	authDao "github.com/GoSimplicity/AI-CloudOps/internal/auth/dao/casbin"
	menuDao "github.com/GoSimplicity/AI-CloudOps/internal/auth/dao/menu"
	roleDao "github.com/GoSimplicity/AI-CloudOps/internal/auth/dao/role"
	apiService "github.com/GoSimplicity/AI-CloudOps/internal/auth/service/api"
	menuService "github.com/GoSimplicity/AI-CloudOps/internal/auth/service/menu"
	roleService "github.com/GoSimplicity/AI-CloudOps/internal/auth/service/role"
	k8sHandler "github.com/GoSimplicity/AI-CloudOps/internal/k8s/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	k8sDao "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	k8sService "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	treeHandler "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	ecsDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/ecs"
	elbDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/elb"
	rdsDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/rds"
	nodeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/tree_node"
	treeService "github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	userHandler "github.com/GoSimplicity/AI-CloudOps/internal/user/api"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	userService "github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/google/wire"
	_ "github.com/google/wire"
)

func InitWebServer() *Cmd {
	wire.Build(
		InitMiddlewares,
		ijwt.NewJWTHandler,
		InitGinServer,
		InitLogger,
		InitRedis,
		InitDB,
		InitCasbin,
		InitAndRefreshK8sClient,
		userHandler.NewUserHandler,
		authHandler.NewAuthHandler,
		treeHandler.NewTreeHandler,
		k8sHandler.NewK8sHandler,
		client.NewK8sClient,
		userService.NewUserService,
		treeService.NewTreeService,
		apiService.NewApiService,
		roleService.NewRoleService,
		menuService.NewMenuService,
		k8sService.NewK8sService,
		userDao.NewUserDAO,
		apiDao.NewApiDAO,
		roleDao.NewRoleDAO,
		menuDao.NewMenuDAO,
		authDao.NewCasbinDAO,
		ecsDao.NewTreeEcsDAO,
		rdsDao.NewTreeRdsDAO,
		elbDao.NewTreeElbDAO,
		k8sDao.NewK8sDAO,
		nodeDao.NewTreeNodeDAO,
		wire.Struct(new(Cmd), "*"),
	)
	return new(Cmd)
}
