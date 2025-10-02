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

func (h *K8sDeploymentHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Deployment基础管理
		k8sGroup.GET("/deployment/:cluster_id/list", h.GetDeploymentList)                              // 获取Deployment列表
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail", h.GetDeploymentDetails)        // 获取Deployment详情
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail/yaml", h.GetDeploymentYaml)      // 获取Deployment YAML
		k8sGroup.POST("/deployment/:cluster_id/create", h.CreateDeployment)                            // 创建Deployment
		k8sGroup.POST("/deployment/:cluster_id/create/yaml", h.CreateDeploymentByYaml)                 // 通过YAML创建Deployment
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update", h.UpdateDeployment)            // 更新Deployment
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update/yaml", h.UpdateDeploymentByYaml) // 通过YAML更新Deployment
		k8sGroup.DELETE("/deployment/:cluster_id/:namespace/:name/delete", h.DeleteDeployment)         // 删除Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/restart", h.RestartDeployment)         // 重启Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/scale", h.ScaleDeployment)             // 扩缩容Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/pause", h.PauseDeployment)             // 暂停Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/resume", h.ResumeDeployment)           // 恢复Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/rollback", h.RollbackDeployment)       // 回滚Deployment
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/pods", h.GetDeploymentPods)             // 获取Deployment Pod列表
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/history", h.GetDeploymentHistory)       // 获取Deployment版本历史
	}
}

// GetDeploymentList 获取Deployment列表
func (h *K8sDeploymentHandler) GetDeploymentList(ctx *gin.Context) {
	var req model.GetDeploymentListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentList(ctx, &req)
	})
}

// GetDeploymentDetails 获取Deployment详情
func (h *K8sDeploymentHandler) GetDeploymentDetails(ctx *gin.Context) {
	var req model.GetDeploymentDetailsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentDetails(ctx, &req)
	})
}

// GetDeploymentYaml 获取Deployment YAML
func (h *K8sDeploymentHandler) GetDeploymentYaml(ctx *gin.Context) {
	var req model.GetDeploymentYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentYaml(ctx, &req)
	})
}

// CreateDeployment 创建Deployment
func (h *K8sDeploymentHandler) CreateDeployment(ctx *gin.Context) {
	var req model.CreateDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.CreateDeployment(ctx, &req)
	})
}

// UpdateDeployment 更新Deployment
func (h *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.UpdateDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.UpdateDeployment(ctx, &req)
	})
}

// DeleteDeployment 删除Deployment
func (h *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
	var req model.DeleteDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.DeleteDeployment(ctx, &req)
	})
}

// RestartDeployment 重启Deployment
func (h *K8sDeploymentHandler) RestartDeployment(ctx *gin.Context) {
	var req model.RestartDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.RestartDeployment(ctx, &req)
	})
}

// ScaleDeployment 伸缩Deployment
func (h *K8sDeploymentHandler) ScaleDeployment(ctx *gin.Context) {
	var req model.ScaleDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.ScaleDeployment(ctx, &req)
	})
}

// GetDeploymentPods 获取Deployment的Pod列表
func (h *K8sDeploymentHandler) GetDeploymentPods(ctx *gin.Context) {
	var req model.GetDeploymentPodsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentPods(ctx, &req)
	})
}

// GetDeploymentHistory 获取Deployment版本历史
func (h *K8sDeploymentHandler) GetDeploymentHistory(ctx *gin.Context) {
	var req model.GetDeploymentHistoryReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.deploymentService.GetDeploymentHistory(ctx, &req)
	})
}

// RollbackDeployment 回滚Deployment
func (h *K8sDeploymentHandler) RollbackDeployment(ctx *gin.Context) {
	var req model.RollbackDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.RollbackDeployment(ctx, &req)
	})
}

// PauseDeployment 暂停Deployment
func (h *K8sDeploymentHandler) PauseDeployment(ctx *gin.Context) {
	var req model.PauseDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.PauseDeployment(ctx, &req)
	})
}

// ResumeDeployment 恢复Deployment
func (h *K8sDeploymentHandler) ResumeDeployment(ctx *gin.Context) {
	var req model.ResumeDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.ResumeDeployment(ctx, &req)
	})
}

// YAML操作方法

// CreateDeploymentByYaml 通过YAML创建deployment
func (h *K8sDeploymentHandler) CreateDeploymentByYaml(ctx *gin.Context) {
	var req model.CreateDeploymentByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.CreateDeploymentByYaml(ctx, &req)
	})
}

// UpdateDeploymentByYaml 通过YAML更新deployment
func (h *K8sDeploymentHandler) UpdateDeploymentByYaml(ctx *gin.Context) {
	var req model.UpdateDeploymentByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetParamCustomName(ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.deploymentService.UpdateDeploymentByYaml(ctx, &req)
	})
}
