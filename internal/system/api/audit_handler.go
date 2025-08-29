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
func (h *AuditHandler) CreateAuditLog(ctx *gin.Context) {
	var req model.CreateAuditLogRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.CreateAuditLog(ctx, &req)
	})
}

// BatchCreateAuditLogs 批量创建审计日志 - 高性能批处理
func (h *AuditHandler) BatchCreateAuditLogs(ctx *gin.Context) {
	var req model.AuditLogBatch

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchCreateAuditLogs(ctx, req.Logs)
	})
}

// ListAuditLogs 获取审计日志列表
func (h *AuditHandler) ListAuditLogs(ctx *gin.Context) {
	var req model.ListAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.ListAuditLogs(ctx, &req)
	})
}

// GetAuditLogDetail 获取审计日志详情
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
func (h *AuditHandler) SearchAuditLogs(ctx *gin.Context) {
	var req model.SearchAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.svc.SearchAuditLogs(ctx, &req)
	})
}

// GetAuditStatistics 获取审计统计信息
func (h *AuditHandler) GetAuditStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditStatistics(ctx)
	})
}

// GetAuditTypes 获取审计类型列表
func (h *AuditHandler) GetAuditTypes(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.svc.GetAuditTypes(ctx)
	})
}

// DeleteAuditLog 删除审计日志
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
func (h *AuditHandler) BatchDeleteLogs(ctx *gin.Context) {
	var req model.BatchDeleteRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.BatchDeleteAuditLogs(ctx, req.IDs)
	})
}

// ArchiveAuditLogs 归档审计日志
func (h *AuditHandler) ArchiveAuditLogs(ctx *gin.Context) {
	var req model.ArchiveAuditLogsRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.svc.ArchiveAuditLogs(ctx, &req)
	})
}
