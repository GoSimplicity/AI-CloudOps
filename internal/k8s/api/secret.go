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

// K8sSecretHandler Secret处理器
type K8sSecretHandler struct {
	secretService service.SecretService
}

// NewK8sSecretHandler 创建Secret处理器
func NewK8sSecretHandler(secretService service.SecretService) *K8sSecretHandler {
	return &K8sSecretHandler{
		secretService: secretService,
	}
}

// RegisterRouters 注册路由
func (h *K8sSecretHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/secret/:cluster_id/list", h.GetSecretList)
		k8sGroup.GET("/secret/:cluster_id/:namespace/:name/detail", h.GetSecret)
		k8sGroup.GET("/secret/:cluster_id/:namespace/:name/detail/yaml", h.GetSecretYAML)
		k8sGroup.POST("/secret/:cluster_id/create", h.CreateSecret)
		k8sGroup.POST("/secret/:cluster_id/create/yaml", h.CreateSecretByYaml)
		k8sGroup.PUT("/secret/:cluster_id/:namespace/:name/update", h.UpdateSecret)
		k8sGroup.PUT("/secret/:cluster_id/:namespace/:name/update/yaml", h.UpdateSecretByYaml)
		k8sGroup.DELETE("/secret/:cluster_id/:namespace/:name/delete", h.DeleteSecret)
	}
}

func (h *K8sSecretHandler) GetSecretList(ctx *gin.Context) {
	var req model.GetSecretListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.secretService.GetSecretList(ctx, &req)
	})
}

func (h *K8sSecretHandler) GetSecret(ctx *gin.Context) {
	var req model.GetSecretDetailsReq

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
		return h.secretService.GetSecret(ctx, &req)
	})
}

func (h *K8sSecretHandler) CreateSecret(ctx *gin.Context) {
	var req model.CreateSecretReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecret(ctx, &req)
	})
}

func (h *K8sSecretHandler) UpdateSecret(ctx *gin.Context) {
	var req model.UpdateSecretReq

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
		return nil, h.secretService.UpdateSecret(ctx, &req)
	})
}

func (h *K8sSecretHandler) DeleteSecret(ctx *gin.Context) {
	var req model.DeleteSecretReq

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
		return nil, h.secretService.DeleteSecret(ctx, &req)
	})
}

func (h *K8sSecretHandler) GetSecretYAML(ctx *gin.Context) {
	var req model.GetSecretYamlReq

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
		return h.secretService.GetSecretYAML(ctx, &req)
	})
}

func (h *K8sSecretHandler) CreateSecretByYaml(ctx *gin.Context) {
	var req model.CreateSecretByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecretByYaml(ctx, &req)
	})
}

func (h *K8sSecretHandler) UpdateSecretByYaml(ctx *gin.Context) {
	var req model.UpdateSecretByYamlReq

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
		return nil, h.secretService.UpdateSecretByYaml(ctx, &req)
	})
}
