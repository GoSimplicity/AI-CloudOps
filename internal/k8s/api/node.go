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

func (k *K8sNodeHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/nodes/:cluster_id/list", k.GetNodeList)
		k8sGroup.GET("/nodes/:cluster_id/:node_name/detail", k.GetNodeDetail)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/labels/add", k.AddLabelNodes)
		k8sGroup.DELETE("/nodes/:cluster_id/:node_name/labels/delete", k.DeleteLabelNodes)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/drain", k.DrainNode)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/cordon", k.CordonNode)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/uncordon", k.UncordonNode)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/schedule/switch", k.SwitchNodeSchedule)
		k8sGroup.GET("/nodes/:cluster_id/:node_name/taints", k.GetNodeTaints)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/taints/add", k.AddNodeTaints)
		k8sGroup.DELETE("/nodes/:cluster_id/:node_name/taints/delete", k.DeleteNodeTaints)
		k8sGroup.POST("/nodes/:cluster_id/:node_name/taints/check", k.CheckTaintYaml)
	}
}

func (k *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	var req model.GetNodeListReq

	clusterID, err := utils.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.nodeService.GetNodeList(ctx, &req)
	})
}

func (k *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
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
		return k.nodeService.GetNodeDetail(ctx, &req)
	})
}

func (k *K8sNodeHandler) AddLabelNodes(ctx *gin.Context) {
	var req model.AddLabelNodesReq

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
		return nil, k.nodeService.AddOrUpdateNodeLabel(ctx, &req)
	})
}

func (k *K8sNodeHandler) DeleteLabelNodes(ctx *gin.Context) {
	var req model.DeleteLabelNodesReq

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
		return nil, k.nodeService.DeleteNodeLabel(ctx, &req)
	})
}

// DrainNode 驱逐节点上的所有Pod
func (k *K8sNodeHandler) DrainNode(ctx *gin.Context) {
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
		return nil, k.nodeService.DrainNode(ctx, &req)
	})
}

// CordonNode 禁止节点调度新的Pod
func (k *K8sNodeHandler) CordonNode(ctx *gin.Context) {
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
		return nil, k.nodeService.CordonNode(ctx, &req)
	})
}

// UncordonNode 解除节点调度限制
func (k *K8sNodeHandler) UncordonNode(ctx *gin.Context) {
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
		return nil, k.nodeService.UncordonNode(ctx, &req)
	})
}

// GetNodeTaints 获取节点污点列表
func (k *K8sNodeHandler) GetNodeTaints(ctx *gin.Context) {
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
		return k.nodeService.GetNodeTaints(ctx, &req)
	})
}

// AddNodeTaints 添加节点污点
func (k *K8sNodeHandler) AddNodeTaints(ctx *gin.Context) {
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
		return nil, k.taintService.AddNodeTaint(ctx, &req)
	})
}

// DeleteNodeTaints 删除节点污点
func (k *K8sNodeHandler) DeleteNodeTaints(ctx *gin.Context) {
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
		return nil, k.taintService.DeleteNodeTaint(ctx, &req)
	})
}

// CheckTaintYaml 检查污点YAML配置
func (k *K8sNodeHandler) CheckTaintYaml(ctx *gin.Context) {
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
		return nil, k.taintService.CheckTaintYaml(ctx, &req)
	})
}

// SwitchNodeSchedule 切换节点调度状态
func (k *K8sNodeHandler) SwitchNodeSchedule(ctx *gin.Context) {
	var req model.SwitchNodeScheduleReq

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
		return nil, k.taintService.SwitchNodeSchedule(ctx, &req)
	})
}
