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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// K8sTolerationHandler K8s容忍度管理API处理器
// 负责处理Kubernetes容忍度相关的HTTP请求，包括容忍度的增删改查、批量操作、模板管理等功能
type K8sTolerationHandler struct {
	tolerationService admin.TolerationService // 容忍度服务接口
	logger            *zap.Logger             // 日志记录器
}

// NewK8sTolerationHandler 创建新的K8s容忍度处理器实例
// 参数:
//   - logger: 日志记录器，用于记录API调用和错误信息
//   - tolerationService: 容忍度服务实例，提供业务逻辑处理
// 返回: K8sTolerationHandler指针
func NewK8sTolerationHandler(logger *zap.Logger, tolerationService admin.TolerationService) *K8sTolerationHandler {
	return &K8sTolerationHandler{
		logger:            logger,
		tolerationService: tolerationService,
	}
}

// RegisterRouters 注册容忍度管理相关的路由
// 为容忍度管理功能注册HTTP路由，包括基本的CRUD操作、批量操作、模板管理和时间配置等
func (k *K8sTolerationHandler) RegisterRouters(server *gin.Engine) {
	// 创建K8s API组
	k8sGroup := server.Group("/api/k8s")

	// 容忍度管理路由组
	tolerations := k8sGroup.Group("/tolerations")
	{
		tolerations.POST("/add", k.AddTolerations)                        // 为指定资源添加容忍度
		tolerations.POST("/update", k.UpdateTolerations)                  // 更新指定资源的容忍度
		tolerations.DELETE("/delete", k.DeleteTolerations)               // 从指定资源删除容忍度
		tolerations.POST("/validate", k.ValidateTolerations)             // 验证容忍度配置是否正确
		tolerations.GET("/list", k.ListTolerations)                      // 获取指定资源的容忍度列表
		tolerations.POST("/time/config", k.ConfigTolerationTime)         // 配置容忍度时间参数
		tolerations.POST("/time/validate", k.ValidateTolerationTime)     // 验证容忍度时间配置
		tolerations.POST("/batch", k.BatchUpdateTolerations)             // 批量更新多个资源的容忍度
		tolerations.POST("/template", k.CreateTolerationTemplate)        // 创建容忍度模板
		tolerations.GET("/template/:name", k.GetTolerationTemplate)      // 根据名称获取容忍度模板
		tolerations.DELETE("/template/:name", k.DeleteTolerationTemplate) // 根据名称删除容忍度模板
	}
}

// AddTolerations 为指定的K8s资源添加容忍度
// 支持为Pod、Deployment、StatefulSet、DaemonSet等资源添加新的容忍度配置
// 请求体: K8sTaintTolerationRequest - 包含集群ID、资源类型、资源名称、命名空间和容忍度列表
// 响应: K8sTaintTolerationResponse - 包含操作结果和兼容的节点信息
func (k *K8sTolerationHandler) AddTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	// 使用通用请求处理器处理请求，自动完成参数绑定、验证和错误处理
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.AddTolerations(ctx, &req)
	})
}

// UpdateTolerations 更新指定K8s资源的容忍度配置
// 完全替换现有的容忍度设置，不是增量更新
// 请求体: K8sTaintTolerationRequest - 包含新的容忍度配置
// 响应: K8sTaintTolerationResponse - 包含更新结果和兼容的节点信息
func (k *K8sTolerationHandler) UpdateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	// 调用容忍度服务更新指定资源的容忍度配置
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.UpdateTolerations(ctx, &req)
	})
}

// DeleteTolerations 从指定K8s资源中删除特定的容忍度
// 根据请求中指定的容忍度列表，从目标资源中移除匹配的容忍度配置
// 请求体: K8sTaintTolerationRequest - 包含要删除的容忍度列表
// 响应: 无内容，仅返回HTTP状态码
func (k *K8sTolerationHandler) DeleteTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	// 删除操作不返回具体内容，只返回操作状态
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.tolerationService.DeleteTolerations(ctx, &req)
	})
}

// ValidateTolerations 验证容忍度配置的有效性
// 检查容忍度配置是否符合K8s规范，并分析与集群节点的兼容性
// 请求体: K8sTaintTolerationValidationRequest - 包含要验证的容忍度配置和验证选项
// 响应: K8sTaintTolerationValidationResponse - 包含验证结果、兼容节点列表和调度建议
func (k *K8sTolerationHandler) ValidateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationValidationRequest

	// 验证容忍度配置并分析节点兼容性
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ValidateTolerations(ctx, &req)
	})
}

// ListTolerations 获取指定K8s资源的当前容忍度配置列表
// 查询并返回目标资源当前配置的所有容忍度信息
// 请求体: K8sTaintTolerationRequest - 包含集群ID、资源类型、资源名称和命名空间
// 响应: K8sTaintTolerationResponse - 包含当前的容忍度配置列表
func (k *K8sTolerationHandler) ListTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	// 获取指定资源的容忍度配置列表
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ListTolerations(ctx, &req)
	})
}

// ConfigTolerationTime 配置容忍度的时间参数
// 设置容忍度的超时时间、默认时间和条件化超时配置，支持全局和资源级别的配置
// 请求体: K8sTolerationTimeRequest - 包含时间配置参数和应用范围
// 响应: K8sTolerationTimeResponse - 包含配置结果和应用的超时设置
func (k *K8sTolerationHandler) ConfigTolerationTime(ctx *gin.Context) {
	var req model.K8sTolerationTimeRequest

	// 配置容忍度时间参数，支持全局和资源级别的配置
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ConfigTolerationTime(ctx, &req)
	})
}

// ValidateTolerationTime 验证容忍度时间配置的有效性
// 检查时间配置参数是否合理，包括最大值、最小值和条件化超时的有效性
// 请求体: K8sTolerationTimeRequest - 包含要验证的时间配置参数
// 响应: K8sTolerationTimeResponse - 包含验证结果和详细的验证信息
func (k *K8sTolerationHandler) ValidateTolerationTime(ctx *gin.Context) {
	var req model.K8sTolerationTimeRequest

	// 验证容忍度时间配置的合理性
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ValidateTolerationTime(ctx, &req)
	})
}

// BatchUpdateTolerations 批量更新多个资源的容忍度配置
// 同时对指定命名空间下的所有同类型资源进行容忍度更新，支持并发处理提高效率
// 请求体: K8sTaintTolerationRequest - 包含统一的容忍度配置和目标资源类型
// 响应: K8sTaintTolerationResponse - 包含批量操作结果和成功更新的资源数量
func (k *K8sTolerationHandler) BatchUpdateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	// 批量更新操作，使用并发处理提高效率
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.BatchUpdateTolerations(ctx, &req)
	})
}

// CreateTolerationTemplate 创建容忍度模板
// 保存常用的容忍度配置为模板，方便后续快速应用到多个资源
// 请求体: K8sTolerationConfigRequest - 包含模板名称、容忍度配置和应用选项
// 响应: K8sTolerationTemplate - 返回创建的模板信息
func (k *K8sTolerationHandler) CreateTolerationTemplate(ctx *gin.Context) {
	var req model.K8sTolerationConfigRequest

	// 创建容忍度模板，支持立即应用到现有资源
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.CreateTolerationTemplate(ctx, &req)
	})
}

// GetTolerationTemplate 根据名称获取容忍度模板
// 查询并返回指定名称的容忍度模板配置信息
// URL参数: name - 模板名称 (path), cluster_id - 集群ID (query)
// 响应: K8sTolerationTemplate - 返回模板的详细配置信息
func (k *K8sTolerationHandler) GetTolerationTemplate(ctx *gin.Context) {
	// 从 URL路径中获取模板名称
	templateName := ctx.Param("name")
	// 从查询参数中获取集群ID
	clusterId := ctx.Query("cluster_id")

	// 验证集群ID参数的有效性
	clusterID, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestWithDetails(ctx, "Invalid cluster_id", "cluster_id must be a valid integer")
		return
	}

	// 构建请求对象
	req := model.K8sTolerationConfigRequest{
		ClusterID: clusterID,
		TolerationTemplate: model.K8sTolerationTemplate{
			Name: templateName,
		},
	}

	// 调用服务获取模板信息
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.GetTolerationTemplate(ctx, &req)
	})
}

// DeleteTolerationTemplate 根据名称删除容忍度模板
// 永久删除指定名称的容忍度模板，不影响已经应用了该模板的资源
// URL参数: name - 模板名称 (path), cluster_id - 集群ID (query)
// 响应: 无内容，仅返回HTTP状态码
func (k *K8sTolerationHandler) DeleteTolerationTemplate(ctx *gin.Context) {
	// 从 URL路径中获取模板名称
	templateName := ctx.Param("name")
	// 从查询参数中获取集群ID
	clusterId := ctx.Query("cluster_id")

	// 验证集群ID参数的有效性
	clusterID, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestWithDetails(ctx, "Invalid cluster_id", "cluster_id must be a valid integer")
		return
	}

	// 构建请求对象
	req := model.K8sTolerationConfigRequest{
		ClusterID: clusterID,
		TolerationTemplate: model.K8sTolerationTemplate{
			Name: templateName,
		},
	}

	// 调用服务删除模板
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.tolerationService.DeleteTolerationTemplate(ctx, &req)
	})
}
