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
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/user/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
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
	userGroup.POST("/signup", u.SignUp)                  // 注册
	userGroup.POST("/login", u.Login)                    // 登陆
	userGroup.POST("/refresh_token", u.RefreshToken)     // 刷新token
	userGroup.POST("/logout", u.Logout)                  // 退出登陆
	userGroup.GET("/profile", u.Profile)                 // 用户信息
	userGroup.GET("/codes", u.GetPermCode)               // 前端所需状态码
	userGroup.GET("/list", u.GetUserList)                // 用户列表
	userGroup.POST("/change_password", u.ChangePassword) // 修改密码
	userGroup.POST("/write_off", u.WriteOff)             // 注销账号
	userGroup.POST("/profile/update", u.UpdateProfile)   // 更新用户信息
	userGroup.DELETE("/:id", u.DeleteUser)               // 删除用户
}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	var req model.User

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := u.service.SignUp(ctx, &req); err != nil {
		if errors.Is(err, constants.ErrorUserExist) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorUserExist.Error())
			return
		}

		u.l.Error("signup failed", zap.Error(err))

		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) Login(ctx *gin.Context) {
	var req model.User

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	ur, err := u.service.Login(ctx, &req)
	if err != nil {
		if errors.Is(err, constants.ErrorUserNotExist) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorUserNotExist.Error())
			return
		}

		if errors.Is(err, constants.ErrorPasswordIncorrect) {
			apiresponse.ErrorWithMessage(ctx, constants.ErrorPasswordIncorrect.Error())
			return
		}

		u.l.Error("login failed", zap.Error(err))

		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	accessToken, refreshToken, err := u.ijwt.SetLoginToken(ctx, ur.ID)
	if err != nil {
		u.l.Error("set login token failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, gin.H{
		"id":           ur.ID,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"roles":        ur.Roles,
		"desc":         ur.Desc,
		"realName":     ur.RealName,
		"userId":       ur.ID,
		"username":     ur.Username,
	})
}

func (u *UserHandler) Logout(ctx *gin.Context) {
	if err := u.ijwt.ClearToken(ctx); err != nil {
		u.l.Error("clear token failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) Profile(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)
	user, err := u.service.GetProfile(ctx, uc.Uid)
	if err != nil {
		u.l.Error("get user info failed", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, gin.H{
		"id":           user.ID,
		"roles":        user.Roles,
		"realName":     user.RealName,
		"userId":       user.ID,
		"username":     user.Username,
		"desc":         user.Desc,
		"homePath":     user.HomePath,
		"mobile":       user.Mobile,
		"feiShuUserId": user.FeiShuUserId,
	})
}

func (u *UserHandler) RefreshToken(ctx *gin.Context) {
	var req TokenRequest

	rc := ijwt.RefreshClaims{}

	if err := ctx.BindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	// 获取密钥
	key := viper.GetString("jwt.key2")

	// 解析 token 并获取刷新 claims
	token, err := jwt.ParseWithClaims(req.RefreshToken, &rc, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		u.l.Error("failed to parse token", zap.Error(err))
		apiresponse.Unauthorized(ctx, http.StatusUnauthorized, "token parsing failed", "token解析失败")
		return
	}

	// 检查 token 是否有效
	if token == nil || !token.Valid {
		u.l.Warn("invalid token")
		apiresponse.Unauthorized(ctx, http.StatusUnauthorized, "token is invalid", "token无效")
		return
	}

	// 检查会话状态是否异常
	if err = u.ijwt.CheckSession(ctx, rc.Ssid); err != nil {
		u.l.Error("session check failed", zap.Error(err))
		apiresponse.Unauthorized(ctx, http.StatusUnauthorized, "session check failed", "会话检查失败")
		return
	}

	// 刷新短 token
	newToken, err := u.ijwt.SetJWTToken(ctx, rc.Uid, rc.Ssid)
	if err != nil {
		u.l.Error("failed to generate new token", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "生成新token失败")
		return
	}

	apiresponse.SuccessWithData(ctx, newToken)
}

func (u *UserHandler) GetPermCode(ctx *gin.Context) {
	uc := ctx.MustGet("user").(ijwt.UserClaims)

	codes, err := u.service.GetPermCode(ctx, uc.Uid)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, codes)
}

func (u *UserHandler) GetUserList(ctx *gin.Context) {
	list, err := u.service.GetUserList(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

func (u *UserHandler) ChangePassword(ctx *gin.Context) {
	var req ChangePasswordRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	// 验证新密码和确认密码是否一致
	if req.NewPassword != req.ConfirmPassword {
		apiresponse.ErrorWithMessage(ctx, "新密码和确认密码不一致")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	if err := u.service.ChangePassword(ctx, uc.Uid, req.Password, req.NewPassword); err != nil {
		if errors.Is(err, constants.ErrorPasswordIncorrect) {
			apiresponse.ErrorWithMessage(ctx, "原密码不正确")
			return
		}
		u.l.Error("修改密码失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "修改密码失败")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) WriteOff(ctx *gin.Context) {
	var req WriteOffRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	// 验证用户名不能为空
	if req.Username == "" {
		apiresponse.ErrorWithMessage(ctx, "用户名不能为空")
		return
	}

	if err := u.service.WriteOff(ctx, req.Username, req.Password); err != nil {
		u.l.Error("注销账号失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "注销账号失败")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) UpdateProfile(ctx *gin.Context) {
	var req UpdateProfileRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserId = uc.Uid

	user := &model.User{
		RealName:     req.RealName,
		Desc:         req.Desc,
		Mobile:       req.Mobile,
		FeiShuUserId: req.FeiShuUserId,
		AccountType:  req.AccountType,
		HomePath:     req.HomePath,
		Enable:       req.Enable,
	}

	if err := u.service.UpdateProfile(ctx, req.UserId, user); err != nil {
		u.l.Error("更新用户信息失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "更新用户信息失败")
		return
	}

	apiresponse.Success(ctx)
}

func (u *UserHandler) DeleteUser(ctx *gin.Context) {
	id := ctx.Param("id")

	idInt, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	if err := u.service.DeleteUser(ctx, idInt); err != nil {
		u.l.Error("删除用户失败", zap.Error(err))
		apiresponse.InternalServerError(ctx, http.StatusInternalServerError, err.Error(), "删除用户失败")
		return
	}

	apiresponse.Success(ctx)
}
