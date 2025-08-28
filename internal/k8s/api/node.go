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

type K8sNodeHandler struct {
	logger      *zap.Logger
	nodeService service.NodeService
}

func NewK8sNodeHandler(logger *zap.Logger, nodeService service.NodeService) *K8sNodeHandler {
	return &K8sNodeHandler{
		logger:      logger,
		nodeService: nodeService,
	}
}

func (k *K8sNodeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	nodes := k8sGroup.Group("/nodes")
	{
		nodes.GET("/list/:id", k.GetNodeList)                              // 获取节点列表
		nodes.GET("/:node_name", k.GetNodeDetail)                          // 获取指定节点详情
		nodes.POST("/labels/add", k.AddLabelNodes)                         // 添加节点标签
		nodes.DELETE("/labels/delete", k.DeleteLabelNodes)                 // 删除节点标签
		nodes.GET("/:cluster_id/:node_name/resources", k.GetNodeResources) // 获取集群节点资源
		nodes.GET("/:cluster_id/:node_name/events", k.GetNodeEvents)       // 获取集群节点事件
	}
}

// GetNodeList 获取节点列表
// @Summary 获取集群节点列表
// @Description 根据集群ID获取指定K8s集群中的所有节点列表
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} utils.ApiResponse{data=[]model.K8sNode} "成功获取节点列表"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/nodes/list/{id} [get]
// @Security BearerAuth
func (k *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	clusterID, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeService.ListNodeByClusterName(ctx, clusterID)
	})
}

// GetNodeDetail 获取指定名称的节点详情
// @Summary 获取节点详细信息
// @Description 根据节点名称获取指定节点的详细信息，包括状态、资源使用情况等
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param name path string true "节点名称"
// @Param id query int true "集群ID"
// @Success 200 {object} utils.ApiResponse "成功获取节点详情"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/nodes/{name} [get]
// @Security BearerAuth
func (k *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
	name, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	id, err := utils.GetQueryParam[int](ctx, "id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return k.nodeService.GetNodeDetail(ctx, id, name)
	})
}

// AddLabelNodes 为节点添加标签
// @Summary 为节点添加标签
// @Description 为指定的K8s节点添加标签，支持批量操作
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param request body model.LabelK8sNodesReq true "添加标签请求参数"
// @Success 200 {object} utils.ApiResponse "成功添加标签"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/nodes/labels/add [post]
// @Security BearerAuth
func (k *K8sNodeHandler) AddLabelNodes(ctx *gin.Context) {
	var req model.LabelK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.nodeService.AddOrUpdateNodeLabel(ctx, &req)
	})
}

// DeleteLabelNodes 删除节点标签
// @Summary 删除节点标签
// @Description 删除指定K8s节点的标签，支持批量操作
// @Tags 节点管理
// @Accept json
// @Produce json
// @Param request body model.LabelK8sNodesReq true "删除标签请求参数"
// @Success 200 {object} utils.ApiResponse "成功删除标签"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/k8s/nodes/labels/delete [delete]
// @Security BearerAuth
func (k *K8sNodeHandler) DeleteLabelNodes(ctx *gin.Context) {
	var req model.LabelK8sNodesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.nodeService.AddOrUpdateNodeLabel(ctx, &req)
	})
}

func (k *K8sNodeHandler) GetNodeResources(ctx *gin.Context) {
	var req model.NodeResourcesReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeService.GetNodeResources(ctx, req.ClusterID)
	})
}

func (k *K8sNodeHandler) GetNodeEvents(ctx *gin.Context) {
	var req model.NodeEventsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	name, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = name

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeService.GetNodeEvents(ctx, req.ClusterID, req.NodeName)
	})
}
