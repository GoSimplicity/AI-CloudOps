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
	"encoding/base64"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"net/http"
	"strings"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	ws "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
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

		tokenStr = m.ExtractToken(ctx)
		// 仅升级请求使用 WebSocket 凭据，普通 execute API 继续使用请求头
		if browserWebSocketRequest(ctx) && tokenStr == "" {
			protocols := ctx.GetHeader("Sec-WebSocket-Protocol")
			tokenStr = ws.TokenFromProtocols(protocols)
			if tokenStr == "" && !strings.Contains(protocols, ws.CredentialProtocolPrefix) {
				// 保留已发布查询令牌格式，新的浏览器客户端使用子协议
				tokenStr = ctx.Query("token")
			}
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

// browserWebSocketRequest 仅在已注册终端路由与完整握手上使用浏览器凭据
func browserWebSocketRequest(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet || !websocket.IsWebSocketUpgrade(c.Request) || c.GetHeader("Sec-WebSocket-Version") != "13" {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(c.GetHeader("Sec-WebSocket-Key"))
	if err != nil || len(key) != 16 {
		return false
	}
	switch c.FullPath() {
	case "/api/tree/local/terminal/:id", "/api/tree/cloud/:id/terminal", "/api/k8s/pod/:cluster_id/:namespace/:name/containers/:container/exec":
		return true
	case "/api/v1/*path":
		return c.Request.URL.Path == "/api/v1/assistant/stream"
	default:
		return false
	}
}
