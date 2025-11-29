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

type K8sServiceAccountHandler struct {
	serviceAccountService service.ServiceAccountService
}

func NewK8sServiceAccountHandler(serviceAccountService service.ServiceAccountService) *K8sServiceAccountHandler {
	return &K8sServiceAccountHandler{
		serviceAccountService: serviceAccountService,
	}
}

func (h *K8sServiceAccountHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/serviceaccount/:cluster_id/list", h.GetServiceAccountList)
		k8sGroup.GET("/serviceaccount/:cluster_id/:namespace/:name/detail", h.GetServiceAccountDetails)
		k8sGroup.GET("/serviceaccount/:cluster_id/:namespace/:name/detail/yaml", h.GetServiceAccountYaml)
		k8sGroup.POST("/serviceaccount/:cluster_id/create", h.CreateServiceAccount)
		k8sGroup.POST("/serviceaccount/:cluster_id/create/yaml", h.CreateServiceAccountByYaml)
		k8sGroup.PUT("/serviceaccount/:cluster_id/:namespace/:name/update", h.UpdateServiceAccount)
		k8sGroup.PUT("/serviceaccount/:cluster_id/:namespace/:name/update/yaml", h.UpdateServiceAccountYaml)
		k8sGroup.DELETE("/serviceaccount/:cluster_id/:namespace/:name/delete", h.DeleteServiceAccount)
		k8sGroup.GET("/serviceaccount/:cluster_id/:namespace/:name/token", h.GetServiceAccountToken)
		k8sGroup.POST("/serviceaccount/:cluster_id/:namespace/:name/token", h.CreateServiceAccountToken)
	}
}

func (h *K8sServiceAccountHandler) GetServiceAccountList(ctx *gin.Context) {
	var req model.GetServiceAccountListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.serviceAccountService.GetServiceAccountList(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) GetServiceAccountDetails(ctx *gin.Context) {
	var req model.GetServiceAccountDetailsReq

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
		return h.serviceAccountService.GetServiceAccountDetails(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) CreateServiceAccount(ctx *gin.Context) {
	var req model.CreateServiceAccountReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.serviceAccountService.CreateServiceAccount(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) CreateServiceAccountByYaml(ctx *gin.Context) {
	var req model.CreateServiceAccountByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.serviceAccountService.CreateServiceAccountByYaml(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) UpdateServiceAccount(ctx *gin.Context) {
	var req model.UpdateServiceAccountReq

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
		return nil, h.serviceAccountService.UpdateServiceAccount(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) DeleteServiceAccount(ctx *gin.Context) {
	var req model.DeleteServiceAccountReq

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
		return nil, h.serviceAccountService.DeleteServiceAccount(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) GetServiceAccountYaml(ctx *gin.Context) {
	var req model.GetServiceAccountYamlReq

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
		return h.serviceAccountService.GetServiceAccountYaml(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) UpdateServiceAccountYaml(ctx *gin.Context) {
	var req model.UpdateServiceAccountByYamlReq

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
		return nil, h.serviceAccountService.UpdateServiceAccountYaml(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) GetServiceAccountToken(ctx *gin.Context) {
	var req model.GetServiceAccountTokenReq

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
		return h.serviceAccountService.GetServiceAccountToken(ctx, &req)
	})
}

func (h *K8sServiceAccountHandler) CreateServiceAccountToken(ctx *gin.Context) {
	var req model.CreateServiceAccountTokenReq

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
	req.ServiceAccountName = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.serviceAccountService.CreateServiceAccountToken(ctx, &req)
	})
}
