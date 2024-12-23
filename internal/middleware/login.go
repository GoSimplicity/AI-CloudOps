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

package middleware

import (
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"strings"

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
		if path == "/api/user/login" ||
			//path == "/api/user/signup" ||   // 不允许用户自己注册账号
			path == "/api/user/logout" ||
			strings.Contains(path, "hello") ||
			path == "/api/user/refresh_token" ||
			path == "/api/user/signup" ||
			path == "/api/not_auth/getTreeNodeBindIps" ||
			path == "/api/monitor/prometheus_configs/prometheus" ||
			path == "/api/monitor/prometheus_configs/prometheus_alert" ||
			path == "/api/monitor/prometheus_configs/prometheus_record" ||
			path == "/api/monitor/prometheus_configs/alertManager" {
			return
		}

		var uc ijwt.UserClaims
		// 从请求中提取token
		tokenStr := m.ExtractToken(ctx)
		key := []byte(viper.GetString("jwt.key1"))
		token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
			return key, nil
		})

		if err != nil {
			// token 错误
			ctx.AbortWithStatus(401)
			return
		}

		if token == nil || !token.Valid {
			// token 非法或过期
			ctx.AbortWithStatus(401)
			return
		}

		// 检查是否携带ua头
		if uc.UserAgent == "" {
			ctx.AbortWithStatus(401)
			return
		}

		// 检查会话是否有效
		err = m.CheckSession(ctx, uc.Ssid)

		if err != nil {
			ctx.AbortWithStatus(401)
			return
		}

		ctx.Set("user", uc)
	}
}
