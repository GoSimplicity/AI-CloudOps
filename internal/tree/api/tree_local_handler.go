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
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TreeLocalHandler struct {
	service service.TreeLocalService
	ssh     ssh.EcsSSH
	logger  *zap.Logger
}

func NewTreeLocalHandler(service service.TreeLocalService, ssh ssh.EcsSSH, logger *zap.Logger) *TreeLocalHandler {
	return &TreeLocalHandler{
		service: service,
		ssh:     ssh,
		logger:  logger,
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
	}
}

func (h *TreeLocalHandler) GetTreeLocalList(ctx *gin.Context) {
	var req model.GetTreeLocalListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalList(ctx, &req)
	})
}

func (h *TreeLocalHandler) GetTreeLocalDetail(ctx *gin.Context) {
	var req model.GetTreeLocalDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalDetail(ctx, &req)
	})
}

func (h *TreeLocalHandler) CreateTreeLocal(ctx *gin.Context) {
	var req model.CreateTreeLocalReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.CreateTreeLocal(ctx, &req)
	})
}

func (h *TreeLocalHandler) UpdateTreeLocal(ctx *gin.Context) {
	var req model.UpdateTreeLocalReq

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

func (h *TreeLocalHandler) DeleteTreeLocal(ctx *gin.Context) {
	var req model.DeleteTreeLocalReq

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

func (h *TreeLocalHandler) ConnectTerminal(ctx *gin.Context) {
	var req model.GetTreeLocalDetailReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的资源ID")
		return
	}

	uc := ctx.MustGet("user").(utils.UserClaims)

	req.ID = id

	ld, err := h.service.GetTreeLocalDetail(ctx, &req)
	if err != nil {
		return
	}

	// 升级 websocket 连接
	ws, err := utils.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		h.logger.Error("升级 websocket 失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "升级 websocket 连接失败: "+err.Error())
		return
	}
	defer func() {
		err := ws.Close()
		if err != nil {
			h.logger.Error("关闭 websocket 连接失败", zap.Error(err))
			return
		}
		if h.ssh.Sessions != nil {
			for _, session := range h.ssh.Sessions {
				err := session.Close()
				if err != nil {
					h.logger.Error("关闭 ssh 会话失败", zap.Error(err))
					return
				}
			}
		}
		if h.ssh.Client != nil {
			err := h.ssh.Client.Close()
			if err != nil {
				h.logger.Error("关闭 ssh 客户端失败", zap.Error(err))
				return
			}
		}
	}()

	err = h.ssh.Connect(ld.IpAddr, ld.Port, ld.Username, ld.Password, ld.Key, ld.AuthMode, uc.Uid)
	if err != nil {
		h.logger.Error("连接主机失败", zap.Error(err))
		utils.ErrorWithMessage(ctx, "连接ECS实例失败: "+err.Error())
		return
	}

	// 进行 web-ssh 命令通信
	h.ssh.Web2SSH(ws)
}
