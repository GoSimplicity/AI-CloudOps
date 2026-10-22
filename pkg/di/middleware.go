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

package di

import (
	"os"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func InitMiddlewares(ih ijwt.Handler, l *zap.Logger, roleSvc service.RoleService, auditSvc service.AuditService, userSvc service.UserService) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowCredentials: true,
			AllowHeaders:     []string{"Content-Type", "Authorization", "X-Refresh-Token"},
			ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
			AllowOriginFunc: func(origin string) bool {
				for _, allowed := range strings.Split(os.Getenv("SERVER_ALLOWED_ORIGINS"), ",") {
					if strings.TrimSpace(allowed) == origin {
						return true
					}
				}
				return false
			},
			MaxAge: 12 * time.Hour,
		}),
		middleware.NewJWTMiddleware(ih, userSvc).CheckLogin(),
		middleware.NewLogMiddleware(l).Log(),
		middleware.NewAuthMiddleware(roleSvc).CheckAuth(),
		middleware.NewAuditLogMiddleware(auditSvc, l).AuditLog(),
	}
}
