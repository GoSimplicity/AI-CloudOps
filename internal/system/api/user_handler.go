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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	jwt2 "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type UserHandler struct {
	service service.UserService
	jwt     jwt2.Handler
}

func NewUserHandler(service service.UserService, jwt jwt2.Handler) *UserHandler {
	return &UserHandler{
		service: service,
		jwt:     jwt,
	}
}

func (h *UserHandler) RegisterRoutes(server *gin.Engine) {
	userGroup := server.Group("/api/user")
	{
		userGroup.POST("/signup", h.SignUp)
		userGroup.POST("/login", h.Login)
		userGroup.POST("/refresh_token", h.RefreshToken)
		userGroup.POST("/logout", h.Logout)
		userGroup.GET("/profile", h.Profile)
		userGroup.GET("/codes", h.GetPermCode)
		userGroup.GET("/detail/:id", h.GetUserDetail)
		userGroup.GET("/list", h.GetUserList)
		userGroup.POST("/change_password", h.ChangePassword)
		userGroup.POST("/write_off", h.WriteOff)
		userGroup.PUT("/profile/update/:id", h.UpdateProfile)
		userGroup.DELETE("/:id/delete", h.DeleteUser)
		userGroup.GET("/statistics", h.GetUserStatistics)
	}
}

// SignUp 用户注册处理
func (h *UserHandler) SignUp(ctx *gin.Context) {
	var req model.UserSignUpReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.SignUp(ctx, &req)
	})
}

// Login 用户登录处理
func (h *UserHandler) Login(ctx *gin.Context) {
	var req model.UserLoginReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		user, err := h.service.Login(ctx, &req)
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

		accessToken, refreshToken, err := h.jwt.SetLoginToken(ctx, user.ID, user.Username, user.AccountType)
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
func (h *UserHandler) Logout(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.jwt.ClearToken(ctx)
	})
}

// Profile 获取用户信息
func (h *UserHandler) Profile(ctx *gin.Context) {
	var req model.ProfileReq

	uc, err := userutils.ExtractClaims(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = uc.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetProfile(ctx, req.ID)
	})
}

// RefreshToken 刷新令牌
func (h *UserHandler) RefreshToken(ctx *gin.Context) {
	var req model.TokenRequest

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		rc := jwt2.RefreshClaims{}

		key := viper.GetString("jwt.key2")
		token, err := jwt.ParseWithClaims(req.RefreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
			return []byte(key), nil
		})

		if err != nil || token == nil || !token.Valid {
			return nil, fmt.Errorf("令牌无效，请重新登录")
		}

		if err = h.jwt.CheckSession(ctx, rc.Ssid); err != nil {
			return nil, fmt.Errorf("会话已过期，请重新登录")
		}

		return h.jwt.SetJWTToken(ctx, rc.Uid, rc.Username, rc.Ssid, rc.AccountType)
	})
}

// GetPermCode 获取权限码
func (h *UserHandler) GetPermCode(ctx *gin.Context) {
	var req model.GetPermCodeReq

	uc, err := userutils.ExtractClaims(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.ID = uc.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetPermCode(ctx, req.ID)
	})
}

// GetUserList 获取用户列表
func (h *UserHandler) GetUserList(ctx *gin.Context) {
	var req model.GetUserListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetUserList(ctx, &req)
	})
}

// ChangePassword 修改密码
func (h *UserHandler) ChangePassword(ctx *gin.Context) {
	var req model.ChangePasswordReq

	uc, err := userutils.ExtractClaims(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}
	req.UserID = uc.Uid

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.ChangePassword(ctx, &req)
	})
}

// WriteOff 注销账号
func (h *UserHandler) WriteOff(ctx *gin.Context) {
	var req model.WriteOffReq

	uc, err := userutils.ExtractClaims(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.WriteOff(ctx, uc.Uid, req.Password)
	})
}

// UpdateProfile 更新用户信息
func (h *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req model.UpdateProfileReq

	uc, err := userutils.ExtractClaims(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, err.Error())
		return
	}

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}
	req.ID = id

	if uc.Username != "admin" && uc.AccountType != constants.AccountTypeService && uc.Uid != req.ID {
		base.ForbiddenError(ctx, "无权限修改该用户信息")
		return
	}

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateProfile(ctx, &req)
	})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser(ctx *gin.Context) {
	var req model.DeleteUserReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.service.DeleteUser(ctx, req.ID)
	})
}

// GetUserDetail 获取用户详情
func (h *UserHandler) GetUserDetail(ctx *gin.Context) {
	var req model.GetUserDetailReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetUserDetail(ctx, req.ID)
	})
}

// GetUserStatistics 获取用户统计
func (h *UserHandler) GetUserStatistics(ctx *gin.Context) {
	base.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.GetUserStatistics(ctx)
	})
}
