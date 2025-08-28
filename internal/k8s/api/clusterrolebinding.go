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
	k8sGroup := server.Group("/api/v1/k8s")

	clusterRoleBindings := k8sGroup.Group("/cluster-role-binding")
	{
		clusterRoleBindings.GET("/list", crba.GetClusterRoleBindingList)                         // 获取ClusterRoleBinding列表
		clusterRoleBindings.GET("/details/:cluster_id/:name", crba.GetClusterRoleBindingDetails) // 获取ClusterRoleBinding详情
		clusterRoleBindings.POST("/create", crba.CreateClusterRoleBinding)                       // 创建ClusterRoleBinding
		clusterRoleBindings.PUT("/update", crba.UpdateClusterRoleBinding)                        // 更新ClusterRoleBinding
		clusterRoleBindings.DELETE("/delete/:cluster_id/:name", crba.DeleteClusterRoleBinding)   // 删除ClusterRoleBinding
		clusterRoleBindings.POST("/batch-delete", crba.BatchDeleteClusterRoleBinding)            // 批量删除ClusterRoleBinding
		clusterRoleBindings.GET("/yaml/:cluster_id/:name", crba.GetClusterRoleBindingYaml)       // 获取ClusterRoleBinding YAML
		clusterRoleBindings.PUT("/yaml", crba.UpdateClusterRoleBindingYaml)                      // 更新ClusterRoleBinding YAML
	}
}

// GetClusterRoleBindingList 获取ClusterRoleBinding列表
// @Summary 获取ClusterRoleBinding列表
// @Description 获取指定集群中的ClusterRoleBinding列表，支持分页和关键字搜索
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} apiresponse.ApiResponse{data=model.ListResp[model.ClusterRoleBindingInfo]}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/list [get]
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
// @Summary 获取ClusterRoleBinding详情
// @Description 获取指定ClusterRoleBinding的详细信息
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse{data=model.ClusterRoleBindingInfo}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 404 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/details/{cluster_id}/{name} [get]
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
// @Summary 创建ClusterRoleBinding
// @Description 在指定集群中创建新的ClusterRoleBinding
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param clusterrolebinding body model.CreateClusterRoleBindingReq true "ClusterRoleBinding创建信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/create [post]
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
// @Summary 更新ClusterRoleBinding
// @Description 更新指定ClusterRoleBinding的配置信息
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param clusterrolebinding body model.UpdateClusterRoleBindingReq true "ClusterRoleBinding更新信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/update [put]
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
// @Summary 删除ClusterRoleBinding
// @Description 删除指定的ClusterRoleBinding
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/delete/{cluster_id}/{name} [delete]
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

// BatchDeleteClusterRoleBinding 批量删除ClusterRoleBinding
// @Summary 批量删除ClusterRoleBinding
// @Description 批量删除指定的多个ClusterRoleBinding
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param clusterrolebindings body model.BatchDeleteClusterRoleBindingReq true "批量删除ClusterRoleBinding信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/batch-delete [post]
func (crba *ClusterRoleBindingAPI) BatchDeleteClusterRoleBinding(c *gin.Context) {
	var req model.BatchDeleteClusterRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := crba.clusterRoleBindingService.BatchDeleteClusterRoleBinding(c.Request.Context(), &req)
	if err != nil {
		crba.logger.Error("批量删除ClusterRoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "批量删除ClusterRoleBinding失败")
		return
	}

	utils.Success(c)
}

// GetClusterRoleBindingYaml 获取ClusterRoleBinding的YAML配置
// @Summary 获取ClusterRoleBinding的YAML配置
// @Description 获取指定ClusterRoleBinding的YAML格式配置
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse{data=string}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/yaml/{cluster_id}/{name} [get]
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
// @Summary 更新ClusterRoleBinding的YAML配置
// @Description 通过YAML更新指定ClusterRoleBinding的配置
// @Tags RBAC ClusterRoleBinding管理
// @Accept json
// @Produce json
// @Param yaml body model.ClusterRoleBindingYamlReq true "ClusterRoleBinding YAML更新信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role-binding/yaml [put]
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
