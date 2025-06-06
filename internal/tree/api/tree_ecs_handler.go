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

type TreeEcsHandler struct {
	ecsService service.TreeEcsService
}

func NewTreeEcsHandler(ecsService service.TreeEcsService) *TreeEcsHandler {
	return &TreeEcsHandler{
		ecsService: ecsService,
	}
}

func (h *TreeEcsHandler) RegisterRouters(server *gin.Engine) {
	ecsGroup := server.Group("/ecs")
	{
		ecsGroup.POST("/list", h.ListEcsResources)
		ecsGroup.POST("/instance_options", h.ListInstanceOptions)
		ecsGroup.POST("/detail", h.GetEcsDetail)
		ecsGroup.POST("/create", h.CreateEcsResource)
		ecsGroup.DELETE("/delete", h.DeleteEcs)
		ecsGroup.POST("/start", h.StartEcs)
		ecsGroup.POST("/stop", h.StopEcs)
		ecsGroup.POST("/restart", h.RestartEcs)
	}
}

func (h *TreeEcsHandler) ListEcsResources(c *gin.Context) {

}

func (h *TreeEcsHandler) ListInstanceOptions(c *gin.Context) {

}

func (h *TreeEcsHandler) GetEcsDetail(c *gin.Context) {

}

func (h *TreeEcsHandler) CreateEcsResource(c *gin.Context) {

}

func (h *TreeEcsHandler) DeleteEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) StartEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) StopEcs(c *gin.Context) {

}

func (h *TreeEcsHandler) RestartEcs(c *gin.Context) {

}
