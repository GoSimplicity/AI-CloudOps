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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

// K8sRBACHandler RBAC权限管理处理器
type K8sRBACHandler struct {
	rbacService service.RBACService
}

func NewK8sRBACHandler(rbacService service.RBACService) *K8sRBACHandler {
	return &K8sRBACHandler{
		rbacService: rbacService,
	}
}

// RegisterRouters 注册路由
func (h *K8sRBACHandler) RegisterRouters(server *gin.Engine) {
	rbacGroup := server.Group("/api/k8s")
	{
		// RBAC权限分析
		rbacGroup.POST("/rbac/:cluster_id/analyze", h.AnalyzeRBACPermissions)
		rbacGroup.POST("/rbac/:cluster_id/check-permission", h.CheckRBACPermission)
	}
}

// AnalyzeRBACPermissions 分析RBAC权限
func (h *K8sRBACHandler) AnalyzeRBACPermissions(ctx *gin.Context) {
	var req model.AnalyzeRBACPermissionsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rbacService.AnalyzeRBACPermissions(ctx, &req)
	})
}

func (h *K8sRBACHandler) CheckRBACPermission(ctx *gin.Context) {
	var req model.CheckRBACPermissionReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rbacService.CheckRBACPermission(ctx, &req)
	})
}
