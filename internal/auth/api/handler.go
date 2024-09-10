package api

import (
	"github.com/GoSimplicity/CloudOps/internal/auth/service"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

func (a *AuthHandler) RegisterRouters(server *gin.Engine) {
	authGroup := server.Group("/api/auth")

	//TODO 菜单相关路由

	// 获取菜单列表
	authGroup.GET("/menu/list", a.GetMenuList)
	// 获取所有菜单列表
	authGroup.GET("/menu/all", a.GetAllMenuList)
	// 更新菜单
	authGroup.POST("/menu/update", a.UpdateMenu)
	// 创建菜单
	authGroup.POST("/menu/create", a.CreateMenu)
	// 删除菜单
	authGroup.DELETE("/menu/:id", a.DeleteMenu)

	//TODO 权限管理相关路由

	// 获取所有角色列表
	authGroup.GET("/role/list", a.GetAllRoleList)
	// 创建角色
	authGroup.POST("/role/create", a.CreateRole)
	// 更新角色
	authGroup.POST("/role/update", a.UpdateRole)
	// 设置角色状态
	authGroup.POST("/role/status", a.SetRoleStatus)
	// 删除角色
	authGroup.DELETE("/role/:id", a.DeleteRole)

	//TODO API 管理相关路由

	// 获取 API 列表
	authGroup.GET("/api/list", a.GetApiList)
	// 获取所有 API 列表
	authGroup.GET("/api/all", a.GetApiListAll)
	// 删除 API
	authGroup.DELETE("/api/:id", a.DeleteApi)
	// 创建 API
	authGroup.POST("/api/create", a.CreateApi)
	// 更新 API
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

	if err := ctx.ShouldBindJSON(req); err != nil {
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
	// TODO: Implement CreateMenu
}

func (a *AuthHandler) DeleteMenu(ctx *gin.Context) {
	// TODO: Implement DeleteMenu
}

func (a *AuthHandler) GetAllRoleList(ctx *gin.Context) {
	// TODO: Implement GetAllRoleList
}

func (a *AuthHandler) CreateRole(ctx *gin.Context) {
	// TODO: Implement CreateRole
}

func (a *AuthHandler) UpdateRole(ctx *gin.Context) {
	// TODO: Implement UpdateRole
}

func (a *AuthHandler) SetRoleStatus(ctx *gin.Context) {
	// TODO: Implement SetRoleStatus
}

func (a *AuthHandler) DeleteRole(ctx *gin.Context) {
	// TODO: Implement DeleteRole
}

func (a *AuthHandler) CreateAccount(ctx *gin.Context) {
	// TODO: Implement CreateAccount
}

func (a *AuthHandler) AccountExist(ctx *gin.Context) {
	// TODO: Implement AccountExist
}

func (a *AuthHandler) UpdateAccount(ctx *gin.Context) {
	// TODO: Implement UpdateAccount
}

func (a *AuthHandler) ChangePassword(ctx *gin.Context) {
	// TODO: Implement ChangePassword
}

func (a *AuthHandler) GetAccountList(ctx *gin.Context) {
	// TODO: Implement GetAccountList
}

func (a *AuthHandler) GetAllUserAndRoles(ctx *gin.Context) {
	// TODO: Implement GetAllUserAndRoles
}

func (a *AuthHandler) DeleteAccount(ctx *gin.Context) {
	// TODO: Implement DeleteAccount
}

func (a *AuthHandler) GetApiList(ctx *gin.Context) {
	// TODO: Implement GetApiList
}

func (a *AuthHandler) GetApiListAll(ctx *gin.Context) {
	// TODO: Implement GetApiListAll
}

func (a *AuthHandler) DeleteApi(ctx *gin.Context) {
	// TODO: Implement DeleteApi
}

func (a *AuthHandler) CreateApi(ctx *gin.Context) {
	// TODO: Implement CreateApi
}

func (a *AuthHandler) UpdateApi(ctx *gin.Context) {
	// TODO: Implement UpdateApi
}
