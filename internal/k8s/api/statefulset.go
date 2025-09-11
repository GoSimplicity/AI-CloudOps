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

type K8sStatefulSetHandler struct {
	statefulSetService service.StatefulSetService
}

func NewK8sStatefulSetHandler(statefulSetService service.StatefulSetService) *K8sStatefulSetHandler {
	return &K8sStatefulSetHandler{
		statefulSetService: statefulSetService,
	}
}

func (k *K8sStatefulSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// StatefulSet基础管理
		k8sGroup.GET("/statefulset/:cluster_id/list", k.GetStatefulSetList)                              // 获取StatefulSet列表
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/detail", k.GetStatefulSetDetails)        // 获取StatefulSet详情
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/detail/yaml", k.GetStatefulSetYaml)      // 获取StatefulSet YAML
		k8sGroup.POST("/statefulset/:cluster_id/create", k.CreateStatefulSet)                            // 创建StatefulSet
		k8sGroup.POST("/statefulset/:cluster_id/create/yaml", k.CreateStatefulSetByYaml)                 // 通过YAML创建StatefulSet
		k8sGroup.PUT("/statefulset/:cluster_id/:namespace/:name/update", k.UpdateStatefulSet)            // 更新StatefulSet
		k8sGroup.PUT("/statefulset/:cluster_id/:namespace/:name/update/yaml", k.UpdateStatefulSetByYaml) // 通过YAML更新StatefulSet
		k8sGroup.DELETE("/statefulset/:cluster_id/:namespace/:name/delete", k.DeleteStatefulSet)         // 删除StatefulSet
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/restart", k.RestartStatefulSet)         // 重启StatefulSet
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/scale", k.ScaleStatefulSet)             // 扩缩容StatefulSet
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/rollback", k.RollbackStatefulSet)       // 回滚StatefulSet
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/pods", k.GetStatefulSetPods)             // 获取StatefulSet Pod列表
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/history", k.GetStatefulSetHistory)       // 获取StatefulSet版本历史
	}
}

// GetStatefulSetList 获取StatefulSet列表
func (k *K8sStatefulSetHandler) GetStatefulSetList(ctx *gin.Context) {
	var req model.GetStatefulSetListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.statefulSetService.GetStatefulSetList(ctx, &req)
	})
}

// GetStatefulSetDetails 获取StatefulSet详情
func (k *K8sStatefulSetHandler) GetStatefulSetDetails(ctx *gin.Context) {
	var req model.GetStatefulSetDetailsReq

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
		return k.statefulSetService.GetStatefulSetDetails(ctx, &req)
	})
}

// GetStatefulSetYaml 获取StatefulSet YAML
func (k *K8sStatefulSetHandler) GetStatefulSetYaml(ctx *gin.Context) {
	var req model.GetStatefulSetYamlReq

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
		return k.statefulSetService.GetStatefulSetYaml(ctx, &req)
	})
}

// CreateStatefulSet 创建StatefulSet
func (k *K8sStatefulSetHandler) CreateStatefulSet(ctx *gin.Context) {
	var req model.CreateStatefulSetReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.CreateStatefulSet(ctx, &req)
	})
}

// CreateStatefulSetByYaml 通过YAML创建StatefulSet
func (k *K8sStatefulSetHandler) CreateStatefulSetByYaml(ctx *gin.Context) {
	var req model.CreateStatefulSetByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.statefulSetService.CreateStatefulSetByYaml(ctx, &req)
	})
}

// UpdateStatefulSet 更新StatefulSet
func (k *K8sStatefulSetHandler) UpdateStatefulSet(ctx *gin.Context) {
	var req model.UpdateStatefulSetReq

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
		return nil, k.statefulSetService.UpdateStatefulSet(ctx, &req)
	})
}

// UpdateStatefulSetByYaml 通过YAML更新StatefulSet
func (k *K8sStatefulSetHandler) UpdateStatefulSetByYaml(ctx *gin.Context) {
	var req model.UpdateStatefulSetByYamlReq

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
		return nil, k.statefulSetService.UpdateStatefulSetByYaml(ctx, &req)
	})
}

// DeleteStatefulSet 删除StatefulSet
func (k *K8sStatefulSetHandler) DeleteStatefulSet(ctx *gin.Context) {
	var req model.DeleteStatefulSetReq

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
		return nil, k.statefulSetService.DeleteStatefulSet(ctx, &req)
	})
}

// RestartStatefulSet 重启StatefulSet
func (k *K8sStatefulSetHandler) RestartStatefulSet(ctx *gin.Context) {
	var req model.RestartStatefulSetReq

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
		return nil, k.statefulSetService.RestartStatefulSet(ctx, &req)
	})
}

// ScaleStatefulSet 缩放StatefulSet
func (k *K8sStatefulSetHandler) ScaleStatefulSet(ctx *gin.Context) {
	var req model.ScaleStatefulSetReq

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
		return nil, k.statefulSetService.ScaleStatefulSet(ctx, &req)
	})
}

// RollbackStatefulSet 回滚StatefulSet
func (k *K8sStatefulSetHandler) RollbackStatefulSet(ctx *gin.Context) {
	var req model.RollbackStatefulSetReq

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
		return nil, k.statefulSetService.RollbackStatefulSet(ctx, &req)
	})
}

// GetStatefulSetPods 获取StatefulSet下的Pod列表
func (k *K8sStatefulSetHandler) GetStatefulSetPods(ctx *gin.Context) {
	var req model.GetStatefulSetPodsReq

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
		return k.statefulSetService.GetStatefulSetPods(ctx, &req)
	})
}

// GetStatefulSetHistory 获取StatefulSet历史
func (k *K8sStatefulSetHandler) GetStatefulSetHistory(ctx *gin.Context) {
	var req model.GetStatefulSetHistoryReq

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
		return k.statefulSetService.GetStatefulSetHistory(ctx, &req)
	})
}
