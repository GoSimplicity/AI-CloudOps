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
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeRdsHandler struct {
	rdsService service.TreeRdsService
}

func NewTreeRdsHandler(rdsService service.TreeRdsService) *TreeRdsHandler {
	return &TreeRdsHandler{
		rdsService: rdsService,
	}
}

func (h *TreeRdsHandler) RegisterRouters(server *gin.Engine) {
	rdsGroup := server.Group("/rds")
	{
		rdsGroup.POST("/list", h.ListRdsResources)
		rdsGroup.POST("/detail", h.GetRdsDetail)
		rdsGroup.POST("/create", h.CreateRdsResource)
		rdsGroup.POST("/start", h.StartRds)
		rdsGroup.POST("/stop", h.StopRds)
		rdsGroup.POST("/restart", h.RestartRds)
		rdsGroup.POST("/delete", h.DeleteRds)
	}
}

func (h *TreeRdsHandler) ListRdsResources(ctx *gin.Context) {

}

func (h *TreeRdsHandler) GetRdsDetail(ctx *gin.Context) {

}

func (h *TreeRdsHandler) CreateRdsResource(ctx *gin.Context) {

}

func (h *TreeRdsHandler) StartRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) StopRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) RestartRds(ctx *gin.Context) {

}

func (h *TreeRdsHandler) DeleteRds(ctx *gin.Context) {

}
