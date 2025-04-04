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
	user2 "github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/user"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type K8sAppHandler struct {
	l               *zap.Logger
	appService      user2.AppService
	instanceService user2.InstanceService
	projectService  user2.ProjectService
	cronjobService  user2.CronjobService
}

func NewK8sAppHandler(l *zap.Logger, instanceService user2.InstanceService, appService user2.AppService, projectService user2.ProjectService, cronjobService user2.CronjobService) *K8sAppHandler {
	return &K8sAppHandler{
		l:               l,
		projectService:  projectService,
		instanceService: instanceService,
		cronjobService:  cronjobService,
		appService:      appService,
	}
}

func (k *K8sAppHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	k8sAppApiGroup := k8sGroup.Group("/k8sApp")
	{
		// 命名空间
		k8sAppApiGroup.GET("/namespaces/unique", k.GetClusterNamespacesUnique) // 获取唯一的命名空间列表

		// 实例
		instances := k8sAppApiGroup.Group("/instances")
		{
			instances.POST("/create", k.CreateK8sInstanceOne)     // 创建单个 Kubernetes 实例
			instances.PUT("/update/:id", k.UpdateK8sInstanceOne)  // 更新单个 Kubernetes 实例
			instances.DELETE("/delete", k.BatchDeleteK8sInstance) // 批量删除 Kubernetes 实例
			instances.POST("/restart", k.BatchRestartK8sInstance) // 批量重启 Kubernetes 实例
			instances.GET("/by-app", k.GetK8sInstanceByApp)       // 根据应用获取 Kubernetes 实例
			instances.GET("/instances", k.GetK8sInstanceList)     // 获取 Kubernetes 实例列表
			instances.GET("/:id", k.GetK8sInstanceOne)            // 获取单个 Kubernetes 实例
		}

		// 应用 Deployment 和 Service 的抽象
		apps := k8sAppApiGroup.Group("/apps")
		{
			apps.POST("/create", k.CreateK8sAppOne)        // 创建单个 Kubernetes 应用
			apps.PUT("/update/:id", k.UpdateK8sAppOne)     // 更新单个 Kubernetes 应用
			apps.DELETE("/:id", k.DeleteK8sAppOne)         // 删除单个 Kubernetes 应用
			apps.GET("/:id", k.GetK8sAppOne)               // 获取单个 Kubernetes 应用
			apps.GET("/by-app", k.GetK8sAppList)           // 获取 Kubernetes 应用列表 // TODO:需求未懂
			apps.GET("/:id/pods", k.GetK8sPodListByDeploy) // 根据部署获取 Kubernetes Pod 列表
			apps.GET("/select", k.GetK8sAppListForSelect)  // 获取用于选择的 Kubernetes 应用列表
		}

		// 项目
		projects := k8sAppApiGroup.Group("/projects")
		{
			projects.GET("/all", k.GetK8sProjectList)             // 获取 Kubernetes 项目列表
			projects.GET("/select", k.GetK8sProjectListForSelect) // 获取用于选择的 Kubernetes 项目列表
			projects.POST("/create", k.CreateK8sProject)          // 创建 Kubernetes 项目
			projects.PUT("/update/:id", k.UpdateK8sProject)       // 更新 Kubernetes 项目
			projects.DELETE("/:id", k.DeleteK8sProjectOne)        // 删除单个 Kubernetes 项目
		}

		// CronJob
		cronJobs := k8sAppApiGroup.Group("/cronJobs")
		{
			cronJobs.GET("/cronJoblist", k.GetK8sCronjobList)     // 获取 CronJob 列表
			cronJobs.POST("/create", k.CreateK8sCronjobOne)       // 创建单个 CronJob
			cronJobs.PUT("/:id", k.UpdateK8sCronjobOne)           // 更新单个 CronJob
			cronJobs.GET("/:id", k.GetK8sCronjobOne)              // 获取单个 CronJob
			cronJobs.GET("/:id/last-pod", k.GetK8sCronjobLastPod) // 获取 CronJob 最近的 Pod
			cronJobs.DELETE("/delete", k.BatchDeleteK8sCronjob)   // 批量删除 CronJob
		}
	}
}

// GetClusterNamespacesUnique 获取唯一的命名空间列表
func (k *K8sAppHandler) GetClusterNamespacesUnique(ctx *gin.Context) {
	// TODO: 实现获取唯一命名空间列表的逻辑

}

// CreateK8sInstanceOne 创建单个 Kubernetes 实例
func (k *K8sAppHandler) CreateK8sInstanceOne(ctx *gin.Context) {
	var req model.K8sInstance
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.instanceService.CreateInstanceOne(ctx, &req)
	})
}

// UpdateK8sInstanceOne 更新单个 Kubernetes 实例
func (k *K8sAppHandler) UpdateK8sInstanceOne(ctx *gin.Context) {
	// 拿到id参数
	instanceId := ctx.Param("id")
	instanceId_int, err2 := strconv.ParseInt(instanceId, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	// 拿到req
	var req model.K8sInstance
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	err := k.instanceService.UpdateInstanceOne(ctx, instanceId_int, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "instance updated successfully"})
}

// BatchDeleteK8sInstance 批量删除 Kubernetes 实例
func (k *K8sAppHandler) BatchDeleteK8sInstance(ctx *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	// 解析 JSON 体
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ids cannot be empty"})
		return
	}
	// 调用服务方法进行批量删除
	if err := k.instanceService.BatchDeleteInstance(ctx, req.IDs); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "instances deleted successfully"})

}

// BatchRestartK8sInstance 批量重启 Kubernetes 实例
func (k *K8sAppHandler) BatchRestartK8sInstance(ctx *gin.Context) {
	var req struct {
		InstanceIDs []int64 `json:"instance_ids" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if len(req.InstanceIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "instance_ids cannot be empty"})
		return
	}

	// 调用服务方法进行批量重启
	if err := k.instanceService.BatchRestartInstance(ctx, req.InstanceIDs); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "instances restarted successfully"})
}

// GetK8sInstanceByApp 根据应用获取 Kubernetes 实例
func (k *K8sAppHandler) GetK8sInstanceByApp(ctx *gin.Context) {
	appID := ctx.Query("app_id") // 获取 app_id 的值
	if appID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
		return
	}
	appID64, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
		return
	}

	// 2.调用服务方法获取实例列表
	instances, err := k.instanceService.GetInstanceByApp(ctx, appID64)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// 4.返回实例列表
	ctx.JSON(http.StatusOK, instances)
}

// GetK8sInstanceList 获取 Kubernetes 实例列表
func (k *K8sAppHandler) GetK8sInstanceList(ctx *gin.Context) {
	res, err := k.instanceService.GetInstanceAll(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

// GetK8sInstanceOne 获取单个 Kubernetes 实例
func (k *K8sAppHandler) GetK8sInstanceOne(ctx *gin.Context) {
	instanceId := ctx.Param("id")
	instanceId_int, err2 := strconv.ParseInt(instanceId, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	instance, err := k.instanceService.GetInstanceOne(ctx, instanceId_int)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, instance)
}

// GetK8sAppList 获取 Kubernetes 应用列表
func (k *K8sAppHandler) GetK8sAppList(ctx *gin.Context) {
	//Todo:
	//ID := ctx.Query("id") // 获取 app_id 的值
	//if ID == "" {
	//	ctx.JSON(http.StatusBadRequest, gin.H{"error": "app_id is required"})
	//	return
	//}
	//ID64, err := strconv.ParseInt(ID, 10, 64)
	//if err != nil {
	//	ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
	//	return
	//}
	//k.appService.GetK8sAppList(ctx, ID)
}

// CreateK8sAppOne 创建单个 Kubernetes 应用
func (k *K8sAppHandler) CreateK8sAppOne(ctx *gin.Context) {
	var req model.K8sApp
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.appService.CreateAppOne(ctx, &req)
	})
}

// UpdateK8sAppOne 更新单个 Kubernetes 应用
func (k *K8sAppHandler) UpdateK8sAppOne(ctx *gin.Context) {
	// 拿到id参数
	Id := ctx.Param("id")
	Id_int, err := strconv.ParseInt(Id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	// 拿到req
	var req model.K8sApp
	if err = ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	err = k.appService.UpdateAppOne(ctx, Id_int, req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "app updated successfully"})
}

// DeleteK8sAppOne 删除单个 Kubernetes 应用
func (k *K8sAppHandler) DeleteK8sAppOne(ctx *gin.Context) {
	Id := ctx.Param("id")
	Id_int, err := strconv.ParseInt(Id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid app_id"})
	}
	err = k.appService.DeleteAppOne(ctx, Id_int)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "app deleted successfully"})

}

// GetK8sAppOne 获取单个 Kubernetes 应用
func (k *K8sAppHandler) GetK8sAppOne(ctx *gin.Context) {
	Id := ctx.Param("id")
	Id_int, err2 := strconv.ParseInt(Id, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	app, err := k.appService.GetAppOne(ctx, Id_int)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, app)
}

// GetK8sPodListByDeploy 根据部署获取 Kubernetes Pod 列表
func (k *K8sAppHandler) GetK8sPodListByDeploy(ctx *gin.Context) {
	ID := ctx.Param("id")
	IDInt, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid app ID"})
		return
	}
	resources, err := k.appService.GetPodListByDeploy(ctx, IDInt)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, resources)
}

// GetK8sAppListForSelect 获取用于选择的 Kubernetes 应用列表
func (k *K8sAppHandler) GetK8sAppListForSelect(ctx *gin.Context) {

	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "instance_ids cannot be empty"})
		return
	}

	// 调用服务方法进行批量重启
	apps, err1 := k.appService.GetAppByIds(ctx, req.IDs)
	if err1 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err1.Error()})
		return
	}
	ctx.JSON(http.StatusOK, apps)
}

// GetK8sProjectList 获取 Kubernetes 项目列表
func (k *K8sAppHandler) GetK8sProjectList(ctx *gin.Context) {
	projectList, err1 := k.projectService.GetprojectList(ctx)
	if err1 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err1.Error()})
		return
	}
	ctx.JSON(http.StatusOK, projectList)
}

// GetK8sProjectListForSelect 获取用于选择的 Kubernetes 项目列表
func (k *K8sAppHandler) GetK8sProjectListForSelect(ctx *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	// 解析 JSON 体
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ids cannot be empty"})
		return
	}
	projectList, err1 := k.projectService.GetprojectListByIds(ctx, req.IDs)
	if err1 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err1.Error()})
		return
	}
	ctx.JSON(http.StatusOK, projectList)
}

// CreateK8sProject 创建 Kubernetes 项目
func (k *K8sAppHandler) CreateK8sProject(ctx *gin.Context) {
	var req model.K8sProject
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.projectService.CreateProjectOne(ctx, &req)
	})
}

// UpdateK8sProject 更新 Kubernetes 项目
func (k *K8sAppHandler) UpdateK8sProject(ctx *gin.Context) {
	Id := ctx.Param("id")
	Id_int, err := strconv.ParseInt(Id, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	// 拿到req
	var req model.K8sProject

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	// 调用服务层更新项目
	if err := k.projectService.UpdateProjectOne(ctx, Id_int, &req); err != nil {
		k.l.Error("项目更新失败",
			zap.Int64("projectId", Id_int),
			zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "项目更新失败"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "项目更新成功",
		"id":      req.ID,
	})
}

// DeleteK8sProjectOne 删除单个 Kubernetes 项目
func (k *K8sAppHandler) DeleteK8sProjectOne(ctx *gin.Context) {
	ID := ctx.Param("id")
	IDInt, err := strconv.ParseInt(ID, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid app ID"})
		return
	}
	err1 := k.projectService.DeleteProjectOne(ctx, IDInt)
	if err1 != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err1.Error()})
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "project deleted successfully"})
}

// GetK8sCronjobList 获取 CronJob 列表
func (k *K8sAppHandler) GetK8sCronjobList(ctx *gin.Context) {
	res, err := k.cronjobService.GetCronjobList(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, res)
}

// CreateK8sCronjobOne 创建单个 CronJob
func (k *K8sAppHandler) CreateK8sCronjobOne(ctx *gin.Context) {
	// 从请求中解析出 CronJob 信息到 req 结构体
	var req model.K8sCronjob
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 若解析失败，记录错误日志并返回 400 错误响应
		k.l.Error("解析请求 JSON 失败", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求体",
			"details": err.Error(),
		})
		return
	}

	// 调用服务层方法创建 CronJob
	err := k.cronjobService.CreateCronjobOne(ctx, req)
	if err != nil {
		// 若创建失败，记录错误日志并返回 500 错误响应
		k.l.Error("创建 CronJob 失败", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建 CronJob 时出错",
		})
		return
	}

	// 若创建成功，返回 201 状态码和成功消息
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "CronJob 创建成功",
	})
}

// UpdateK8sCronjobOne 更新单个 CronJob
func (k *K8sAppHandler) UpdateK8sCronjobOne(ctx *gin.Context) {
	Id := ctx.Param("id")
	Id_int, err2 := strconv.ParseInt(Id, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}

	var req model.K8sCronjob
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// 若解析失败，记录错误日志并返回 400 错误响应
		k.l.Error("解析请求 JSON 失败", zap.Error(err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "无效的请求体",
			"details": err.Error(),
		})
		return
	}
	err := k.cronjobService.UpdateCronjobOne(ctx, Id_int, req)
	if err != nil {
		// 若创建失败，记录错误日志并返回 500 错误响应
		k.l.Error("创建 CronJob 失败", zap.Error(err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建 CronJob 时出错",
		})
	}
	// 若创建成功，返回 201 状态码和成功消息
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "CronJob 创建成功",
	})
}

// GetK8sCronjobOne 获取单个 CronJob
func (k *K8sAppHandler) GetK8sCronjobOne(ctx *gin.Context) {

	Id := ctx.Param("id")
	Id_int, err2 := strconv.ParseInt(Id, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	cronjob, err := k.cronjobService.GetCronjobOne(ctx, Id_int)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	ctx.JSON(http.StatusOK, cronjob)
}

// GetK8sCronjobLastPod 获取 CronJob 最近的 Pod
func (k *K8sAppHandler) GetK8sCronjobLastPod(ctx *gin.Context) {
	cronjobID := ctx.Param("id") // 获取 URL 参数中的 CronJob ID
	Id_int, err2 := strconv.ParseInt(cronjobID, 10, 64)
	if err2 != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid instance_id"})
	}
	// 调用服务层方法获取最近的 Pod
	pod, err := k.cronjobService.GetCronjobLastPod(ctx, Id_int)
	// 处理错误
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 返回最近的 Pod
	ctx.JSON(http.StatusOK, pod)
}

// BatchDeleteK8sCronjob 批量删除 CronJob
func (k *K8sAppHandler) BatchDeleteK8sCronjob(ctx *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "instance_ids cannot be empty"})
		return
	}
	// 调用服务方法进行批量删除
	if err := k.cronjobService.BatchDeleteCronjob(ctx, req.IDs); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	//	返回成功响应
	ctx.JSON(http.StatusOK, gin.H{"message": "cronjobs deleted successfully"})

}
