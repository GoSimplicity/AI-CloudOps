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
 */

package middleware

import (
	"strconv"
	"strings"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

type GatewayAuthMiddleware struct {
	ijwt.Handler
}

func NewGatewayAuthMiddleware(hdl ijwt.Handler) *GatewayAuthMiddleware {
	return &GatewayAuthMiddleware{
		Handler: hdl,
	}
}

func (m *GatewayAuthMiddleware) CheckAuth() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		path := ctx.Request.URL.Path

		// 跳过认证的路径（网关层已经处理）
		if m.shouldSkipAuth(path) {
			ctx.Next()
			return
		}

		// 检查是否启用网关模式
		gatewayMode := viper.GetBool("gateway.enabled")
		if !gatewayMode {
			// 如果未启用网关模式，回退到传统JWT认证
			m.fallbackToJWTAuth(ctx)
			return
		}

		// 从网关传递的请求头中获取用户信息
		userID := ctx.GetHeader("X-User-ID")
		userName := ctx.GetHeader("X-User-Name")
		sessionID := ctx.GetHeader("X-Session-ID")
		accountTypeStr := ctx.GetHeader("X-Account-Type")

		// 如果网关头信息不存在，尝试从网关认证头获取token
		gatewayAuth := ctx.GetHeader("X-Gateway-Auth")
		if userID == "" && gatewayAuth != "" {
			// 尝试从JWT token中解析用户信息作为降级方案
			m.fallbackToJWTAuth(ctx)
			return
		}

		// 检查必需的用户信息是否存在
		if userID == "" || userName == "" || sessionID == "" {
			// 如果网关信息不完整，尝试从JWT中解析
			m.fallbackToJWTAuth(ctx)
			return
		}

		// 解析用户ID
		uid, err := strconv.Atoi(userID)
		if err != nil {
			ctx.AbortWithStatus(401)
			return
		}

		// 解析账户类型
		var accountType int8 = 1 // 默认为普通用户
		if accountTypeStr != "" {
			if at, err := strconv.Atoi(accountTypeStr); err == nil {
				accountType = int8(at)
			}
		}

		// 构造用户信息
		userClaims := ijwt.UserClaims{
			Uid:         uid,
			Username:    userName,
			Ssid:        sessionID,
			AccountType: accountType,
			UserAgent:   ctx.GetHeader("User-Agent"),
		}

		// 验证会话（如果需要的话）
		if viper.GetBool("gateway.verify_session") {
			err := m.CheckSession(ctx, sessionID)
			if err != nil {
				ctx.AbortWithStatus(401)
				return
			}
		}

		// 将用户信息设置到上下文中
		ctx.Set("user", userClaims)
		ctx.Next()
	}
}

func (m *GatewayAuthMiddleware) shouldSkipAuth(path string) bool {
	skipPaths := []string{
		"/api/user/login",
		"/api/user/logout",
		"/api/user/refresh_token",
		"/api/user/signup",
		"/api/not_auth/getBindIps",
		"/api/not_auth/getTreeNodeBindIps",
		"/favicon.ico",
		"/",
	}

	skipPrefixes := []string{
		"/swagger/",
		"/api/monitor/prometheus_configs/",
	}

	// 检查完整路径匹配
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	// 检查路径前缀匹配
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

func (m *GatewayAuthMiddleware) fallbackToJWTAuth(ctx *gin.Context) {
	var uc ijwt.UserClaims
	var tokenStr string

	// WebSocket路径从查询参数获取token
	if strings.HasPrefix(ctx.Request.URL.Path, "/api/tree/local/terminal") ||
		strings.Contains(ctx.Request.URL.Path, "/exec") {
		tokenStr = ctx.Query("token")
	} else {
		// 从请求头提取token
		tokenStr = m.ExtractToken(ctx)
	}

	if tokenStr == "" {
		ctx.AbortWithStatus(401)
		return
	}

	key := []byte(viper.GetString("jwt.key1"))
	token, err := jwt.ParseWithClaims(tokenStr, &uc, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if err != nil || token == nil || !token.Valid {
		ctx.AbortWithStatus(401)
		return
	}

	// 检查UserAgent
	if uc.UserAgent == "" {
		ctx.AbortWithStatus(401)
		return
	}

	// 验证会话
	err = m.CheckSession(ctx, uc.Ssid)
	if err != nil {
		ctx.AbortWithStatus(401)
		return
	}

	ctx.Set("user", uc)
	ctx.Next()
}
