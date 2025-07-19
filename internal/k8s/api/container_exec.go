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
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sContainerExecHandler struct {
	logger               *zap.Logger
	containerExecService admin.ContainerExecService
}

func NewK8sContainerExecHandler(logger *zap.Logger, containerExecService admin.ContainerExecService) *K8sContainerExecHandler {
	return &K8sContainerExecHandler{
		logger:               logger,
		containerExecService: containerExecService,
	}
}

func (h *K8sContainerExecHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	containers := k8sGroup.Group("/containers/:id")
	{
		// 单次命令执行 - 需要操作权限
		containers.POST("/exec", h.ExecuteCommand)
		// 终端会话 - 需要操作权限
		containers.POST("/exec/terminal", h.CreateTerminalSession)
		// WebSocket 终端连接 - 需要操作权限
		containers.GET("/exec/ws", h.TerminalWebSocket)
		// 获取命令执行历史 - 只需查看权限
		containers.GET("/exec/history", h.GetExecutionHistory)
		// 获取会话列表 - 只需查看权限
		containers.GET("/sessions", h.GetSessions)
		// 关闭会话 - 需要操作权限
		containers.DELETE("/sessions/:sessionId", h.CloseSession)

		// 文件管理
		containers.GET("/files", h.GetFiles)                    // 查看权限
		containers.POST("/files/upload", h.UploadFile)          // 管理员权限
		containers.GET("/files/download", h.DownloadFile)       // 查看权限
		containers.PUT("/files/edit", h.EditFile)               // 管理员权限
		containers.DELETE("/files/delete", h.DeleteFile)        // 管理员权限

		// 日志管理
		containers.GET("/logs", h.GetLogs)                      // 查看权限
		containers.GET("/logs/stream", h.StreamLogs)            // 查看权限
		containers.GET("/logs/search", h.SearchLogs)            // 查看权限
		containers.POST("/logs/export", h.ExportLogs)           // 操作权限
		containers.GET("/logs/history", h.GetLogsHistory)       // 查看权限
	}
}

// ExecuteCommand 执行容器命令
func (h *K8sContainerExecHandler) ExecuteCommand(ctx *gin.Context) {
	var req model.K8sContainerExecRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.ExecuteCommand(ctx, containerId, &req)
	})
}

// CreateTerminalSession 创建终端会话
func (h *K8sContainerExecHandler) CreateTerminalSession(ctx *gin.Context) {
	var req model.K8sContainerTerminalRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.CreateTerminalSession(ctx, containerId, &req)
	})
}

// TerminalWebSocket WebSocket 终端连接
func (h *K8sContainerExecHandler) TerminalWebSocket(ctx *gin.Context) {
	containerId := ctx.Param("id")
	sessionId := ctx.Query("session")
	tty := ctx.Query("tty") == "true"

	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	if sessionId == "" {
		utils.BadRequestError(ctx, "缺少会话ID参数")
		return
	}

	err := h.containerExecService.HandleWebSocketTerminal(ctx, containerId, sessionId, tty)
	if err != nil {
		h.logger.Error("WebSocket终端连接失败", zap.Error(err))
		utils.InternalServerErrorWithDetails(ctx, nil, "WebSocket连接失败")
		return
	}
}

// GetExecutionHistory 获取命令执行历史
func (h *K8sContainerExecHandler) GetExecutionHistory(ctx *gin.Context) {
	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	var req model.K8sContainerExecHistoryRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.GetExecutionHistory(ctx, containerId, &req)
	})
}

// GetSessions 获取会话列表
func (h *K8sContainerExecHandler) GetSessions(ctx *gin.Context) {
	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.containerExecService.GetSessions(ctx, containerId)
	})
}

// CloseSession 关闭会话
func (h *K8sContainerExecHandler) CloseSession(ctx *gin.Context) {
	containerId := ctx.Param("id")
	sessionId := ctx.Param("sessionId")

	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	if sessionId == "" {
		utils.BadRequestError(ctx, "缺少会话ID参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.containerExecService.CloseSession(ctx, containerId, sessionId)
	})
}

// GetFiles 获取文件列表
func (h *K8sContainerExecHandler) GetFiles(ctx *gin.Context) {
	var req model.K8sContainerFilesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.GetFiles(ctx, containerId, &req)
	})
}

// UploadFile 上传文件
func (h *K8sContainerExecHandler) UploadFile(ctx *gin.Context) {
	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		utils.BadRequestError(ctx, "获取上传文件失败: "+err.Error())
		return
	}
	defer file.Close()

	path := ctx.PostForm("path")
	overwrite := ctx.PostForm("overwrite") == "true"

	if path == "" {
		utils.BadRequestError(ctx, "缺少目标路径参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.containerExecService.UploadFile(ctx, containerId, file, header, path, overwrite)
	})
}

// DownloadFile 下载文件
func (h *K8sContainerExecHandler) DownloadFile(ctx *gin.Context) {
	containerId := ctx.Param("id")
	path := ctx.Query("path")

	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	if path == "" {
		utils.BadRequestError(ctx, "缺少文件路径参数")
		return
	}

	err := h.containerExecService.DownloadFile(ctx, containerId, path)
	if err != nil {
		h.logger.Error("下载文件失败", zap.Error(err))
		utils.InternalServerErrorWithDetails(ctx, nil, "下载文件失败")
		return
	}
}

// EditFile 编辑文件
func (h *K8sContainerExecHandler) EditFile(ctx *gin.Context) {
	var req model.K8sContainerFileEditRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.containerExecService.EditFile(ctx, containerId, &req)
	})
}

// DeleteFile 删除文件
func (h *K8sContainerExecHandler) DeleteFile(ctx *gin.Context) {
	var req model.K8sContainerFileDeleteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.containerExecService.DeleteFile(ctx, containerId, &req)
	})
}

// GetLogs 获取容器日志
func (h *K8sContainerExecHandler) GetLogs(ctx *gin.Context) {
	var req model.K8sContainerLogsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.GetLogs(ctx, containerId, &req)
	})
}

// StreamLogs 实时日志流
func (h *K8sContainerExecHandler) StreamLogs(ctx *gin.Context) {
	var req model.K8sContainerLogsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	err := h.containerExecService.StreamLogs(ctx, containerId, &req)
	if err != nil {
		h.logger.Error("日志流失败", zap.Error(err))
		utils.InternalServerErrorWithDetails(ctx, nil, "日志流失败")
		return
	}
}

// SearchLogs 搜索容器日志
func (h *K8sContainerExecHandler) SearchLogs(ctx *gin.Context) {
	var req model.K8sContainerLogsRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.containerExecService.SearchLogs(ctx, containerId, &req)
	})
}

// ExportLogs 导出容器日志
func (h *K8sContainerExecHandler) ExportLogs(ctx *gin.Context) {
	var req model.K8sContainerLogsExportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.BadRequestError(ctx, err.Error())
		return
	}

	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	err := h.containerExecService.ExportLogs(ctx, containerId, &req)
	if err != nil {
		h.logger.Error("导出日志失败", zap.Error(err))
		utils.InternalServerErrorWithDetails(ctx, nil, "导出日志失败")
		return
	}
}

// GetLogsHistory 获取日志历史记录
func (h *K8sContainerExecHandler) GetLogsHistory(ctx *gin.Context) {
	containerId := ctx.Param("id")
	if containerId == "" {
		utils.BadRequestError(ctx, "缺少容器ID参数")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.containerExecService.GetLogsHistory(ctx, containerId)
	})
}
