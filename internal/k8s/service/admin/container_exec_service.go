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

package admin

import (
	"context"
	"mime/multipart"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

// ContainerExecService 容器执行服务接口
type ContainerExecService interface {
	// ExecuteCommand 执行单次命令
	ExecuteCommand(ctx context.Context, containerId string, req *model.K8sContainerExecRequest) (*model.K8sContainerExecResponse, error)
	
	// CreateTerminalSession 创建终端会话
	CreateTerminalSession(ctx context.Context, containerId string, req *model.K8sContainerTerminalRequest) (*model.K8sContainerTerminalResponse, error)
	
	// HandleWebSocketTerminal 处理WebSocket终端连接
	HandleWebSocketTerminal(ctx *gin.Context, containerId, sessionId string, tty bool) error
	
	// GetSessions 获取会话列表
	GetSessions(ctx context.Context, containerId string) ([]model.K8sContainerSession, error)
	
	// CloseSession 关闭会话
	CloseSession(ctx context.Context, containerId, sessionId string) error
	
	// GetExecutionHistory 获取执行历史
	GetExecutionHistory(ctx context.Context, containerId string, req *model.K8sContainerExecHistoryRequest) (*model.K8sContainerExecHistoryResponse, error)
	
	// GetFiles 获取文件列表
	GetFiles(ctx context.Context, containerId string, req *model.K8sContainerFilesRequest) (*model.K8sContainerFilesResponse, error)
	
	// UploadFile 上传文件
	UploadFile(ctx context.Context, containerId string, file multipart.File, header *multipart.FileHeader, path string, overwrite bool) error
	
	// DownloadFile 下载文件
	DownloadFile(ctx *gin.Context, containerId, path string) error
	
	// EditFile 编辑文件
	EditFile(ctx context.Context, containerId string, req *model.K8sContainerFileEditRequest) error
	
	// DeleteFile 删除文件
	DeleteFile(ctx context.Context, containerId string, req *model.K8sContainerFileDeleteRequest) error
	
	// GetLogs 获取日志
	GetLogs(ctx context.Context, containerId string, req *model.K8sContainerLogsRequest) (*model.K8sContainerLogsResponse, error)
	
	// StreamLogs 实时日志流
	StreamLogs(ctx *gin.Context, containerId string, req *model.K8sContainerLogsRequest) error
	
	// SearchLogs 搜索日志
	SearchLogs(ctx context.Context, containerId string, req *model.K8sContainerLogsRequest) (*model.K8sContainerLogsResponse, error)
	
	// ExportLogs 导出日志
	ExportLogs(ctx *gin.Context, containerId string, req *model.K8sContainerLogsExportRequest) error
	
	// GetLogsHistory 获取日志历史
	GetLogsHistory(ctx context.Context, containerId string) ([]model.K8sContainerExecHistory, error)
}