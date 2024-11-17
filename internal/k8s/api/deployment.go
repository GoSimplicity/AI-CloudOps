package api

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

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sDeploymentHandler struct {
	l                 *zap.Logger
	deploymentService admin.DeploymentService
}

func NewK8sDeploymentHandler(l *zap.Logger, deploymentService admin.DeploymentService) *K8sDeploymentHandler {
	return &K8sDeploymentHandler{
		l:                 l,
		deploymentService: deploymentService,
	}
}

func (k *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	deployments := k8sGroup.Group("/deployments")
	{
		deployments.GET("/:id", k.GetDeployListByNamespace)          // 根据命名空间获取部署列表
		deployments.GET("/:id/yaml", k.GetDeployYaml)                // 获取指定部署的 YAML 配置
		deployments.POST("/update", k.UpdateDeployment)              // 更新指定 deployment
		deployments.DELETE("/batch_delete", k.BatchDeleteDeployment) // 批量删除 deployment
		deployments.DELETE("/delete/:id", k.DeleteDeployment)
		deployments.POST("/batch_restart", k.BatchRestartDeployments) // 批量重启部署
		deployments.POST("/restart/:id", k.RestartDeployment)
	}
}

// GetDeployListByNamespace 根据命名空间获取部署列表
func (k *K8sDeploymentHandler) GetDeployListByNamespace(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.deploymentService.GetDeploymentsByNamespace(ctx, id, namespace)
	})
}

// UpdateDeployment 更新指定 Name 的部署
func (k *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.UpdateDeployment(ctx, &req)
	})
}

// BatchDeleteDeployment 删除指定 Name 的部署
func (k *K8sDeploymentHandler) BatchDeleteDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.BatchDeleteDeployment(ctx, req.ClusterId, req.Namespace, req.DeploymentNames)
	})
}

// BatchRestartDeployments 批量重启部署
func (k *K8sDeploymentHandler) BatchRestartDeployments(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.BatchRestartDeployments(ctx, &req)
	})
}

// GetDeployYaml 获取部署的 YAML 配置
func (k *K8sDeploymentHandler) GetDeployYaml(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	deploymentName := ctx.Query("deployment_name")
	if deploymentName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'deployment_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.deploymentService.GetDeploymentYaml(ctx, id, namespace, deploymentName)
	})
}

func (k *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	deploymentName := ctx.Query("deployment_name")
	if deploymentName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'deployment_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.deploymentService.DeleteDeployment(ctx, id, namespace, deploymentName)
	})
}

func (k *K8sDeploymentHandler) RestartDeployment(ctx *gin.Context) {
	id, err := apiresponse.GetParamID(ctx)
	if err != nil {
		apiresponse.BadRequestError(ctx, err.Error())
		return
	}

	deploymentName := ctx.Query("deployment_name")
	if deploymentName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'deployment_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	apiresponse.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.deploymentService.RestartDeployment(ctx, id, namespace, deploymentName)
	})
}
