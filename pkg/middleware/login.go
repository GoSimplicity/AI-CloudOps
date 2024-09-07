package middleware

import (
	"strings"

	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type JWTMiddleware struct {
	ijwt.Handler
}

func NewJWTMiddleware(hdl ijwt.Handler) *JWTMiddleware {
	return &JWTMiddleware{
		Handler: hdl,
	}
}

// CheckLogin 校验JWT
func (m *JWTMiddleware) CheckLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		// 如果请求的路径是下述路径，则不进行token验证
		if path == "/api/users/signup" ||
			path == "/api/users/login" ||
			path == "/api/users/refresh_token" ||
			path == "/api/users/change_password" ||
			strings.Contains(path, "hello") {
			return
		}

		var uc ijwt.UserClaims
		// 从请求中提取token
		tokenStr := m.ExtractToken(ctx)
		key1 := viper.GetString("jwt.key1")

		token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
			return key1, nil
		})

		if err != nil {
			// token 错误
			ctx.AbortWithStatus(400)
			return
		}

		if token == nil || !token.Valid {
			// token 非法或过期
			ctx.AbortWithStatus(400)
			return
		}

		// 检查是否携带ua头
		if uc.UserAgent == "" {
			ctx.AbortWithStatus(400)
			return
		}

		// 检查会话是否有效
		err = m.CheckSession(ctx, uc.Ssid)

		if err != nil {
			ctx.AbortWithStatus(400)
			return
		}

		ctx.Set("user", uc)
	}
}
