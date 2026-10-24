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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/permission"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	roleService service.RoleService
}

func NewAuthMiddleware(roleService service.RoleService) *AuthMiddleware {
	return &AuthMiddleware{
		roleService: roleService,
	}
}

// CheckAuth 按“HTTP 方法 + gin 路由模板（c.FullPath()）”对角色授予做精确比较。
// 授予中的 /* 与末尾 * 只对模板做前缀匹配，:param 不再被当作通配符；
// 匹配规则见 internal/permission 与 docs/api/user-permissions-v2.md。
func (am *AuthMiddleware) CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if permission.SkipsAuthorization(path) {
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

		// 特权只来自内置超管角色（code=super_admin 且 is_system=1 且启用），不再按用户名判断
		roles, err := am.roleService.GetUserRoles(c, user.Uid)
		if err != nil {
			base.ErrorWithMessage(c, "获取用户角色失败")
			c.Abort()
			return
		}
		if roles == nil {
			roles = &model.ListResp[*model.Role]{}
		}
		super := permission.HasSuperAdminRole(roles.Items)
		permission.MarkSuperAdmin(c, super)

		if path == "/api/user/signup" && !super {
			base.ForbiddenError(c, "仅管理员可创建用户")
			c.Abort()
			return
		}

		// 超级管理员放行（未命中路由时交给 gin 返回 404）
		if super {
			c.Next()
			return
		}

		// 获取HTTP方法代码
		method := c.Request.Method
		methodCode, exists := permission.MethodCode(method)
		if !exists {
			base.ErrorWithMessage(c, "不支持的HTTP方法")
			c.Abort()
			return
		}

		// 未命中任何路由（404）时 FullPath 为空：对非超管统一返回 403，
		// 既不放行也不暴露路由是否存在。
		fullPath := c.FullPath()
		if fullPath == "" {
			base.ForbiddenError(c, "无权限访问该接口")
			c.Abort()
			return
		}

		// 代理通配入口（AIOps）借助接口目录把真实路径还原为具体模板
		resolved := permission.ResolveProxyTemplate(method, fullPath, path)
		if permission.Allowed(permission.GrantsFromRoles(roles.Items), methodCode, fullPath, resolved) {
			c.Next()
			return
		}

		// 无权限访问
		base.ForbiddenError(c, "无权限访问该接口")
		c.Abort()
	}
}
