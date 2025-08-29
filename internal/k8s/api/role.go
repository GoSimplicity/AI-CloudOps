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

type RoleAPI struct {
	roleService *service.RoleService
	logger      *zap.Logger
}

func NewRoleAPI(roleService *service.RoleService, logger *zap.Logger) *RoleAPI {
	return &RoleAPI{
		roleService: roleService,
		logger:      logger,
	}
}

func (ra *RoleAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/v1/k8s")

	roles := k8sGroup.Group("/role")
	{
		roles.GET("/list", ra.GetRoleList)                                    // 获取Role列表
		roles.GET("/details/:cluster_id/:namespace/:name", ra.GetRoleDetails) // 获取Role详情
		roles.POST("/create", ra.CreateRole)                                  // 创建Role
		roles.PUT("/update", ra.UpdateRole)                                   // 更新Role
		roles.DELETE("/delete/:cluster_id/:namespace/:name", ra.DeleteRole)   // 删除Role

		roles.GET("/yaml/:cluster_id/:namespace/:name", ra.GetRoleYaml) // 获取Role YAML
		roles.PUT("/yaml", ra.UpdateRoleYaml)                           // 更新Role YAML
	}
}

// GetRoleList 获取Role列表
// @Summary 获取Role列表
// @Description 获取指定集群中的Role列表，支持分页和关键字搜索
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param cluster_id query int true "集群ID"
// @Param namespace query string false "命名空间"
// @Param keyword query string false "搜索关键字"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认10"
// @Success 200 {object} utils.ApiResponse{data=model.ListResp[model.RoleInfo]}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/list [get]
func (ra *RoleAPI) GetRoleList(c *gin.Context) {
	var req model.RoleListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return ra.roleService.GetRoleList(c.Request.Context(), &req)
	})
}

// GetRoleDetails 获取Role详情
// @Summary 获取Role详情
// @Description 获取指定Role的详细信息
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Role名称"
// @Success 200 {object} utils.ApiResponse{data=model.RoleInfo}
// @Failure 400 {object} utils.ApiResponse
// @Failure 404 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/details/{cluster_id}/{namespace}/{name} [get]
func (ra *RoleAPI) GetRoleDetails(c *gin.Context) {
	var req model.RoleGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return ra.roleService.GetRoleDetails(c.Request.Context(), &req)
	})
}

// CreateRole 创建Role
// @Summary 创建Role
// @Description 在指定集群和命名空间中创建新的Role
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param role body model.CreateRoleReq true "Role创建信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/create [post]
func (ra *RoleAPI) CreateRole(c *gin.Context) {
	var req model.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return nil, ra.roleService.CreateRole(c.Request.Context(), &req)
	})
}

// UpdateRole 更新Role
// @Summary 更新Role
// @Description 更新指定Role的配置信息
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param role body model.UpdateRoleReq true "Role更新信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/update [put]
func (ra *RoleAPI) UpdateRole(c *gin.Context) {
	var req model.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return nil, ra.roleService.UpdateRole(c.Request.Context(), &req)
	})
}

// DeleteRole 删除Role
// @Summary 删除Role
// @Description 删除指定的Role
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Role名称"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/delete/{cluster_id}/{namespace}/{name} [delete]
func (ra *RoleAPI) DeleteRole(c *gin.Context) {
	var req model.DeleteRoleReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return nil, ra.roleService.DeleteRole(c.Request.Context(), &req)
	})
}

// GetRoleYaml 获取Role的YAML配置
// @Summary 获取Role的YAML配置
// @Description 获取指定Role的YAML格式配置
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间"
// @Param name path string true "Role名称"
// @Success 200 {object} utils.ApiResponse{data=string}
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/yaml/{cluster_id}/{namespace}/{name} [get]
func (ra *RoleAPI) GetRoleYaml(c *gin.Context) {
	var req model.RoleGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return ra.roleService.GetRoleYaml(c.Request.Context(), &req)
	})
}

// UpdateRoleYaml 更新Role的YAML配置
// @Summary 更新Role的YAML配置
// @Description 通过YAML更新指定Role的配置
// @Tags RBAC Role管理
// @Accept json
// @Produce json
// @Param yaml body model.RoleYamlReq true "Role YAML更新信息"
// @Success 200 {object} utils.ApiResponse
// @Failure 400 {object} utils.ApiResponse
// @Failure 500 {object} utils.ApiResponse
// @Router /api/v1/k8s/role/yaml [put]
func (ra *RoleAPI) UpdateRoleYaml(c *gin.Context) {
	var req model.RoleYamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	utils.HandleRequest(c, nil, func() (interface{}, error) {
		return nil, ra.roleService.UpdateRoleYaml(c.Request.Context(), &req)
	})
}
