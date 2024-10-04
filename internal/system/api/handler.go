package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao/casbin"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service/api"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service/menu"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service/role"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuthHandler struct {
	apiService  api.ApiService
	roleService role.RoleService
	menuService menu.MenuService
	ijwt        ijwt.Handler
	l           *zap.Logger
	casbinDao   casbin.CasbinDAO
	userDao     dao.UserDAO
}

func NewAuthHandler(apiService api.ApiService, roleService role.RoleService, menuService menu.MenuService, handler ijwt.Handler, l *zap.Logger, casbinDao casbin.CasbinDAO, userDao dao.UserDAO) *AuthHandler {
	return &AuthHandler{
		apiService:  apiService,
		roleService: roleService,
		menuService: menuService,
		ijwt:        handler,
		l:           l,
		casbinDao:   casbinDao,
		userDao:     userDao,
	}
}

func (a *AuthHandler) RegisterRouters(server *gin.Engine) {
	authGroup := server.Group("/api/auth")
	authGroup.Use(middleware.NewCasbinMiddleware(a.l, a.userDao, a.casbinDao).CheckPermission())

	// 菜单相关路由
	authGroup.GET("/menu/list", a.GetMenuList)
	authGroup.GET("/menu/all", a.GetAllMenuList)
	authGroup.POST("/menu/update", a.UpdateMenu)
	authGroup.POST("/menu/update_status", a.UpdateMenuStatus)
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

	roles, err := a.menuService.GetMenuList(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, roles)
}

func (a *AuthHandler) GetAllMenuList(ctx *gin.Context) {
	menus, err := a.menuService.GetAllMenuList(ctx)
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

	if err := a.menuService.UpdateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (a *AuthHandler) UpdateMenuStatus(ctx *gin.Context) {
	var req struct {
		ID     int    `json:"id"`
		Status string `json:"status"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := a.menuService.UpdateMenuStatus(ctx, req.ID, req.Status); err != nil {
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

	if err := a.menuService.CreateMenu(ctx, req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (a *AuthHandler) DeleteMenu(ctx *gin.Context) {
	id := ctx.Param("id")

	if err := a.menuService.DeleteMenu(ctx, id); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) GetAllRoleList(ctx *gin.Context) {
	roles, err := a.roleService.GetAllRoleList(ctx)
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

	err := a.roleService.CreateRole(ctx, req)
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

	err := a.roleService.UpdateRole(ctx, req)
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

	err := a.roleService.SetRoleStatus(ctx, req.ID, req.Status)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}

func (a *AuthHandler) DeleteRole(ctx *gin.Context) {
	id := ctx.Param("id")

	err := a.roleService.DeleteRole(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) GetApiList(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	Apis, err := a.apiService.GetApiList(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, Apis)
}

func (a *AuthHandler) GetApiListAll(ctx *gin.Context) {
	Apis, err := a.apiService.GetApiListAll(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, Apis)
}

func (a *AuthHandler) DeleteApi(ctx *gin.Context) {
	id := ctx.Param("id")

	err := a.apiService.DeleteApi(ctx, id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "删除成功")
}

func (a *AuthHandler) CreateApi(ctx *gin.Context) {
	var ma model.Api

	if err := ctx.ShouldBindJSON(&ma); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.apiService.CreateApi(ctx, &ma)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "创建成功")
}

func (a *AuthHandler) UpdateApi(ctx *gin.Context) {
	var ma model.Api

	if err := ctx.ShouldBindJSON(&ma); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	err := a.apiService.UpdateApi(ctx, &ma)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithMessage(ctx, "更新成功")
}
