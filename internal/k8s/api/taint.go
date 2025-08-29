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
func (k *K8sTaintHandler) AddTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// ScheduleEnableSwitchNodes 启用或切换节点调度
func (k *K8sTaintHandler) ScheduleEnableSwitchNodes(ctx *gin.Context) {
	var req model.ScheduleK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.BatchEnableSwitchNodes(ctx, &req)
	})
}

// TaintYamlCheck 检查节点 Taint 的 YAML 配置
func (k *K8sTaintHandler) TaintYamlCheck(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.CheckTaintYaml(ctx, &req)
	})
}

// DeleteTaintsNodes 删除节点 Taint
func (k *K8sTaintHandler) DeleteTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// DrainPods 清空节点上的 Pods
func (k *K8sTaintHandler) DrainPods(ctx *gin.Context) {
	var req model.K8sClusterNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.DrainPods(ctx, &req)
	})
}
