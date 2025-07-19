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
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gorm.io/gorm"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// ContainerInfo 容器信息结构
type ContainerInfo struct {
	ClusterId     int    `json:"cluster_id"`
	Namespace     string `json:"namespace"`
	PodName       string `json:"pod_name"`
	ContainerName string `json:"container_name"`
}

// containerExecService 容器执行服务实现
type containerExecService struct {
	logger     *zap.Logger
	db         *gorm.DB
	clusterDao admin.ClusterDAO
	upgrader   websocket.Upgrader
}

// NewContainerExecService 创建容器执行服务实例
func NewContainerExecService(logger *zap.Logger, db *gorm.DB, clusterDao admin.ClusterDAO) ContainerExecService {
	return &containerExecService{
		logger:     logger,
		db:         db,
		clusterDao: clusterDao,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 在生产环境中应该进行适当的来源检查
			},
		},
	}
}

// ExecuteCommand 在容器中执行单次命令
func (s *containerExecService) ExecuteCommand(ctx context.Context, containerId string, req *model.K8sContainerExecRequest) (*model.K8sContainerExecResponse, error) {
	s.logger.Info("开始执行容器命令",
		zap.String("containerId", containerId),
		zap.Int("clusterId", req.ClusterId),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName),
		zap.Strings("command", req.Command))

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, req.ClusterId)
	if err != nil {
		s.logger.Error("获取集群配置失败", zap.Error(err))
		return nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		s.logger.Error("创建Kubernetes客户端失败", zap.Error(err))
		return nil, fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 生成会话ID
	sessionId := uuid.New().String()

	// 记录执行历史
	history := &model.K8sContainerExecHistory{
		SessionId:     sessionId,
		ClusterId:     req.ClusterId,
		Namespace:     req.Namespace,
		PodName:       req.PodName,
		ContainerName: req.ContainerName,
		Command:       strings.Join(req.Command, " "),
		SessionType:   "exec",
		ExecutedAt:    time.Now().Format(time.RFC3339),
	}

	// 从上下文获取用户信息（这里需要根据实际的认证系统调整）
	if userInfo := ctx.Value("user"); userInfo != nil {
		// 假设用户信息包含ID和用户名
		// 实际实现需要根据你的认证系统调整
		if user, ok := userInfo.(map[string]interface{}); ok {
			if userId, exists := user["id"]; exists {
				history.UserId = int(userId.(float64))
			}
			if userName, exists := user["username"]; exists {
				history.UserName = userName.(string)
			}
		}
	}

	startTime := time.Now()

	// 执行命令
	stdout, stderr, exitCode, err := s.executeK8sCommand(clientset, config, req)

	executionTime := time.Since(startTime).Seconds()
	history.ExecutionTime = executionTime
	history.ExitCode = exitCode
	history.Stdout = stdout
	history.Stderr = stderr

	if err != nil {
		history.Status = "failed"
		history.ErrorMessage = err.Error()
		s.logger.Error("命令执行失败", zap.Error(err))
	} else {
		history.Status = "success"
	}

	// 保存执行历史
	if err := s.db.Create(history).Error; err != nil {
		s.logger.Error("保存执行历史失败", zap.Error(err))
		// 不返回错误，因为命令执行成功
	}

	response := &model.K8sContainerExecResponse{
		SessionId:     sessionId,
		Stdout:        stdout,
		Stderr:        stderr,
		ExitCode:      exitCode,
		ExecutionTime: executionTime,
	}

	if err != nil {
		return response, fmt.Errorf("命令执行失败: %w", err)
	}

	return response, nil
}

// executeK8sCommand 执行Kubernetes命令的底层实现
func (s *containerExecService) executeK8sCommand(clientset *kubernetes.Clientset, config *rest.Config, req *model.K8sContainerExecRequest) (string, string, int, error) {
	// 构建exec请求
	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec")

	// 设置exec参数
	execReq.VersionedParams(&v1.PodExecOptions{
		Container: req.ContainerName,
		Command:   req.Command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	// 创建executor
	executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
	if err != nil {
		return "", "", -1, fmt.Errorf("创建执行器失败: %w", err)
	}

	// 准备输出缓冲区
	var stdout, stderr bytes.Buffer

	// 执行命令
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	exitCode := 0
	if err != nil {
		// 检查错误类型
		if strings.Contains(err.Error(), "exit status") {
			// 从错误消息中提取退出码
			if strings.Contains(err.Error(), "exit status 1") {
				exitCode = 1
			} else if strings.Contains(err.Error(), "exit status 127") {
				exitCode = 127
			} else {
				exitCode = 1
			}
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// parseLogContent 解析日志内容为结构化日志条目
func (s *containerExecService) parseLogContent(content string, req *model.K8sContainerLogsRequest) []model.K8sContainerLogEntry {
	var logs []model.K8sContainerLogEntry
	lines := strings.Split(content, "\n")
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// 解析日志级别
		level := s.extractLogLevel(line)
		
		// 如果设置了级别过滤，且不匹配则跳过
		if req.Level != "" && level != req.Level {
			continue
		}
		
		// 如果设置了搜索关键词，且不包含则跳过
		if req.Search != "" && !strings.Contains(line, req.Search) {
			continue
		}
		
		logs = append(logs, model.K8sContainerLogEntry{
			Timestamp:     time.Now().Format(time.RFC3339), // 简化处理，实际需要从日志中解析时间戳
			Level:         level,
			Message:       line,
			ContainerName: req.ContainerName,
			PodName:       req.PodName,
			Namespace:     req.Namespace,
		})
	}
	
	return logs
}

// extractLogLevel 从日志行中提取日志级别
func (s *containerExecService) extractLogLevel(line string) string {
	line = strings.ToUpper(line)
	
	if strings.Contains(line, "ERROR") || strings.Contains(line, "ERR") {
		return "ERROR"
	}
	if strings.Contains(line, "WARN") || strings.Contains(line, "WARNING") {
		return "WARN"
	}
	if strings.Contains(line, "INFO") {
		return "INFO"
	}
	if strings.Contains(line, "DEBUG") {
		return "DEBUG"
	}
	
	return "INFO" // 默认级别
}

// CreateTerminalSession 创建终端会话
func (s *containerExecService) CreateTerminalSession(ctx context.Context, containerId string, req *model.K8sContainerTerminalRequest) (*model.K8sContainerTerminalResponse, error) {
	s.logger.Info("创建终端会话",
		zap.String("containerId", containerId),
		zap.Int("clusterId", req.ClusterId),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName))

	// 生成会话ID
	sessionId := uuid.New().String()

	// 创建会话记录
	session := &model.K8sContainerSession{
		SessionId:     sessionId,
		ClusterId:     req.ClusterId,
		Namespace:     req.Namespace,
		PodName:       req.PodName,
		ContainerName: req.ContainerName,
		SessionType:   "terminal",
		Status:        "active",
		StartTime:     time.Now().Format(time.RFC3339),
		LastActivity:  time.Now().Format(time.RFC3339),
		TTY:           req.TTY,
		WorkingDir:    req.WorkingDir,
	}

	// 从上下文获取用户信息
	if userInfo := ctx.Value("user"); userInfo != nil {
		if user, ok := userInfo.(map[string]interface{}); ok {
			if userId, exists := user["id"]; exists {
				session.UserId = int(userId.(float64))
			}
			if userName, exists := user["username"]; exists {
				session.UserName = userName.(string)
			}
		}
	}

	// 保存会话信息
	if err := s.db.Create(session).Error; err != nil {
		s.logger.Error("创建会话记录失败", zap.Error(err))
		return nil, fmt.Errorf("创建会话记录失败: %w", err)
	}

	// 构建WebSocket URL
	websocketURL := fmt.Sprintf("ws://localhost:8080/api/k8s/containers/%s/exec/ws?session=%s&tty=%t",
		containerId, sessionId, req.TTY)

	response := &model.K8sContainerTerminalResponse{
		SessionId:    sessionId,
		WebSocketURL: websocketURL,
	}

	return response, nil
}

// HandleWebSocketTerminal 处理WebSocket终端连接
func (s *containerExecService) HandleWebSocketTerminal(ctx *gin.Context, containerId, sessionId string, tty bool) error {
	s.logger.Info("处理WebSocket终端连接",
		zap.String("containerId", containerId),
		zap.String("sessionId", sessionId),
		zap.Bool("tty", tty))

	// 查找会话信息
	var session model.K8sContainerSession
	if err := s.db.Where("session_id = ?", sessionId).First(&session).Error; err != nil {
		s.logger.Error("会话不存在", zap.Error(err))
		return fmt.Errorf("会话不存在: %w", err)
	}

	// 升级HTTP连接为WebSocket
	conn, err := s.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		s.logger.Error("WebSocket升级失败", zap.Error(err))
		return fmt.Errorf("WebSocket升级失败: %w", err)
	}
	defer conn.Close()

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(context.Background(), session.ClusterId)
	if err != nil {
		s.logger.Error("获取集群配置失败", zap.Error(err))
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		s.logger.Error("创建Kubernetes客户端失败", zap.Error(err))
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 创建终端连接
	return s.handleTerminalSession(conn, clientset, config, &session)
}

// handleTerminalSession 处理终端会话的具体逻辑
func (s *containerExecService) handleTerminalSession(conn *websocket.Conn, clientset *kubernetes.Clientset, config *rest.Config, session *model.K8sContainerSession) error {
	// 构建exec请求
	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(session.PodName).
		Namespace(session.Namespace).
		SubResource("exec")

	// 设置exec参数
	execReq.VersionedParams(&v1.PodExecOptions{
		Container: session.ContainerName,
		Command:   []string{"/bin/bash"}, // 默认使用bash
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       session.TTY,
	}, scheme.ParameterCodec)

	// 创建executor
	executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("创建执行器失败: %w", err)
	}

	// 创建管道
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	// 启动goroutine处理WebSocket消息
	go s.handleWebSocketInput(conn, stdinWriter, session.SessionId)
	go s.handleWebSocketOutput(conn, stdoutReader, "stdout")
	go s.handleWebSocketOutput(conn, stderrReader, "stderr")

	// 执行命令
	err = executor.Stream(remotecommand.StreamOptions{
		Stdin:  stdinReader,
		Stdout: stdoutWriter,
		Stderr: stderrWriter,
		Tty:    session.TTY,
	})

	// 更新会话状态
	s.db.Model(&model.K8sContainerSession{}).
		Where("session_id = ?", session.SessionId).
		Updates(map[string]interface{}{
			"status":   "closed",
			"end_time": time.Now().Format(time.RFC3339),
		})

	return err
}

// handleWebSocketInput 处理WebSocket输入
func (s *containerExecService) handleWebSocketInput(conn *websocket.Conn, stdin io.WriteCloser, sessionId string) {
	defer stdin.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			s.logger.Error("读取WebSocket消息失败", zap.Error(err))
			break
		}

		// 更新最后活动时间
		s.db.Model(&model.K8sContainerSession{}).
			Where("session_id = ?", sessionId).
			Update("last_activity", time.Now().Format(time.RFC3339))

		// 写入标准输入
		if _, err := stdin.Write(message); err != nil {
			s.logger.Error("写入标准输入失败", zap.Error(err))
			break
		}
	}
}

// handleWebSocketOutput 处理WebSocket输出
func (s *containerExecService) handleWebSocketOutput(conn *websocket.Conn, reader io.Reader, outputType string) {
	buffer := make([]byte, 1024)
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err != io.EOF {
				s.logger.Error("读取输出失败", zap.Error(err))
			}
			break
		}

		// 发送到WebSocket
		message := map[string]interface{}{
			"type": outputType,
			"data": string(buffer[:n]),
		}

		if err := conn.WriteJSON(message); err != nil {
			s.logger.Error("发送WebSocket消息失败", zap.Error(err))
			break
		}
	}
}

// GetSessions 获取会话列表
func (s *containerExecService) GetSessions(ctx context.Context, containerId string) ([]model.K8sContainerSession, error) {
	var sessions []model.K8sContainerSession

	// 这里需要根据containerId解析出集群、命名空间、Pod等信息
	// 简化实现，实际需要更复杂的逻辑
	if err := s.db.Where("status = ?", "active").Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("获取会话列表失败: %w", err)
	}

	return sessions, nil
}

// CloseSession 关闭会话
func (s *containerExecService) CloseSession(ctx context.Context, containerId, sessionId string) error {
	s.logger.Info("关闭会话", zap.String("sessionId", sessionId))

	err := s.db.Model(&model.K8sContainerSession{}).
		Where("session_id = ?", sessionId).
		Updates(map[string]interface{}{
			"status":   "closed",
			"end_time": time.Now().Format(time.RFC3339),
		}).Error

	if err != nil {
		return fmt.Errorf("关闭会话失败: %w", err)
	}

	return nil
}

// GetExecutionHistory 获取执行历史
func (s *containerExecService) GetExecutionHistory(ctx context.Context, containerId string, req *model.K8sContainerExecHistoryRequest) (*model.K8sContainerExecHistoryResponse, error) {
	var histories []model.K8sContainerExecHistory
	var total int64

	query := s.db.Model(&model.K8sContainerExecHistory{})

	// 应用过滤条件
	if req.ClusterId > 0 {
		query = query.Where("cluster_id = ?", req.ClusterId)
	}
	if req.Namespace != "" {
		query = query.Where("namespace = ?", req.Namespace)
	}
	if req.PodName != "" {
		query = query.Where("pod_name = ?", req.PodName)
	}
	if req.ContainerName != "" {
		query = query.Where("container_name = ?", req.ContainerName)
	}
	if req.UserId > 0 {
		query = query.Where("user_id = ?", req.UserId)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.StartTime != "" {
		query = query.Where("executed_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		query = query.Where("executed_at <= ?", req.EndTime)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取历史记录总数失败: %w", err)
	}

	// 应用分页
	if req.Limit > 0 {
		query = query.Limit(req.Limit)
	}
	if req.Offset > 0 {
		query = query.Offset(req.Offset)
	}

	// 获取记录
	if err := query.Order("executed_at DESC").Find(&histories).Error; err != nil {
		return nil, fmt.Errorf("获取执行历史失败: %w", err)
	}

	return &model.K8sContainerExecHistoryResponse{
		History:    histories,
		TotalCount: int(total),
	}, nil
}

// createK8sClient 创建Kubernetes客户端
func (s *containerExecService) createK8sClient(cluster *model.K8sCluster) (*kubernetes.Clientset, *rest.Config, error) {
	// 这里需要根据实际的集群配置创建客户端
	// 简化实现，实际需要处理kubeconfig等
	config := &rest.Config{
		Host: cluster.ApiServerAddr, // 使用ApiServerAddr字段
		// 其他配置...
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	return clientset, config, nil
}

// 以下是文件管理和日志管理的其他方法的实现框架
// 由于篇幅限制，这里提供主要方法的实现示例

// GetFiles 获取文件列表
func (s *containerExecService) GetFiles(ctx context.Context, containerId string, req *model.K8sContainerFilesRequest) (*model.K8sContainerFilesResponse, error) {
	s.logger.Info("获取容器文件列表",
		zap.String("containerId", containerId),
		zap.Int("clusterId", req.ClusterId),
		zap.String("path", req.Path))

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, req.ClusterId)
	if err != nil {
		return nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 构建ls命令
	path := req.Path
	if path == "" {
		path = "/"
	}
	
	var command []string
	if req.Recursive {
		command = []string{"find", path, "-ls"}
	} else {
		command = []string{"ls", "-la", path}
	}

	// 执行ls命令获取文件列表
	stdout, stderr, exitCode, err := s.executeFileCommand(clientset, config, req, command)
	if err != nil || exitCode != 0 {
		return nil, fmt.Errorf("执行文件列表命令失败: stdout=%s, stderr=%s, err=%v", stdout, stderr, err)
	}

	// 解析ls命令输出
	files := s.parseFileList(stdout, req.Recursive)

	return &model.K8sContainerFilesResponse{
		Files: files,
	}, nil
}

// executeFileCommand 执行文件相关命令
func (s *containerExecService) executeFileCommand(clientset *kubernetes.Clientset, config *rest.Config, req *model.K8sContainerFilesRequest, command []string) (string, string, int, error) {
	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec")

	execReq.VersionedParams(&v1.PodExecOptions{
		Container: req.ContainerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
	if err != nil {
		return "", "", -1, fmt.Errorf("创建执行器失败: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	exitCode := 0
	if err != nil {
		// 检查错误类型
		if strings.Contains(err.Error(), "exit status") {
			// 从错误消息中提取退出码
			if strings.Contains(err.Error(), "exit status 1") {
				exitCode = 1
			} else if strings.Contains(err.Error(), "exit status 127") {
				exitCode = 127
			} else {
				exitCode = 1
			}
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// parseFileList 解析ls命令输出
func (s *containerExecService) parseFileList(output string, recursive bool) []model.K8sContainerFile {
	var files []model.K8sContainerFile
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" || strings.HasPrefix(line, "total") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		permissions := fields[0]
		sizeStr := fields[4]
		fileName := fields[8]
		
		// 跳过 . 和 .. 目录
		if fileName == "." || fileName == ".." {
			continue
		}

		// 解析文件大小
		size := int64(0)
		if s, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			size = s
		}

		// 判断文件类型
		fileType := "file"
		if strings.HasPrefix(permissions, "d") {
			fileType = "directory"
		} else if strings.HasPrefix(permissions, "l") {
			fileType = "symlink"
		}

		// 构建完整路径
		fullPath := fileName
		if recursive && len(fields) > 9 {
			// find命令输出包含完整路径
			fullPath = strings.Join(fields[10:], " ")
		}

		files = append(files, model.K8sContainerFile{
			Name:        fileName,
			Path:        fullPath,
			Size:        size,
			Type:        fileType,
			Permissions: permissions,
			ModifiedTime: time.Now().Format(time.RFC3339), // 简化处理，实际需要解析时间
		})
	}

	return files
}

// UploadFile 上传文件
func (s *containerExecService) UploadFile(ctx context.Context, containerId string, file multipart.File, header *multipart.FileHeader, path string, overwrite bool) error {
	s.logger.Info("上传文件到容器",
		zap.String("containerId", containerId),
		zap.String("fileName", header.Filename),
		zap.String("path", path),
		zap.Bool("overwrite", overwrite))

	// 解析containerId获取集群和Pod信息
	clusterInfo, err := s.parseContainerId(containerId)
	if err != nil {
		return fmt.Errorf("解析容器ID失败: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, clusterInfo.ClusterId)
	if err != nil {
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 读取文件内容
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("读取上传文件失败: %w", err)
	}

	// 构建目标文件路径
	targetPath := fmt.Sprintf("%s/%s", strings.TrimSuffix(path, "/"), header.Filename)

	// 检查文件是否存在
	if !overwrite {
		checkCmd := []string{"test", "-f", targetPath}
		_, _, exitCode, _ := s.executeFileCommandWithInfo(clientset, config, clusterInfo, checkCmd)
		if exitCode == 0 {
			return fmt.Errorf("文件已存在且不允许覆盖: %s", targetPath)
		}
	}

	// 使用base64编码传输文件内容
	encodedContent := base64.StdEncoding.EncodeToString(fileContent)
	
	// 分块传输大文件
	const chunkSize = 1024 * 1024 // 1MB chunks
	if len(encodedContent) > chunkSize {
		return s.uploadLargeFile(clientset, config, clusterInfo, encodedContent, targetPath)
	}

	// 直接传输小文件
	command := []string{"sh", "-c", fmt.Sprintf("echo '%s' | base64 -d > '%s'", encodedContent, targetPath)}
	_, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, command)
	
	if err != nil || exitCode != 0 {
		return fmt.Errorf("上传文件失败: stderr=%s, err=%v", stderr, err)
	}

	s.logger.Info("文件上传成功", zap.String("targetPath", targetPath))
	return nil
}

// uploadLargeFile 上传大文件
func (s *containerExecService) uploadLargeFile(clientset *kubernetes.Clientset, config *rest.Config, clusterInfo *ContainerInfo, encodedContent, targetPath string) error {
	const chunkSize = 1024 * 1024
	
	// 创建临时文件
	tempFile := fmt.Sprintf("/tmp/upload_%s", uuid.New().String())
	
	// 分块上传
	for i := 0; i < len(encodedContent); i += chunkSize {
		end := i + chunkSize
		if end > len(encodedContent) {
			end = len(encodedContent)
		}
		
		chunk := encodedContent[i:end]
		var command []string
		
		if i == 0 {
			// 第一块：创建文件
			command = []string{"sh", "-c", fmt.Sprintf("echo '%s' > '%s'", chunk, tempFile)}
		} else {
			// 后续块：追加内容
			command = []string{"sh", "-c", fmt.Sprintf("echo '%s' >> '%s'", chunk, tempFile)}
		}
		
		_, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, command)
		if err != nil || exitCode != 0 {
			return fmt.Errorf("上传文件块失败: stderr=%s, err=%v", stderr, err)
		}
	}
	
	// 解码并移动到目标位置
	decodeCmd := []string{"sh", "-c", fmt.Sprintf("base64 -d '%s' > '%s' && rm '%s'", tempFile, targetPath, tempFile)}
	_, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, decodeCmd)
	
	if err != nil || exitCode != 0 {
		return fmt.Errorf("解码文件失败: stderr=%s, err=%v", stderr, err)
	}
	
	return nil
}

// DownloadFile 下载文件
func (s *containerExecService) DownloadFile(ctx *gin.Context, containerId, path string) error {
	s.logger.Info("从容器下载文件",
		zap.String("containerId", containerId),
		zap.String("path", path))

	// 解析containerId获取集群和Pod信息
	clusterInfo, err := s.parseContainerId(containerId)
	if err != nil {
		return fmt.Errorf("解析容器ID失败: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(context.Background(), clusterInfo.ClusterId)
	if err != nil {
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 检查文件是否存在
	checkCmd := []string{"test", "-f", path}
	_, _, exitCode, _ := s.executeFileCommandWithInfo(clientset, config, clusterInfo, checkCmd)
	if exitCode != 0 {
		return fmt.Errorf("文件不存在: %s", path)
	}

	// 获取文件信息
	statCmd := []string{"stat", "-c", "%s", path}
	sizeOutput, _, _, _ := s.executeFileCommandWithInfo(clientset, config, clusterInfo, statCmd)
	fileSize, _ := strconv.ParseInt(strings.TrimSpace(sizeOutput), 10, 64)

	// 使用base64编码读取文件内容
	readCmd := []string{"base64", path}
	stdout, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, readCmd)
	
	if err != nil || exitCode != 0 {
		return fmt.Errorf("读取文件失败: stderr=%s, err=%v", stderr, err)
	}

	// 解码文件内容
	decodedContent, err := base64.StdEncoding.DecodeString(strings.TrimSpace(stdout))
	if err != nil {
		return fmt.Errorf("解码文件内容失败: %w", err)
	}

	// 设置响应头
	fileName := filepath.Base(path)
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Content-Length", fmt.Sprintf("%d", fileSize))

	// 返回文件内容
	ctx.Data(200, "application/octet-stream", decodedContent)
	
	s.logger.Info("文件下载成功", zap.String("fileName", fileName))
	return nil
}

// EditFile 编辑文件
func (s *containerExecService) EditFile(ctx context.Context, containerId string, req *model.K8sContainerFileEditRequest) error {
	s.logger.Info("编辑容器文件",
		zap.String("containerId", containerId),
		zap.String("path", req.Path),
		zap.Bool("backup", req.Backup))

	// 解析containerId获取集群和Pod信息
	clusterInfo, err := s.parseContainerId(containerId)
	if err != nil {
		return fmt.Errorf("解析容器ID失败: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, clusterInfo.ClusterId)
	if err != nil {
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 备份原文件
	if req.Backup {
		backupPath := fmt.Sprintf("%s.backup.%d", req.Path, time.Now().Unix())
		backupCmd := []string{"cp", req.Path, backupPath}
		_, stderr, exitCode, _ := s.executeFileCommandWithInfo(clientset, config, clusterInfo, backupCmd)
		if exitCode != 0 {
			s.logger.Warn("备份文件失败", zap.String("stderr", stderr))
		} else {
			s.logger.Info("文件备份成功", zap.String("backupPath", backupPath))
		}
	}

	// 使用base64编码写入新内容
	encodedContent := base64.StdEncoding.EncodeToString([]byte(req.Content))
	writeCmd := []string{"sh", "-c", fmt.Sprintf("echo '%s' | base64 -d > '%s'", encodedContent, req.Path)}
	
	_, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, writeCmd)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("编辑文件失败: stderr=%s, err=%v", stderr, err)
	}

	s.logger.Info("文件编辑成功", zap.String("path", req.Path))
	return nil
}

// DeleteFile 删除文件
func (s *containerExecService) DeleteFile(ctx context.Context, containerId string, req *model.K8sContainerFileDeleteRequest) error {
	s.logger.Info("删除容器文件",
		zap.String("containerId", containerId),
		zap.String("path", req.Path),
		zap.Bool("recursive", req.Recursive))

	// 解析containerId获取集群和Pod信息
	clusterInfo, err := s.parseContainerId(containerId)
	if err != nil {
		return fmt.Errorf("解析容器ID失败: %w", err)
	}

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, clusterInfo.ClusterId)
	if err != nil {
		return fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, config, err := s.createK8sClient(cluster)
	if err != nil {
		return fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 构建删除命令
	var command []string
	if req.Recursive {
		command = []string{"rm", "-rf", req.Path}
	} else {
		command = []string{"rm", "-f", req.Path}
	}

	_, stderr, exitCode, err := s.executeFileCommandWithInfo(clientset, config, clusterInfo, command)
	if err != nil || exitCode != 0 {
		return fmt.Errorf("删除文件失败: stderr=%s, err=%v", stderr, err)
	}

	s.logger.Info("文件删除成功", zap.String("path", req.Path))
	return nil
}

// GetLogs 获取日志
func (s *containerExecService) GetLogs(ctx context.Context, containerId string, req *model.K8sContainerLogsRequest) (*model.K8sContainerLogsResponse, error) {
	s.logger.Info("获取容器日志",
		zap.String("containerId", containerId),
		zap.Int("clusterId", req.ClusterId),
		zap.String("namespace", req.Namespace),
		zap.String("podName", req.PodName),
		zap.String("containerName", req.ContainerName))

	// 获取集群配置
	cluster, err := s.clusterDao.GetClusterByID(ctx, req.ClusterId)
	if err != nil {
		return nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建Kubernetes客户端
	clientset, _, err := s.createK8sClient(cluster)
	if err != nil {
		return nil, fmt.Errorf("创建Kubernetes客户端失败: %w", err)
	}

	// 构建日志查询选项
	podLogOpts := &v1.PodLogOptions{
		Container: req.ContainerName,
		Follow:    false,
	}

	if req.Tail > 0 {
		tailLines := int64(req.Tail)
		podLogOpts.TailLines = &tailLines
	}

	if req.Since != "" {
		if sinceTime, err := time.Parse(time.RFC3339, req.Since); err == nil {
			podLogOpts.SinceTime = &metav1.Time{Time: sinceTime}
		}
	}

	// 获取日志
	logStream, err := clientset.CoreV1().Pods(req.Namespace).GetLogs(req.PodName, podLogOpts).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取日志流失败: %w", err)
	}
	defer logStream.Close()

	// 读取日志内容
	logContent, err := io.ReadAll(logStream)
	if err != nil {
		return nil, fmt.Errorf("读取日志内容失败: %w", err)
	}

	// 解析日志内容
	logs := s.parseLogContent(string(logContent), req)

	response := &model.K8sContainerLogsResponse{
		Logs:    logs,
		Total:   len(logs),
		HasMore: len(logs) >= req.Tail && req.Tail > 0,
	}

	return response, nil
}

// StreamLogs 流式日志 - 实现框架
func (s *containerExecService) StreamLogs(ctx *gin.Context, containerId string, req *model.K8sContainerLogsRequest) error {
	// TODO: 实现流式日志逻辑
	return nil
}

// SearchLogs 搜索日志 - 实现框架
func (s *containerExecService) SearchLogs(ctx context.Context, containerId string, req *model.K8sContainerLogsRequest) (*model.K8sContainerLogsResponse, error) {
	// TODO: 实现日志搜索逻辑
	return &model.K8sContainerLogsResponse{
		Logs:    []model.K8sContainerLogEntry{},
		Total:   0,
		HasMore: false,
	}, nil
}

// ExportLogs 导出日志 - 实现框架
func (s *containerExecService) ExportLogs(ctx *gin.Context, containerId string, req *model.K8sContainerLogsExportRequest) error {
	// TODO: 实现日志导出逻辑
	return nil
}

// GetLogsHistory 获取日志历史 - 实现框架
func (s *containerExecService) GetLogsHistory(ctx context.Context, containerId string) ([]model.K8sContainerExecHistory, error) {
	// TODO: 实现日志历史获取逻辑
	return []model.K8sContainerExecHistory{}, nil
}

// parseContainerId 解析容器ID获取集群和Pod信息
func (s *containerExecService) parseContainerId(containerId string) (*ContainerInfo, error) {
	// 简化实现：containerId格式假设为 clusterId:namespace:podName:containerName
	// 实际项目中可能需要更复杂的解析逻辑或从数据库查询
	parts := strings.Split(containerId, ":")
	if len(parts) != 4 {
		return nil, fmt.Errorf("无效的容器ID格式: %s, 期望格式: clusterId:namespace:podName:containerName", containerId)
	}

	clusterId, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("无效的集群ID: %s", parts[0])
	}

	return &ContainerInfo{
		ClusterId:     clusterId,
		Namespace:     parts[1],
		PodName:       parts[2],
		ContainerName: parts[3],
	}, nil
}

// executeFileCommandWithInfo 使用ContainerInfo执行文件相关命令
func (s *containerExecService) executeFileCommandWithInfo(clientset *kubernetes.Clientset, config *rest.Config, containerInfo *ContainerInfo, command []string) (string, string, int, error) {
	execReq := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(containerInfo.PodName).
		Namespace(containerInfo.Namespace).
		SubResource("exec")

	execReq.VersionedParams(&v1.PodExecOptions{
		Container: containerInfo.ContainerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", execReq.URL())
	if err != nil {
		return "", "", -1, fmt.Errorf("创建执行器失败: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	exitCode := 0
	if err != nil {
		// 检查错误类型
		if strings.Contains(err.Error(), "exit status") {
			// 从错误消息中提取退出码
			if strings.Contains(err.Error(), "exit status 1") {
				exitCode = 1
			} else if strings.Contains(err.Error(), "exit status 127") {
				exitCode = 127
			} else {
				exitCode = 1
			}
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}
