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

type K8sStorageClassHandler struct {
	l                    *zap.Logger
	storageClassService  admin.StorageClassService
}

func NewK8sStorageClassHandler(l *zap.Logger, storageClassService admin.StorageClassService) *K8sStorageClassHandler {
	return &K8sStorageClassHandler{
		l:                    l,
		storageClassService:  storageClassService,
	}
}

func (k *K8sStorageClassHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	storageClasses := k8sGroup.Group("/storageclasses")
	{
		storageClasses.GET("/:id", k.GetStorageClasses)                    // 获取 StorageClass 列表
		storageClasses.POST("/create", k.CreateStorageClass)               // 创建 StorageClass
		storageClasses.DELETE("/delete/:id", k.DeleteStorageClass)         // 删除指定 StorageClass
		storageClasses.DELETE("/batch_delete", k.BatchDeleteStorageClass)  // 批量删除 StorageClass
		storageClasses.GET("/:id/yaml", k.GetStorageClassYaml)            // 获取 StorageClass YAML 配置
		storageClasses.GET("/:id/status", k.GetStorageClassStatus)        // 获取 StorageClass 状态
		storageClasses.GET("/:id/config", k.GetStorageClassConfig)        // 获取 StorageClass 配置参数
		storageClasses.GET("/:id/default", k.GetDefaultStorageClass)      // 获取默认存储类
	}
}

// GetStorageClasses 获取 StorageClass 列表
func (k *K8sStorageClassHandler) GetStorageClasses(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.storageClassService.GetStorageClasses(ctx, id)
	})
}

// CreateStorageClass 创建 StorageClass
func (k *K8sStorageClassHandler) CreateStorageClass(ctx *gin.Context) {
	var req model.K8sStorageClassRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.storageClassService.CreateStorageClass(ctx, &req)
	})
}

// BatchDeleteStorageClass 批量删除 StorageClass
func (k *K8sStorageClassHandler) BatchDeleteStorageClass(ctx *gin.Context) {
	var req model.K8sStorageClassRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.storageClassService.BatchDeleteStorageClass(ctx, req.ClusterID, req.StorageClassNames)
	})
}

// GetStorageClassYaml 获取 StorageClass 的 YAML 配置
func (k *K8sStorageClassHandler) GetStorageClassYaml(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	storageClassName := ctx.Query("storage_class_name")
	if storageClassName == "" {
		k.l.Error("缺少必需的 storage_class_name 参数")
		utils.BadRequestError(ctx, "缺少 'storage_class_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.storageClassService.GetStorageClassYaml(ctx, id, storageClassName)
	})
}

// DeleteStorageClass 删除指定的 StorageClass
func (k *K8sStorageClassHandler) DeleteStorageClass(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	storageClassName := ctx.Query("storage_class_name")
	if storageClassName == "" {
		k.l.Error("缺少必需的 storage_class_name 参数")
		utils.BadRequestError(ctx, "缺少 'storage_class_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, k.storageClassService.DeleteStorageClass(ctx, id, storageClassName)
	})
}

// GetStorageClassStatus 获取 StorageClass 状态
func (k *K8sStorageClassHandler) GetStorageClassStatus(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	storageClassName := ctx.Query("storage_class_name")
	if storageClassName == "" {
		k.l.Error("缺少必需的 storage_class_name 参数")
		utils.BadRequestError(ctx, "缺少 'storage_class_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.storageClassService.GetStorageClassStatus(ctx, id, storageClassName)
	})
}

// GetStorageClassConfig 获取 StorageClass 配置参数
func (k *K8sStorageClassHandler) GetStorageClassConfig(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	storageClassName := ctx.Query("storage_class_name")
	if storageClassName == "" {
		k.l.Error("缺少必需的 storage_class_name 参数")
		utils.BadRequestError(ctx, "缺少 'storage_class_name' 参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.storageClassService.GetStorageClassConfig(ctx, id, storageClassName)
	})
}

// GetDefaultStorageClass 获取默认存储类
func (k *K8sStorageClassHandler) GetDefaultStorageClass(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		k.l.Error("获取参数 ID 失败", zap.Error(err))
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.storageClassService.GetDefaultStorageClass(ctx, id)
	})
}