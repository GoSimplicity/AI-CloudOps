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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// K8sAppHandler 处理 Kubernetes 应用相关的 API 请求
type K8sAppHandler struct {
	logger          *zap.Logger
	appService      user.AppService
	instanceService user.InstanceService
	projectService  user.ProjectService
	cronjobService  user.CronjobService
}

// NewK8sAppHandler 创建新的 K8sAppHandler 实例
func NewK8sAppHandler(
	logger *zap.Logger,
	instanceService user.InstanceService,
	appService user.AppService,
	projectService user.ProjectService,
	cronjobService user.CronjobService,
) *K8sAppHandler {
	return &K8sAppHandler{
		logger:          logger,
		projectService:  projectService,
		instanceService: instanceService,
		cronjobService:  cronjobService,
		appService:      appService,
	}
}

// RegisterRouters 注册 K8s 应用相关的路由
func (h *K8sAppHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	k8sAppGroup := k8sGroup.Group("/k8sApp")
	{
		// 命名空间
		k8sAppGroup.GET("/namespaces/unique", h.GetClusterNamespacesUnique)

		// 实例
		instances := k8sAppGroup.Group("/instances")
		{
			instances.POST("/create", h.CreateK8sInstance)        // 创建 Kubernetes 实例
			instances.PUT("/update/:id", h.UpdateK8sInstance)     // 更新 Kubernetes 实例
			instances.DELETE("/delete", h.BatchDeleteK8sInstance) // 批量删除 Kubernetes 实例
			instances.POST("/restart", h.BatchRestartK8sInstance) // 批量重启 Kubernetes 实例
			instances.GET("/by-app", h.GetK8sInstanceByApp)       // 根据应用获取 Kubernetes 实例
			instances.GET("/instances", h.GetK8sInstanceList)     // 获取 Kubernetes 实例列表
			instances.GET("/:id", h.GetK8sInstance)               // 获取单个 Kubernetes 实例
		}

		// 应用 Deployment 和 Service 的抽象
		apps := k8sAppGroup.Group("/apps")
		{
			apps.POST("/create", h.CreateK8sApp)            // 创建 Kubernetes 应用
			apps.PUT("/update/:id", h.UpdateK8sApp)         // 更新 Kubernetes 应用
			apps.DELETE("/:id", h.DeleteK8sApp)             // 删除 Kubernetes 应用
			apps.GET("/:id", h.GetK8sApp)                   // 获取单个 Kubernetes 应用
			apps.GET("/by-app", h.GetK8sAppList)            // 获取 Kubernetes 应用列表
			apps.GET("/:id/pods", h.GetK8sPodsByDeployment) // 根据部署获取 Kubernetes Pod 列表
			apps.GET("/select", h.GetK8sAppListForSelect)   // 获取用于选择的 Kubernetes 应用列表
		}

		// 项目
		projects := k8sAppGroup.Group("/projects")
		{
			projects.GET("/all", h.GetK8sProjectList)             // 获取 Kubernetes 项目列表
			projects.GET("/select", h.GetK8sProjectListForSelect) // 获取用于选择的 Kubernetes 项目列表
			projects.POST("/create", h.CreateK8sProject)          // 创建 Kubernetes 项目
			projects.PUT("/update/:id", h.UpdateK8sProject)       // 更新 Kubernetes 项目
			projects.DELETE("/:id", h.DeleteK8sProject)           // 删除 Kubernetes 项目
		}

		// CronJob
		cronJobs := k8sAppGroup.Group("/cronJobs")
		{
			cronJobs.GET("/list", h.GetK8sCronjobList)            // 获取 CronJob 列表
			cronJobs.POST("/create", h.CreateK8sCronjob)          // 创建 CronJob
			cronJobs.PUT("/:id", h.UpdateK8sCronjob)              // 更新 CronJob
			cronJobs.GET("/:id", h.GetK8sCronjob)                 // 获取单个 CronJob
			cronJobs.GET("/:id/last-pod", h.GetK8sCronjobLastPod) // 获取 CronJob 最近的 Pod
			cronJobs.DELETE("/delete", h.BatchDeleteK8sCronjob)   // 批量删除 CronJob
		}
	}
}

// GetClusterNamespacesUnique 获取唯一的命名空间列表
func (h *K8sAppHandler) GetClusterNamespacesUnique(ctx *gin.Context) {
	return
}

// CreateK8sInstance 创建 Kubernetes 实例
func (h *K8sAppHandler) CreateK8sInstance(ctx *gin.Context) {
	var req model.CreateK8sInstanceRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.instanceService.CreateInstance(ctx, &req)
	})
}

// UpdateK8sInstance 更新 Kubernetes 实例
func (h *K8sAppHandler) UpdateK8sInstance(ctx *gin.Context) {
	var req model.UpdateK8sInstanceRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.instanceService.UpdateInstance(ctx, &req)
	})
}

// BatchDeleteK8sInstance 批量删除 Kubernetes 实例
func (h *K8sAppHandler) BatchDeleteK8sInstance(ctx *gin.Context) {
	var req model.BatchDeleteK8sInstanceRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.instanceService.BatchDeleteInstance(ctx, req.InstanceIDs)
	})
}

// BatchRestartK8sInstance 批量重启 Kubernetes 实例
func (h *K8sAppHandler) BatchRestartK8sInstance(ctx *gin.Context) {
	var req model.BatchRestartK8sInstanceRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.instanceService.BatchRestartInstance(ctx, req.InstanceIDs)
	})
}

// GetK8sInstanceByApp 根据应用获取 Kubernetes 实例
func (h *K8sAppHandler) GetK8sInstanceByApp(ctx *gin.Context) {
	appID, err := utils.GetQueryParam[int64](ctx, "app_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	instances, err := h.instanceService.GetInstanceByApp(ctx, appID)
	if err != nil {
		utils.ErrorWithDetails(ctx, err.Error(), "获取实例失败")
		return
	}
	utils.SuccessWithData(ctx, instances)
}

// GetK8sInstanceList 获取 Kubernetes 实例列表
func (h *K8sAppHandler) GetK8sInstanceList(ctx *gin.Context) {
	var req model.GetK8sInstanceListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.GetInstanceList(ctx, &req)
	})
}

// GetK8sInstance 获取单个 Kubernetes 实例
func (h *K8sAppHandler) GetK8sInstance(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	instance, err := h.instanceService.GetInstance(ctx, id)
	if err != nil {
		utils.ErrorWithDetails(ctx, err.Error(), "获取实例失败")
		return
	}
	utils.SuccessWithData(ctx, instance)
}

// GetK8sAppList 获取 Kubernetes 应用列表
func (h *K8sAppHandler) GetK8sAppList(ctx *gin.Context) {
	var req model.GetK8sAppListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.appService.GetAppList(ctx, &req)
	})
}

// CreateK8sApp 创建 Kubernetes 应用
func (h *K8sAppHandler) CreateK8sApp(ctx *gin.Context) {
	var req model.CreateK8sAppRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.appService.CreateApp(ctx, &req)
	})
}

// UpdateK8sApp 更新 Kubernetes 应用
func (h *K8sAppHandler) UpdateK8sApp(ctx *gin.Context) {
	var req model.UpdateK8sAppRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.appService.UpdateApp(ctx, &req)
	})
}

// DeleteK8sApp 删除 Kubernetes 应用
func (h *K8sAppHandler) DeleteK8sApp(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.appService.DeleteApp(ctx, id)
	})
}

// GetK8sApp 获取单个 Kubernetes 应用
func (h *K8sAppHandler) GetK8sApp(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.appService.GetApp(ctx, id)
	})
}

// GetK8sPodsByDeployment 根据部署获取 Kubernetes Pod 列表
func (h *K8sAppHandler) GetK8sPodsByDeployment(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.appService.GetPodListByDeploy(ctx, id)
	})
}

// GetK8sAppListForSelect 获取用于选择的 Kubernetes 应用列表
func (h *K8sAppHandler) GetK8sAppListForSelect(ctx *gin.Context) {
	// TODO: 暂未实现
}

// GetK8sProjectList 获取 Kubernetes 项目列表
func (h *K8sAppHandler) GetK8sProjectList(ctx *gin.Context) {
	var req model.GetK8sProjectListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.projectService.GetProjectList(ctx, &req)
	})
}

// GetK8sProjectListForSelect 获取用于选择的 Kubernetes 项目列表
func (h *K8sAppHandler) GetK8sProjectListForSelect(ctx *gin.Context) {
	// TODO: 暂未实现
}

// CreateK8sProject 创建 Kubernetes 项目
func (h *K8sAppHandler) CreateK8sProject(ctx *gin.Context) {
	var req model.CreateK8sProjectRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.projectService.CreateProject(ctx, &req)
	})
}

// UpdateK8sProject 更新 Kubernetes 项目
func (h *K8sAppHandler) UpdateK8sProject(ctx *gin.Context) {
	var req model.UpdateK8sProjectRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.projectService.UpdateProject(ctx, &req)
	})
}

// DeleteK8sProject 删除 Kubernetes 项目
func (h *K8sAppHandler) DeleteK8sProject(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.projectService.DeleteProject(ctx, id)
	})
}

// GetK8sCronjobList 获取 CronJob 列表
func (h *K8sAppHandler) GetK8sCronjobList(ctx *gin.Context) {
	var req model.GetK8sCronjobListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cronjobService.GetCronjobList(ctx, &req)
	})
}

// CreateK8sCronjob 创建 CronJob
func (h *K8sAppHandler) CreateK8sCronjob(ctx *gin.Context) {
	var req model.CreateK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.CreateCronjob(ctx, &req)
	})
}

// UpdateK8sCronjob 更新 CronJob
func (h *K8sAppHandler) UpdateK8sCronjob(ctx *gin.Context) {
	var req model.UpdateK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.UpdateCronjob(ctx, &req)
	})
}

// GetK8sCronjob 获取单个 CronJob
func (h *K8sAppHandler) GetK8sCronjob(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cronjobService.GetCronjob(ctx, id)
	})
}

// GetK8sCronjobLastPod 获取 CronJob 最近的 Pod
func (h *K8sAppHandler) GetK8sCronjobLastPod(ctx *gin.Context) {
	id, err := utils.GetQueryParam[int64](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cronjobService.GetCronjobLastPod(ctx, id)
	})
}

// BatchDeleteK8sCronjob 批量删除 CronJob
func (h *K8sAppHandler) BatchDeleteK8sCronjob(ctx *gin.Context) {
	var req model.BatchDeleteK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.BatchDeleteCronjob(ctx, req.CronjobIDs)
	})
}
