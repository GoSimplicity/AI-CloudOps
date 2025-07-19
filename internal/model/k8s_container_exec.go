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

package model

// K8sContainerExecRequest 容器执行命令请求
type K8sContainerExecRequest struct {
	ClusterId     int      `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string   `json:"namespace" binding:"required"`      // 命名空间
	PodName       string   `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string   `json:"container_name" binding:"required"` // 容器名称
	Command       []string `json:"command" binding:"required"`        // 执行的命令
	Timeout       int      `json:"timeout"`                           // 执行超时时间（秒）
	WorkingDir    string   `json:"working_dir"`                       // 工作目录
}

// K8sContainerExecResponse 容器执行命令响应
type K8sContainerExecResponse struct {
	SessionId     string  `json:"session_id"`     // 会话ID
	Stdout        string  `json:"stdout"`         // 标准输出
	Stderr        string  `json:"stderr"`         // 标准错误输出
	ExitCode      int     `json:"exit_code"`      // 退出码
	ExecutionTime float64 `json:"execution_time"` // 执行时间（秒）
}

// K8sContainerTerminalRequest 终端会话请求
type K8sContainerTerminalRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	TTY           bool   `json:"tty"`                               // 是否分配TTY
	Stdin         bool   `json:"stdin"`                             // 是否支持标准输入
	WorkingDir    string `json:"working_dir"`                       // 工作目录
}

// K8sContainerTerminalResponse 终端会话响应
type K8sContainerTerminalResponse struct {
	SessionId    string `json:"session_id"`    // 会话ID
	WebSocketURL string `json:"websocket_url"` // WebSocket连接URL
}

// K8sContainerExecHistory 容器执行历史记录
type K8sContainerExecHistory struct {
	Model
	SessionId      string  `json:"session_id" gorm:"size:100;comment:会话ID"`                   // 会话ID
	ClusterId      int     `json:"cluster_id" gorm:"index;comment:集群ID"`                      // 集群ID
	Namespace      string  `json:"namespace" gorm:"size:100;comment:命名空间"`                    // 命名空间
	PodName        string  `json:"pod_name" gorm:"size:200;comment:Pod名称"`                     // Pod名称
	ContainerName  string  `json:"container_name" gorm:"size:200;comment:容器名称"`               // 容器名称
	Command        string  `json:"command" gorm:"type:text;comment:执行的命令"`                     // 执行的命令
	UserId         int     `json:"user_id" gorm:"index;comment:执行用户ID"`                       // 执行用户ID
	UserName       string  `json:"user_name" gorm:"size:100;comment:执行用户名"`                   // 执行用户名
	ExitCode       int     `json:"exit_code" gorm:"comment:退出码"`                             // 退出码
	ExecutionTime  float64 `json:"execution_time" gorm:"comment:执行时间（秒）"`                     // 执行时间（秒）
	Stdout         string  `json:"stdout" gorm:"type:text;comment:标准输出"`                      // 标准输出
	Stderr         string  `json:"stderr" gorm:"type:text;comment:标准错误输出"`                    // 标准错误输出
	Status         string  `json:"status" gorm:"size:50;comment:执行状态"`                        // 执行状态：success, failed, timeout
	ErrorMessage   string  `json:"error_message" gorm:"type:text;comment:错误信息"`               // 错误信息
	ExecutedAt     string  `json:"executed_at" gorm:"comment:执行时间"`                           // 执行时间
	SessionType    string  `json:"session_type" gorm:"size:50;comment:会话类型"`                  // 会话类型：exec, terminal
}

// K8sContainerSession 容器会话管理
type K8sContainerSession struct {
	Model
	SessionId     string `json:"session_id" gorm:"size:100;unique;comment:会话ID"`              // 会话ID
	ClusterId     int    `json:"cluster_id" gorm:"index;comment:集群ID"`                        // 集群ID
	Namespace     string `json:"namespace" gorm:"size:100;comment:命名空间"`                      // 命名空间
	PodName       string `json:"pod_name" gorm:"size:200;comment:Pod名称"`                       // Pod名称
	ContainerName string `json:"container_name" gorm:"size:200;comment:容器名称"`                 // 容器名称
	UserId        int    `json:"user_id" gorm:"index;comment:用户ID"`                           // 用户ID
	UserName      string `json:"user_name" gorm:"size:100;comment:用户名"`                       // 用户名
	SessionType   string `json:"session_type" gorm:"size:50;comment:会话类型"`                    // 会话类型：exec, terminal
	Status        string `json:"status" gorm:"size:50;comment:会话状态"`                          // 会话状态：active, closed, expired
	StartTime     string `json:"start_time" gorm:"comment:开始时间"`                              // 开始时间
	EndTime       string `json:"end_time" gorm:"comment:结束时间"`                                // 结束时间
	LastActivity  string `json:"last_activity" gorm:"comment:最后活动时间"`                         // 最后活动时间
	TTY           bool   `json:"tty" gorm:"comment:是否为TTY"`                                   // 是否为TTY
	WorkingDir    string `json:"working_dir" gorm:"size:500;comment:工作目录"`                    // 工作目录
}

// K8sContainerExecHistoryRequest 获取执行历史请求
type K8sContainerExecHistoryRequest struct {
	ClusterId     int    `json:"cluster_id"`      // 集群ID
	Namespace     string `json:"namespace"`       // 命名空间
	PodName       string `json:"pod_name"`        // Pod名称
	ContainerName string `json:"container_name"`  // 容器名称
	UserId        int    `json:"user_id"`         // 用户ID
	Status        string `json:"status"`          // 执行状态
	StartTime     string `json:"start_time"`      // 开始时间
	EndTime       string `json:"end_time"`        // 结束时间
	Limit         int    `json:"limit"`           // 限制数量
	Offset        int    `json:"offset"`          // 偏移量
}

// K8sContainerExecHistoryResponse 执行历史响应
type K8sContainerExecHistoryResponse struct {
	History    []K8sContainerExecHistory `json:"history"`     // 历史记录列表
	TotalCount int                       `json:"total_count"` // 总数
}

// K8sContainerFilesRequest 文件管理请求
type K8sContainerFilesRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	Path          string `json:"path"`                              // 文件路径
	Recursive     bool   `json:"recursive"`                         // 是否递归
}

// K8sContainerFile 容器文件信息
type K8sContainerFile struct {
	Name         string `json:"name"`          // 文件名
	Path         string `json:"path"`          // 文件路径
	Size         int64  `json:"size"`          // 文件大小
	Type         string `json:"type"`          // 文件类型：file, directory
	Permissions  string `json:"permissions"`   // 文件权限
	ModifiedTime string `json:"modified_time"` // 修改时间
}

// K8sContainerFilesResponse 文件列表响应
type K8sContainerFilesResponse struct {
	Files []K8sContainerFile `json:"files"` // 文件列表
}

// K8sContainerFileUploadRequest 文件上传请求
type K8sContainerFileUploadRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	Path          string `json:"path" binding:"required"`           // 目标路径
	Overwrite     bool   `json:"overwrite"`                         // 是否覆盖
}

// K8sContainerFileEditRequest 文件编辑请求
type K8sContainerFileEditRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	Path          string `json:"path" binding:"required"`           // 文件路径
	Content       string `json:"content" binding:"required"`        // 文件内容
	Backup        bool   `json:"backup"`                            // 是否备份
}

// K8sContainerFileDeleteRequest 文件删除请求
type K8sContainerFileDeleteRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	Path          string `json:"path" binding:"required"`           // 文件路径
	Recursive     bool   `json:"recursive"`                         // 是否递归删除
}

// K8sContainerLogsRequest 容器日志请求
type K8sContainerLogsRequest struct {
	ClusterId     int    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string `json:"namespace" binding:"required"`      // 命名空间
	PodName       string `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string `json:"container_name" binding:"required"` // 容器名称
	Tail          int    `json:"tail"`                              // 返回的日志行数
	Since         string `json:"since"`                             // 开始时间
	Until         string `json:"until"`                             // 结束时间
	Level         string `json:"level"`                             // 日志级别
	Search        string `json:"search"`                            // 搜索关键词
	Follow        bool   `json:"follow"`                            // 是否跟踪
}

// K8sContainerLogEntry 容器日志条目
type K8sContainerLogEntry struct {
	Timestamp     string `json:"timestamp"`      // 时间戳
	Level         string `json:"level"`          // 日志级别
	Message       string `json:"message"`        // 日志消息
	ContainerName string `json:"container_name"` // 容器名称
	PodName       string `json:"pod_name"`       // Pod名称
	Namespace     string `json:"namespace"`      // 命名空间
}

// K8sContainerLogsResponse 容器日志响应
type K8sContainerLogsResponse struct {
	Logs     []K8sContainerLogEntry `json:"logs"`      // 日志列表
	Total    int                    `json:"total"`     // 总数
	HasMore  bool                   `json:"has_more"`  // 是否有更多
}

// K8sContainerLogsExportRequest 日志导出请求
type K8sContainerLogsExportRequest struct {
	ClusterId     int                    `json:"cluster_id" binding:"required"`     // 集群ID
	Namespace     string                 `json:"namespace" binding:"required"`      // 命名空间
	PodName       string                 `json:"pod_name" binding:"required"`       // Pod名称
	ContainerName string                 `json:"container_name" binding:"required"` // 容器名称
	Format        string                 `json:"format"`                            // 导出格式：json, csv, txt
	StartTime     string                 `json:"start_time"`                        // 开始时间
	EndTime       string                 `json:"end_time"`                          // 结束时间
	Filters       map[string]interface{} `json:"filters"`                           // 过滤条件
}