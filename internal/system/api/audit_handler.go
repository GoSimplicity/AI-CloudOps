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
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuditHandler struct {
	svc    service.AuditService
	logger *zap.Logger
}

func NewAuditHandler(svc service.AuditService, logger *zap.Logger) *AuditHandler {
	return &AuditHandler{
		svc:    svc,
		logger: logger,
	}
}

func (h *AuditHandler) RegisterRouters(server *gin.Engine) {
	auditGroup := server.Group("/api/audit")

	// 查询相关接口
	auditGroup.GET("/list", h.ListAuditLogs)
	auditGroup.GET("/detail/:id", h.GetAuditLogDetail)
	auditGroup.GET("/search", h.SearchAuditLogs)

	// 统计和分析接口
	auditGroup.GET("/statistics", h.GetAuditStatistics)
	auditGroup.GET("/types", h.GetAuditTypes)

	// 管理接口 - 需要管理员权限
	auditGroup.DELETE("/:id", h.DeleteAuditLog)
	auditGroup.POST("/batch-delete", h.BatchDeleteLogs)
	auditGroup.POST("/archive", h.ArchiveAuditLogs)

	// 创建接口 - 通常由系统内部调用
	auditGroup.POST("/create", h.CreateAuditLog)
	auditGroup.POST("/batch-create", h.BatchCreateAuditLogs)
}

// CreateAuditLog 创建单个审计日志
// @Summary 创建审计日志
// @Description 创建一条新的审计日志记录
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param request body model.CreateAuditLogRequest true "创建审计日志请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/create [post]
func (h *AuditHandler) CreateAuditLog(ctx *gin.Context) {
	var req model.CreateAuditLogRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateAuditLog(ctx, &req)
	})
}

// BatchCreateAuditLogs 批量创建审计日志 - 高性能批处理
// @Summary 批量创建审计日志
// @Description 高性能批量创建多条审计日志记录
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param request body model.AuditLogBatch true "批量创建审计日志请求参数"
// @Success 200 {object} utils.ApiResponse "创建成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/batch-create [post]
func (h *AuditHandler) BatchCreateAuditLogs(ctx *gin.Context) {
	var req model.AuditLogBatch

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchCreateAuditLogs(ctx, req.Logs)
	})
}

// ListAuditLogs 获取审计日志列表
// @Summary 获取审计日志列表
// @Description 分页获取系统审计日志列表
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param user_id query int false "用户ID"
// @Param action query string false "操作类型"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} utils.ApiResponse{data=[]model.AuditLog} "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/list [get]
func (h *AuditHandler) ListAuditLogs(ctx *gin.Context) {
	var req model.ListAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListAuditLogs(ctx, &req)
	})
}

// GetAuditLogDetail 获取审计日志详情
// @Summary 获取审计日志详情
// @Description 根据ID获取指定审计日志的详细信息
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param id path int true "审计日志ID"
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/detail/{id} [get]
func (h *AuditHandler) GetAuditLogDetail(ctx *gin.Context) {
	var req model.GetAuditLogDetailRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的审计日志ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.GetAuditLogDetail(ctx, req.ID)
	})
}

// SearchAuditLogs 搜索审计日志
// @Summary 搜索审计日志
// @Description 根据条件搜索审计日志记录
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param keyword query string false "关键字"
// @Param user_name query string false "用户名"
// @Param ip query string false "IP地址"
// @Param action query string false "操作类型"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} utils.ApiResponse{data=[]model.AuditLog} "搜索成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/search [get]
func (h *AuditHandler) SearchAuditLogs(ctx *gin.Context) {
	var req model.SearchAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.SearchAuditLogs(ctx, &req)
	})
}

// GetAuditStatistics 获取审计统计信息
// @Summary 获取审计统计信息
// @Description 获取审计日志相关的统计数据
// @Tags 审计管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/statistics [get]
func (h *AuditHandler) GetAuditStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditStatistics(ctx)
	})
}

// GetAuditTypes 获取审计类型列表
// @Summary 获取审计类型列表
// @Description 获取系统支持的所有审计类型
// @Tags 审计管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=[]string} "获取成功"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/types [get]
func (h *AuditHandler) GetAuditTypes(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditTypes(ctx)
	})
}

// DeleteAuditLog 删除审计日志
// @Summary 删除审计日志
// @Description 根据ID删除指定的审计日志
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param id path int true "审计日志ID"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/{id} [delete]
func (h *AuditHandler) DeleteAuditLog(ctx *gin.Context) {
	var req model.DeleteAuditLogRequest

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, "无效的审计日志ID")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.DeleteAuditLog(ctx, req.ID)
	})
}

// BatchDeleteLogs 批量删除审计日志
// @Summary 批量删除审计日志
// @Description 根据ID列表批量删除审计日志
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteRequest true "批量删除请求参数"
// @Success 200 {object} utils.ApiResponse "删除成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/batch-delete [post]
func (h *AuditHandler) BatchDeleteLogs(ctx *gin.Context) {
	var req model.BatchDeleteRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchDeleteAuditLogs(ctx, req.IDs)
	})
}

// ArchiveAuditLogs 归档审计日志
// @Summary 归档审计日志
// @Description 将指定时间范围的审计日志进行归档处理
// @Tags 审计管理
// @Accept json
// @Produce json
// @Param request body model.ArchiveAuditLogsRequest true "归档请求参数"
// @Success 200 {object} utils.ApiResponse "归档成功"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Security BearerAuth
// @Router /api/audit/archive [post]
func (h *AuditHandler) ArchiveAuditLogs(ctx *gin.Context) {
	var req model.ArchiveAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.ArchiveAuditLogs(ctx, &req)
	})
}
