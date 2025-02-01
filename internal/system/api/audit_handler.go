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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	svc service.AuditService
}

func NewAuditHandler(svc service.AuditService) *AuditHandler {
	return &AuditHandler{
		svc: svc,
	}
}

func (h *AuditHandler) RegisterRouters(server *gin.Engine) {
	auditGroup := server.Group("/api/audit")

	auditGroup.POST("/list", h.ListAuditLogs)
	auditGroup.GET("/detail/:id", h.GetAuditLogDetail)
	auditGroup.GET("/types", h.GetAuditTypes)
	auditGroup.GET("/statistics", h.GetAuditStatistics)
	auditGroup.POST("/search", h.SearchAuditLogs)
	auditGroup.POST("/export", h.ExportAuditLogs)
	auditGroup.DELETE("/:id", h.DeleteAuditLog)
	auditGroup.POST("/batch-delete", h.BatchDeleteLogs)
	auditGroup.POST("/archive", h.ArchiveAuditLogs)
}

// ListAuditLogs 获取审计日志列表
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	var req model.ListAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	logs, total, err := h.svc.ListAuditLogs(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, gin.H{
		"list":  logs,
		"total": total,
	})
}

// GetAuditLogDetail 获取审计日志详情
func (h *AuditHandler) GetAuditLogDetail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c)
		return
	}

	detail, err := h.svc.GetAuditLogDetail(c.Request.Context(), uint(id))
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, detail)
}

// GetAuditTypes 获取审计类型列表
func (h *AuditHandler) GetAuditTypes(c *gin.Context) {
	types, err := h.svc.GetAuditTypes(c.Request.Context())
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, types)
}

// GetAuditStatistics 获取审计统计信息
func (h *AuditHandler) GetAuditStatistics(c *gin.Context) {
	stats, err := h.svc.GetAuditStatistics(c.Request.Context())
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, stats)
}

// SearchAuditLogs 搜索审计日志
func (h *AuditHandler) SearchAuditLogs(c *gin.Context) {
	var req model.ListAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	logs, total, err := h.svc.SearchAuditLogs(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, gin.H{
		"list":  logs,
		"total": total,
	})
}

// ExportAuditLogs 导出审计日志
func (h *AuditHandler) ExportAuditLogs(c *gin.Context) {
	var req model.ListAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	data, err := h.svc.ExportAuditLogs(c.Request.Context(), &req)
	if err != nil {
		utils.Error(c)
		return
	}

	utils.SuccessWithData(c, data)
}

// DeleteAuditLog 删除单条审计日志
func (h *AuditHandler) DeleteAuditLog(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		utils.Error(c)
		return
	}

	if err := h.svc.DeleteAuditLog(c.Request.Context(), uint(id)); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// BatchDeleteLogs 批量删除审计日志
func (h *AuditHandler) BatchDeleteLogs(c *gin.Context) {
	var ids []uint
	if err := c.ShouldBindJSON(&ids); err != nil {
		utils.Error(c)
		return
	}

	if err := h.svc.BatchDeleteLogs(c.Request.Context(), ids); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}

// ArchiveAuditLogs 归档审计日志
func (h *AuditHandler) ArchiveAuditLogs(c *gin.Context) {
	var req model.ListAuditLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error(c)
		return
	}

	if err := h.svc.ArchiveAuditLogs(c.Request.Context(), &req); err != nil {
		utils.Error(c)
		return
	}

	utils.Success(c)
}
