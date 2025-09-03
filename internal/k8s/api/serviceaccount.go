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

type K8sServiceAccountHandler struct {
	serviceAccountService service.ServiceAccountService
}

func NewK8sServiceAccountHandler(serviceAccountService service.ServiceAccountService) *K8sServiceAccountHandler {
	return &K8sServiceAccountHandler{
		serviceAccountService: serviceAccountService,
	}
}

func (s *K8sServiceAccountHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/serviceaccounts", s.GetServiceAccountList)
		k8sGroup.GET("/serviceaccounts/:cluster_id/:namespace/:name/details", s.GetServiceAccountDetails)
		k8sGroup.POST("/serviceaccounts", s.CreateServiceAccount)
		k8sGroup.PUT("/serviceaccounts/:cluster_id/:namespace/:name/update", s.UpdateServiceAccount)
		k8sGroup.DELETE("/serviceaccounts/:cluster_id/:namespace/:name", s.DeleteServiceAccount)
		k8sGroup.GET("/serviceaccounts/:cluster_id/:namespace/:name/yaml", s.GetServiceAccountYaml)
		k8sGroup.PUT("/serviceaccounts/:cluster_id/:namespace/:name/yaml", s.UpdateServiceAccountYaml)
		k8sGroup.GET("/serviceaccounts/:cluster_id/:namespace/:name/token", s.GetServiceAccountToken)
		k8sGroup.POST("/serviceaccounts/token", s.CreateServiceAccountToken)
	}
}

// GetServiceAccountList 获取 ServiceAccount 列表
func (s *K8sServiceAccountHandler) GetServiceAccountList(ctx *gin.Context) {
	var req model.GetServiceAccountListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return s.serviceAccountService.GetServiceAccountList(ctx, &req)
	})
}

// GetServiceAccountDetails 获取 ServiceAccount 详情
func (s *K8sServiceAccountHandler) GetServiceAccountDetails(ctx *gin.Context) {
	var req model.GetServiceAccountDetailsReq

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
		return s.serviceAccountService.GetServiceAccountDetails(ctx, &req)
	})
}

// CreateServiceAccount 创建 ServiceAccount
func (s *K8sServiceAccountHandler) CreateServiceAccount(ctx *gin.Context) {
	var req model.CreateServiceAccountReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, s.serviceAccountService.CreateServiceAccount(ctx, &req)
	})
}

// UpdateServiceAccount 更新 ServiceAccount
func (s *K8sServiceAccountHandler) UpdateServiceAccount(ctx *gin.Context) {
	var req model.UpdateServiceAccountReq

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
		return nil, s.serviceAccountService.UpdateServiceAccount(ctx, &req)
	})
}

// DeleteServiceAccount 删除 ServiceAccount
func (s *K8sServiceAccountHandler) DeleteServiceAccount(ctx *gin.Context) {
	var req model.DeleteServiceAccountReq

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
		return nil, s.serviceAccountService.DeleteServiceAccount(ctx, &req)
	})
}

// GetServiceAccountYaml 获取 ServiceAccount YAML
func (s *K8sServiceAccountHandler) GetServiceAccountYaml(ctx *gin.Context) {
	var req model.GetServiceAccountYamlReq

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
		return s.serviceAccountService.GetServiceAccountYaml(ctx, &req)
	})
}

// UpdateServiceAccountYaml 更新 ServiceAccount YAML
func (s *K8sServiceAccountHandler) UpdateServiceAccountYaml(ctx *gin.Context) {
	var req model.UpdateServiceAccountYamlReq

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
		return nil, s.serviceAccountService.UpdateServiceAccountYaml(ctx, &req)
	})
}

// GetServiceAccountToken 获取 ServiceAccount 令牌
func (s *K8sServiceAccountHandler) GetServiceAccountToken(ctx *gin.Context) {
	var req model.GetServiceAccountTokenReq

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
		return s.serviceAccountService.GetServiceAccountToken(ctx, &req)
	})
}

// CreateServiceAccountToken 创建 ServiceAccount 令牌
func (s *K8sServiceAccountHandler) CreateServiceAccountToken(ctx *gin.Context) {
	var req model.CreateServiceAccountTokenReq

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
		return s.serviceAccountService.CreateServiceAccountToken(ctx, &req)
	})
}
