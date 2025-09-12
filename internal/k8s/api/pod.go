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

type K8sPodHandler struct {
	podService service.PodService
}

func NewK8sPodHandler(podService service.PodService) *K8sPodHandler {
	return &K8sPodHandler{

		podService: podService,
	}
}

func (k *K8sPodHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Pod基础管理
		k8sGroup.GET("/pod/:cluster_id/list", k.GetPodList)                                                           // 获取Pod列表
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail", k.GetPodDetails)                                     // 获取Pod详情
		k8sGroup.GET("/pod/:cluster_id/:namespace/:name/detail/yaml", k.GetPodYaml)                                   // 获取Pod YAML
		k8sGroup.POST("/pod/:cluster_id/create", k.CreatePod)                                                         // 创建Pod
		k8sGroup.POST("/pod/:cluster_id/create/yaml", k.CreatePodByYaml)                                              // 通过YAML创建Pod
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update", k.UpdatePod)                                         // 更新Pod
		k8sGroup.PUT("/pod/:cluster_id/:namespace/:name/update/yaml", k.UpdatePodByYaml)                              // 通过YAML更新Pod
		k8sGroup.DELETE("/pod/:cluster_id/:namespace/:name/delete", k.DeletePod)                                      // 删除Pod
		k8sGroup.GET("/pod/:cluster_id/:namespace/:pod_name/containers", k.GetPodContainers)                          // 获取Pod容器列表
		k8sGroup.GET("/pod/:cluster_id/:namespace/:pod_name/containers/:container/logs", k.GetPodLogs)                // 获取容器日志
		k8sGroup.POST("/pod/:cluster_id/:namespace/:pod_name/containers/:container/exec", k.PodExec)                  // Pod执行命令
		k8sGroup.POST("/pod/:cluster_id/:namespace/:pod_name/port-forward", k.PodPortForward)                         // Pod端口转发
		k8sGroup.POST("/pod/:cluster_id/:namespace/:pod_name/containers/:container/files/upload", k.PodFileUpload)    // Pod文件上传
		k8sGroup.GET("/pod/:cluster_id/:namespace/:pod_name/containers/:container/files/download", k.PodFileDownload) // Pod文件下载
	}
}

// GetPodDetails 获取Pod详情
func (k *K8sPodHandler) GetPodDetails(ctx *gin.Context) {
	var req model.GetPodDetailsReq

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
		return k.podService.GetPodDetails(ctx, &req)
	})
}

// GetPodList 获取Pod列表
func (k *K8sPodHandler) GetPodList(ctx *gin.Context) {
	var req model.GetPodListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podService.GetPodList(ctx, &req)
	})
}

// GetPodContainers 获取Pod的容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	var req model.GetPodContainersReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.podService.GetPodContainers(ctx, &req)
	})
}

// GetPodLogs 获取容器日志
func (k *K8sPodHandler) GetPodLogs(ctx *gin.Context) {
	var req model.GetPodLogsReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.Container = container

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.GetPodLogs(ctx, &req)
	})
}

// GetPodYaml 获取Pod的YAML配置
func (k *K8sPodHandler) GetPodYaml(ctx *gin.Context) {
	var req model.GetPodYamlReq

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
		return k.podService.GetPodYaml(ctx, &req)
	})
}

// CreatePod 创建Pod
func (k *K8sPodHandler) CreatePod(ctx *gin.Context) {
	var req model.CreatePodReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.CreatePod(ctx, &req)
	})
}

// CreatePodByYaml 通过YAML创建Pod
func (k *K8sPodHandler) CreatePodByYaml(ctx *gin.Context) {
	var req model.CreatePodByYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.CreatePodByYaml(ctx, &req)
	})
}

// UpdatePod 更新Pod
func (k *K8sPodHandler) UpdatePod(ctx *gin.Context) {
	var req model.UpdatePodReq

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
		return nil, k.podService.UpdatePod(ctx, &req)
	})
}

// UpdatePodByYaml 通过YAML更新Pod
func (k *K8sPodHandler) UpdatePodByYaml(ctx *gin.Context) {
	var req model.UpdatePodByYamlReq

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
		return nil, k.podService.UpdatePodByYaml(ctx, &req)
	})
}

// DeletePod 删除Pod
func (k *K8sPodHandler) DeletePod(ctx *gin.Context) {
	var req model.DeletePodReq

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
		return nil, k.podService.DeletePod(ctx, &req)
	})
}

// PodExec Pod执行命令
func (k *K8sPodHandler) PodExec(ctx *gin.Context) {
	var req model.PodExecReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.Container = container

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.PodExec(ctx, &req)
	})
}

// PodPortForward Pod端口转发
func (k *K8sPodHandler) PodPortForward(ctx *gin.Context) {
	var req model.PodPortForwardReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.PodPortForward(ctx, &req)
	})
}

// PodFileUpload Pod文件上传
func (k *K8sPodHandler) PodFileUpload(ctx *gin.Context) {
	var req model.PodFileUploadReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.PodFileUpload(ctx, &req)
	})
}

// PodFileDownload Pod文件下载
func (k *K8sPodHandler) PodFileDownload(ctx *gin.Context) {
	var req model.PodFileDownloadReq

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

	podName, err := utils.GetParamCustomName(ctx, "pod_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	container, err := utils.GetParamCustomName(ctx, "container")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.PodName = podName
	req.ContainerName = container

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.podService.PodFileDownload(ctx, &req)
	})
}
