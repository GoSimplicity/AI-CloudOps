package di

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	casbinDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao/casbin"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
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
		middleware.NewJWTMiddleware(ih).CheckLogin(),
		middleware.NewLogMiddleware(l).Log(),
	}
}
