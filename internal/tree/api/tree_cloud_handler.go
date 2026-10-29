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
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type TreeCloudHandler struct {
	service    service.TreeCloudService
	newSSH     TerminalClientFactory
	wsUpgrader *websocket.Upgrader
	logger     *zap.Logger
}

func cloudOperator(ctx *gin.Context) (jwt.UserClaims, bool) {
	value, ok := ctx.Get("user")
	user, valid := value.(jwt.UserClaims)
	if !ok || !valid || user.Uid <= 0 {
		base.ForbiddenError(ctx, "缺少可信操作人")
		return jwt.UserClaims{}, false
	}
	return user, true
}

func NewTreeCloudHandler(service service.TreeCloudService, factory TerminalClientFactory, wsUpgrader *websocket.Upgrader, logger *zap.Logger) *TreeCloudHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TreeCloudHandler{
		service:    service,
		newSSH:     factory,
		wsUpgrader: wsUpgrader,
		logger:     logger,
	}
}

func (h *TreeCloudHandler) RegisterRouters(server *gin.Engine) {
	cloudGroup := server.Group("/api/tree/cloud")
	{
		cloudGroup.GET("/list", h.GetTreeCloudResourceList)
		cloudGroup.GET("/:id/detail", h.GetTreeCloudResourceDetail)
		cloudGroup.GET("/:id/node", h.GetTreeNodeCloudResources)
		cloudGroup.GET("/:id/terminal", h.ConnectCloudResourceTerminal)
		cloudGroup.POST("/sync", h.SyncTreeCloudResource)
		cloudGroup.GET("/sync/history", h.GetSyncHistory)
		cloudGroup.PUT("/:id/update", h.UpdateTreeCloudResource)
		cloudGroup.DELETE("/:id/delete", h.DeleteTreeCloudResource)
		cloudGroup.PUT("/:id/status", h.UpdateCloudResourceStatus)
		cloudGroup.POST("/:id/bind", h.BindTreeCloudResource)
		cloudGroup.POST("/:id/unbind", h.UnBindTreeCloudResource)
		cloudGroup.GET("/changelog", h.GetChangeLog)
		cloudGroup.POST("/batch/delete", h.BatchDeleteTreeCloudResource)
		cloudGroup.PUT("/batch/status", h.BatchUpdateCloudResourceStatus)
	}
}

func (h *TreeCloudHandler) GetTreeCloudResourceList(ctx *gin.Context) {
	var req model.GetTreeCloudResourceListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeCloudResourceList(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) GetTreeCloudResourceDetail(ctx *gin.Context) {
	var req model.GetTreeCloudResourceDetailReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
		return h.service.GetTreeCloudResourceDetail(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) UpdateTreeCloudResource(ctx *gin.Context) {
	var req model.UpdateTreeCloudResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc, ok := cloudOperator(ctx)
	if !ok {
		return
	}
	req.ID = id
	req.OperatorID = uc.Uid
	req.OperatorName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID, req.OperatorID, req.OperatorName = id, uc.Uid, uc.Username
		return nil, h.service.UpdateTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) DeleteTreeCloudResource(ctx *gin.Context) {
	var req model.DeleteTreeCloudResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc, ok := cloudOperator(ctx)
	if !ok {
		return
	}
	req.ID = id
	req.OperatorID = uc.Uid
	req.OperatorName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID, req.OperatorID, req.OperatorName = id, uc.Uid, uc.Username
		return nil, h.service.DeleteTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) BindTreeCloudResource(ctx *gin.Context) {
	var req model.BindTreeCloudResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
		return nil, h.service.BindTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) UnBindTreeCloudResource(ctx *gin.Context) {
	var req model.UnBindTreeCloudResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
		return nil, h.service.UnBindTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) SyncTreeCloudResource(ctx *gin.Context) {
	var req model.SyncTreeCloudResourceReq

	uc, ok := cloudOperator(ctx)
	if !ok {
		return
	}
	req.OperatorID = uc.Uid
	req.OperatorName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.OperatorID, req.OperatorName = uc.Uid, uc.Username
		return h.service.SyncTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) GetSyncHistory(ctx *gin.Context) {
	var req model.GetCloudResourceSyncHistoryReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSyncHistory(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) GetChangeLog(ctx *gin.Context) {
	var req model.GetCloudResourceChangeLogReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetChangeLog(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) GetTreeNodeCloudResources(ctx *gin.Context) {
	var req model.GetTreeNodeCloudResourcesReq

	nodeId, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的节点ID")
		return
	}

	req.NodeID = nodeId

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeNodeCloudResources(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) ConnectCloudResourceTerminal(ctx *gin.Context) {
	options, err := ssh.ParseTerminalOptions(ctx.Request.URL.Query())
	if err != nil {
		base.BadRequest(ctx, base.StatusError, nil, err.Error())
		return
	}
	var req model.ConnectTreeCloudResourceTerminalReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	req.ID = id
	req.UserID = uc.Uid

	detailReq := &model.GetTreeCloudResourceDetailReq{ID: req.ID}
	cloud, err := h.service.GetTreeCloudResourceForConnection(ctx.Request.Context(), detailReq)
	if err != nil {
		base.ErrorWithMessage(ctx, "获取云资源信息失败: "+err.Error())
		return
	}

	// 仅支持ECS类型的云资源连接终端
	if cloud.ResourceType != model.ResourceTypeECS {
		base.ErrorWithMessage(ctx, "仅支持ECS类型的云资源连接终端")
		return
	}

	// 如果没有公网IP，尝试使用私网IP
	ipAddr := cloud.PublicIP
	if ipAddr == "" {
		ipAddr = cloud.PrivateIP
	}

	if ipAddr == "" {
		base.ErrorWithMessage(ctx, "云资源没有可用的IP地址")
		return
	}

	port := cloud.Port
	if port == 0 {
		port = 22
	}

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

	if h.newSSH == nil {
		base.ErrorWithMessage(ctx, "SSH客户端工厂未初始化")
		return
	}
	client := h.newSSH()
	if client == nil {
		base.ErrorWithMessage(ctx, "SSH客户端未初始化")
		return
	}
	defer func() {
		if client.Close() != nil {
			h.logger.Warn("关闭独立SSH连接失败")
		}
	}()
	if err := client.ConnectContext(ctx.Request.Context(), sshConfig); err != nil {
		base.ErrorWithMessage(ctx, "连接SSH失败: "+err.Error())
		return
	}

	if h.wsUpgrader == nil {
		base.ErrorWithMessage(ctx, "WebSocket升级器未初始化")
		return
	}

	// 升级WebSocket连接
	ws, err := h.wsUpgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		base.ErrorWithMessage(ctx, "升级WebSocket连接失败: "+err.Error())
		return
	}
	defer func() {
		if closeErr := ws.Close(); closeErr != nil {
			h.logger.Error("关闭WebSocket连接失败", zap.Error(closeErr))
		}
	}()

	// 启动终端会话
	if err := client.WebTerminalWithOptionsContext(ctx.Request.Context(), uc.Uid, ws, options); err != nil {
		h.logger.Warn("Web终端会话已结束")
		return
	}
}

func (h *TreeCloudHandler) UpdateCloudResourceStatus(ctx *gin.Context) {
	var req model.UpdateCloudResourceStatusReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.ID = id
		return nil, h.service.UpdateCloudResourceStatus(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) BatchDeleteTreeCloudResource(ctx *gin.Context) {
	var req model.BatchDeleteTreeCloudResourceReq

	uc, ok := cloudOperator(ctx)
	if !ok {
		return
	}
	req.OperatorID = uc.Uid
	req.OperatorName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.OperatorID, req.OperatorName = uc.Uid, uc.Username
		return nil, h.service.BatchDeleteTreeCloudResource(ctx.Request.Context(), &req)
	})
}

func (h *TreeCloudHandler) BatchUpdateCloudResourceStatus(ctx *gin.Context) {
	var req model.BatchUpdateCloudResourceStatusReq

	uc, ok := cloudOperator(ctx)
	if !ok {
		return
	}
	req.OperatorID = uc.Uid
	req.OperatorName = uc.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		req.OperatorID, req.OperatorName = uc.Uid, uc.Username
		return nil, h.service.BatchUpdateCloudResourceStatus(ctx.Request.Context(), &req)
	})
}
