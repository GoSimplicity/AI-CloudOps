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
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AccessLog struct {
	Path     string        `json:"path"`     // 请求路径
	Method   string        `json:"method"`   // 请求方法
	ReqBody  string        `json:"reqBody"`  // 请求体内容
	Status   int           `json:"status"`   // 响应状态码
	RespBody string        `json:"respBody"` // 响应体内容
	Duration time.Duration `json:"duration"` // 请求处理耗时
}

type LogMiddleware struct {
	l *zap.Logger
}

func NewLogMiddleware(l *zap.Logger) *LogMiddleware {
	return &LogMiddleware{
		l: l,
	}
}

// Log 日志中间件
func (lm *LogMiddleware) Log() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()
		// 请求路径
		path := c.Request.URL.Path
		// 请求方法
		method := c.Request.Method
		// 读取请求体
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			lm.l.Error("请求体读取失败", zap.Error(err))
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		// 由于读取请求体会消耗掉c.Request.Body，所以需要重新设置回上下文
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		al := AccessLog{
			Path:    path,
			Method:  method,
			ReqBody: string(bodyBytes),
		}
		c.Next()
		// 记录响应状态码和响应体
		al.Status = c.Writer.Status()
		al.RespBody = c.Writer.Header().Get("Content-Type")
		al.Duration = time.Since(start)
		lm.l.Info("请求日志", zap.Any("accessLog", al))
	}
}
