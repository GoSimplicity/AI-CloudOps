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
 */

package api

import (
	"net/http"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type InternalHandler struct {
	roleService service.RoleService
}

func NewInternalHandler(roleService service.RoleService) *InternalHandler {
	return &InternalHandler{
		roleService: roleService,
	}
}

type PermissionCheckRequest struct {
	UserID int    `json:"user_id" binding:"required"`
	Path   string `json:"path" binding:"required"`
	Method string `json:"method" binding:"required"`
}

type PermissionCheckResponse struct {
	Allowed bool   `json:"allowed"`
	Message string `json:"message,omitempty"`
}

// HTTP方法映射
var methodMapping = map[string]int8{
	"GET":    1,
	"POST":   2,
	"PUT":    3,
	"DELETE": 4,
}

// CheckPermission 检查用户权限
func (h *InternalHandler) CheckPermission(c *gin.Context) {
	// 验证内部请求
	internalFlag := c.GetHeader("X-Internal-Request")
	if internalFlag != "true" {
		utils.ErrorWithMessage(c, "禁止外部访问")
		return
	}

	var req PermissionCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 管理员用户直接放行
	if req.UserID == 1 { // 假设管理员用户ID为1
		c.JSON(http.StatusOK, PermissionCheckResponse{
			Allowed: true,
			Message: "管理员用户",
		})
		return
	}

	// 获取HTTP方法代码
	methodCode, exists := methodMapping[req.Method]
	if !exists {
		c.JSON(http.StatusOK, PermissionCheckResponse{
			Allowed: false,
			Message: "不支持的HTTP方法",
		})
		return
	}

	// 获取用户角色
	roles, err := h.roleService.GetUserRoles(c, req.UserID)
	if err != nil {
		utils.ErrorWithMessage(c, "获取用户角色失败: "+err.Error())
		return
	}

	// 检查权限
	allowed := false
	var message string

	for _, role := range roles.Items {
		// 跳过禁用角色
		if role.Status != 1 {
			continue
		}

		// 检查API权限
		for _, api := range role.Apis {
			if h.matchWildcardPath(api.Path, req.Path, methodCode, api.Method) {
				allowed = true
				message = "权限检查通过"
				break
			}
		}

		if allowed {
			break
		}
	}

	if !allowed {
		message = "无权限访问该接口"
	}

	c.JSON(http.StatusOK, PermissionCheckResponse{
		Allowed: allowed,
		Message: message,
	})
}

// matchWildcardPath 检查通配符路径匹配
func (h *InternalHandler) matchWildcardPath(apiPath, requestPath string, methodCode int8, apiMethod int8) bool {
	// 方法不匹配则返回false
	if apiMethod != methodCode {
		return false
	}

	// 完全匹配
	if apiPath == requestPath {
		return true
	}

	// 全局通配符匹配所有路径
	if apiPath == "/*" {
		return true
	}

	// 不包含通配符直接返回
	if !strings.Contains(apiPath, "*") {
		return false
	}

	// 末尾通配符：/api/user/*
	if strings.HasSuffix(apiPath, "*") {
		prefix := strings.TrimSuffix(apiPath, "*")
		return strings.HasPrefix(requestPath, prefix)
	}

	// 开头通配符：*/logs
	if strings.HasPrefix(apiPath, "*") {
		suffix := strings.TrimPrefix(apiPath, "*")
		return strings.HasSuffix(requestPath, suffix)
	}

	// 中间通配符：/api/*/logs
	if strings.Count(apiPath, "*") == 1 {
		parts := strings.Split(apiPath, "*")
		return strings.HasPrefix(requestPath, parts[0]) && strings.HasSuffix(requestPath, parts[1])
	}

	return false
}

func RegisterInternalRoutes(r *gin.Engine, handler *InternalHandler) {
	internal := r.Group("/api/internal")
	{
		internal.POST("/check_permission", handler.CheckPermission)
	}
}
