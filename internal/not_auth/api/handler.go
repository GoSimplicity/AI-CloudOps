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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
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
	// 兼容旧地址
	notAuthGroup.GET("/getTreeNodeBindIps", n.GetBindIps)
}

// GetBindIps 获取绑定IP地址
func (n *NotAuthHandler) GetBindIps(ctx *gin.Context) {
	if len(ctx.Request.URL.RawQuery) > service.MaxDiscoveryQueryLength {
		base.BadRequest(ctx, base.StatusError, nil, "查询参数过长")
		return
	}
	port, err := base.GetQueryParam[int](ctx, "port")
	if err != nil {
		base.BadRequest(ctx, base.StatusError, nil, "port 必须是整数")
		return
	}

	idsStr, err := base.GetQueryParam[string](ctx, "tree_node_ids")
	if err != nil || idsStr == "" {
		base.BadRequest(ctx, base.StatusError, nil, "参数错误: 缺少tree_node_ids")
		return
	}

	if len(idsStr) > service.MaxDiscoveryIDsLength {
		base.BadRequest(ctx, base.StatusError, nil, "tree_node_ids 过长")
		return
	}
	// 解析后统一去重与限量，拒绝无效片段，避免悄悄进行部分查询
	var ids []int
	for _, seg := range strings.Split(idsStr, ",") {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		num, convErr := strconv.Atoi(seg)
		if convErr != nil {
			base.BadRequest(ctx, base.StatusError, nil, "tree_node_ids 必须是正整数")
			return
		}
		ids = append(ids, num)
	}

	ids, err = service.NormalizeDiscoveryNodeIDs(ids)
	if err != nil {
		base.BadRequest(ctx, base.StatusError, nil, err.Error())
		return
	}
	if port < 1 || port > 65535 {
		base.BadRequest(ctx, base.StatusError, nil, "端口无效")
		return
	}

	res, err := n.svc.BuildPrometheusServiceDiscovery(ctx.Request.Context(), port, ids)
	if err != nil {
		base.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	ctx.JSON(http.StatusOK, res)
}
