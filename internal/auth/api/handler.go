package api

import (
	"github.com/GoSimplicity/CloudOps/internal/auth/dao/casbin"
	"github.com/GoSimplicity/CloudOps/internal/auth/service"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/dao"
	"github.com/GoSimplicity/CloudOps/pkg/middleware"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	service   service.AuthService
	ijwt      ijwt.Handler
	l         *zap.Logger
	casbinDao casbin.CasbinDAO
	userDao   dao.UserDAO
}

func NewAuthHandler(service service.AuthService, handler ijwt.Handler, l *zap.Logger, casbinDao casbin.CasbinDAO, userDao dao.UserDAO) *AuthHandler {
	return &AuthHandler{
		service:   service,
		ijwt:      handler,
		l:         l,
		casbinDao: casbinDao,
		userDao:   userDao,
	}
}

func (a *AuthHandler) RegisterRouters(server *gin.Engine) {
	authGroup := server.Group("/api/auth")
	authGroup.Use(middleware.NewCasbinMiddleware(a.l, a.userDao, a.casbinDao).CheckPermission())

	// 菜单相关路由
	authGroup.GET("/menu/list", a.GetMenuList)
	authGroup.GET("/menu/all", a.GetAllMenuList)
	authGroup.POST("/menu/update", a.UpdateMenu)
	authGroup.POST("/menu/create", a.CreateMenu)
	authGroup.DELETE("/menu/:id", a.DeleteMenu)

	// 权限管理相关路由
	authGroup.GET("/role/list", a.GetAllRoleList)
	authGroup.POST("/role/create", a.CreateRole)
	authGroup.POST("/role/update", a.UpdateRole)
	authGroup.POST("/role/status", a.SetRoleStatus)
	authGroup.DELETE("/role/:id", a.DeleteRole)

	// API 管理相关路由
	authGroup.GET("/api/list", a.GetApiList)
	authGroup.GET("/api/all", a.GetApiListAll)
	authGroup.DELETE("/api/:id", a.DeleteApi)
	authGroup.POST("/api/create", a.CreateApi)
	authGroup.POST("/api/update", a.UpdateApi)
}

func (a *AuthHandler) GetMenuList(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	roles, err := a.service.GetMenuList(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, roles)
}

func (a *AuthHandler) GetAllMenuList(ctx *gin.Context) {
	menus, err := a.service.GetAllMenuList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, menus)
}

func (a *AuthHandler) UpdateMenu(ctx *gin.Context) {
	var req model.Menu

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := a.service.UpdateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (a *AuthHandler) CreateMenu(ctx *gin.Context) {
	var req model.Menu

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := a.service.CreateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (a *AuthHandler) DeleteMenu(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := a.service.DeleteMenu(ctx, id); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) GetAllRoleList(ctx *gin.Context) {
	roles, err := a.service.GetAllRoleList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, roles)
}

func (a *AuthHandler) CreateRole(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.service.CreateRole(ctx, req)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (a *AuthHandler) UpdateRole(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.service.UpdateRole(ctx, req)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (a *AuthHandler) SetRoleStatus(ctx *gin.Context) {
	var req model.Role

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.service.SetRoleStatus(ctx, req.ID, req.Status)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (a *AuthHandler) DeleteRole(ctx *gin.Context) {
	id := ctx.Param("id")

	err := a.service.DeleteRole(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) GetApiList(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	Apis, err := a.service.GetApiList(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, Apis)
}

func (a *AuthHandler) GetApiListAll(ctx *gin.Context) {
	Apis, err := a.service.GetApiListAll(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, Apis)
}

func (a *AuthHandler) DeleteApi(ctx *gin.Context) {
	id := ctx.Param("id")

	err := a.service.DeleteApi(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) CreateApi(ctx *gin.Context) {
	var api model.Api

	if err := ctx.ShouldBindJSON(&api); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.service.CreateApi(ctx, &api)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (a *AuthHandler) UpdateApi(ctx *gin.Context) {
	var api model.Api

	if err := ctx.ShouldBindJSON(&api); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.service.UpdateApi(ctx, &api)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}
