package middleware

import (
	"io"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// SafeRecovery 只记录固定路由与栈，不转储请求、认证头或 panic 原文
func SafeRecovery(output io.Writer) gin.HandlerFunc {
	logger := log.New(output, "[Recovery] ", log.LstdFlags)
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				stack := debug.Stack()
				if len(stack) > 16*1024 {
					stack = stack[:16*1024]
				}
				logger.Printf("method=%s route=%q panic_type=%T\n%s", c.Request.Method, c.FullPath(), recovered, stack)
				if c.Writer.Written() {
					c.Abort()
					return
				}
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务内部错误", "data": nil})
			}
		}()
		c.Next()
	}
}
