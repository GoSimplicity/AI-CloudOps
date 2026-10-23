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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

var skipAuthPaths = map[string]bool{
	"/api/user/login":                  true,
	"/api/user/logout":                 true,
	"/api/user/refresh_token":          true,
	"/api/user/profile":                true,
	"/api/user/codes":                  true,
	"/api/not_auth/getBindIps":         true,
	"/api/not_auth/getTreeNodeBindIps": true,
	"/favicon.ico":                     true,
}

// HTTP方法映射
var methodMapping = map[string]int8{
	"GET":    1,
	"POST":   2,
	"PUT":    3,
	"DELETE": 4,
}

type AuthMiddleware struct {
	roleService service.RoleService
}

func NewAuthMiddleware(roleService service.RoleService) *AuthMiddleware {
	return &AuthMiddleware{
		roleService: roleService,
	}
}

func matchWildcardPath(apiPath, requestPath string, methodCode int8, apiMethod int8) bool {
	// 方法不匹配则返回false
	if apiMethod != methodCode {
		return false
	}

	// 完全匹配
	if apiPath == requestPath {
		return true
	}

	// 全局通配符匹配所有路径
	if apiPath == "/*" {
		return true
	}

	if !strings.Contains(apiPath, "*") {
		return false
	}

	// 末尾通配符：/api/user/*
	if strings.HasSuffix(apiPath, "*") {
		prefix := strings.TrimSuffix(apiPath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// 开头通配符：*/logs
	if strings.HasPrefix(apiPath, "*") {
		suffix := strings.TrimPrefix(apiPath, "*")
		return strings.HasSuffix(requestPath, suffix)
	}

	// 中间通配符：/api/*/logs
	if strings.Count(apiPath, "*") == 1 {
		parts := strings.Split(apiPath, "*")
		return strings.HasPrefix(requestPath, parts[0]) && strings.HasSuffix(requestPath, parts[1])
	}

	return false
}

func (am *AuthMiddleware) CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if skipAuthPaths[path] {
			c.Next()
			return
		}

		// 跳过静态资源和WebSocket路径
		if path == "/" || path == "/health" || path == "/ready" {
			c.Next()
			return
		}

		// 获取用户信息（兼容nginx代理）
		userVal, exists := c.Get("user")
		if !exists {
			// 兼容未登录时放行登录接口
			if skipAuthPaths[path] {
				c.Next()
				return
			}
			base.ForbiddenError(c, "未登录或登录已过期")
			c.Abort()
			return
		}
		user, ok := userVal.(jwt.UserClaims)
		if !ok {
			base.ForbiddenError(c, "用户信息异常")
			c.Abort()
			return
		}

		if path == "/api/user/signup" && user.Username != "admin" {
			base.ForbiddenError(c, "仅管理员可创建用户")
			c.Abort()
			return
		}

		// 管理员放行
		if user.Username == "admin" {
			c.Next()
			return
		}

		// 获取HTTP方法代码
		method := c.Request.Method
		methodCode, exists := methodMapping[method]
		if !exists {
			base.ErrorWithMessage(c, "不支持的HTTP方法")
			c.Abort()
			return
		}

		roles, err := am.roleService.GetUserRoles(c, user.Uid)
		if err != nil {
			base.ErrorWithMessage(c, "获取用户角色失败")
			c.Abort()
			return
		}

		for _, role := range roles.Items {
			// 跳过禁用角色
			if role.Status != 1 {
				continue
			}

			// 检查API权限
			for _, api := range role.Apis {
				if (api.Path == c.FullPath() && api.Method == methodCode) || matchWildcardPath(api.Path, path, methodCode, api.Method) {
					c.Next()
					return
				}
			}
		}

		// 无权限访问
		base.ForbiddenError(c, "无权限访问该接口")
		c.Abort()
	}
}
