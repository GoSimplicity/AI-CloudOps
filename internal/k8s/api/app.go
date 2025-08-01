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
// @Summary 获取集群唯一命名空间列表
// @Description 查询集群中所有不重复的命名空间名称，用于下拉选择和过滤操作
// @Tags 应用管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]string} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/namespaces/unique [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetClusterNamespacesUnique(ctx *gin.Context) {
	return
}

// CreateK8sInstance 创建 Kubernetes 实例
// @Summary 创建K8s实例
// @Description 创建新的Kubernetes工作负载实例，支持Deployment、StatefulSet等类型
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param request body model.K8sInstance true "实例创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/create [post]
// @Security BearerAuth
func (h *K8sAppHandler) CreateK8sInstance(ctx *gin.Context) {
	var req model.K8sInstance
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.CreateInstance(ctx, &req)
	})
}

// UpdateK8sInstance 更新 Kubernetes 实例
// @Summary 更新K8s实例配置
// @Description 更新指定Kubernetes实例的配置信息，包括镜像版本、资源限制等
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param request body model.K8sInstance true "实例更新信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/update/{id} [put]
// @Security BearerAuth
func (h *K8sAppHandler) UpdateK8sInstance(ctx *gin.Context) {
	var req model.K8sInstance
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.UpdateInstance(ctx, &req)
	})
}

// BatchDeleteK8sInstance 批量删除 Kubernetes 实例
// @Summary 批量删除K8s实例
// @Description 同时删除多个Kubernetes实例，支持跨命名空间和集群的批量操作
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteK8sInstanceReq true "批量删除实例请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/delete [delete]
// @Security BearerAuth
func (h *K8sAppHandler) BatchDeleteK8sInstance(ctx *gin.Context) {
	var req model.BatchDeleteK8sInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.BatchDeleteInstance(ctx, &req)
	})
}

// BatchRestartK8sInstance 批量重启 Kubernetes 实例
// @Summary 批量重启K8s实例
// @Description 同时重启多个Kubernetes实例，触发滚动更新以应用最新配置
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param request body model.BatchRestartK8sInstanceReq true "批量重启实例请求"
// @Success 200 {object} utils.ApiResponse "重启成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/restart [post]
// @Security BearerAuth
func (h *K8sAppHandler) BatchRestartK8sInstance(ctx *gin.Context) {
	var req model.BatchRestartK8sInstanceReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.BatchRestartInstance(ctx, &req)
	})
}

// GetK8sInstanceByApp 根据应用获取 Kubernetes 实例
// @Summary 根据应用获取K8s实例列表
// @Description 根据应用ID和集群信息查询对应的所有Kubernetes实例，支持命名空间过滤
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param app_id query int64 true "应用ID"
// @Param cluster_id query int64 true "集群ID"
// @Param namespace query string true "命名空间"
// @Success 200 {object} utils.ApiResponse{data=[]interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/by-app [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sInstanceByApp(ctx *gin.Context) {
	appID, err := utils.GetQueryParam[int64](ctx, "app_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterID, err := utils.GetQueryParam[int64](ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetQueryParam[string](ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req := &model.GetK8sInstanceByAppReq{
		AppID:     int(appID),
		ClusterID: int(clusterID),
		Namespace: namespace,
	}
	
	resp, err := h.instanceService.GetInstanceByApp(ctx, req)
	if err != nil {
		utils.ErrorWithDetails(ctx, err.Error(), "获取实例失败")
		return
	}
	utils.SuccessWithData(ctx, resp.Items)
}

// GetK8sInstanceList 获取 Kubernetes 实例列表
// @Summary 获取K8s实例列表
// @Description 查询Kubernetes实例列表，支持分页、过滤和排序功能
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param request body model.GetK8sInstanceListReq true "查询条件"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/instances [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sInstanceList(ctx *gin.Context) {
	var req model.GetK8sInstanceListReq
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.instanceService.GetInstanceList(ctx, &req)
	})
}

// GetK8sInstance 获取单个 Kubernetes 实例
// @Summary 获取K8s实例详情
// @Description 根据ID和相关参数获取单个Kubernetes实例的详细信息和运行状态
// @Tags 实例管理
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param name query string true "实例名称"
// @Param namespace query string true "命名空间"
// @Param cluster_id query int64 true "集群ID"
// @Param type query string true "实例类型（deployment、statefulset等）"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/instances/{id} [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sInstance(ctx *gin.Context) {
	name, err := utils.GetQueryParam[string](ctx, "name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := utils.GetQueryParam[string](ctx, "namespace")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	clusterID, err := utils.GetQueryParam[int64](ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	t, err := utils.GetQueryParam[string](ctx, "type")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req := &model.GetK8sInstanceReq{
		Name:      name,
		Namespace: namespace,
		ClusterID: int(clusterID),
		Type:      t,
	}

	resp, err := h.instanceService.GetInstance(ctx, req)
	if err != nil {
		utils.ErrorWithDetails(ctx, err.Error(), "获取实例失败")
		return
	}
	utils.SuccessWithData(ctx, resp.Item)
}

// GetK8sAppList 获取 Kubernetes 应用列表
// @Summary 获取K8s应用列表
// @Description 查询Kubernetes应用列表，支持按项目、集群和命名空间过滤，包含分页功能
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param request body model.GetK8sAppListRequest true "查询条件"
// @Success 200 {object} utils.ApiResponse{data=interface{}} "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/apps/by-app [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sAppList(ctx *gin.Context) {
	var req model.GetK8sAppListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.appService.GetAppList(ctx, &req)
	})
}

// CreateK8sApp 创建 Kubernetes 应用
// @Summary 创建K8s应用
// @Description 创建新的Kubernetes应用，自动生成Deployment和Service资源
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param request body model.CreateK8sAppRequest true "应用创建信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/k8sApp/apps/create [post]
// @Security BearerAuth
func (h *K8sAppHandler) CreateK8sApp(ctx *gin.Context) {
	var req model.CreateK8sAppRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.appService.CreateApp(ctx, &req)
	})
}

// UpdateK8sApp 更新 Kubernetes 应用
// @Summary 更新 Kubernetes 应用
// @Description 更新指定的 Kubernetes 应用信息
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Param request body model.UpdateK8sAppRequest true "应用信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/apps/update/{id} [put]
// @Security BearerAuth
func (h *K8sAppHandler) UpdateK8sApp(ctx *gin.Context) {
	var req model.UpdateK8sAppRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.appService.UpdateApp(ctx, &req)
	})
}

// DeleteK8sApp 删除 Kubernetes 应用
// @Summary 删除 Kubernetes 应用
// @Description 删除指定的 Kubernetes 应用
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/apps/{id} [delete]
// @Security BearerAuth
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
// @Summary 获取单个 Kubernetes 应用
// @Description 根据ID获取单个 Kubernetes 应用的详细信息
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/apps/{id} [get]
// @Security BearerAuth
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
// @Summary 根据部署获取 Pod 列表
// @Description 根据部署ID获取对应的 Kubernetes Pod 列表
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "部署ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/apps/{id}/pods [get]
// @Security BearerAuth
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
// @Summary 获取用于选择的应用列表
// @Description 获取用于下拉选择的 Kubernetes 应用列表
// @Tags K8s管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/apps/select [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sAppListForSelect(ctx *gin.Context) {
	// TODO: 暂未实现
}

// GetK8sProjectList 获取 Kubernetes 项目列表
// @Summary 获取 Kubernetes 项目列表
// @Description 获取 Kubernetes 项目列表，支持分页和筛选
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param request body model.GetK8sProjectListRequest true "查询条件"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/projects/all [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sProjectList(ctx *gin.Context) {
	var req model.GetK8sProjectListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.projectService.GetProjectList(ctx, &req)
	})
}

// GetK8sProjectListForSelect 获取用于选择的 Kubernetes 项目列表
// @Summary 获取用于选择的项目列表
// @Description 获取用于下拉选择的 Kubernetes 项目列表
// @Tags K8s管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/projects/select [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sProjectListForSelect(ctx *gin.Context) {
	// TODO: 暂未实现
}

// CreateK8sProject 创建 Kubernetes 项目
// @Summary 创建 Kubernetes 项目
// @Description 创建新的 Kubernetes 项目
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param request body model.CreateK8sProjectRequest true "项目信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/projects/create [post]
// @Security BearerAuth
func (h *K8sAppHandler) CreateK8sProject(ctx *gin.Context) {
	var req model.CreateK8sProjectRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.projectService.CreateProject(ctx, &req)
	})
}

// UpdateK8sProject 更新 Kubernetes 项目
// @Summary 更新 Kubernetes 项目
// @Description 更新指定的 Kubernetes 项目信息
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Param request body model.UpdateK8sProjectRequest true "项目信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/projects/update/{id} [put]
// @Security BearerAuth
func (h *K8sAppHandler) UpdateK8sProject(ctx *gin.Context) {
	var req model.UpdateK8sProjectRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.projectService.UpdateProject(ctx, &req)
	})
}

// DeleteK8sProject 删除 Kubernetes 项目
// @Summary 删除 Kubernetes 项目
// @Description 删除指定的 Kubernetes 项目
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "项目ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/projects/{id} [delete]
// @Security BearerAuth
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
// @Summary 获取 CronJob 列表
// @Description 获取 Kubernetes CronJob 列表，支持分页和筛选
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param request body model.GetK8sCronjobListRequest true "查询条件"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/list [get]
// @Security BearerAuth
func (h *K8sAppHandler) GetK8sCronjobList(ctx *gin.Context) {
	var req model.GetK8sCronjobListRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cronjobService.GetCronjobList(ctx, &req)
	})
}

// CreateK8sCronjob 创建 CronJob
// @Summary 创建 CronJob
// @Description 创建新的 Kubernetes CronJob
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param request body model.CreateK8sCronjobRequest true "CronJob信息"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/create [post]
// @Security BearerAuth
func (h *K8sAppHandler) CreateK8sCronjob(ctx *gin.Context) {
	var req model.CreateK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.CreateCronjob(ctx, &req)
	})
}

// UpdateK8sCronjob 更新 CronJob
// @Summary 更新 CronJob
// @Description 更新指定的 Kubernetes CronJob
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "CronJob ID"
// @Param request body model.UpdateK8sCronjobRequest true "CronJob信息"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/{id} [put]
// @Security BearerAuth
func (h *K8sAppHandler) UpdateK8sCronjob(ctx *gin.Context) {
	var req model.UpdateK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.UpdateCronjob(ctx, &req)
	})
}

// GetK8sCronjob 获取单个 CronJob
// @Summary 获取单个 CronJob
// @Description 根据ID获取单个 Kubernetes CronJob 的详细信息
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "CronJob ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/{id} [get]
// @Security BearerAuth
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
// @Summary 获取 CronJob 最近的 Pod
// @Description 获取指定 CronJob 最近执行的 Pod 信息
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param id path int true "CronJob ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/{id}/last-pod [get]
// @Security BearerAuth
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
// @Summary 批量删除 CronJob
// @Description 批量删除指定的 Kubernetes CronJob
// @Tags K8s管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteK8sCronjobRequest true "批量删除请求"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "内部服务器错误"
// @Router /api/k8s/k8sApp/cronJobs/delete [delete]
// @Security BearerAuth
func (h *K8sAppHandler) BatchDeleteK8sCronjob(ctx *gin.Context) {
	var req model.BatchDeleteK8sCronjobRequest
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.cronjobService.BatchDeleteCronjob(ctx, req.CronjobIDs)
	})
}
