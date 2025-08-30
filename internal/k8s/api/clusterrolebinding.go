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

type ClusterRoleBindingAPI struct {
	clusterRoleBindingService *service.ClusterRoleBindingService
	logger                    *zap.Logger
}

func NewClusterRoleBindingAPI(clusterRoleBindingService *service.ClusterRoleBindingService, logger *zap.Logger) *ClusterRoleBindingAPI {
	return &ClusterRoleBindingAPI{
		clusterRoleBindingService: clusterRoleBindingService,
		logger:                    logger,
	}
}

func (crba *ClusterRoleBindingAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	{
		k8sGroup.GET("/cluster-role-binding/list", crba.GetClusterRoleBindingList)                         // 获取ClusterRoleBinding列表
		k8sGroup.GET("/cluster-role-binding/details/:cluster_id/:name", crba.GetClusterRoleBindingDetails) // 获取ClusterRoleBinding详情
		k8sGroup.POST("/cluster-role-binding/create", crba.CreateClusterRoleBinding)                       // 创建ClusterRoleBinding
		k8sGroup.PUT("/cluster-role-binding/update", crba.UpdateClusterRoleBinding)                        // 更新ClusterRoleBinding
		k8sGroup.DELETE("/cluster-role-binding/delete/:cluster_id/:name", crba.DeleteClusterRoleBinding)   // 删除ClusterRoleBinding
		k8sGroup.GET("/cluster-role-binding/yaml/:cluster_id/:name", crba.GetClusterRoleBindingYaml)       // 获取ClusterRoleBinding YAML
		k8sGroup.PUT("/cluster-role-binding/yaml", crba.UpdateClusterRoleBindingYaml)                      // 更新ClusterRoleBinding YAML
	}
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表
func (crba *ClusterRoleBindingAPI) GetClusterRoleBindingList(c *gin.Context) {
	var req model.ClusterRoleBindingListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := crba.clusterRoleBindingService.GetClusterRoleBindingList(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("获取ClusterRoleBinding列表失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRoleBinding列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetClusterRoleBindingDetails 获取ClusterRoleBinding详情
func (crba *ClusterRoleBindingAPI) GetClusterRoleBindingDetails(c *gin.Context) {
	var req model.ClusterRoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := crba.clusterRoleBindingService.GetClusterRoleBindingDetails(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("获取ClusterRoleBinding详情失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRoleBinding详情失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
func (crba *ClusterRoleBindingAPI) CreateClusterRoleBinding(c *gin.Context) {
	var req model.CreateClusterRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := crba.clusterRoleBindingService.CreateClusterRoleBinding(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("创建ClusterRoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "创建ClusterRoleBinding失败")
		return
	}

	utils.Success(c)
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
func (crba *ClusterRoleBindingAPI) UpdateClusterRoleBinding(c *gin.Context) {
	var req model.UpdateClusterRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := crba.clusterRoleBindingService.UpdateClusterRoleBinding(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("更新ClusterRoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新ClusterRoleBinding失败")
		return
	}

	utils.Success(c)
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
func (crba *ClusterRoleBindingAPI) DeleteClusterRoleBinding(c *gin.Context) {
	var req model.DeleteClusterRoleBindingReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := crba.clusterRoleBindingService.DeleteClusterRoleBinding(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("删除ClusterRoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "删除ClusterRoleBinding失败")
		return
	}

	utils.Success(c)
}

// GetClusterRoleBindingYaml 获取ClusterRoleBinding的YAML配置
func (crba *ClusterRoleBindingAPI) GetClusterRoleBindingYaml(c *gin.Context) {
	var req model.ClusterRoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	yamlContent, err := crba.clusterRoleBindingService.GetClusterRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("获取ClusterRoleBinding YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取ClusterRoleBinding YAML失败")
		return
	}

	utils.SuccessWithData(c, yamlContent)
}

// UpdateClusterRoleBindingYaml 更新ClusterRoleBinding的YAML配置
func (crba *ClusterRoleBindingAPI) UpdateClusterRoleBindingYaml(c *gin.Context) {
	var req model.ClusterRoleBindingYamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := crba.clusterRoleBindingService.UpdateClusterRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("更新ClusterRoleBinding YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新ClusterRoleBinding YAML失败")
		return
	}

	utils.Success(c)
}
