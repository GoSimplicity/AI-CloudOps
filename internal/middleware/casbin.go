package middleware

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

import (
	casbinDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CasbinMiddleware 结构体，负责通过 Casbin 检查用户权限
type CasbinMiddleware struct {
	l         *zap.Logger         // Logger 用于日志记录
	userDAO   userDao.UserDAO     // 用户 DAO，用于获取用户信息
	casbinDAO casbinDao.CasbinDAO // Casbin DAO，用于权限检查
}

// NewCasbinMiddleware 创建一个新的 CasbinMiddleware 实例
func NewCasbinMiddleware(l *zap.Logger, userDAO userDao.UserDAO, casbinDAO casbinDao.CasbinDAO) *CasbinMiddleware {
	return &CasbinMiddleware{
		l:         l,
		userDAO:   userDAO,
		casbinDAO: casbinDAO,
	}
}

// CheckPermission 通过 Casbin 检查权限的中间件
func (m *CasbinMiddleware) CheckPermission() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 获取经过身份验证的用户信息
		uc := ctx.MustGet("user").(ijwt.UserClaims)

		// 根据用户 ID 获取用户详细信息
		user, err := m.userDAO.GetUserByID(ctx, uc.Uid)
		if err != nil {
			m.l.Error("get user by id failed", zap.Error(err))
			apiresponse.InternalServerError(ctx, 0, nil, "failed to retrieve user info") // 优化: 返回适当的错误响应
			ctx.Abort()
			return
		}

		// 如果用户是服务账户（AccountType == 2），跳过权限检查，直接允许访问
		if user.AccountType == 2 {
			ctx.Next()
			return
		}

		// 获取当前请求的路径和方法
		path := ctx.Request.URL.Path
		method := ctx.Request.Method

		// 检查用户的每个角色是否有权限
		pass := false
		for _, role := range user.Roles {
			// 使用 Casbin 检查角色是否有权限访问该路径
			permission, err := m.casbinDAO.CheckPermission(ctx, role.RoleValue, path, method)
			if err != nil {
				m.l.Error("casbin permission check failed", zap.Error(err))
				apiresponse.InternalServerError(ctx, 0, nil, "failed to check permission") // 优化: 返回适当的错误响应
				ctx.Abort()
				return
			}

			// 记录 Casbin 校验结果
			m.l.Debug("Casbin permission check result",
				zap.Int("userID", uc.Uid),
				zap.String("RoleValue", role.RoleValue),
				zap.String("path", path),
				zap.String("method", method),
				zap.Bool("permission", permission))

			// 如果其中一个角色有权限，则停止进一步检查
			if permission {
				pass = true
				break
			}
		}

		// 如果没有角色通过权限检查，则返回 403 Forbidden
		if !pass {
			apiresponse.Forbidden(ctx, 1, nil, "no permission")
			ctx.Abort()
		} else {
			// 权限校验通过，继续处理请求
			ctx.Next()
		}
	}
}
