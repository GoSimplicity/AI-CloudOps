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
	"go.uber.org/zap"
)

type K8sTaintHandler struct {
	taintService service.TaintService
	logger       *zap.Logger
}

func NewK8sTaintHandler(logger *zap.Logger, taintService service.TaintService) *K8sTaintHandler {
	return &K8sTaintHandler{
		logger:       logger,
		taintService: taintService,
	}
}

func (k *K8sTaintHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	nodes := k8sGroup.Group("/taints")
	{
		nodes.POST("/add", k.AddTaintsNodes)                      // 为节点添加 Taint
		nodes.POST("/enable_switch", k.ScheduleEnableSwitchNodes) // 启用或切换节点调度
		nodes.POST("/taint_check", k.TaintYamlCheck)              // 检查节点 Taint 的 YAML 配置
		nodes.DELETE("/delete", k.DeleteTaintsNodes)              // 删除节点 Taint
		nodes.POST("/drain", k.DrainPods)                         // 清空节点上的 Pods
	}
}

// AddTaintsNodes 为节点添加 Taint
// @Summary 为节点添加污点
// @Description 为指定的Kubernetes节点添加或更新污点配置
// @Tags 污点管理
// @Accept json
// @Produce json
// @Param request body model.TaintK8sNodesReq true "添加污点请求参数"
// @Success 200 {object} utils.ApiResponse "添加成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/taints/add [post]
// @Security BearerAuth
func (k *K8sTaintHandler) AddTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// ScheduleEnableSwitchNodes 启用或切换节点调度
// @Summary 启用或禁用节点调度
// @Description 批量启用或禁用Kubernetes节点的调度功能
// @Tags 污点管理
// @Accept json
// @Produce json
// @Param request body model.ScheduleK8sNodesReq true "节点调度切换请求参数"
// @Success 200 {object} utils.ApiResponse "操作成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/taints/enable_switch [post]
// @Security BearerAuth
func (k *K8sTaintHandler) ScheduleEnableSwitchNodes(ctx *gin.Context) {
	var req model.ScheduleK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.BatchEnableSwitchNodes(ctx, &req)
	})
}

// TaintYamlCheck 检查节点 Taint 的 YAML 配置
// @Summary 检查污点YAML配置
// @Description 验证节点污点的YAML配置是否正确
// @Tags 污点管理
// @Accept json
// @Produce json
// @Param request body model.TaintK8sNodesReq true "污点YAML检查请求参数"
// @Success 200 {object} utils.ApiResponse "检查成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/taints/taint_check [post]
// @Security BearerAuth
func (k *K8sTaintHandler) TaintYamlCheck(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.CheckTaintYaml(ctx, &req)
	})
}

// DeleteTaintsNodes 删除节点 Taint
// @Summary 删除节点污点
// @Description 删除指定Kubernetes节点的污点配置
// @Tags 污点管理
// @Accept json
// @Produce json
// @Param request body model.TaintK8sNodesReq true "删除污点请求参数"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/taints/delete [delete]
// @Security BearerAuth
func (k *K8sTaintHandler) DeleteTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// DrainPods 清空节点上的 Pods
// @Summary 清空节点Pod
// @Description 驱逐指定Kubernetes节点上的所有Pod，为节点维护做准备
// @Tags 污点管理
// @Accept json
// @Produce json
// @Param request body model.K8sClusterNodesReq true "清空Pod请求参数"
// @Success 200 {object} utils.ApiResponse "清空成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/taints/drain [post]
// @Security BearerAuth
func (k *K8sTaintHandler) DrainPods(ctx *gin.Context) {
	var req model.K8sClusterNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.DrainPods(ctx, &req)
	})
}
