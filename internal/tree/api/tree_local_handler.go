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
)

type TreeLocalHandler struct {
	service service.TreeLocalService
	ssh     ssh.EcsSSH
}

func NewTreeLocalHandler(service service.TreeLocalService, ssh ssh.EcsSSH) *TreeLocalHandler {
	return &TreeLocalHandler{
		service: service,
		ssh:     ssh,
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
	var req model.GetTreeLocalListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetTreeLocalList(ctx, &req)
	})
}

func (h *TreeLocalHandler) GetTreeLocalDetail(ctx *gin.Context) {
	var req model.GetTreeLocalDetailReq

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

	ld, err := h.service.GetTreeLocalForConnection(ctx, &req)
	if err != nil {
		utils.ErrorWithMessage(ctx, "获取主机信息失败: "+err.Error())
		return
	}

	// 升级 websocket 连接
	ws, err := utils.UpGrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		utils.ErrorWithMessage(ctx, "升级 websocket 连接失败: "+err.Error())
		return
	}
	defer func() {
		err := ws.Close()
		if err != nil {
			utils.ErrorWithMessage(ctx, "关闭 websocket 连接失败: "+err.Error())
			return
		}
		err = h.ssh.Close()
		if err != nil {
			utils.ErrorWithMessage(ctx, "关闭 ssh 连接失败: "+err.Error())
			return
		}
	}()

	err = h.ssh.Connect(ld.IpAddr, ld.Port, ld.Username, ld.Password, ld.Key, string(ld.AuthMode), uc.Uid)
	if err != nil {
		utils.ErrorWithMessage(ctx, "连接ECS实例失败: "+err.Error())
		return
	}

	// 进行 web-ssh 命令通信
	h.ssh.Web2SSH(ws)
}

func (h *TreeLocalHandler) BindTreeLocal(ctx *gin.Context) {
	var req model.BindLocalResourceReq

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

func (h *TreeLocalHandler) UnbindTreeLocal(ctx *gin.Context) {
	var req model.UnBindLocalResourceReq

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
