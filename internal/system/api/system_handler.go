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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	systemutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	svc service.SystemService
}

func NewSystemHandler(svc service.SystemService) *SystemHandler {
	return &SystemHandler{
		svc: svc,
	}
}

func (h *SystemHandler) RegisterRouters(server *gin.Engine) {
	systemGroup := server.Group("/api/system")

	systemGroup.GET("/info", h.GetSystemInfo)
	systemGroup.GET("/metrics", h.GetSystemMetrics)
	systemGroup.POST("/refresh", h.RefreshSystemInfo)
}

// GetSystemInfo 获取系统基本信息
func (h *SystemHandler) GetSystemInfo(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		system, err := h.svc.GetCurrentSystemInfo(ctx)
		if err != nil {
			return nil, err
		}
		return systemutils.ToResponse(system), nil
	})
}

// GetSystemMetrics 获取系统性能指标
func (h *SystemHandler) GetSystemMetrics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		system, err := h.svc.GetSystemMetrics(ctx)
		if err != nil {
			return nil, err
		}
		return systemutils.ToResponse(system), nil
	})
}

// RefreshSystemInfo 刷新系统信息
func (h *SystemHandler) RefreshSystemInfo(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		system, err := h.svc.RefreshSystemInfo(ctx)
		if err != nil {
			return nil, err
		}
		return systemutils.ToResponse(system), nil
	})
}
