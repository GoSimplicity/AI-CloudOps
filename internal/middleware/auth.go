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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

// 预定义跳过权限校验的路径
var skipAuthPaths = map[string]bool{
	"/api/user/login":                                   true,
	"/api/user/logout":                                  true,
	"/api/user/refresh_token":                           true,
	"/api/user/signup":                                  true,
	"/api/not_auth/getTreeNodeBindIps":                  true,
	"/api/monitor/prometheus_configs/prometheus":        true,
	"/api/monitor/prometheus_configs/prometheus_alert":  true,
	"/api/monitor/prometheus_configs/prometheus_record": true,
	"/api/monitor/prometheus_configs/alertManager":      true,
}

type AuthMiddleware struct {
	roleService service.RoleService
}

func NewAuthMiddleware(roleService service.RoleService) *AuthMiddleware {
	return &AuthMiddleware{
		roleService: roleService,
	}
}

func (am *AuthMiddleware) CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		
		// 快速检查是否需要跳过权限校验
		if skipAuthPaths[path] {
			c.Next()
			return
		}
		
		// 跳过静态资源和WebSocket路径
		if path == "/" ||
			strings.HasPrefix(path, "/api/ai/chat/ws") ||
			strings.HasPrefix(path, "/assets") ||
			strings.HasPrefix(path, "/_app.config.js") ||
			strings.HasPrefix(path, "/jse/") ||
			strings.HasPrefix(path, "/favicon.ico") ||
			strings.HasPrefix(path, "/js/") ||
			strings.HasPrefix(path, "/css/") {
			c.Next()
			return
		}
		
		user := c.MustGet("user").(utils.UserClaims)
		if user.Username == "admin" {
			c.Next()
			return
		}
		// TODO: 实现权限校验
		c.Next()
	}
}
