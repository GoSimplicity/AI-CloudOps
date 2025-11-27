package utils

import (
	"fmt"

	pkgutils "github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

// ExtractClaims 从 gin 上下文安全提取登录用户信息
func ExtractClaims(ctx *gin.Context) (pkgutils.UserClaims, error) {
	value, exists := ctx.Get("user")
	if !exists {
		return pkgutils.UserClaims{}, fmt.Errorf("用户信息缺失")
	}

	claims, ok := value.(pkgutils.UserClaims)
	if !ok {
		return pkgutils.UserClaims{}, fmt.Errorf("用户信息格式错误")
	}

	return claims, nil
}
