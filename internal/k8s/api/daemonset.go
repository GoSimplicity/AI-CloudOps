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

type K8sDaemonSetHandler struct {
	daemonSetService service.DaemonSetService
}

func NewK8sDaemonSetHandler(daemonSetService service.DaemonSetService) *K8sDaemonSetHandler {
	return &K8sDaemonSetHandler{
		daemonSetService: daemonSetService,
	}
}

func (k *K8sDaemonSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/daemonsets", k.GetDaemonSetList)
		k8sGroup.GET("/daemonsets/:cluster_id/:namespace/:name", k.GetDaemonSetDetails)
		k8sGroup.GET("/daemonsets/:cluster_id/:namespace/:name/yaml", k.GetDaemonSetYaml)
		k8sGroup.POST("/daemonsets", k.CreateDaemonSet)
		k8sGroup.PUT("/daemonsets/:cluster_id/:namespace/:name", k.UpdateDaemonSet)
		k8sGroup.DELETE("/daemonsets/:cluster_id/:namespace/:name", k.DeleteDaemonSet)
		k8sGroup.POST("/daemonsets/:cluster_id/:namespace/:name/restart", k.RestartDaemonSet)
		k8sGroup.GET("/daemonsets/:cluster_id/:namespace/:name/pods", k.GetDaemonSetPods)
		k8sGroup.GET("/daemonsets/:cluster_id/:namespace/:name/history", k.GetDaemonSetHistory)
		k8sGroup.POST("/daemonsets/:cluster_id/:namespace/:name/rollback", k.RollbackDaemonSet)
	}
}

// GetDaemonSetList 获取DaemonSet列表
func (k *K8sDaemonSetHandler) GetDaemonSetList(ctx *gin.Context) {
	var req model.GetDaemonSetListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.daemonSetService.GetDaemonSetList(ctx, &req)
	})
}

// GetDaemonSetDetails 获取DaemonSet详情
func (k *K8sDaemonSetHandler) GetDaemonSetDetails(ctx *gin.Context) {
	var req model.GetDaemonSetDetailsReq

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
		return k.daemonSetService.GetDaemonSetDetails(ctx, &req)
	})
}

// GetDaemonSetYaml 获取DaemonSet YAML
func (k *K8sDaemonSetHandler) GetDaemonSetYaml(ctx *gin.Context) {
	var req model.GetDaemonSetYamlReq

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
		return k.daemonSetService.GetDaemonSetYaml(ctx, &req)
	})
}

// CreateDaemonSet 创建DaemonSet
func (k *K8sDaemonSetHandler) CreateDaemonSet(ctx *gin.Context) {
	var req model.CreateDaemonSetReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.daemonSetService.CreateDaemonSet(ctx, &req)
	})
}

// UpdateDaemonSet 更新DaemonSet
func (k *K8sDaemonSetHandler) UpdateDaemonSet(ctx *gin.Context) {
	var req model.UpdateDaemonSetReq

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
		return nil, k.daemonSetService.UpdateDaemonSet(ctx, &req)
	})
}

// DeleteDaemonSet 删除DaemonSet
func (k *K8sDaemonSetHandler) DeleteDaemonSet(ctx *gin.Context) {
	var req model.DeleteDaemonSetReq

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
		return nil, k.daemonSetService.DeleteDaemonSet(ctx, &req)
	})
}

// RestartDaemonSet 重启DaemonSet
func (k *K8sDaemonSetHandler) RestartDaemonSet(ctx *gin.Context) {
	var req model.RestartDaemonSetReq

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
		return nil, k.daemonSetService.RestartDaemonSet(ctx, &req)
	})
}

// GetDaemonSetPods 获取DaemonSet下的Pod列表
func (k *K8sDaemonSetHandler) GetDaemonSetPods(ctx *gin.Context) {
	var req model.GetDaemonSetPodsReq

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
		return k.daemonSetService.GetDaemonSetPods(ctx, &req)
	})
}

// GetDaemonSetHistory 获取DaemonSet历史
func (k *K8sDaemonSetHandler) GetDaemonSetHistory(ctx *gin.Context) {
	var req model.GetDaemonSetHistoryReq

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
		return k.daemonSetService.GetDaemonSetHistory(ctx, &req)
	})
}

// RollbackDaemonSet 回滚DaemonSet
func (k *K8sDaemonSetHandler) RollbackDaemonSet(ctx *gin.Context) {
	var req model.RollbackDaemonSetReq

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
		return nil, k.daemonSetService.RollbackDaemonSet(ctx, &req)
	})
}
