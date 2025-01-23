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
	"net/http"
	"strconv"
	"strings"

	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
)

type CasbinMiddleware struct {
	enforcer *casbin.Enforcer
}

func NewCasbinMiddleware(enforcer *casbin.Enforcer) *CasbinMiddleware {
	return &CasbinMiddleware{
		enforcer: enforcer,
	}
}

func (cm *CasbinMiddleware) CheckCasbin() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		// 如果请求的路径是下述路径，则不进行权限验证
		if path == "/api/user/login" ||
			path == "/api/user/logout" ||
			strings.Contains(path, "hello") ||
			path == "/api/user/refresh_token" ||
			path == "/api/user/codes" ||
			path == "/api/user/signup" ||
			path == "/api/not_auth/getTreeNodeBindIps" ||
			path == "/api/monitor/prometheus_configs/prometheus" ||
			path == "/api/monitor/prometheus_configs/prometheus_alert" ||
			path == "/api/monitor/prometheus_configs/prometheus_record" ||
			path == "/api/monitor/prometheus_configs/alertManager" {
			c.Next()
			return
		}

		// 获取用户身份
		userClaims, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    1,
				"message": "User not authenticated",
			})
			c.Abort()
			return
		}

		sub, ok := userClaims.(ijwt.UserClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    1,
				"message": "Invalid user claims",
			})
			c.Abort()
			return
		}

		if sub.Uid == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    1,
				"message": "Invalid user ID",
			})
			c.Abort()
			return
		}

		userIDStr := strconv.Itoa(sub.Uid)
		act := c.Request.Method

		cm.enforcer.LoadPolicy()

		// 使用 Casbin 检查权限
		ok, err := cm.enforcer.Enforce(userIDStr, path, act)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1,
				"message": "Error occurred when enforcing policy",
			})
			c.Abort()
			return
		}

		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    1,
				"message": "You don't have permission to access this resource",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
