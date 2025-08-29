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
	"go.uber.org/zap"
)

type RBACAPI struct {
	rbacService *service.RBACService
	logger      *zap.Logger
}

func NewRBACAPI(rbacService *service.RBACService, logger *zap.Logger) *RBACAPI {
	return &RBACAPI{
		rbacService: rbacService,
		logger:      logger,
	}
}

func (ra *RBACAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/v1/k8s")

	rbac := k8sGroup.Group("/rbac")
	{
		rbac.GET("/statistics/:cluster_id", ra.GetRBACStatistics)               // 获取RBAC统计信息
		rbac.POST("/check-permissions", ra.CheckPermissions)                    // 检查权限
		rbac.POST("/subject-permissions/:cluster_id", ra.GetSubjectPermissions) // 获取主体权限
		rbac.GET("/resource-verbs", ra.GetResourceVerbs)                        // 获取资源动作列表
	}
}

// GetRBACStatistics 获取RBAC统计信息
// @Summary 获取RBAC统计信息
// @Description 获取指定集群的RBAC统计信息，包括角色、绑定和主体数量
// @Tags RBAC 统计和权限检查
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=model.RBACStatistics}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/rbac/statistics/{cluster_id} [get]
func (ra *RBACAPI) GetRBACStatistics(c *gin.Context) {
	clusterIDStr := c.Param("cluster_id")
	clusterID, err := strconv.Atoi(clusterIDStr)
	if err != nil {
		utils.BadRequestError(c, "invalid cluster_id")
		return
	}

	result, err := ra.rbacService.GetRBACStatistics(c.Request.Context(), clusterID)
	if err != nil {
		ra.logger.Error("获取RBAC统计信息失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取RBAC统计信息失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CheckPermissions 检查权限
// @Summary 检查权限
// @Description 检查指定主体对资源的访问权限
// @Tags RBAC 统计和权限检查
// @Accept json
// @Produce json
// @Param permissions body model.CheckPermissionsReq true "权限检查信息"
// @Success 200 {object} utils.ApiResponse{data=[]model.PermissionResult}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/rbac/check-permissions [post]
func (ra *RBACAPI) CheckPermissions(c *gin.Context) {
	var req model.CheckPermissionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := ra.rbacService.CheckPermissions(c.Request.Context(), &req)
	if err != nil {
		ra.logger.Error("检查权限失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "检查权限失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetSubjectPermissions 获取主体的有效权限列表
// @Summary 获取主体的有效权限列表
// @Description 获取指定主体在集群中的所有有效权限
// @Tags RBAC 统计和权限检查
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param subject body model.Subject true "主体信息"
// @Success 200 {object} utils.ApiResponse{data=model.SubjectPermissionsResponse}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/rbac/subject-permissions/{cluster_id} [post]
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
		ra.logger.Error("获取主体权限失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取主体权限失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetResourceVerbs 获取预定义的资源和动作列表
// @Summary 获取预定义的资源和动作列表
// @Description 获取Kubernetes中预定义的资源类型和可用动作列表
// @Tags RBAC 统计和权限检查
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=model.ResourceVerbsResponse}
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/rbac/resource-verbs [get]
func (ra *RBACAPI) GetResourceVerbs(c *gin.Context) {
	result, err := ra.rbacService.GetResourceVerbs(c.Request.Context())
	if err != nil {
		ra.logger.Error("获取资源动作列表失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取资源动作列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}
