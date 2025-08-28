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

type RoleBindingAPI struct {
	roleBindingService *service.RoleBindingService
	logger             *zap.Logger
}

func NewRoleBindingAPI(roleBindingService *service.RoleBindingService, logger *zap.Logger) *RoleBindingAPI {
	return &RoleBindingAPI{
		roleBindingService: roleBindingService,
		logger:             logger,
	}
}

func (rba *RoleBindingAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/v1/k8s")

	roleBindings := k8sGroup.Group("/role-binding")
	{
		roleBindings.GET("/list", rba.GetRoleBindingList)                                    // 获取RoleBinding列表
		roleBindings.GET("/details/:cluster_id/:namespace/:name", rba.GetRoleBindingDetails) // 获取RoleBinding详情
		roleBindings.POST("/create", rba.CreateRoleBinding)                                  // 创建RoleBinding
		roleBindings.PUT("/update", rba.UpdateRoleBinding)                                   // 更新RoleBinding
		roleBindings.DELETE("/delete/:cluster_id/:namespace/:name", rba.DeleteRoleBinding)   // 删除RoleBinding
		roleBindings.POST("/batch-delete", rba.BatchDeleteRoleBinding)                       // 批量删除RoleBinding
		roleBindings.GET("/yaml/:cluster_id/:namespace/:name", rba.GetRoleBindingYaml)       // 获取RoleBinding YAML
		roleBindings.PUT("/yaml", rba.UpdateRoleBindingYaml)                                 // 更新RoleBinding YAML
	}
}

// GetRoleBindingList 获取RoleBinding列表
// @Summary 获取RoleBinding列表
// @Description 获取指定集群中的RoleBinding列表，支持分页和关键字搜索
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} apiresponse.ApiResponse{data=model.ListResp[model.ClusterRoleBindingInfo]}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/list [get]
func (rba *RoleBindingAPI) GetRoleBindingList(c *gin.Context) {
	var req model.RoleBindingListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := rba.roleBindingService.GetRoleBindingList(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("获取RoleBinding列表失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取RoleBinding列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetRoleBindingDetails 获取RoleBinding详情
// @Summary 获取RoleBinding详情
// @Description 获取指定RoleBinding的详细信息
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "RoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse{data=model.RoleBindingInfo}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 404 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/details/{cluster_id}/{namespace}/{name} [get]
func (rba *RoleBindingAPI) GetRoleBindingDetails(c *gin.Context) {
	var req model.RoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := rba.roleBindingService.GetRoleBindingDetails(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("获取RoleBinding详情失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取RoleBinding详情失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CreateRoleBinding 创建RoleBinding
// @Summary 创建RoleBinding
// @Description 在指定集群和命名空间中创建新的RoleBinding
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param rolebinding body model.CreateRoleBindingReq true "RoleBinding创建信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/create [post]
func (rba *RoleBindingAPI) CreateRoleBinding(c *gin.Context) {
	var req model.CreateRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.CreateRoleBinding(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("创建RoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "创建RoleBinding失败")
		return
	}

	utils.Success(c)
}

// UpdateRoleBinding 更新RoleBinding
// @Summary 更新RoleBinding
// @Description 更新指定RoleBinding的配置信息
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param rolebinding body model.UpdateRoleBindingReq true "RoleBinding更新信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/update [put]
func (rba *RoleBindingAPI) UpdateRoleBinding(c *gin.Context) {
	var req model.UpdateRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.UpdateRoleBinding(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("更新RoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新RoleBinding失败")
		return
	}

	utils.Success(c)
}

// DeleteRoleBinding 删除RoleBinding
// @Summary 删除RoleBinding
// @Description 删除指定的RoleBinding
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "RoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/delete/{cluster_id}/{namespace}/{name} [delete]
func (rba *RoleBindingAPI) DeleteRoleBinding(c *gin.Context) {
	var req model.DeleteRoleBindingReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.DeleteRoleBinding(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("删除RoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "删除RoleBinding失败")
		return
	}

	utils.Success(c)
}

// BatchDeleteRoleBinding 批量删除RoleBinding
// @Summary 批量删除RoleBinding
// @Description 批量删除指定的多个RoleBinding
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param rolebindings body model.BatchDeleteRoleBindingReq true "批量删除RoleBinding信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/batch-delete [post]
func (rba *RoleBindingAPI) BatchDeleteRoleBinding(c *gin.Context) {
	var req model.BatchDeleteRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.BatchDeleteRoleBinding(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("批量删除RoleBinding失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "批量删除RoleBinding失败")
		return
	}

	utils.Success(c)
}

// GetRoleBindingYaml 获取RoleBinding的YAML配置
// @Summary 获取RoleBinding的YAML配置
// @Description 获取指定RoleBinding的YAML格式配置
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "RoleBinding名称"
// @Success 200 {object} apiresponse.ApiResponse{data=string}
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/yaml/{cluster_id}/{namespace}/{name} [get]
func (rba *RoleBindingAPI) GetRoleBindingYaml(c *gin.Context) {
	var req model.RoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	yamlContent, err := rba.roleBindingService.GetRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("获取RoleBinding YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "获取RoleBinding YAML失败")
		return
	}

	utils.SuccessWithData(c, yamlContent)
}

// UpdateRoleBindingYaml 更新RoleBinding的YAML配置
// @Summary 更新RoleBinding的YAML配置
// @Description 通过YAML更新指定RoleBinding的配置
// @Tags RBAC RoleBinding管理
// @Accept json
// @Produce json
// @Param yaml body model.RoleBindingYamlReq true "RoleBinding YAML更新信息"
// @Success 200 {object} apiresponse.ApiResponse
// @Failure 400 {object} apiresponse.ApiResponse
// @Failure 500 {object} apiresponse.ApiResponse
// @Router /api/v1/k8s/role-binding/yaml [put]
func (rba *RoleBindingAPI) UpdateRoleBindingYaml(c *gin.Context) {
	var req model.RoleBindingYamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.UpdateRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {
		rba.logger.Error("更新RoleBinding YAML失败", zap.Error(err))
		utils.InternalServerError(c, 500, nil, "更新RoleBinding YAML失败")
		return
	}

	utils.Success(c)
}
