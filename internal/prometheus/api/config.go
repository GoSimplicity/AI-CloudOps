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
		configs.POST("/create", h.CreateMonitorConfig)
		configs.PUT("/update/:id", h.UpdateMonitorConfig)
		configs.DELETE("/delete/:id", h.DeleteMonitorConfig)
	}
}

// GetMonitorConfigList 获取监控配置列表
// @Summary 获取监控配置列表
// @Description 获取所有监控配置的分页列表
// @Tags 监控配置
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/configs/list [get]
// @Security BearerAuth
func (h *MonitorConfigHandler) GetMonitorConfigList(ctx *gin.Context) {
	var req model.GetMonitorConfigListReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetMonitorConfigList(ctx, &req)
	})
}

// GetMonitorConfig 获取监控配置
// @Summary 获取监控配置详情
// @Description 根据ID获取指定监控配置的详细信息
// @Tags 监控配置
// @Accept json
// @Produce json
// @Param id path int true "监控配置ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/configs/detail/{id} [get]
// @Security BearerAuth
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

// CreateMonitorConfig 创建监控配置
// @Summary 创建监控配置
// @Description 创建新的监控配置
// @Tags 监控配置
// @Accept json
// @Produce json
// @Param request body model.CreateMonitorConfigReq true "创建监控配置请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/configs/create [post]
// @Security BearerAuth
func (h *MonitorConfigHandler) CreateMonitorConfig(ctx *gin.Context) {
	var req model.CreateMonitorConfigReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateMonitorConfig(ctx, &req)
	})
}

// UpdateMonitorConfig 更新监控配置
// @Summary 更新监控配置
// @Description 更新指定的监控配置
// @Tags 监控配置
// @Accept json
// @Produce json
// @Param id path int true "监控配置ID"
// @Param request body model.UpdateMonitorConfigReq true "更新监控配置请求参数"
// @Success 200 {object} utils.ApiResponse "更新成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/configs/update/{id} [put]
// @Security BearerAuth
func (h *MonitorConfigHandler) UpdateMonitorConfig(ctx *gin.Context) {
	var req model.UpdateMonitorConfigReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.UpdateMonitorConfig(ctx, &req)
	})
}

// DeleteMonitorConfig 删除监控配置
// @Summary 删除监控配置
// @Description 删除指定ID的监控配置
// @Tags 监控配置
// @Accept json
// @Produce json
// @Param id path int true "监控配置ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/monitor/configs/delete/{id} [delete]
// @Security BearerAuth
func (h *MonitorConfigHandler) DeleteMonitorConfig(ctx *gin.Context) {
	var req model.DeleteMonitorConfigReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteMonitorConfig(ctx, &req)
	})
}
