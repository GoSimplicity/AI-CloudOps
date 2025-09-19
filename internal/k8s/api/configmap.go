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

// K8sConfigMapHandler ConfigMap处理器
type K8sConfigMapHandler struct {
	configMapService service.ConfigMapService
}

// NewK8sConfigMapHandler 创建ConfigMap处理器
func NewK8sConfigMapHandler(configMapService service.ConfigMapService) *K8sConfigMapHandler {
	return &K8sConfigMapHandler{
		configMapService: configMapService,
	}
}

// RegisterRouters 注册路由
func (h *K8sConfigMapHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/configmap/:cluster_id/list", h.GetConfigMapList)                              // 获取ConfigMap列表
		k8sGroup.GET("/configmap/:cluster_id/:namespace/:name/detail", h.GetConfigMap)               // 获取ConfigMap详情
		k8sGroup.GET("/configmap/:cluster_id/:namespace/:name/detail/yaml", h.GetConfigMapYAML)      // 获取ConfigMap YAML
		k8sGroup.POST("/configmap/:cluster_id/create", h.CreateConfigMap)                            // 创建ConfigMap
		k8sGroup.POST("/configmap/:cluster_id/create/yaml", h.CreateConfigMapByYaml)                 // 通过YAML创建ConfigMap
		k8sGroup.PUT("/configmap/:cluster_id/:namespace/:name/update", h.UpdateConfigMap)            // 更新ConfigMap
		k8sGroup.PUT("/configmap/:cluster_id/:namespace/:name/update/yaml", h.UpdateConfigMapByYaml) // 通过YAML更新ConfigMap
		k8sGroup.DELETE("/configmap/:cluster_id/:namespace/:name/delete", h.DeleteConfigMap)         // 删除ConfigMap
	}
}

// GetConfigMapList 获取ConfigMap列表
func (h *K8sConfigMapHandler) GetConfigMapList(ctx *gin.Context) {
	var req model.GetConfigMapListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMapList(ctx, &req)
	})
}

// GetConfigMap 获取单个ConfigMap详情
func (h *K8sConfigMapHandler) GetConfigMap(ctx *gin.Context) {
	var req model.GetConfigMapDetailsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name

	if req.Namespace == "" || req.Name == "" {
		utils.BadRequestError(ctx, "命名空间和ConfigMap名称不能为空")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMap(ctx, &req)
	})
}

// CreateConfigMap 创建ConfigMap
func (h *K8sConfigMapHandler) CreateConfigMap(ctx *gin.Context) {
	var req model.CreateConfigMapReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.CreateConfigMap(ctx, &req)
	})
}

// UpdateConfigMap 更新ConfigMap
func (h *K8sConfigMapHandler) UpdateConfigMap(ctx *gin.Context) {
	var req model.UpdateConfigMapReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.UpdateConfigMap(ctx, &req)
	})
}

// DeleteConfigMap 删除ConfigMap
func (h *K8sConfigMapHandler) DeleteConfigMap(ctx *gin.Context) {
	var req model.DeleteConfigMapReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.configMapService.DeleteConfigMap(ctx, &req)
	})
}

// GetConfigMapYAML 获取ConfigMap的YAML配置
func (h *K8sConfigMapHandler) GetConfigMapYAML(ctx *gin.Context) {
	var req model.GetConfigMapYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.configMapService.GetConfigMapYAML(ctx, &req)
	})
}

// CreateConfigMapByYaml 通过YAML创建ConfigMap
func (h *K8sConfigMapHandler) CreateConfigMapByYaml(ctx *gin.Context) {
	var req model.CreateConfigMapByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.CreateConfigMapByYaml(ctx, &req)
	})
}

// UpdateConfigMapByYaml 通过YAML更新ConfigMap
func (h *K8sConfigMapHandler) UpdateConfigMapByYaml(ctx *gin.Context) {
	var req model.UpdateConfigMapByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	ns, err := utils.GetParamCustomName(ctx, "namespace")
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
	req.Namespace = ns
	req.Name = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.configMapService.UpdateConfigMapByYaml(ctx, &req)
	})
}
