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

type K8sDeploymentHandler struct {
	deploymentService service.DeploymentService
}

func NewK8sDeploymentHandler(deploymentService service.DeploymentService) *K8sDeploymentHandler {
	return &K8sDeploymentHandler{

		deploymentService: deploymentService,
	}
}

func (k *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/deployments/:id", k.GetDeployListByNamespace) // 根据命名空间获取部署列表
		k8sGroup.GET("/deployments/:id/yaml", k.GetDeployYaml)       // 获取指定部署的 YAML 配置
		k8sGroup.POST("/deployments/update", k.UpdateDeployment)     // 更新指定 deployment
		k8sGroup.DELETE("/deployments/delete/:id", k.DeleteDeployment)
		k8sGroup.POST("/deployments/restart/:id", k.RestartDeployment)
	}
}

// GetDeployListByNamespace 根据命名空间获取部署列表
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
func (k *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.UpdateDeployment(ctx, &req)
	})
}

// GetDeployYaml 获取部署的YAML配置
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
