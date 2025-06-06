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

type TreeElbHandler struct {
	elbService service.TreeElbService
}

func NewTreeElbHandler(elbService service.TreeElbService) *TreeElbHandler {
	return &TreeElbHandler{
		elbService: elbService,
	}
}

func (h *TreeElbHandler) RegisterRouters(server *gin.Engine) {
	elbGroup := server.Group("/elb")
	{
		elbGroup.POST("/list", h.ListElbResources)
		elbGroup.POST("/detail", h.GetElbDetail)
		elbGroup.POST("/create", h.CreateElbResource)
		elbGroup.POST("/delete", h.DeleteElb)
	}
}

func (h *TreeElbHandler) ListElbResources(ctx *gin.Context) {

}

func (h *TreeElbHandler) GetElbDetail(ctx *gin.Context) {

}

func (h *TreeElbHandler) CreateElbResource(ctx *gin.Context) {

}

func (h *TreeElbHandler) DeleteElb(ctx *gin.Context) {

}
