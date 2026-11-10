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
	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// InitMiddlewares 只保留访问日志。
// webhook 没有浏览器跨域调用方：Alertmanager 的 /receive 是服务端请求，
// 飞书卡片是顶层导航，确认页表单与页面同源。原先 AllowOriginFunc 对任意 Origin 返回 true
// 且 AllowCredentials 为 true，因此这里不再挂 CORS。
func InitMiddlewares(l *zap.Logger) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewLogMiddleware(l).Log(),
	}
}
