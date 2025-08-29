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
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ClusterRoleAPI struct {
	clusterRoleService *service.ClusterRoleService
	logger             *zap.Logger
}

func NewClusterRoleAPI(clusterRoleService *service.ClusterRoleService, logger *zap.Logger) *ClusterRoleAPI {
	return &ClusterRoleAPI{
		clusterRoleService: clusterRoleService,
		logger:             logger,
	}
}

func (cra *ClusterRoleAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/v1/k8s")

	clusterRoles := k8sGroup.Group("/cluster-role")
	{
		clusterRoles.GET("/list", cra.GetClusterRoleList)                         // 获取ClusterRole列表
		clusterRoles.GET("/details/:cluster_id/:name", cra.GetClusterRoleDetails) // 获取ClusterRole详情
		clusterRoles.POST("/create", cra.CreateClusterRole)                       // 创建ClusterRole
		clusterRoles.PUT("/update", cra.UpdateClusterRole)                        // 更新ClusterRole
		clusterRoles.DELETE("/delete/:cluster_id/:name", cra.DeleteClusterRole)   // 删除ClusterRole

		clusterRoles.GET("/yaml/:cluster_id/:name", cra.GetClusterRoleYaml) // 获取ClusterRole YAML
		clusterRoles.PUT("/yaml", cra.UpdateClusterRoleYaml)                // 更新ClusterRole YAML
	}
}

// GetClusterRoleList 获取ClusterRole列表
func (cra *ClusterRoleAPI) GetClusterRoleList(c *gin.Context) {
	var req model.ClusterRoleListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := cra.clusterRoleService.GetClusterRoleList(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("获取ClusterRole列表失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRole列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetClusterRoleDetails 获取ClusterRole详情
func (cra *ClusterRoleAPI) GetClusterRoleDetails(c *gin.Context) {
	var req model.ClusterRoleGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := cra.clusterRoleService.GetClusterRoleDetails(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("获取ClusterRole详情失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRole详情失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CreateClusterRole 创建ClusterRole
func (cra *ClusterRoleAPI) CreateClusterRole(c *gin.Context) {
	var req model.CreateClusterRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := cra.clusterRoleService.CreateClusterRole(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("创建ClusterRole失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "创建ClusterRole失败")
		return
	}

	utils.Success(c)
}

// UpdateClusterRole 更新ClusterRole
func (cra *ClusterRoleAPI) UpdateClusterRole(c *gin.Context) {
	var req model.UpdateClusterRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := cra.clusterRoleService.UpdateClusterRole(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("更新ClusterRole失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新ClusterRole失败")
		return
	}

	utils.Success(c)
}

// DeleteClusterRole 删除ClusterRole
func (cra *ClusterRoleAPI) DeleteClusterRole(c *gin.Context) {
	var req model.DeleteClusterRoleReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := cra.clusterRoleService.DeleteClusterRole(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("删除ClusterRole失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "删除ClusterRole失败")
		return
	}

	utils.Success(c)
}

// GetClusterRoleYaml 获取ClusterRole的YAML配置
func (cra *ClusterRoleAPI) GetClusterRoleYaml(c *gin.Context) {
	var req model.ClusterRoleGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	yamlContent, err := cra.clusterRoleService.GetClusterRoleYaml(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("获取ClusterRole YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRole YAML失败")
		return
	}

	utils.SuccessWithData(c, yamlContent)
}

// UpdateClusterRoleYaml 更新ClusterRole的YAML配置
func (cra *ClusterRoleAPI) UpdateClusterRoleYaml(c *gin.Context) {
	var req model.ClusterRoleYamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := cra.clusterRoleService.UpdateClusterRoleYaml(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("更新ClusterRole YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新ClusterRole YAML失败")
		return
	}

	utils.Success(c)
}
