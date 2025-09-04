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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type RBACAPI struct {
	rbacService *service.RBACService
}

func NewRBACAPI(rbacService *service.RBACService) *RBACAPI {
	return &RBACAPI{
		rbacService: rbacService,
	}
}

func (ra *RBACAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/rbac/statistics/:cluster_id", ra.GetRBACStatistics)               // 获取RBAC统计信息
		k8sGroup.POST("/rbac/check-permissions", ra.CheckPermissions)                    // 检查权限
		k8sGroup.POST("/rbac/subject-permissions/:cluster_id", ra.GetSubjectPermissions) // 获取主体权限
		k8sGroup.GET("/rbac/resource-verbs", ra.GetResourceVerbs)                        // 获取资源动作列表
	}
}

// GetRBACStatistics 获取RBAC统计信息
func (ra *RBACAPI) GetRBACStatistics(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(c, "invalid cluster_id")
		return
	}

	result, err := ra.rbacService.GetRBACStatistics(c.Request.Context(), clusterID)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取RBAC统计信息失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CheckPermissions 检查权限
func (ra *RBACAPI) CheckPermissions(c *gin.Context) {
	var req model.CheckPermissionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := ra.rbacService.CheckPermissions(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "检查权限失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetSubjectPermissions 获取主体的有效权限列表
func (ra *RBACAPI) GetSubjectPermissions(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(c, "invalid cluster_id")
		return
	}

	var subject model.Subject
	if err := c.ShouldBindJSON(&subject); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	req := &model.SubjectPermissionsReq{
		ClusterID: clusterID,
		Subject:   subject,
	}

	result, err := ra.rbacService.GetSubjectPermissions(c.Request.Context(), req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取主体权限失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetResourceVerbs 获取预定义的资源和动作列表
func (ra *RBACAPI) GetResourceVerbs(c *gin.Context) {
	result, err := ra.rbacService.GetResourceVerbs(c.Request.Context())
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取资源动作列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}
