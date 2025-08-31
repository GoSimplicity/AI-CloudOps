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
)

type RoleBindingAPI struct {
	roleBindingService *service.RoleBindingService
}

func NewRoleBindingAPI(roleBindingService *service.RoleBindingService) *RoleBindingAPI {
	return &RoleBindingAPI{
		roleBindingService: roleBindingService,
	}
}

func (rba *RoleBindingAPI) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/role-binding/list", rba.GetRoleBindingList)                                    // 获取RoleBinding列表
		k8sGroup.GET("/role-binding/details/:cluster_id/:namespace/:name", rba.GetRoleBindingDetails) // 获取RoleBinding详情
		k8sGroup.POST("/role-binding/create", rba.CreateRoleBinding)                                  // 创建RoleBinding
		k8sGroup.PUT("/role-binding/update", rba.UpdateRoleBinding)                                   // 更新RoleBinding
		k8sGroup.DELETE("/role-binding/delete/:cluster_id/:namespace/:name", rba.DeleteRoleBinding)   // 删除RoleBinding
		k8sGroup.GET("/role-binding/yaml/:cluster_id/:namespace/:name", rba.GetRoleBindingYaml)       // 获取RoleBinding YAML
		k8sGroup.PUT("/role-binding/yaml", rba.UpdateRoleBindingYaml)                                 // 更新RoleBinding YAML
	}
}

// GetRoleBindingList 获取RoleBinding列表
func (rba *RoleBindingAPI) GetRoleBindingList(c *gin.Context) {
	var req model.RoleBindingListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := rba.roleBindingService.GetRoleBindingList(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取RoleBinding列表失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// GetRoleBindingDetails 获取RoleBinding详情
func (rba *RoleBindingAPI) GetRoleBindingDetails(c *gin.Context) {
	var req model.RoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	result, err := rba.roleBindingService.GetRoleBindingDetails(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取RoleBinding详情失败")
		return
	}

	utils.SuccessWithData(c, result)
}

// CreateRoleBinding 创建RoleBinding
func (rba *RoleBindingAPI) CreateRoleBinding(c *gin.Context) {
	var req model.CreateRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.CreateRoleBinding(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "创建RoleBinding失败")
		return
	}

	utils.Success(c)
}

// UpdateRoleBinding 更新RoleBinding
func (rba *RoleBindingAPI) UpdateRoleBinding(c *gin.Context) {
	var req model.UpdateRoleBindingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.UpdateRoleBinding(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "更新RoleBinding失败")
		return
	}

	utils.Success(c)
}

// DeleteRoleBinding 删除RoleBinding
func (rba *RoleBindingAPI) DeleteRoleBinding(c *gin.Context) {
	var req model.DeleteRoleBindingReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.DeleteRoleBinding(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "删除RoleBinding失败")
		return
	}

	utils.Success(c)
}

// GetRoleBindingYaml 获取RoleBinding的YAML配置
func (rba *RoleBindingAPI) GetRoleBindingYaml(c *gin.Context) {
	var req model.RoleBindingGetReq
	if err := c.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	yamlContent, err := rba.roleBindingService.GetRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "获取RoleBinding YAML失败")
		return
	}

	utils.SuccessWithData(c, yamlContent)
}

// UpdateRoleBindingYaml 更新RoleBinding的YAML配置
func (rba *RoleBindingAPI) UpdateRoleBindingYaml(c *gin.Context) {
	var req model.RoleBindingYamlReq
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(c, err.Error())
		return
	}

	err := rba.roleBindingService.UpdateRoleBindingYaml(c.Request.Context(), &req)
	if err != nil {

		utils.InternalServerError(c, 500, nil, "更新RoleBinding YAML失败")
		return
	}

	utils.Success(c)
}
