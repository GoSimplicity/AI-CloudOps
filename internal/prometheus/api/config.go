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
	"github.com/gin-gonic/gin"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	configService "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/config"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
)

type MonitorConfigHandler struct {
	svc configService.MonitorConfigService
}

func NewMonitorConfigHandler(svc configService.MonitorConfigService) *MonitorConfigHandler {
	return &MonitorConfigHandler{
		svc: svc,
	}
}

func (h *MonitorConfigHandler) RegisterRouters(server *gin.Engine) {
	monitorGroup := server.Group("/api/monitor")

	configs := monitorGroup.Group("/configs")
	{
		configs.GET("/list", h.GetMonitorConfigList)
		configs.GET("/detail/:id", h.GetMonitorConfig)
	}
}

// GetMonitorConfigList 获取监控配置列表
func (h *MonitorConfigHandler) GetMonitorConfigList(ctx *gin.Context) {
	var req model.GetMonitorConfigListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetMonitorConfigList(ctx, &req)
	})
}

// GetMonitorConfig 获取监控配置
func (h *MonitorConfigHandler) GetMonitorConfig(ctx *gin.Context) {
	var req model.GetMonitorConfigReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetMonitorConfigByID(ctx, &req)
	})
}
