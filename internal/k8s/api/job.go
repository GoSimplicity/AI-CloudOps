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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sJobHandler struct {
	l          *zap.Logger
	jobService admin.JobService
}

func NewK8sJobHandler(l *zap.Logger, jobService admin.JobService) *K8sJobHandler {
	return &K8sJobHandler{
		l:          l,
		jobService: jobService,
	}
}

func (k *K8sJobHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	jobs := k8sGroup.Group("/jobs")
	{
		jobs.GET("/:id", k.GetJobsByNamespace)          // 根据命名空间获取 Job 列表
		jobs.GET("/:id/yaml", k.GetJobYaml)            // 获取指定 Job 的 YAML 配置
		jobs.POST("/create", k.CreateJob)              // 创建 Job
		jobs.DELETE("/batch_delete", k.BatchDeleteJob) // 批量删除 Job
		jobs.DELETE("/delete/:id", k.DeleteJob)        // 删除指定 Job
		jobs.GET("/:id/status", k.GetJobStatus)        // 获取 Job 状态
		jobs.GET("/:id/history", k.GetJobHistory)      // 获取 Job 执行历史
		jobs.GET("/:id/pods", k.GetJobPods)           // 获取 Job 关联的 Pod 列表
	}
}

// GetJobsByNamespace 根据命名空间获取 Job 列表
func (k *K8sJobHandler) GetJobsByNamespace(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.jobService.GetJobsByNamespace(ctx, id, namespace)
	})
}

// CreateJob 创建 Job
func (k *K8sJobHandler) CreateJob(ctx *gin.Context) {
	var req model.K8sJobRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.jobService.CreateJob(ctx, &req)
	})
}

// BatchDeleteJob 批量删除 Job
func (k *K8sJobHandler) BatchDeleteJob(ctx *gin.Context) {
	var req model.K8sJobRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.jobService.BatchDeleteJob(ctx, req.ClusterID, req.Namespace, req.JobNames)
	})
}

// GetJobYaml 获取 Job 的 YAML 配置
func (k *K8sJobHandler) GetJobYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	jobName := ctx.Query("job_name")
	if jobName == "" {
		utils.BadRequestError(ctx, "缺少 'job_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.jobService.GetJobYaml(ctx, id, namespace, jobName)
	})
}

// DeleteJob 删除指定的 Job
func (k *K8sJobHandler) DeleteJob(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	jobName := ctx.Query("job_name")
	if jobName == "" {
		utils.BadRequestError(ctx, "缺少 'job_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.jobService.DeleteJob(ctx, id, namespace, jobName)
	})
}

// GetJobStatus 获取 Job 状态
func (k *K8sJobHandler) GetJobStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	jobName := ctx.Query("job_name")
	if jobName == "" {
		utils.BadRequestError(ctx, "缺少 'job_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.jobService.GetJobStatus(ctx, id, namespace, jobName)
	})
}

// GetJobHistory 获取 Job 执行历史
func (k *K8sJobHandler) GetJobHistory(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.jobService.GetJobHistory(ctx, id, namespace)
	})
}

// GetJobPods 获取 Job 关联的 Pod 列表
func (k *K8sJobHandler) GetJobPods(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	jobName := ctx.Query("job_name")
	if jobName == "" {
		utils.BadRequestError(ctx, "缺少 'job_name' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		utils.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.jobService.GetJobPods(ctx, id, namespace, jobName)
	})
}