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

type K8sDeploymentHandler struct {
	logger            *zap.Logger
	deploymentService service.DeploymentService
}

func NewK8sDeploymentHandler(logger *zap.Logger, deploymentService service.DeploymentService) *K8sDeploymentHandler {
	return &K8sDeploymentHandler{
		logger:            logger,
		deploymentService: deploymentService,
	}
}

func (k *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	deployments := k8sGroup.Group("/deployments")
	{
		deployments.GET("/:id", k.GetDeployListByNamespace) // 根据命名空间获取部署列表
		deployments.GET("/:id/yaml", k.GetDeployYaml)       // 获取指定部署的 YAML 配置
		deployments.POST("/update", k.UpdateDeployment)     // 更新指定 deployment

		deployments.DELETE("/delete/:id", k.DeleteDeployment)

		deployments.POST("/restart/:id", k.RestartDeployment)
	}
}

// GetDeployListByNamespace 根据命名空间获取部署列表
// @Summary 根据命名空间获取部署列表
// @Description 根据指定的命名空间获取K8s集群中的Deployment列表
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace query string false "命名空间，为空则获取所有命名空间"
// @Param label_selector query string false "标签选择器"
// @Param field_selector query string false "字段选择器"
// @Param limit query int false "限制结果数量"
// @Success 200 {object} utils.ApiResponse{data=[]object} "成功获取部署列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/deployments/{cluster_id} [get]
func (k *K8sDeploymentHandler) GetDeployListByNamespace(ctx *gin.Context) {
	var req model.K8sGetResourceListReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.deploymentService.GetDeploymentsByNamespace(ctx, req.ClusterID, req.Namespace)
	})
}

// UpdateDeployment 更新部署
// @Summary 更新部署
// @Description 更新指定的Deployment资源配置
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param request body model.K8sDeploymentReq true "部署更新请求"
// @Success 200 {object} utils.ApiResponse "成功更新部署"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/deployments/update [post]
func (k *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.UpdateDeployment(ctx, &req)
	})
}

// GetDeployYaml 获取部署的YAML配置
// @Summary 获取部署的YAML配置
// @Description 获取指定Deployment的完整YAML配置文件
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param resource_name path string true "部署名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=string} "成功获取YAML配置"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/deployments/{cluster_id}/{resource_name}/yaml [get]
func (k *K8sDeploymentHandler) GetDeployYaml(ctx *gin.Context) {
	var req model.K8sGetResourceYamlReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.deploymentService.GetDeploymentYaml(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// DeleteDeployment 删除部署
// @Summary 删除部署
// @Description 删除指定命名空间中的单个Deployment
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param resource_name path string true "部署名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "成功删除部署"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/deployments/{cluster_id}/{resource_name} [delete]
func (k *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
	var req model.K8sDeleteResourceReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.deploymentService.DeleteDeployment(ctx, req.ClusterID, req.Namespace, req.ResourceName)
	})
}

// RestartDeployment 重启部署
// @Summary 重启部署
// @Description 重启指定命名空间中的单个Deployment
// @Tags 部署管理
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param resource_name path string true "部署名称"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse "成功重启部署"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/k8s/deployments/{cluster_id}/{resource_name}/restart [post]
func (k *K8sDeploymentHandler) RestartDeployment(ctx *gin.Context) {
	var req model.DeploymentRestartReq
	if err := ctx.ShouldBindUri(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.deploymentService.RestartDeployment(ctx, req.ClusterID, req.Namespace, req.DeploymentName)
	})
}
