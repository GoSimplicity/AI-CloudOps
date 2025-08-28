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
		clusterRoles.POST("/batch-delete", cra.BatchDeleteClusterRole)            // 批量删除ClusterRole
		clusterRoles.GET("/yaml/:cluster_id/:name", cra.GetClusterRoleYaml)       // 获取ClusterRole YAML
		clusterRoles.PUT("/yaml", cra.UpdateClusterRoleYaml)                      // 更新ClusterRole YAML
	}
}

// GetClusterRoleList 获取ClusterRole列表
// @Summary 获取ClusterRole列表
// @Description 获取指定集群中的ClusterRole列表，支持分页和关键字搜索
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} utils.ApiResponse{data=model.ListResp[model.ClusterRoleBindingInfo]}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/list [get]
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
// @Summary 获取ClusterRole详情
// @Description 获取指定ClusterRole的详细信息
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRole名称"
// @Success 200 {object} utils.ApiResponse{data=model.ClusterRoleInfo}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 404 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/details/{cluster_id}/{name} [get]
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
// @Summary 创建ClusterRole
// @Description 在指定集群中创建新的ClusterRole
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param clusterrole body model.CreateClusterRoleReq true "ClusterRole创建信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/create [post]
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
// @Summary 更新ClusterRole
// @Description 更新指定ClusterRole的配置信息
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param clusterrole body model.UpdateClusterRoleReq true "ClusterRole更新信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/update [put]
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
// @Summary 删除ClusterRole
// @Description 删除指定的ClusterRole
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRole名称"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/delete/{cluster_id}/{name} [delete]
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

// BatchDeleteClusterRole 批量删除ClusterRole
// @Summary 批量删除ClusterRole
// @Description 批量删除指定的多个ClusterRole
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param clusterroles body model.BatchDeleteClusterRoleReq true "批量删除ClusterRole信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/batch-delete [post]
func (cra *ClusterRoleAPI) BatchDeleteClusterRole(c *gin.Context) {
	var req model.BatchDeleteClusterRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := cra.clusterRoleService.BatchDeleteClusterRole(c.Request.Context(), &req)
	if err != nil {
		cra.logger.Error("批量删除ClusterRole失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "批量删除ClusterRole失败")
		return
	}

	utils.Success(c)
}

// GetClusterRoleYaml 获取ClusterRole的YAML配置
// @Summary 获取ClusterRole的YAML配置
// @Description 获取指定ClusterRole的YAML格式配置
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRole名称"
// @Success 200 {object} utils.ApiResponse{data=string}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/yaml/{cluster_id}/{name} [get]
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
// @Summary 更新ClusterRole的YAML配置
// @Description 通过YAML更新指定ClusterRole的配置
// @Tags RBAC ClusterRole管理
// @Accept json
// @Produce json
// @Param yaml body model.ClusterRoleYamlReq true "ClusterRole YAML更新信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/cluster-role/yaml [put]
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
