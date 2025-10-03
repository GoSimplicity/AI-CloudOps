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

type K8sNodeHandler struct {
	nodeService  service.NodeService
	taintService service.TaintService
}

func NewK8sNodeHandler(nodeService service.NodeService, taintService service.TaintService) *K8sNodeHandler {
	return &K8sNodeHandler{
		nodeService:  nodeService,
		taintService: taintService,
	}
}

func (h *K8sNodeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		// Node管理
		k8sGroup.GET("/node/:cluster_id/list", h.GetNodeList)                           // 获取Node列表
		k8sGroup.GET("/node/:cluster_id/:node_name/detail", h.GetNodeDetail)            // 获取Node详情
		k8sGroup.POST("/node/:cluster_id/:node_name/labels/update", h.UpdateNodeLabels) // 更新Node标签（完全覆盖）
		k8sGroup.POST("/node/:cluster_id/:node_name/drain", h.DrainNode)                // 驱逐Node
		k8sGroup.POST("/node/:cluster_id/:node_name/cordon", h.CordonNode)              // 封锁Node（禁止调度）
		k8sGroup.POST("/node/:cluster_id/:node_name/uncordon", h.UncordonNode)          // 解封Node（允许调度）
		// 污点管理
		k8sGroup.GET("/node/:cluster_id/:node_name/taints/list", h.GetNodeTaints)         // 获取Node污点
		k8sGroup.POST("/node/:cluster_id/:node_name/taints/add", h.AddNodeTaints)         // 添加Node污点
		k8sGroup.DELETE("/node/:cluster_id/:node_name/taints/delete", h.DeleteNodeTaints) // 删除Node污点
		k8sGroup.POST("/node/:cluster_id/:node_name/taints/check", h.CheckTaintYaml)      // 检查污点YAML
	}
}

// GetNodeList 获取Node列表
func (h *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	var req model.GetNodeListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeList(ctx, &req)
	})
}

func (h *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.GetNodeDetailReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeDetail(ctx, &req)
	})
}

// UpdateNodeLabels 更新节点标签
func (h *K8sNodeHandler) UpdateNodeLabels(ctx *gin.Context) {
	var req model.UpdateNodeLabelsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.UpdateNodeLabels(ctx, &req)
	})
}

// DrainNode 驱逐节点上的所有Pod
func (h *K8sNodeHandler) DrainNode(ctx *gin.Context) {
	var req model.DrainNodeReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.DrainNode(ctx, &req)
	})
}

// CordonNode 禁止节点调度新的Pod
func (h *K8sNodeHandler) CordonNode(ctx *gin.Context) {
	var req model.NodeCordonReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.CordonNode(ctx, &req)
	})
}

// UncordonNode 解除节点调度限制
func (h *K8sNodeHandler) UncordonNode(ctx *gin.Context) {
	var req model.NodeUncordonReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.UncordonNode(ctx, &req)
	})
}

// GetNodeTaints 获取节点污点列表
func (h *K8sNodeHandler) GetNodeTaints(ctx *gin.Context) {
	var req model.GetNodeTaintsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeTaints(ctx, &req)
	})
}

// AddNodeTaints 添加节点污点
func (h *K8sNodeHandler) AddNodeTaints(ctx *gin.Context) {
	var req model.AddNodeTaintsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.AddNodeTaint(ctx, &req)
	})
}

// DeleteNodeTaints 删除节点污点
func (h *K8sNodeHandler) DeleteNodeTaints(ctx *gin.Context) {
	var req model.DeleteNodeTaintsReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.DeleteNodeTaint(ctx, &req)
	})
}

// CheckTaintYaml 检查污点YAML配置
func (h *K8sNodeHandler) CheckTaintYaml(ctx *gin.Context) {
	var req model.CheckTaintYamlReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := utils.GetParamCustomName(ctx, "node_name")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.CheckTaintYaml(ctx, &req)
	})
}
