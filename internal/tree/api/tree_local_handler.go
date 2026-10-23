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
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type TerminalClient interface {
	ConnectContext(context.Context, *ssh.Config) error
	WebTerminalContext(context.Context, int, *websocket.Conn) error
	Close() error
}
type TerminalClientFactory func() TerminalClient

type TreeLocalHandler struct {
	service    service.TreeLocalService
	newSSH     TerminalClientFactory
	wsUpgrader *websocket.Upgrader
	logger     *zap.Logger
}

func NewTreeLocalHandler(service service.TreeLocalService, factory TerminalClientFactory, wsUpgrader *websocket.Upgrader, logger *zap.Logger) *TreeLocalHandler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &TreeLocalHandler{
		service:    service,
		newSSH:     factory,
		wsUpgrader: wsUpgrader,
		logger:     logger,
	}
}

func (h *TreeLocalHandler) RegisterRouters(server *gin.Engine) {
	localGroup := server.Group("/api/tree/local")
	{
		localGroup.GET("/list", h.GetTreeLocalList)
		localGroup.GET("/detail/:id", h.GetTreeLocalDetail)
		localGroup.POST("/create", h.CreateTreeLocal)
		localGroup.PUT("/update/:id", h.UpdateTreeLocal)
		localGroup.DELETE("/delete/:id", h.DeleteTreeLocal)
		localGroup.GET("/terminal/:id", h.ConnectTerminal)
		localGroup.POST("/bind/:id", h.BindTreeLocal)
		localGroup.POST("/unbind/:id", h.UnbindTreeLocal)
	}
}

func (h *TreeLocalHandler) GetTreeLocalList(ctx *gin.Context) {
	var req model.GetTreeLocalResourceListReq

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalList(ctx.Request.Context(), &req)
	})
}

func (h *TreeLocalHandler) GetTreeLocalDetail(ctx *gin.Context) {
	var req model.GetTreeLocalResourceDetailReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalDetail(ctx.Request.Context(), &req)
	})
}

func (h *TreeLocalHandler) CreateTreeLocal(ctx *gin.Context) {
	var req model.CreateTreeLocalResourceReq

	user := ctx.MustGet("user").(jwt.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTreeLocal(ctx.Request.Context(), &req)
	})
}

func (h *TreeLocalHandler) UpdateTreeLocal(ctx *gin.Context) {
	var req model.UpdateTreeLocalResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTreeLocal(ctx.Request.Context(), &req)
	})
}

func (h *TreeLocalHandler) DeleteTreeLocal(ctx *gin.Context) {
	var req model.DeleteTreeLocalResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteTreeLocal(ctx.Request.Context(), &req)
	})
}

// ConnectTerminal 连接终端
func (h *TreeLocalHandler) ConnectTerminal(ctx *gin.Context) {
	var req model.GetTreeLocalResourceDetailReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc := ctx.MustGet("user").(jwt.UserClaims)
	req.ID = id

	ld, err := h.service.GetTreeLocalForConnection(ctx.Request.Context(), &req)
	if err != nil {
		base.ErrorWithMessage(ctx, "获取主机信息失败: "+err.Error())
		return
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

	// 配置SSH连接
	sshConfig := &ssh.Config{
		Host:     ld.IpAddr,
		Port:     ld.Port,
		Username: ld.Username,
		Password: ld.Password,
		Key:      ld.Key,
		Mode:     ssh.AuthMode(ld.AuthMode),
		Timeout:  10,
	}

	// 建立SSH连接
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
	if err := client.WebTerminalContext(ctx.Request.Context(), uc.Uid, ws); err != nil {
		h.logger.Warn("Web终端会话已结束")
		return
	}
}

func (h *TreeLocalHandler) BindTreeLocal(ctx *gin.Context) {
	var req model.BindTreeLocalResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindTreeLocal(ctx.Request.Context(), &req)
	})

}

func (h *TreeLocalHandler) UnbindTreeLocal(ctx *gin.Context) {
	var req model.UnBindTreeLocalResourceReq

	id, err := base.GetParamID(ctx)
	if err != nil {
		base.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnBindLocalResource(ctx.Request.Context(), &req)
	})
}
