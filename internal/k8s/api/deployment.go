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
		// Deployment基础管理
		k8sGroup.GET("/deployment/:cluster_id/list", k.GetDeploymentList)                              // 获取Deployment列表
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail", k.GetDeploymentDetails)        // 获取Deployment详情
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/detail/yaml", k.GetDeploymentYaml)      // 获取Deployment YAML
		k8sGroup.POST("/deployment/:cluster_id/create", k.CreateDeployment)                            // 创建Deployment
		k8sGroup.POST("/deployment/:cluster_id/create/yaml", k.CreateDeploymentByYaml)                 // 通过YAML创建Deployment
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update", k.UpdateDeployment)            // 更新Deployment
		k8sGroup.PUT("/deployment/:cluster_id/:namespace/:name/update/yaml", k.UpdateDeploymentByYaml) // 通过YAML更新Deployment
		k8sGroup.DELETE("/deployment/:cluster_id/:namespace/:name/delete", k.DeleteDeployment)         // 删除Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/restart", k.RestartDeployment)         // 重启Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/scale", k.ScaleDeployment)             // 扩缩容Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/pause", k.PauseDeployment)             // 暂停Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/resume", k.ResumeDeployment)           // 恢复Deployment
		k8sGroup.POST("/deployment/:cluster_id/:namespace/:name/rollback", k.RollbackDeployment)       // 回滚Deployment
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/pods", k.GetDeploymentPods)             // 获取Deployment Pod列表
		k8sGroup.GET("/deployment/:cluster_id/:namespace/:name/history", k.GetDeploymentHistory)       // 获取Deployment版本历史
	}
}

// GetDeploymentList 获取Deployment列表
func (k *K8sDeploymentHandler) GetDeploymentList(ctx *gin.Context) {
	var req model.GetDeploymentListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.deploymentService.GetDeploymentList(ctx, &req)
	})
}

// GetDeploymentDetails 获取Deployment详情
func (k *K8sDeploymentHandler) GetDeploymentDetails(ctx *gin.Context) {
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
		return k.deploymentService.GetDeploymentDetails(ctx, &req)
	})
}

// GetDeploymentYaml 获取Deployment YAML
func (k *K8sDeploymentHandler) GetDeploymentYaml(ctx *gin.Context) {
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
		return k.deploymentService.GetDeploymentYaml(ctx, &req)
	})
}

// CreateDeployment 创建Deployment
func (k *K8sDeploymentHandler) CreateDeployment(ctx *gin.Context) {
	var req model.CreateDeploymentReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.CreateDeployment(ctx, &req)
	})
}

// UpdateDeployment 更新Deployment
func (k *K8sDeploymentHandler) UpdateDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.UpdateDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) DeleteDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.DeleteDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) RestartDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.RestartDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) ScaleDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.ScaleDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) GetDeploymentPods(ctx *gin.Context) {
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
		return k.deploymentService.GetDeploymentPods(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) GetDeploymentHistory(ctx *gin.Context) {
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
		return k.deploymentService.GetDeploymentHistory(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) RollbackDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.RollbackDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) PauseDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.PauseDeployment(ctx, &req)
	})
}

func (k *K8sDeploymentHandler) ResumeDeployment(ctx *gin.Context) {
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
		return nil, k.deploymentService.ResumeDeployment(ctx, &req)
	})
}

// YAML操作方法

// CreateDeploymentByYaml 通过YAML创建deployment
func (k *K8sDeploymentHandler) CreateDeploymentByYaml(ctx *gin.Context) {
	var req model.CreateDeploymentByYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.CreateDeploymentByYaml(ctx, &req)
	})
}

// UpdateDeploymentByYaml 通过YAML更新deployment
func (k *K8sDeploymentHandler) UpdateDeploymentByYaml(ctx *gin.Context) {
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
		return nil, k.deploymentService.UpdateDeploymentByYaml(ctx, &req)
	})
}
