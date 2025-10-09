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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type TreeCloudHandler struct {
	service   service.TreeCloudService
	sshClient ssh.Client
}

func NewTreeCloudHandler(service service.TreeCloudService, sshClient ssh.Client) *TreeCloudHandler {
	return &TreeCloudHandler{
		service:   service,
		sshClient: sshClient,
	}
}

func (h *TreeCloudHandler) RegisterRouters(server *gin.Engine) {
	cloudGroup := server.Group("/api/tree/cloud")
	{
		cloudGroup.GET("/list", h.GetTreeCloudResourceList)
		cloudGroup.GET("/:id/detail", h.GetTreeCloudResourceDetail)
		cloudGroup.POST("/create", h.CreateTreeCloudResource)
		cloudGroup.PUT("/:id/update", h.UpdateTreeCloudResource)
		cloudGroup.DELETE("/:id/delete", h.DeleteTreeCloudResource)
		cloudGroup.POST("/:id/bind", h.BindTreeCloudResource)
		cloudGroup.POST("/:id/unbind", h.UnBindTreeCloudResource)
		cloudGroup.POST("/sync", h.SyncTreeCloudResource)
		cloudGroup.POST("/batch_import", h.BatchImportCloudResource)
		cloudGroup.GET("/:id/node", h.GetTreeNodeCloudResources)
		cloudGroup.GET("/:id/terminal", h.ConnectCloudResourceTerminal)
		cloudGroup.PUT("/:id/status", h.UpdateCloudResourceStatus)
	}
}

// GetTreeCloudResourceList 获取云资源列表
func (h *TreeCloudHandler) GetTreeCloudResourceList(ctx *gin.Context) {
	var req model.GetTreeCloudResourceListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeCloudResourceList(ctx, &req)
	})
}

// GetTreeCloudResourceDetail 获取云资源详情
func (h *TreeCloudHandler) GetTreeCloudResourceDetail(ctx *gin.Context) {
	var req model.GetTreeCloudResourceDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeCloudResourceDetail(ctx, &req)
	})
}

// CreateTreeCloudResource 创建云资源
func (h *TreeCloudHandler) CreateTreeCloudResource(ctx *gin.Context) {
	var req model.CreateTreeCloudResourceReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTreeCloudResource(ctx, &req)
	})
}

// UpdateTreeCloudResource 更新云资源
func (h *TreeCloudHandler) UpdateTreeCloudResource(ctx *gin.Context) {
	var req model.UpdateTreeCloudResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTreeCloudResource(ctx, &req)
	})
}

// DeleteTreeCloudResource 删除云资源
func (h *TreeCloudHandler) DeleteTreeCloudResource(ctx *gin.Context) {
	var req model.DeleteTreeCloudResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteTreeCloudResource(ctx, &req)
	})
}

// BindTreeCloudResource 绑定云资源到树节点
func (h *TreeCloudHandler) BindTreeCloudResource(ctx *gin.Context) {
	var req model.BindTreeCloudResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindTreeCloudResource(ctx, &req)
	})
}

// UnBindTreeCloudResource 解绑云资源与树节点
func (h *TreeCloudHandler) UnBindTreeCloudResource(ctx *gin.Context) {
	var req model.UnBindTreeCloudResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnBindTreeCloudResource(ctx, &req)
	})
}

// SyncTreeCloudResource 从云厂商同步资源
func (h *TreeCloudHandler) SyncTreeCloudResource(ctx *gin.Context) {
	var req model.SyncTreeCloudResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.SyncTreeCloudResource(ctx, &req)
	})
}

// BatchImportCloudResource 批量导入云资源
func (h *TreeCloudHandler) BatchImportCloudResource(ctx *gin.Context) {
	var req model.BatchImportCloudResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.BatchImportCloudResource(ctx, &req)
	})
}

// GetTreeNodeCloudResources 获取树节点下的云资源
func (h *TreeCloudHandler) GetTreeNodeCloudResources(ctx *gin.Context) {
	var req model.GetTreeNodeCloudResourcesReq

	nodeId, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.NodeID = nodeId

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeNodeCloudResources(ctx, &req)
	})
}

// ConnectCloudResourceTerminal 连接云资源终端
func (h *TreeCloudHandler) ConnectCloudResourceTerminal(ctx *gin.Context) {
	var req model.ConnectTreeCloudResourceTerminalReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.ID = id
	req.UserID = uc.Uid

	// 获取云资源详情
	detailReq := &model.GetTreeCloudResourceDetailReq{ID: req.ID}
	cloud, err := h.service.GetTreeCloudResourceForConnection(ctx, detailReq)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取云资源信息失败: "+err.Error())
		return
	}

	// 仅支持ECS类型的云资源连接终端
	if cloud.ResourceType != model.ResourceTypeECS {
		utils.ErrorWithMessage(ctx, "仅支持ECS类型的云资源连接终端")
		return
	}

	// 如果没有公网IP，尝试使用私网IP
	ipAddr := cloud.PublicIP
	if ipAddr == "" {
		ipAddr = cloud.PrivateIP
	}

	if ipAddr == "" {
		utils.ErrorWithMessage(ctx, "云资源没有可用的IP地址")
		return
	}

	// 设置默认端口
	port := cloud.Port
	if port == 0 {
		port = 22
	}

	// 设置默认用户名
	username := cloud.Username
	if username == "" {
		username = "root"
	}

	// 配置SSH连接
	sshConfig := &ssh.Config{
		Host:     ipAddr,
		Port:     port,
		Username: username,
		Password: cloud.Password,
		Key:      cloud.Key,
		Mode:     ssh.AuthMode(cloud.AuthMode),
		Timeout:  10,
	}

	// 建立SSH连接
	if err := h.sshClient.Connect(sshConfig); err != nil {
		utils.ErrorWithMessage(ctx, "连接SSH失败: "+err.Error())
		return
	}

	// 确保SSH连接在函数退出时关闭
	defer func() {
		if closeErr := h.sshClient.Close(); closeErr != nil {
			utils.ErrorWithMessage(ctx, "关闭SSH连接失败: "+closeErr.Error())
		}
	}()

	// 升级WebSocket连接
	ws, err := utils.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		utils.ErrorWithMessage(ctx, "升级WebSocket连接失败: "+err.Error())
		return
	}
	defer ws.Close()

	// 启动终端会话
	if err := h.sshClient.WebTerminal(uc.Uid, ws); err != nil {
		utils.ErrorWithMessage(ctx, "启动Web终端失败: "+err.Error())
		return
	}
}

// UpdateCloudResourceStatus 更新云资源状态
func (h *TreeCloudHandler) UpdateCloudResourceStatus(ctx *gin.Context) {
	var req model.UpdateCloudResourceStatusReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateCloudResourceStatus(ctx, &req)
	})
}
