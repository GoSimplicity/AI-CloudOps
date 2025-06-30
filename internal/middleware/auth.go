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
	"/api/user/login":         true,
	"/api/user/logout":        true,
	"/api/user/refresh_token": true,
	"/api/user/signup":        true,
	"/api/user/profile":       true,
	"/api/user/codes":         true,
}

// 预定义静态资源和WebSocket路径前缀
var skipPrefixes = []string{
	"/api/ai/chat/ws",
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

// 检查路径是否以指定前缀开头
func hasPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// 检查API路径是否匹配通配符路径
func matchWildcardPath(apiPath, requestPath string, methodCode int8, apiMethod int8) bool {
	if apiMethod != methodCode {
		return false
	}

	// 完全匹配
	if apiPath == requestPath {
		return true
	}

	// 不包含通配符，无需进一步检查
	if !strings.Contains(apiPath, "*") {
		return false
	}

	// 处理末尾是*的情况，如/api/user/*
	if strings.HasSuffix(apiPath, "*") {
		prefix := strings.TrimSuffix(apiPath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// 处理开头是*的情况，如*/logs
	if strings.HasPrefix(apiPath, "*") {
		suffix := strings.TrimPrefix(apiPath, "*")
		return strings.HasSuffix(requestPath, suffix)
	}

	// 处理中间带*的情况，如/api/*/logs
	if strings.Count(apiPath, "*") == 1 {
		parts := strings.Split(apiPath, "*")
		return strings.HasPrefix(requestPath, parts[0]) && strings.HasSuffix(requestPath, parts[1])
	}

	return false
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
		if path == "/" || hasPrefix(path, skipPrefixes) {
			c.Next()
			return
		}

		user := c.MustGet("user").(utils.UserClaims)
		// 管理员直接放行
		if user.Username == "admin" {
			c.Next()
			return
		}

		// 服务账号直接放行
		if user.AccountType == 2 {
			c.Next()
			return
		}

		// 获取HTTP方法代码
		method := c.Request.Method
		methodCode, exists := methodMapping[method]
		if !exists {
			utils.ErrorWithMessage(c, "不支持的HTTP方法")
			c.Abort()
			return
		}

		// 获取用户角色
		roles, err := am.roleService.GetUserRoles(c, user.Uid)
		if err != nil {
			utils.ErrorWithMessage(c, "获取用户角色失败")
			c.Abort()
			return
		}

		// 检查用户是否有权限访问当前API
		for _, role := range roles.Items {
			// 跳过禁用的角色
			if role.Status != 1 {
				continue
			}

			// 检查角色是否有权限访问当前API
			for _, api := range role.Apis {
				if matchWildcardPath(api.Path, path, methodCode, api.Method) {
					c.Next()
					return
				}
			}
		}

		// 没有找到匹配的权限
		utils.ForbiddenError(c, "无权限访问该接口")
		c.Abort()
	}
}
