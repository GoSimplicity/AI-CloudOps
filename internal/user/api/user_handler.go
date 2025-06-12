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
	"strconv"

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
		userGroup.POST("/profile/update", u.UpdateProfile)
		userGroup.DELETE("/:id", u.DeleteUser)
	}
}

// SignUp 用户注册处理
func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req model.UserSignUpReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.SignUp(ctx, &req)
	})
}

// Login 用户登录处理
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

		accessToken, refreshToken, err := u.ijwt.SetLoginToken(ctx, user.ID, user.Username)
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
func (u *UserHandler) Logout(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		if err := u.ijwt.ClearToken(ctx); err != nil {
			return nil, fmt.Errorf("登出失败: %w", err)
		}
		return nil, nil
	})
}

// Profile 获取用户信息
func (u *UserHandler) Profile(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		uc := ctx.MustGet("user").(ijwt.UserClaims)
		user, err := u.service.GetProfile(ctx, uc.Uid)
		if err != nil {
			return nil, fmt.Errorf("获取用户信息失败: %w", err)
		}
		return user, nil
	})
}

// RefreshToken 刷新令牌
func (u *UserHandler) RefreshToken(ctx *gin.Context) {
	var req model.TokenRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		rc := ijwt.RefreshClaims{}

		key := viper.GetString("jwt.key2")
		token, err := jwt.ParseWithClaims(req.RefreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
			return []byte(key), nil
		})

		if err != nil || token == nil || !token.Valid {
			return nil, fmt.Errorf("令牌无效，请重新登录")
		}

		if err = u.ijwt.CheckSession(ctx, rc.Ssid); err != nil {
			return nil, fmt.Errorf("会话已过期，请重新登录")
		}

		newToken, err := u.ijwt.SetJWTToken(ctx, rc.Uid, rc.Username, rc.Ssid)
		if err != nil {
			return nil, fmt.Errorf("刷新令牌失败: %w", err)
		}

		return newToken, nil
	})
}

// GetPermCode 获取权限码
func (u *UserHandler) GetPermCode(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		uc := ctx.MustGet("user").(ijwt.UserClaims)
		codes, err := u.service.GetPermCode(ctx, uc.Uid)
		if err != nil {
			return nil, fmt.Errorf("获取权限码失败: %w", err)
		}
		return codes, nil
	})
}

// GetUserList 获取用户列表
func (u *UserHandler) GetUserList(ctx *gin.Context) {
	var req model.ListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return u.service.GetUserList(ctx, &req)
	})
}

// ChangePassword 修改密码
func (u *UserHandler) ChangePassword(ctx *gin.Context) {
	var req model.ChangePasswordReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		if req.NewPassword != req.ConfirmPassword {
			return nil, fmt.Errorf("两次输入的新密码不一致")
		}

		uc := ctx.MustGet("user").(ijwt.UserClaims)
		err := u.service.ChangePassword(ctx, uc.Uid, req.Password, req.NewPassword)
		if err != nil {
			if errors.Is(err, constants.ErrorPasswordIncorrect) {
				return nil, fmt.Errorf("原密码错误")
			}
			return nil, fmt.Errorf("修改密码失败: %w", err)
		}

		return nil, nil
	})
}

// WriteOff 注销账号
func (u *UserHandler) WriteOff(ctx *gin.Context) {
	var req model.WriteOffReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.WriteOff(ctx, req.Username, req.Password)
	})
}

// UpdateProfile 更新用户信息
func (u *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req model.UpdateProfileReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, u.service.UpdateProfile(ctx, req.UserId, &req)
	})
}

// DeleteUser 删除用户
func (u *UserHandler) DeleteUser(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		id := ctx.Param("id")
		idInt, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf("用户ID格式错误")
		}

		return nil, u.service.DeleteUser(ctx, idInt)
	})
}

// GetUserDetail 获取用户详情
func (u *UserHandler) GetUserDetail(ctx *gin.Context) {
	var req model.GetUserDetailReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		id, err := utils.GetParamID(ctx)
		if err != nil {
			return nil, fmt.Errorf("用户ID格式错误")
		}

		req.ID = id
		return u.service.GetUserDetail(ctx, req.ID)
	})
}
