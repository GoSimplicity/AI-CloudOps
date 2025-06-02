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

package api

import (
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type NotAuthHandler struct {
	svc service.NotAuthService
}

func NewNotAuthHandler(svc service.NotAuthService) *NotAuthHandler {
	return &NotAuthHandler{
		svc: svc,
	}
}

func (n *NotAuthHandler) RegisterRouters(server *gin.Engine) {
	notAuthGroup := server.Group("/api/not_auth")
	notAuthGroup.GET("/getBindIps", n.GetBindIps)
}

func (n *NotAuthHandler) GetBindIps(ctx *gin.Context) {
	ipAddress := ctx.Query("ipAddress")
	if ipAddress == "" {
		utils.ErrorWithMessage(ctx, "参数错误")
		return
	}

	res, err := n.svc.BuildPrometheusServiceDiscovery(ctx, ipAddress)
	if err != nil {
		utils.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	ctx.JSON(http.StatusOK, res)
}
