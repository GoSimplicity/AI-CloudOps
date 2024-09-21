package api

import (
	"errors"
	"github.com/GoSimplicity/CloudOps/internal/constants"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"net/http"

	"github.com/GoSimplicity/CloudOps/internal/model"
	"github.com/GoSimplicity/CloudOps/internal/user/service"
	"github.com/GoSimplicity/CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
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
	userGroup.POST("/signup", u.SignUp)              // 注册
	userGroup.POST("/login", u.Login)                // 登陆
	userGroup.POST("/refresh_token", u.RefreshToken) // 刷新token
	userGroup.POST("/logout", u.Logout)              // 退出登陆
	userGroup.GET("/profile", u.Profile)             // 用户信息
	userGroup.GET("/codes", u.GetPermCode)           // 前端所需状态码
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
		"id":       user.ID,
		"roles":    user.Roles,
		"realName": user.RealName,
		"userId":   user.ID,
		"username": user.Username,
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
