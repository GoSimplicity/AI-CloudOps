package di

import (
	casbinDao "github.com/GoSimplicity/CloudOps/internal/auth/dao/casbin"
	userDao "github.com/GoSimplicity/CloudOps/internal/user/dao"
	middleware2 "github.com/GoSimplicity/CloudOps/pkg/middleware"
	ijwt "github.com/GoSimplicity/CloudOps/pkg/utils/jwt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strings"
	"time"
)

// InitMiddlewares 初始化中间件
func InitMiddlewares(ih ijwt.Handler, l *zap.Logger, userDao userDao.UserDAO, casbinDao casbinDao.CasbinDAO) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		cors.New(cors.Config{
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowCredentials: true,
			AllowHeaders:     []string{"Content-Type", "Authorization", "X-Refresh-Token"},
			ExposeHeaders:    []string{"x-jwt-token", "x-refresh-token"},
			AllowOriginFunc: func(origin string) bool {
				if strings.HasPrefix(origin, "http://localhost") {
					return true
				}
				return strings.Contains(origin, "")
			},
			MaxAge: 12 * time.Hour,
		}),
		middleware2.NewJWTMiddleware(ih).CheckLogin(),
		middleware2.NewLogMiddleware(l).Log(),
	}
}
