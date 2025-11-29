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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
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
		k8sGroup.GET("/node/:cluster_id/list", h.GetNodeList)
		k8sGroup.GET("/node/:cluster_id/:node_name/detail", h.GetNodeDetail)
		k8sGroup.POST("/node/:cluster_id/:node_name/labels/update", h.UpdateNodeLabels)
		k8sGroup.POST("/node/:cluster_id/:node_name/drain", h.DrainNode)
		k8sGroup.POST("/node/:cluster_id/:node_name/cordon", h.CordonNode)
		k8sGroup.POST("/node/:cluster_id/:node_name/uncordon", h.UncordonNode)
		k8sGroup.GET("/node/:cluster_id/:node_name/taints/list", h.GetNodeTaints)
		k8sGroup.POST("/node/:cluster_id/:node_name/taints/add", h.AddNodeTaints)
		k8sGroup.DELETE("/node/:cluster_id/:node_name/taints/delete", h.DeleteNodeTaints)
		k8sGroup.POST("/node/:cluster_id/:node_name/taints/check", h.CheckTaintYaml)
	}
}

func (h *K8sNodeHandler) GetNodeList(ctx *gin.Context) {
	var req model.GetNodeListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeList(ctx, &req)
	})
}

func (h *K8sNodeHandler) GetNodeDetail(ctx *gin.Context) {
	var req model.GetNodeDetailReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeDetail(ctx, &req)
	})
}

func (h *K8sNodeHandler) UpdateNodeLabels(ctx *gin.Context) {
	var req model.UpdateNodeLabelsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.UpdateNodeLabels(ctx, &req)
	})
}

func (h *K8sNodeHandler) DrainNode(ctx *gin.Context) {
	var req model.DrainNodeReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.DrainNode(ctx, &req)
	})
}

func (h *K8sNodeHandler) CordonNode(ctx *gin.Context) {
	var req model.NodeCordonReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.CordonNode(ctx, &req)
	})
}

func (h *K8sNodeHandler) UncordonNode(ctx *gin.Context) {
	var req model.NodeUncordonReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.nodeService.UncordonNode(ctx, &req)
	})
}

func (h *K8sNodeHandler) GetNodeTaints(ctx *gin.Context) {
	var req model.GetNodeTaintsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.nodeService.GetNodeTaints(ctx, &req)
	})
}

func (h *K8sNodeHandler) AddNodeTaints(ctx *gin.Context) {
	var req model.AddNodeTaintsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.AddNodeTaint(ctx, &req)
	})
}

func (h *K8sNodeHandler) DeleteNodeTaints(ctx *gin.Context) {
	var req model.DeleteNodeTaintsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.DeleteNodeTaint(ctx, &req)
	})
}

func (h *K8sNodeHandler) CheckTaintYaml(ctx *gin.Context) {
	var req model.CheckTaintYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	nodeName, err := base.GetParamCustomName(ctx, "node_name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.NodeName = nodeName

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.taintService.CheckTaintYaml(ctx, &req)
	})
}
