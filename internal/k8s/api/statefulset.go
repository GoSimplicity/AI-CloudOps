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

type K8sStatefulSetHandler struct {
	statefulSetService service.StatefulSetService
}

func NewK8sStatefulSetHandler(statefulSetService service.StatefulSetService) *K8sStatefulSetHandler {
	return &K8sStatefulSetHandler{
		statefulSetService: statefulSetService,
	}
}

func (h *K8sStatefulSetHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/statefulset/:cluster_id/list", h.GetStatefulSetList)
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/detail", h.GetStatefulSetDetails)
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/detail/yaml", h.GetStatefulSetYaml)
		k8sGroup.POST("/statefulset/:cluster_id/create", h.CreateStatefulSet)
		k8sGroup.POST("/statefulset/:cluster_id/create/yaml", h.CreateStatefulSetByYaml)
		k8sGroup.PUT("/statefulset/:cluster_id/:namespace/:name/update", h.UpdateStatefulSet)
		k8sGroup.PUT("/statefulset/:cluster_id/:namespace/:name/update/yaml", h.UpdateStatefulSetByYaml)
		k8sGroup.DELETE("/statefulset/:cluster_id/:namespace/:name/delete", h.DeleteStatefulSet)
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/restart", h.RestartStatefulSet)
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/scale", h.ScaleStatefulSet)
		k8sGroup.POST("/statefulset/:cluster_id/:namespace/:name/rollback", h.RollbackStatefulSet)
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/pods", h.GetStatefulSetPods)
		k8sGroup.GET("/statefulset/:cluster_id/:namespace/:name/history", h.GetStatefulSetHistory)
	}
}

func (h *K8sStatefulSetHandler) GetStatefulSetList(ctx *gin.Context) {
	var req model.GetStatefulSetListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.statefulSetService.GetStatefulSetList(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) GetStatefulSetDetails(ctx *gin.Context) {
	var req model.GetStatefulSetDetailsReq

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
		return h.statefulSetService.GetStatefulSetDetails(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) GetStatefulSetYaml(ctx *gin.Context) {
	var req model.GetStatefulSetYamlReq

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
		return h.statefulSetService.GetStatefulSetYaml(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) CreateStatefulSet(ctx *gin.Context) {
	var req model.CreateStatefulSetReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.CreateStatefulSet(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) CreateStatefulSetByYaml(ctx *gin.Context) {
	var req model.CreateStatefulSetByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.statefulSetService.CreateStatefulSetByYaml(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) UpdateStatefulSet(ctx *gin.Context) {
	var req model.UpdateStatefulSetReq

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
		return nil, h.statefulSetService.UpdateStatefulSet(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) UpdateStatefulSetByYaml(ctx *gin.Context) {
	var req model.UpdateStatefulSetByYamlReq

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
		return nil, h.statefulSetService.UpdateStatefulSetByYaml(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) DeleteStatefulSet(ctx *gin.Context) {
	var req model.DeleteStatefulSetReq

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
		return nil, h.statefulSetService.DeleteStatefulSet(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) RestartStatefulSet(ctx *gin.Context) {
	var req model.RestartStatefulSetReq

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
		return nil, h.statefulSetService.RestartStatefulSet(ctx, &req)
	})
}

// ScaleStatefulSet 缩放StatefulSet
func (h *K8sStatefulSetHandler) ScaleStatefulSet(ctx *gin.Context) {
	var req model.ScaleStatefulSetReq

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
		return nil, h.statefulSetService.ScaleStatefulSet(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) RollbackStatefulSet(ctx *gin.Context) {
	var req model.RollbackStatefulSetReq

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
		return nil, h.statefulSetService.RollbackStatefulSet(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) GetStatefulSetPods(ctx *gin.Context) {
	var req model.GetStatefulSetPodsReq

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
		return h.statefulSetService.GetStatefulSetPods(ctx, &req)
	})
}

func (h *K8sStatefulSetHandler) GetStatefulSetHistory(ctx *gin.Context) {
	var req model.GetStatefulSetHistoryReq

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
		return h.statefulSetService.GetStatefulSetHistory(ctx, &req)
	})
}
