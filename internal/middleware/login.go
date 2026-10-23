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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"strings"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type JWTMiddleware struct {
	ijwt.Handler
	users service.UserService
}

func NewJWTMiddleware(hdl ijwt.Handler, users service.UserService) *JWTMiddleware {
	return &JWTMiddleware{
		Handler: hdl,
		users:   users,
	}
}

// CheckLogin 校验JWT
func (m *JWTMiddleware) CheckLogin() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path
		// 跳过token验证的路径
		if path == "/api/user/login" ||
			path == "/api/user/logout" ||
			path == "/api/user/refresh_token" ||
			path == "/api/not_auth/getBindIps" || path == "/api/not_auth/getTreeNodeBindIps" ||
			strings.HasPrefix(path, "/api/monitor/prometheus_configs/") ||
			path == "/favicon.ico" ||
			path == "/" || path == "/health" || path == "/ready" {
			ctx.Next()
			return
		}

		var uc ijwt.UserClaims
		var tokenStr string

		// WebSocket路径从查询参数获取token
		if strings.HasPrefix(path, "/api/tree/local/terminal") ||
			strings.Contains(path, "/exec") {
			tokenStr = ctx.Query("token")
		} else {
			// 从请求头提取token
			tokenStr = m.ExtractToken(ctx)
		}

		parsed, err := m.ParseUserClaims(tokenStr)
		if err != nil || parsed == nil {
			ctx.AbortWithStatus(401)
			return
		}
		uc = *parsed
		user, err := m.users.GetProfile(ctx.Request.Context(), uc.Uid)
		if err != nil || user == nil || user.Enable != 1 || user.Username != uc.Username {
			ctx.AbortWithStatus(401)
			return
		}
		// Permissions and account type must reflect persisted state, not stale claims.
		uc.AccountType = user.AccountType
		uc.Username = user.Username

		// 检查UserAgent
		if uc.UserAgent == "" {
			ctx.AbortWithStatus(401)
			return
		}

		err = m.CheckSession(ctx, uc.Ssid)

		if err != nil {
			ctx.AbortWithStatus(401)
			return
		}

		ctx.Set("user", uc)
		ctx.Next()
	}
}
