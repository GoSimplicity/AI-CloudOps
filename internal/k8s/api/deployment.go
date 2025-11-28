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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
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

func (h *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/deployment/:cluster_id/list", h.GetDeploymentList)
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail", h.GetDeploymentDetails)
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail/yaml", h.GetDeploymentYaml)
		k8sGroup.POST("/deployment/:cluster_id/create", h.CreateDeployment)
		k8sGroup.POST("/deployment/:cluster_id/create/yaml", h.CreateDeploymentByYaml)
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update", h.UpdateDeployment)
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update/yaml", h.UpdateDeploymentByYaml)
		k8sGroup.DELETE("/deployment/:cluster_id/:namespace/:name/delete", h.DeleteDeployment)
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/restart", h.RestartDeployment)
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/scale", h.ScaleDeployment)
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/pause", h.PauseDeployment)
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/resume", h.ResumeDeployment)
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/rollback", h.RollbackDeployment)
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/pods", h.GetDeploymentPods)
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/history", h.GetDeploymentHistory)
	}
}

func (h *K8sDeploymentHandler) GetDeploymentList(ctx *gin.Context) {
	var req model.GetDeploymentListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentList(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) GetDeploymentDetails(ctx *gin.Context) {
	var req model.GetDeploymentDetailsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentDetails(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) GetDeploymentYaml(ctx *gin.Context) {
	var req model.GetDeploymentYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentYaml(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) CreateDeployment(ctx *gin.Context) {
	var req model.CreateDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.CreateDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.UpdateDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.UpdateDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
	var req model.DeleteDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.DeleteDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) RestartDeployment(ctx *gin.Context) {
	var req model.RestartDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.RestartDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) ScaleDeployment(ctx *gin.Context) {
	var req model.ScaleDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.ScaleDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) GetDeploymentPods(ctx *gin.Context) {
	var req model.GetDeploymentPodsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentPods(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) GetDeploymentHistory(ctx *gin.Context) {
	var req model.GetDeploymentHistoryReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentHistory(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) RollbackDeployment(ctx *gin.Context) {
	var req model.RollbackDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.RollbackDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) PauseDeployment(ctx *gin.Context) {
	var req model.PauseDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.PauseDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) ResumeDeployment(ctx *gin.Context) {
	var req model.ResumeDeploymentReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.ResumeDeployment(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) CreateDeploymentByYaml(ctx *gin.Context) {
	var req model.CreateDeploymentByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.CreateDeploymentByYaml(ctx, &req)
	})
}

func (h *K8sDeploymentHandler) UpdateDeploymentByYaml(ctx *gin.Context) {
	var req model.UpdateDeploymentByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.UpdateDeploymentByYaml(ctx, &req)
	})
}
