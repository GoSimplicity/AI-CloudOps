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
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
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
	notAuthGroup.GET("/getTreeNodeBindIps", n.GetTreeNodeBindIps)
}

func (n *NotAuthHandler) GetTreeNodeBindIps(ctx *gin.Context) {
	// 获取和验证 leafNodeIds
	leafNodeIds := ctx.DefaultQuery("leafNodeIds", "")
	if leafNodeIds == "" {
		apiresponse.BadRequestError(ctx, "leafNodeIds 参数不能为空")
		return
	}

	leafNodeIdList := strings.Split(leafNodeIds, ",")
	if len(leafNodeIdList) == 0 {
		apiresponse.BadRequestError(ctx, "leafNodeIds 参数格式无效")
		return
	}

	// 获取和验证 port
	port := ctx.DefaultQuery("port", "")
	if port == "" {
		apiresponse.BadRequestError(ctx, "port 参数不能为空")
		return
	}

	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 {
		apiresponse.BadRequestError(ctx, "port 必须为正整数")
		return
	}

	// 构建 Prometheus 服务发现结果
	res, err := n.svc.BuildPrometheusServiceDiscovery(ctx, leafNodeIdList, p)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	ctx.JSON(http.StatusOK, res)
}
