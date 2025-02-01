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
	"net/http"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type UserHandler struct {
	service service.UserService
	l       *zap.Logger
	ijwt    ijwt.Handler
}

func NewUserHandler(service service.UserService, l *zap.Logger, ijwt ijwt.Handler) *UserHandler {
	return &UserHandler{
		service: service,
		l:       l,
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
		userGroup.GET("/list", u.GetUserList)
		userGroup.POST("/change_password", u.ChangePassword)
		userGroup.POST("/write_off", u.WriteOff)
		userGroup.POST("/profile/update", u.UpdateProfile)
		userGroup.DELETE("/:id", u.DeleteUser)
	}
}

// SignUp 用户注册处理
func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req model.User
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	if err := u.service.SignUp(ctx, &req); err != nil {
		if errors.Is(err, constants.ErrorUserExist) {
			utils.ErrorWithMessage(ctx, "用户已存在")
			return
		}
		u.l.Error("注册失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "注册失败")
		return
	}

	utils.Success(ctx)
}

// Login 用户登录处理
func (u *UserHandler) Login(ctx *gin.Context) {
	var req model.User
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	user, err := u.service.Login(ctx, &req)
	if err != nil {
		switch {
		case errors.Is(err, constants.ErrorUserNotExist):
			utils.ErrorWithMessage(ctx, "用户不存在")
		case errors.Is(err, constants.ErrorPasswordIncorrect):
			utils.ErrorWithMessage(ctx, "密码错误")
		default:
			u.l.Error("登录失败", zap.Error(err))
			utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "登录失败")
		}
		return
	}

	accessToken, refreshToken, err := u.ijwt.SetLoginToken(ctx, user.ID)
	if err != nil {
		u.l.Error("生成令牌失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "登录失败")
		return
	}

	utils.SuccessWithData(ctx, gin.H{
		"id":           user.ID,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"roles":        user.Roles,
		"desc":         user.Desc,
		"realName":     user.RealName,
		"userId":       user.ID,
		"username":     user.Username,
	})
}

// Logout 用户登出处理
func (u *UserHandler) Logout(ctx *gin.Context) {
	if err := u.ijwt.ClearToken(ctx); err != nil {
		u.l.Error("清除令牌失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "登出失败")
		return
	}

	utils.Success(ctx)
}

// Profile 获取用户信息
func (u *UserHandler) Profile(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)
	user, err := u.service.GetProfile(ctx, uc.Uid)
	if err != nil {
		u.l.Error("获取用户信息失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "获取用户信息失败")
		return
	}

	utils.SuccessWithData(ctx, user)
}

// RefreshToken 刷新令牌
func (u *UserHandler) RefreshToken(ctx *gin.Context) {
	var req model.TokenRequest
	rc := ijwt.RefreshClaims{}

	if err := ctx.BindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	key := viper.GetString("jwt.key2")
	token, err := jwt.ParseWithClaims(req.RefreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if err != nil || token == nil || !token.Valid {
		u.l.Error("令牌验证失败", zap.Error(err))
		utils.Unauthorized(ctx, http.StatusUnauthorized, "令牌无效", "请重新登录")
		return
	}

	if err = u.ijwt.CheckSession(ctx, rc.Ssid); err != nil {
		u.l.Error("会话验证失败", zap.Error(err))
		utils.Unauthorized(ctx, http.StatusUnauthorized, "会话已过期", "请重新登录")
		return
	}

	newToken, err := u.ijwt.SetJWTToken(ctx, rc.Uid, rc.Ssid)
	if err != nil {
		u.l.Error("生成新令牌失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "刷新令牌失败")
		return
	}

	utils.SuccessWithData(ctx, newToken)
}

// GetPermCode 获取权限码
func (u *UserHandler) GetPermCode(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)
	codes, err := u.service.GetPermCode(ctx, uc.Uid)
	if err != nil {
		u.l.Error("获取权限码失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "获取权限码失败")
		return
	}

	utils.SuccessWithData(ctx, codes)
}

// GetUserList 获取用户列表
func (u *UserHandler) GetUserList(ctx *gin.Context) {
	list, err := u.service.GetUserList(ctx)
	if err != nil {
		u.l.Error("获取用户列表失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "获取用户列表失败")
		return
	}

	utils.SuccessWithData(ctx, list)
}

// ChangePassword 修改密码
func (u *UserHandler) ChangePassword(ctx *gin.Context) {
	var req model.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		utils.ErrorWithMessage(ctx, "两次输入的新密码不一致")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	if err := u.service.ChangePassword(ctx, uc.Uid, req.Password, req.NewPassword); err != nil {
		if errors.Is(err, constants.ErrorPasswordIncorrect) {
			utils.ErrorWithMessage(ctx, "原密码错误")
			return
		}
		u.l.Error("修改密码失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "修改密码失败")
		return
	}

	utils.Success(ctx)
}

// WriteOff 注销账号
func (u *UserHandler) WriteOff(ctx *gin.Context) {
	var req model.WriteOffRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	if req.Username == "" {
		utils.ErrorWithMessage(ctx, "用户名不能为空")
		return
	}

	if err := u.service.WriteOff(ctx, req.Username, req.Password); err != nil {
		u.l.Error("注销账号失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "注销账号失败")
		return
	}

	utils.Success(ctx)
}

// UpdateProfile 更新用户信息
func (u *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req model.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	user := &model.User{
		RealName:     req.RealName,
		Desc:         req.Desc,
		Mobile:       req.Mobile,
		FeiShuUserId: req.FeiShuUserId,
		AccountType:  int8(req.AccountType),
		HomePath:     req.HomePath,
		Enable:       int8(req.Enable),
	}

	if err := u.service.UpdateProfile(ctx, req.UserId, user); err != nil {
		u.l.Error("更新用户信息失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "更新用户信息失败")
		return
	}

	utils.Success(ctx)
}

// DeleteUser 删除用户
func (u *UserHandler) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorWithMessage(ctx, "用户ID格式错误")
		return
	}

	if err := u.service.DeleteUser(ctx, idInt); err != nil {
		u.l.Error("删除用户失败", zap.Error(err))
		utils.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "删除用户失败")
		return
	}

	utils.Success(ctx)
}
