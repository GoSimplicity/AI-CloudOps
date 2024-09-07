package ginp

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// WrapBody 是一个中间件，用于包裹业务逻辑函数，自动绑定请求体、处理响应并集中管理错误处理。
func WrapBody[Req any](bizFn func(ctx *gin.Context, req Req) (Result, error)) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req Req
		// 使用 ShouldBindJSON 替换 Bind，以便于更好地处理错误，避免直接 panic
		if err := ctx.ShouldBindJSON(&req); err != nil {
			// 当请求体解析失败时，返回适当的HTTP错误响应而非 panic
			ctx.AbortWithStatusJSON(http.StatusBadRequest, Result{
				Msg: "无效的请求负载",
			})
			return
		}

		res, err := bizFn(ctx, req)
		if err != nil {
			// 记录错误（在生产环境中建议使用结构化日志）
			log.Printf("执行业务逻辑时发生错误: %v", err)
			// 根据应用需求自定义错误响应
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "服务器内部错误"})
			return
		}

		// 成功处理，返回结果
		ctx.JSON(http.StatusOK, res)
	}
}
