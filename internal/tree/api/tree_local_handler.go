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
	"github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"github.com/gin-gonic/gin"
)

type TreeLocalHandler struct {
	service   service.TreeLocalService
	sshClient ssh.Client
	wsManager websocket.Manager
}

func NewTreeLocalHandler(service service.TreeLocalService, sshClient ssh.Client) *TreeLocalHandler {
	// 初始化WebSocket管理器
	wsManager := websocket.NewManager(nil, nil)

	return &TreeLocalHandler{
		service:   service,
		sshClient: sshClient,
		wsManager: wsManager,
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

// GetTreeLocalList 获取本地资源列表
func (h *TreeLocalHandler) GetTreeLocalList(ctx *gin.Context) {
	var req model.GetTreeLocalResourceListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalList(ctx, &req)
	})
}

// GetTreeLocalDetail 获取本地资源详情
func (h *TreeLocalHandler) GetTreeLocalDetail(ctx *gin.Context) {
	var req model.GetTreeLocalResourceDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "invalid param id")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalDetail(ctx, &req)
	})
}

// CreateTreeLocal 创建本地资源
func (h *TreeLocalHandler) CreateTreeLocal(ctx *gin.Context) {
	var req model.CreateTreeLocalResourceReq

	user := ctx.MustGet("user").(utils.UserClaims)

	req.CreateUserID = user.Uid
	req.CreateUserName = user.Username

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTreeLocal(ctx, &req)
	})
}

// UpdateTreeLocal 更新本地资源
func (h *TreeLocalHandler) UpdateTreeLocal(ctx *gin.Context) {
	var req model.UpdateTreeLocalResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UpdateTreeLocal(ctx, &req)
	})
}

// DeleteTreeLocal 删除本地资源
func (h *TreeLocalHandler) DeleteTreeLocal(ctx *gin.Context) {
	var req model.DeleteTreeLocalResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteTreeLocal(ctx, &req)
	})
}

// ConnectTerminal 连接终端
func (h *TreeLocalHandler) ConnectTerminal(ctx *gin.Context) {
	var req model.GetTreeLocalResourceDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc := ctx.MustGet("user").(utils.UserClaims)
	req.ID = id

	ld, err := h.service.GetTreeLocalForConnection(ctx, &req)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取主机信息失败: "+err.Error())
		return
	}

	defer func() {
		if closeErr := h.sshClient.Close(); closeErr != nil {
			utils.ErrorWithMessage(ctx, "关闭SSH连接失败: "+closeErr.Error())
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
	if err := h.sshClient.Connect(sshConfig); err != nil {
		utils.ErrorWithMessage(ctx, "连接SSH失败: "+err.Error())
		return
	}

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

// BindTreeLocal 绑定本地资源
func (h *TreeLocalHandler) BindTreeLocal(ctx *gin.Context) {
	var req model.BindTreeLocalResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.BindTreeLocal(ctx, &req)
	})

}

// UnbindTreeLocal 解绑本地资源
func (h *TreeLocalHandler) UnbindTreeLocal(ctx *gin.Context) {
	var req model.UnBindTreeLocalResourceReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.UnBindLocalResource(ctx, &req)
	})
}
