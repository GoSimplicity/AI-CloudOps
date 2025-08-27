/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package api

import (
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type UserHandler struct {
	service service.UserService
	ijwt    ijwt.Handler
}

func NewUserHandler(service service.UserService, ijwt ijwt.Handler) *UserHandler {
	return &UserHandler{
		service: service,
		ijwt:    ijwt,
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	userGroup := server.Group("/api/user")
	{
		userGroup.POST("/signup", u.SignUp)
		userGroup.POST("/login", u.Login)
		userGroup.POST("/refresh_token", u.RefreshToken)
		userGroup.POST("/logout", u.Logout)
		userGroup.GET("/profile", u.Profile)
		userGroup.GET("/codes", u.GetPermCode)
		userGroup.GET("/detail/:id", u.GetUserDetail)
		userGroup.GET("/list", u.GetUserList)
		userGroup.POST("/change_password", u.ChangePassword)
		userGroup.POST("/write_off", u.WriteOff)
		userGroup.PUT("/profile/update/:id", u.UpdateProfile)
		userGroup.DELETE("/:id", u.DeleteUser)
		userGroup.GET("/statistics", u.GetUserStatistics)
	}
}

// SignUp 用户注册处理
// @Summary 用户注册
// @Description 创建新用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserSignUpReq true "用户注册请求参数"
// @Success 200 {object} utils.ApiResponse "注册成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/user/signup [post]
func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req model.UserSignUpReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.SignUp(ctx, &req)
	})
}

// Login 用户登录处理
// @Summary 用户登录
// @Description 用户账号密码登录
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UserLoginReq true "用户登录请求参数"
// @Success 200 {object} utils.ApiResponse "登录成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 401 {object} utils.ApiResponse "用户名或密码错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/user/login [post]
func (u *UserHandler) Login(ctx *gin.Context) {
	var req model.UserLoginReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		user, err := u.service.Login(ctx, &req)
		if err != nil {
			switch {
			case errors.Is(err, constants.ErrorUserNotExist):
				return nil, fmt.Errorf("用户不存在")
			case errors.Is(err, constants.ErrorPasswordIncorrect):
				return nil, fmt.Errorf("密码错误")
			default:
				return nil, fmt.Errorf("登录失败: %w", err)
			}
		}

		accessToken, refreshToken, err := u.ijwt.SetLoginToken(ctx, user.ID, user.Username, user.AccountType)
		if err != nil {
			return nil, fmt.Errorf("生成令牌失败: %w", err)
		}

		return gin.H{
			"id":           user.ID,
			"accessToken":  accessToken,
			"refreshToken": refreshToken,
			"desc":         user.Desc,
			"realName":     user.RealName,
			"userId":       user.ID,
			"username":     user.Username,
		}, nil
	})
}

// Logout 用户登出处理
// @Summary 用户登出
// @Description 退出登录并清除令牌
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "登出成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/logout [post]
func (u *UserHandler) Logout(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, u.ijwt.ClearToken(ctx)
	})
}

// Profile 获取用户信息
// @Summary 获取用户资料
// @Description 获取当前登录用户的详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/profile [get]
func (u *UserHandler) Profile(ctx *gin.Context) {
	var req model.ProfileReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.ID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.service.GetProfile(ctx, req.ID)
	})
}

// RefreshToken 刷新令牌
// @Summary 刷新访问令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.TokenRequest true "刷新令牌请求参数"
// @Success 200 {object} utils.ApiResponse "刷新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 401 {object} utils.ApiResponse "令牌无效"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/user/refresh_token [post]
func (u *UserHandler) RefreshToken(ctx *gin.Context) {
	var req model.TokenRequest

	rc := ijwt.RefreshClaims{}

	key := viper.GetString("jwt.key2")
	token, err := jwt.ParseWithClaims(req.RefreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if err != nil || token == nil || !token.Valid {
		utils.ErrorWithMessage(ctx, "令牌无效，请重新登录")
		return
	}

	if err = u.ijwt.CheckSession(ctx, rc.Ssid); err != nil {
		utils.ErrorWithMessage(ctx, "会话已过期，请重新登录")
		return
	}

	req.UserID = rc.Uid
	req.Username = rc.Username
	req.Ssid = rc.Ssid
	req.AccountType = rc.AccountType

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.ijwt.SetJWTToken(ctx, req.UserID, req.Username, req.Ssid, req.AccountType)
	})
}

// GetPermCode 获取权限码
// @Summary 获取用户权限码
// @Description 获取当前用户的权限码列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/codes [get]
func (u *UserHandler) GetPermCode(ctx *gin.Context) {
	var req model.GetPermCodeReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.ID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.service.GetPermCode(ctx, req.ID)
	})
}

// GetUserList 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取系统中的用户列表
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param username query string false "用户名模糊搜索"
// @Success 200 {object} utils.ApiResponse{data=[]model.User} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/list [get]
func (u *UserHandler) GetUserList(ctx *gin.Context) {
	var req model.GetUserListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.service.GetUserList(ctx, &req)
	})
}

// ChangePassword 修改密码
// @Summary 修改用户密码
// @Description 修改当前登录用户的密码
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.ChangePasswordReq true "修改密码请求参数"
// @Success 200 {object} utils.ApiResponse "修改成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/change_password [post]
func (u *UserHandler) ChangePassword(ctx *gin.Context) {
	var req model.ChangePasswordReq

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.ChangePassword(ctx, &req)
	})
}

// WriteOff 注销账号
// @Summary 注销用户账号
// @Description 永久注销用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.WriteOffReq true "注销账号请求参数"
// @Success 200 {object} utils.ApiResponse "注销成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/write_off [post]
func (u *UserHandler) WriteOff(ctx *gin.Context) {
	var req model.WriteOffReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.WriteOff(ctx, req.Username, req.Password)
	})
}

// UpdateProfile 更新用户信息
// @Summary 更新用户资料
// @Description 更新用户个人信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body model.UpdateProfileReq true "更新用户信息请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/profile/update/{id} [post]
func (u *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req model.UpdateProfileReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}
	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.UpdateProfile(ctx, &req)
	})
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 根据用户ID删除用户账号
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/{id} [delete]
func (u *UserHandler) DeleteUser(ctx *gin.Context) {
	var req model.DeleteUserReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, u.service.DeleteUser(ctx, req.ID)
	})
}

// GetUserDetail 获取用户详情
// @Summary 获取用户详情
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/detail/{id} [get]
func (u *UserHandler) GetUserDetail(ctx *gin.Context) {
	var req model.GetUserDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.service.GetUserDetail(ctx, req.ID)
	})
}

// GetUserStatistics 获取用户统计
// @Summary 获取用户统计信息
// @Description 获取系统用户相关的统计数据
// @Tags 用户管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/user/statistics [get]
func (u *UserHandler) GetUserStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return u.service.GetUserStatistics(ctx)
	})
}
