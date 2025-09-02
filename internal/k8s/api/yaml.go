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

type K8sYamlHandler struct {
	yamlService       service.YamlService
	deploymentService service.DeploymentService
	serviceService    service.SvcService
	configMapService  service.ConfigMapService
	secretService     service.SecretService
	ingressService    service.IngressService
	pvService         service.PVService
	pvcService        service.PVCService
}

func NewK8sYamlHandler(
	yamlService service.YamlService,
	deploymentService service.DeploymentService,
	serviceService service.SvcService,
	configMapService service.ConfigMapService,
	secretService service.SecretService,
	ingressService service.IngressService,
	pvService service.PVService,
	pvcService service.PVCService,
) *K8sYamlHandler {
	return &K8sYamlHandler{
		yamlService:       yamlService,
		deploymentService: deploymentService,
		serviceService:    serviceService,
		configMapService:  configMapService,
		secretService:     secretService,
		ingressService:    ingressService,
		pvService:         pvService,
		pvcService:        pvcService,
	}
}

func (k *K8sYamlHandler) RegisterRouters(server *gin.Engine) {
	yamlGroup := server.Group("/api/k8s/yaml")
	{
		yamlGroup.POST("/apply", k.ApplyYaml)                                                    // 应用YAML到集群
		yamlGroup.POST("/validate", k.ValidateYaml)                                              // 验证YAML格式
		yamlGroup.POST("/convert", k.ConvertToYaml)                                              // 将配置转换为YAML
		yamlGroup.POST("/deployments", k.CreateDeploymentByYaml)                                 // 通过YAML创建Deployment
		yamlGroup.PUT("/deployments/:cluster_id/:namespace/:name", k.UpdateDeploymentByYaml)     // 通过YAML更新Deployment
		yamlGroup.POST("/services", k.CreateServiceByYaml)                                       // 通过YAML创建Service
		yamlGroup.PUT("/services/:cluster_id/:namespace/:name", k.UpdateServiceByYaml)           // 通过YAML更新Service
		yamlGroup.POST("/configmaps", k.CreateConfigMapByYaml)                                   // 通过YAML创建ConfigMap
		yamlGroup.PUT("/configmaps/:cluster_id/:namespace/:name", k.UpdateConfigMapByYaml)       // 通过YAML更新ConfigMap
		yamlGroup.POST("/secrets", k.CreateSecretByYaml)                                         // 通过YAML创建Secret
		yamlGroup.PUT("/secrets/:cluster_id/:namespace/:name", k.UpdateSecretByYaml)             // 通过YAML更新Secret
		yamlGroup.POST("/ingresses", k.CreateIngressByYaml)                                      // 通过YAML创建Ingress
		yamlGroup.PUT("/ingresses/:cluster_id/:namespace/:name", k.UpdateIngressByYaml)          // 通过YAML更新Ingress
		yamlGroup.POST("/persistentvolumes", k.CreatePVByYaml)                                   // 通过YAML创建PersistentVolume
		yamlGroup.PUT("/persistentvolumes/:cluster_id/:name", k.UpdatePVByYaml)                  // 通过YAML更新PersistentVolume
		yamlGroup.POST("/persistentvolumeclaims", k.CreatePVCByYaml)                             // 通过YAML创建PersistentVolumeClaim
		yamlGroup.PUT("/persistentvolumeclaims/:cluster_id/:namespace/:name", k.UpdatePVCByYaml) // 通过YAML更新PersistentVolumeClaim
	}
}

// 通用YAML操作接口

// ApplyYaml 应用YAML到K8s集群
func (k *K8sYamlHandler) ApplyYaml(ctx *gin.Context) {
	var req model.ApplyResourceByYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.yamlService.ApplyYaml(ctx, &req)
	})
}

// ValidateYaml 验证YAML格式
func (k *K8sYamlHandler) ValidateYaml(ctx *gin.Context) {
	var req model.ValidateYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.yamlService.ValidateYaml(ctx, &req)
	})
}

// ConvertToYaml 将资源配置转换为YAML
func (k *K8sYamlHandler) ConvertToYaml(ctx *gin.Context) {
	var req model.ConvertToYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.yamlService.ConvertToYaml(ctx, &req)
	})
}

// CreateDeploymentByYaml 通过YAML创建Deployment
func (k *K8sYamlHandler) CreateDeploymentByYaml(ctx *gin.Context) {
	var req model.CreateDeploymentByYamlReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.deploymentService.CreateDeploymentByYaml(ctx, &req)
	})
}

// UpdateDeploymentByYaml 通过YAML更新Deployment
func (k *K8sYamlHandler) UpdateDeploymentByYaml(ctx *gin.Context) {
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

// Service YAML接口

// CreateServiceByYaml 通过YAML创建Service
func (k *K8sYamlHandler) CreateServiceByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypeService

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.serviceService.CreateServiceByYaml(ctx, &req)
	})
}

// UpdateServiceByYaml 通过YAML更新Service
func (k *K8sYamlHandler) UpdateServiceByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypeService

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
		return nil, k.serviceService.UpdateServiceByYaml(ctx, &req)
	})
}

// ConfigMap YAML接口

// CreateConfigMapByYaml 通过YAML创建ConfigMap
func (k *K8sYamlHandler) CreateConfigMapByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypeConfigMap

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.configMapService.CreateConfigMapByYaml(ctx, &req)
	})
}

// UpdateConfigMapByYaml 通过YAML更新ConfigMap
func (k *K8sYamlHandler) UpdateConfigMapByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypeConfigMap

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
		return nil, k.configMapService.UpdateConfigMapByYaml(ctx, &req)
	})
}

// Secret YAML接口

// CreateSecretByYaml 通过YAML创建Secret
func (k *K8sYamlHandler) CreateSecretByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypeSecret

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.secretService.CreateSecretByYaml(ctx, &req)
	})
}

// UpdateSecretByYaml 通过YAML更新Secret
func (k *K8sYamlHandler) UpdateSecretByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypeSecret

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
		return nil, k.secretService.UpdateSecretByYaml(ctx, &req)
	})
}

// Ingress YAML接口

// CreateIngressByYaml 通过YAML创建Ingress
func (k *K8sYamlHandler) CreateIngressByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypeIngress

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.ingressService.CreateIngressByYaml(ctx, &req)
	})
}

// UpdateIngressByYaml 通过YAML更新Ingress
func (k *K8sYamlHandler) UpdateIngressByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypeIngress

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
		return nil, k.ingressService.UpdateIngressByYaml(ctx, &req)
	})
}

// PersistentVolume YAML接口

// CreatePVByYaml 通过YAML创建PersistentVolume
func (k *K8sYamlHandler) CreatePVByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypePV

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.CreatePVByYaml(ctx, &req)
	})
}

// UpdatePVByYaml 通过YAML更新PersistentVolume
func (k *K8sYamlHandler) UpdatePVByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypePV

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
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
	req.Name = name
	// PV是集群级别资源，没有namespace

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvService.UpdatePVByYaml(ctx, &req)
	})
}

// PersistentVolumeClaim YAML接口

// CreatePVCByYaml 通过YAML创建PersistentVolumeClaim
func (k *K8sYamlHandler) CreatePVCByYaml(ctx *gin.Context) {
	var req model.CreateResourceByYamlReq
	req.ResourceType = model.ResourceTypePVC

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.pvcService.CreatePVCByYaml(ctx, &req)
	})
}

// UpdatePVCByYaml 通过YAML更新PersistentVolumeClaim
func (k *K8sYamlHandler) UpdatePVCByYaml(ctx *gin.Context) {
	var req model.UpdateResourceByYamlReq
	req.ResourceType = model.ResourceTypePVC

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
		return nil, k.pvcService.UpdatePVCByYaml(ctx, &req)
	})
}
