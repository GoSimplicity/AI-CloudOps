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

package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	// WebSocket 超时配置
	writeWait         = 10 * time.Second    // WebSocket写入超时
	endOfTransmission = "\u0004"            // 传输结束标志
	pongWait          = 30 * time.Second    // Pong消息等待时间
	pingPeriod        = (pongWait * 9) / 10 // Ping发送间隔（必须小于pongWait）

	// 终端配置
	defaultTerminalRows = 25 // 默认终端行数
	defaultTerminalCols = 80 // 默认终端列数
	maxShellLength      = 50 // Shell名称最大长度
)

// TerminalHandler 定义终端处理接口
type TerminalHandler interface {
	// HandleSession 处理WebSocket终端会话
	HandleSession(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn)
}

// TerminalSessionHandler 终端会话处理器接口
// 组合了io.Reader、io.Writer和终端大小队列接口
type TerminalSessionHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

// Session 终端会话结构体
// 封装了WebSocket连接和终端大小变化通道
type Session struct {
	conn     *websocket.Conn                 // WebSocket连接
	sizeChan chan remotecommand.TerminalSize // 终端大小变化通道
	logger   *zap.Logger                     // 日志记录器
	closed   int32                           // 连接是否已关闭
	mu       sync.RWMutex                    // 读写锁保护连接操作
}

/*
WebSocket 消息协议定义：
 OP      DIRECTION  USED  				DESCRIPTION
 ---------------------------------------------------------------------
 stdin   fe->be     Data           		前端发送的键盘输入/粘贴缓冲区
 resize  fe->be     RowSize, ColSize    前端发送的新终端尺寸
 stdout  be->fe     Data           		后端发送的进程输出
*/
// Message WebSocket消息结构体
// 定义了前后端通信的消息格式
type Message struct {
	Op      string `json:"op"`       // 操作类型: stdin/resize/stdout
	Data    string `json:"data"`     // 消息数据内容
	RowSize uint16 `json:"row_size"` // 终端行数（resize操作使用）
	ColSize uint16 `json:"col_size"` // 终端列数（resize操作使用）
}

// ContainerInfo 容器信息结构体
// 包含容器类型、操作系统等信息，用于优化shell选择
type ContainerInfo struct {
	OS             string   // 操作系统类型: alpine, ubuntu, centos, debian等
	Architecture   string   // 架构: amd64, arm64等
	IsAlpine       bool     // 是否为Alpine Linux
	IsBusyBox      bool     // 是否基于BusyBox
	IsDistroless   bool     // 是否为Distroless镜像
	PackageManager string   // 包管理器: apk, apt, yum等
	ShellFeatures  []string // 容器支持的shell特性
}

// Write 实现io.Writer接口，向WebSocket客户端发送数据
func (t *Session) Write(p []byte) (int, error) {
	// 检查连接是否已关闭
	if atomic.LoadInt32(&t.closed) == 1 {
		return 0, fmt.Errorf("连接已关闭")
	}

	// 空数据直接返回
	if len(p) == 0 {
		return 0, nil
	}

	// 构造stdout消息
	msg, err := json.Marshal(Message{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		t.logger.Error("序列化WebSocket消息失败", zap.Error(err))
		return 0, fmt.Errorf("序列化消息失败: %w", err)
	}

	// 使用读锁保护连接操作
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 再次检查连接状态
	if atomic.LoadInt32(&t.closed) == 1 {
		return 0, fmt.Errorf("连接已关闭")
	}

	// 设置写入超时
	if err := t.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		t.logger.Error("设置WebSocket写入超时失败", zap.Error(err))
		return 0, fmt.Errorf("设置写入超时失败: %w", err)
	}

	// 发送消息
	if err = t.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.logger.Error("向WebSocket发送消息失败", zap.Error(err))
		return 0, fmt.Errorf("发送消息失败: %w", err)
	}

	return len(p), nil
}

// Close 关闭会话，清理资源
func (t *Session) Close() error {
	// 使用原子操作标记连接已关闭，避免重复关闭
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		// 已经关闭过了
		t.logger.Debug("会话已经关闭，跳过重复关闭")
		return nil
	}

	// 使用写锁保护关闭操作
	t.mu.Lock()
	defer t.mu.Unlock()

	// 安全关闭size通道
	defer func() {
		if r := recover(); r != nil {
			t.logger.Error("关闭终端大小通道时发生panic", zap.Any("panic", r))
		}
	}()

	// 关闭通道（可能已经关闭）
	select {
	case <-t.sizeChan:
		// 通道已关闭
	default:
		close(t.sizeChan)
	}

	// 发送关闭帧（graceful关闭）
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "会话结束")
	if err := t.conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
		t.logger.Debug("发送关闭帧失败", zap.Error(err)) // 降级为Debug，因为这在某些情况下是正常的
	}

	// 关闭WebSocket连接
	if err := t.conn.Close(); err != nil {
		// 某些情况下连接可能已经被对端关闭，这是正常的
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			t.logger.Debug("关闭WebSocket连接时出现预期外错误", zap.Error(err))
			return fmt.Errorf("关闭WebSocket连接失败: %w", err)
		}
	}

	t.logger.Debug("终端会话已正常关闭")
	return nil
}

// Read 实现io.Reader接口，从WebSocket客户端读取数据
func (t *Session) Read(p []byte) (int, error) {
	// 检查连接是否已关闭
	if atomic.LoadInt32(&t.closed) == 1 {
		return copy(p, endOfTransmission), io.EOF
	}

	// 使用读锁保护连接操作
	t.mu.RLock()
	defer t.mu.RUnlock()

	// 再次检查连接状态
	if atomic.LoadInt32(&t.closed) == 1 {
		return copy(p, endOfTransmission), io.EOF
	}

	// 尝试读取原始消息
	_, rawMessage, err := t.conn.ReadMessage()
	if err != nil {
		// 检查是否是正常的关闭错误
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			t.logger.Debug("WebSocket连接正常关闭", zap.Error(err))
			return copy(p, endOfTransmission), io.EOF
		}
		t.logger.Error("从WebSocket读取消息失败", zap.Error(err))
		return copy(p, endOfTransmission), fmt.Errorf("读取WebSocket消息失败: %w", err)
	}

	// 空消息处理
	if len(rawMessage) == 0 {
		t.logger.Debug("接收到空消息，忽略")
		return 0, nil
	}

	var msg Message
	// 尝试解析JSON消息
	if err := json.Unmarshal(rawMessage, &msg); err != nil {
		// 如果不是JSON格式，可能是纯文本消息，直接作为stdin处理
		t.logger.Debug("接收到非JSON消息，作为纯文本stdin处理",
			zap.String("消息", string(rawMessage)))
		n := copy(p, rawMessage)
		return n, nil
	}

	// 根据消息类型处理
	switch msg.Op {
	case "stdin":
		// 处理标准输入数据
		n := copy(p, msg.Data)
		t.logger.Debug("接收到标准输入数据", zap.Int("长度", n))
		return n, nil

	case "resize":
		// 处理终端大小调整
		size := remotecommand.TerminalSize{Width: msg.ColSize, Height: msg.RowSize}
		t.logger.Debug("接收到终端大小调整",
			zap.Uint16("宽度", msg.ColSize),
			zap.Uint16("高度", msg.RowSize))

		// 非阻塞发送到大小通道
		select {
		case t.sizeChan <- size:
		default:
			// 通道已满或已关闭，忽略此次调整
			t.logger.Warn("终端大小调整被忽略，通道已满或已关闭")
		}
		return 0, nil

	case "":
		// 空操作类型，可能是心跳或无效消息
		t.logger.Debug("接收到空操作类型消息，忽略")
		return 0, nil

	default:
		// 未知消息类型，但不返回错误，只记录警告
		t.logger.Warn("接收到未知消息类型，忽略", zap.String("类型", msg.Op))
		return 0, nil
	}
}

// Next 实现remotecommand.TerminalSizeQueue接口
// 返回下一个终端大小变化，如果通道关闭则返回nil
func (t *Session) Next() *remotecommand.TerminalSize {
	// 检查连接是否已关闭
	if atomic.LoadInt32(&t.closed) == 1 {
		t.logger.Debug("连接已关闭，返回nil终端大小")
		return nil
	}

	select {
	case size, ok := <-t.sizeChan:
		if !ok {
			// 通道已关闭
			t.logger.Debug("终端大小通道已关闭")
			return nil
		}

		// 验证大小的有效性
		if size.Height == 0 && size.Width == 0 {
			t.logger.Debug("接收到无效的终端大小（0x0）")
			return nil
		}

		// 设置合理的最小值
		if size.Height < 1 {
			size.Height = defaultTerminalRows
		}
		if size.Width < 1 {
			size.Width = defaultTerminalCols
		}

		t.logger.Debug("返回终端大小",
			zap.Uint16("宽度", size.Width),
			zap.Uint16("高度", size.Height))
		return &size
	default:
		// 非阻塞读取，没有新的大小变化
		return nil
	}
}

// terminaler 终端处理器实现
type terminaler struct {
	client kubernetes.Interface // Kubernetes客户端
	config *rest.Config         // Kubernetes配置
	logger *zap.Logger          // 日志记录器
}

// NewTerminalHandler 创建新的终端处理器
// 参数:
//   - client: Kubernetes客户端接口
//   - config: Kubernetes REST配置
//   - logger: 日志记录器
func NewTerminalHandler(client kubernetes.Interface, config *rest.Config, logger *zap.Logger) TerminalHandler {
	if logger == nil {
		// 如果没有提供日志记录器，使用默认的nop logger
		logger = zap.NewNop()
	}

	return &terminaler{
		client: client,
		config: config,
		logger: logger,
	}
}

// HandleSession 处理WebSocket终端会话
// 负责建立和维护WebSocket连接，包括ping/pong心跳检测
func (t *terminaler) HandleSession(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn) {
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 记录会话开始
	t.logger.Info("开始处理终端会话",
		zap.String("命名空间", namespace),
		zap.String("Pod名称", podName),
		zap.String("容器名称", containerName),
		zap.String("Shell类型", shell))

	// 创建终端会话（需要先创建以便心跳机制使用）
	session := &Session{
		conn:     conn,
		sizeChan: make(chan remotecommand.TerminalSize, 1), // 带缓冲的通道防止阻塞
		logger:   t.logger.With(zap.String("组件", "TerminalSession")),
		closed:   0,
	}

	// 启动Ping/Pong心跳机制
	go t.startHeartbeat(ctx, session, cancel)

	// 设置Pong处理器
	t.setupPongHandler(session)

	// 处理终端会话
	t.handleTerminalSession(ctx, shell, namespace, podName, containerName, session)
}

// startHeartbeat 启动WebSocket心跳机制
func (t *terminaler) startHeartbeat(ctx context.Context, session *Session, cancel context.CancelFunc) {
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		// 检查连接是否已关闭
		if atomic.LoadInt32(&session.closed) == 1 {
			t.logger.Debug("连接已关闭，停止心跳")
			cancel() // 取消上下文
			return
		}

		// 使用读锁保护连接操作
		session.mu.RLock()
		defer session.mu.RUnlock()

		// 再次检查连接状态
		if atomic.LoadInt32(&session.closed) == 1 {
			t.logger.Debug("连接已关闭，停止心跳")
			cancel() // 取消上下文
			return
		}

		// 发送Ping消息
		if err := session.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
			// 检查是否是预期的关闭错误
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
				t.logger.Debug("连接已正常关闭，停止心跳", zap.Error(err))
			} else {
				t.logger.Error("发送Ping消息失败", zap.Error(err))
			}
			cancel() // 取消上下文
			return
		}
		t.logger.Debug("发送Ping消息成功")
	}, pingPeriod)
}

// setupPongHandler 设置Pong消息处理器
func (t *terminaler) setupPongHandler(session *Session) {
	// 设置初始读取超时
	session.conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint

	// 设置Pong消息处理器
	session.conn.SetPongHandler(func(string) error {
		// 检查连接是否已关闭
		if atomic.LoadInt32(&session.closed) == 1 {
			t.logger.Debug("连接已关闭，忽略Pong消息")
			return nil
		}

		t.logger.Debug("接收到Pong消息")
		// 使用读锁保护连接操作
		session.mu.RLock()
		defer session.mu.RUnlock()

		// 再次检查连接状态
		if atomic.LoadInt32(&session.closed) == 1 {
			return nil
		}

		// 更新读取超时
		session.conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint
		return nil
	})
}

// handleTerminalSession 处理终端会话的核心逻辑
func (t *terminaler) handleTerminalSession(ctx context.Context, shell, namespace, podName, containerName string, session *Session) {
	// 确保会话清理
	defer func() {
		if err := session.Close(); err != nil {
			t.logger.Error("关闭终端会话失败", zap.Error(err))
		}
	}()

	// 首先检测容器类型和特征
	containerInfo := t.detectContainerInfo(ctx, namespace, podName, containerName)
	t.logger.Debug("检测到容器信息", zap.Any("containerInfo", containerInfo))

	// 检测容器中可用的基本命令
	availableCommands := t.detectAvailableCommands(ctx, namespace, podName, containerName)
	t.logger.Debug("检测到可用命令", zap.Strings("commands", availableCommands))

	// 根据检测结果构建优化的shell fallback列表
	fallbackShells := t.buildOptimizedShellListWithContainerInfo(shell, availableCommands, containerInfo)

	// 如果没有检测到可用命令，记录警告并使用默认列表
	if len(availableCommands) == 0 {
		t.logger.Warn("容器中没有检测到可用的基本命令，将使用默认fallback列表尝试连接")
		// 使用默认shell列表
		fallbackShells = buildShellFallbackList(shell)
	}

	// 尝试执行终端命令，使用fallback机制
	err := t.executeTerminalCommandWithFallback(ctx, namespace, podName, containerName, fallbackShells, session)
	if err != nil && !errors.Is(err, context.Canceled) {
		// 检查连接状态，避免向已关闭连接发送消息
		if atomic.LoadInt32(&session.closed) == 0 {
			// 格式化用户友好的错误消息
			errorMsg := t.formatUserFriendlyError(err, fallbackShells)
			t.logger.Error("终端会话执行失败", zap.Error(err))

			if writeErr := t.writeErrorMessage(session, errorMsg); writeErr != nil {
				t.logger.Error("发送错误消息失败", zap.Error(writeErr))
			}
		} else {
			t.logger.Debug("终端会话执行失败，但连接已关闭，跳过错误消息发送", zap.Error(err))
		}
	}

	t.logger.Info("终端会话处理完成")
}

// validateAndSetupShell 验证并设置Shell命令
// 使用多层fallback机制确保能找到可用的shell
func (t *terminaler) validateAndSetupShell(shell string) []string {
	var preferredShell string

	// 验证shell参数
	if shell != "" && len(shell) <= maxShellLength && isValidShell(shell) {
		preferredShell = shell
		t.logger.Debug("使用指定Shell", zap.String("shell", shell))
	} else if shell != "" {
		if len(shell) > maxShellLength {
			t.logger.Warn("Shell名称过长，使用fallback机制",
				zap.String("shell", shell),
				zap.Int("长度", len(shell)),
				zap.Int("最大长度", maxShellLength))
		} else {
			t.logger.Warn("不支持的Shell类型，使用fallback机制", zap.String("shell", shell))
		}
	}

	// 构建带fallback的shell命令列表
	// 优先级: 用户指定 -> bash -> sh -> /bin/sh -> /bin/bash -> /usr/bin/sh
	fallbackShells := buildShellFallbackList(preferredShell)

	t.logger.Debug("Shell fallback列表", zap.Strings("shells", fallbackShells))

	// 返回第一个shell作为尝试命令，实际会在executeTerminalCommand中处理fallback
	return []string{fallbackShells[0]}
}

// buildShellFallbackListForSession 为会话构建shell fallback列表
func (t *terminaler) buildShellFallbackListForSession(shell string) []string {
	var preferredShell string

	// 验证shell参数
	if shell != "" && len(shell) <= maxShellLength && isValidShell(shell) {
		preferredShell = shell
		t.logger.Debug("使用指定Shell", zap.String("shell", shell))
	} else if shell != "" {
		if len(shell) > maxShellLength {
			t.logger.Warn("Shell名称过长，使用fallback机制",
				zap.String("shell", shell),
				zap.Int("长度", len(shell)),
				zap.Int("最大长度", maxShellLength))
		} else {
			t.logger.Warn("不支持的Shell类型，使用fallback机制", zap.String("shell", shell))
		}
	}

	// 构建shell fallback列表
	fallbackShells := buildShellFallbackList(preferredShell)
	t.logger.Debug("会话Shell fallback列表", zap.Strings("shells", fallbackShells))

	return fallbackShells
}

// executeTerminalCommandWithFallback 使用fallback机制执行终端命令
func (t *terminaler) executeTerminalCommandWithFallback(ctx context.Context, namespace, podName, containerName string, shellList []string, handler TerminalSessionHandler) error {
	var lastErr error

	for i, shell := range shellList {
		t.logger.Debug("尝试执行Shell",
			zap.String("shell", shell),
			zap.Int("尝试次数", i+1),
			zap.Int("总数", len(shellList)))

		// 执行终端命令
		err := t.executeTerminalCommand(ctx, namespace, podName, containerName, []string{shell}, handler)

		if err == nil {
			t.logger.Info("Shell执行成功", zap.String("shell", shell))
			return nil
		}

		// 记录失败原因
		lastErr = err
		t.logger.Warn("Shell执行失败，尝试下一个",
			zap.String("shell", shell),
			zap.Error(err))

		// 如果上下文被取消，立即返回
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			t.logger.Debug("上下文被取消，停止尝试其他Shell")
			return err
		}
	}

	// 所有shell都失败了
	t.logger.Error("所有Shell都执行失败",
		zap.Strings("尝试的shells", shellList),
		zap.Error(lastErr))

	return fmt.Errorf("所有Shell都执行失败，最后一个错误: %w", lastErr)
}

// formatUserFriendlyError 格式化用户友好的错误消息
func (t *terminaler) formatUserFriendlyError(err error, triedShells []string) string {
	errorStr := err.Error()

	// 检查是否为shell不存在错误（退出代码127）
	if strings.Contains(errorStr, "exit code 127") || strings.Contains(errorStr, "command not found") {
		return t.formatShellNotFoundError(triedShells)
	}

	// 检查权限错误
	if strings.Contains(errorStr, "permission denied") || strings.Contains(errorStr, "exit code 126") {
		return t.formatPermissionError(triedShells)
	}

	// 检查连接错误
	if strings.Contains(errorStr, "connection refused") || strings.Contains(errorStr, "dial tcp") {
		return t.formatConnectionError()
	}

	// 检查Pod不存在错误
	if strings.Contains(errorStr, "not found") || strings.Contains(errorStr, "404") {
		return t.formatPodNotFoundError()
	}

	// 检查上下文超时
	if strings.Contains(errorStr, "context deadline exceeded") || strings.Contains(errorStr, "timeout") {
		return t.formatTimeoutError()
	}

	// 检查资源不足
	if strings.Contains(errorStr, "out of memory") || strings.Contains(errorStr, "resource") {
		return t.formatResourceError()
	}

	// 检查容器状态错误
	if strings.Contains(errorStr, "container not running") || strings.Contains(errorStr, "ContainerNotRunning") {
		return t.formatContainerStateError()
	}

	// 检查RBAC权限错误
	if strings.Contains(errorStr, "forbidden") || strings.Contains(errorStr, "403") {
		return t.formatRBACError()
	}

	// 默认错误消息
	return t.formatGenericError(errorStr, triedShells)
}

// formatShellNotFoundError 格式化Shell未找到错误
func (t *terminaler) formatShellNotFoundError(triedShells []string) string {
	return fmt.Sprintf(`容器中未找到可用的Shell程序。

已尝试的Shell: %s

可能的原因：
1. 使用了极简基础镜像（如scratch、distroless、alpine精简版）
2. 容器中的Shell程序被删除或未安装
3. PATH环境变量设置不正确

建议解决方案：
【立即解决】
1. 使用包含基本工具的镜像：
   - 将 FROM scratch 改为 FROM alpine
   - 将 FROM distroless 改为 FROM alpine 或 FROM ubuntu

【Docker镜像修复】
2. 在Dockerfile中添加基本工具：
   Alpine: RUN apk add --no-cache busybox
   Ubuntu: RUN apt-get update && apt-get install -y bash
   CentOS: RUN yum install -y bash

【临时workaround】
3. 尝试使用kubectl exec而不是Web终端：
   kubectl exec -it <pod-name> -- /bin/sh

如需技术支持，请提供Pod的镜像信息给系统管理员。`,
		strings.Join(triedShells, ", "))
}

// formatPermissionError 格式化权限错误
func (t *terminaler) formatPermissionError(triedShells []string) string {
	return fmt.Sprintf(`Shell程序权限不足或无法执行。

已尝试的Shell: %s

可能的原因：
1. 容器以非root用户运行，缺少执行权限
2. SELinux或AppArmor安全策略限制
3. 文件系统只读挂载
4. 容器安全上下文配置过于严格

建议解决方案：
1. 检查Pod的securityContext配置：
   securityContext:
     runAsUser: 0  # 临时使用root用户
     runAsGroup: 0
     
2. 检查文件系统挂载权限：
   kubectl describe pod <pod-name> | grep -A5 "Mounts"
   
3. 验证安全策略：
   kubectl get psp,networkpolicy
   
如需技术支持，请联系系统管理员检查安全策略配置。`,
		strings.Join(triedShells, ", "))
}

// formatConnectionError 格式化连接错误
func (t *terminaler) formatConnectionError() string {
	return `无法连接到Pod容器。

可能的原因：
1. Pod正在重启或启动中
2. 网络策略阻止连接
3. 节点网络问题
4. Kubernetes API Server连接问题

建议解决方案：
1. 检查Pod状态：
   kubectl get pod <pod-name> -o wide
   
2. 查看Pod事件：
   kubectl describe pod <pod-name>
   
3. 检查网络策略：
   kubectl get networkpolicy -A
   
4. 验证节点状态：
   kubectl get nodes
   
请稍后重试，或联系系统管理员检查网络配置。`
}

// formatPodNotFoundError 格式化Pod未找到错误
func (t *terminaler) formatPodNotFoundError() string {
	return `Pod或容器不存在。

可能的原因：
1. Pod名称或容器名称拼写错误
2. Pod已被删除或重新创建
3. 命名空间不正确
4. RBAC权限不足

建议解决方案：
1. 验证Pod是否存在：
   kubectl get pods -A | grep <pod-name>
   
2. 检查正确的命名空间：
   kubectl get pods -n <namespace>
   
3. 查看Pod详细信息：
   kubectl describe pod <pod-name> -n <namespace>
   
4. 检查访问权限：
   kubectl auth can-i get pods --as=<your-user>

请确认Pod名称、容器名称和命名空间是否正确。`
}

// formatTimeoutError 格式化超时错误
func (t *terminaler) formatTimeoutError() string {
	return `连接或操作超时。

可能的原因：
1. 网络延迟过高
2. 容器启动缓慢
3. 系统负载过高
4. 防火墙或代理配置问题

建议解决方案：
1. 检查容器状态：
   kubectl get pod <pod-name> -o wide
   
2. 查看系统负载：
   kubectl top nodes
   kubectl top pods
   
3. 检查网络连接：
   ping <node-ip>
   
4. 稍后重试，或联系管理员检查系统性能

如果问题持续存在，可能需要调整网络超时设置。`
}

// formatResourceError 格式化资源错误
func (t *terminaler) formatResourceError() string {
	return `容器资源不足。

可能的原因：
1. 内存限制过低
2. CPU限制过低
3. 存储空间不足
4. 节点资源耗尽

建议解决方案：
1. 检查资源使用情况：
   kubectl top pod <pod-name>
   kubectl describe pod <pod-name> | grep -A5 "Limits"
   
2. 查看节点资源：
   kubectl top nodes
   kubectl describe node <node-name>
   
3. 调整资源限制：
   resources:
     limits:
       memory: "512Mi"
       cpu: "500m"
     requests:
       memory: "256Mi"
       cpu: "250m"

请联系管理员调整资源配置或扩容集群。`
}

// formatContainerStateError 格式化容器状态错误
func (t *terminaler) formatContainerStateError() string {
	return `容器未运行或状态异常。

可能的原因：
1. 容器正在启动或重启
2. 容器已崩溃或退出
3. 健康检查失败
4. 镜像拉取失败

建议解决方案：
1. 检查容器状态：
   kubectl get pod <pod-name> -o wide
   
2. 查看容器日志：
   kubectl logs <pod-name> -c <container-name>
   
3. 查看Pod事件：
   kubectl describe pod <pod-name>
   
4. 检查健康检查配置：
   livenessProbe和readinessProbe设置
   
等待容器启动完成后重试，或联系管理员检查应用配置。`
}

// formatRBACError 格式化RBAC权限错误
func (t *terminaler) formatRBACError() string {
	return `访问权限不足。

可能的原因：
1. 用户缺少exec权限
2. ServiceAccount配置不正确
3. RBAC策略限制
4. 命名空间访问被拒绝

建议解决方案：
1. 检查用户权限：
   kubectl auth can-i "create" "pods/exec" -n <namespace>
   
2. 查看RBAC配置：
   kubectl get rolebinding,clusterrolebinding -A | grep <user-name>
   
3. 联系管理员申请以下权限：
   - pods/exec 创建权限
   - pods 读取权限
   - 对应命名空间的访问权限

请联系Kubernetes管理员为您分配适当的权限。`
}

// formatGenericError 格式化通用错误
func (t *terminaler) formatGenericError(errorStr string, triedShells []string) string {
	return fmt.Sprintf(`终端会话启动失败。

错误详情: %s
已尝试的Shell: %s

通用解决步骤：
1. 检查Pod状态：kubectl get pod <pod-name> -o wide
2. 查看Pod日志：kubectl logs <pod-name>
3. 检查Pod事件：kubectl describe pod <pod-name>
4. 验证网络连接：ping <pod-ip>
5. 检查用户权限：kubectl auth can-i "*" "pods/exec"

如果问题持续存在，请：
- 记录错误时间和操作步骤
- 收集Pod和节点的详细信息
- 联系系统管理员寻求技术支持

技术支持邮箱：admin@company.com`,
		errorStr, strings.Join(triedShells, ", "))
}

// executeTerminalCommand 执行终端命令，建立与Pod容器的连接
func (t *terminaler) executeTerminalCommand(ctx context.Context, namespace, podName, containerName string, cmd []string, handler TerminalSessionHandler) error {
	// 验证参数
	if namespace == "" {
		return fmt.Errorf("命名空间不能为空")
	}
	if podName == "" {
		return fmt.Errorf("Pod名称不能为空")
	}
	if containerName == "" {
		return fmt.Errorf("容器名称不能为空")
	}
	if len(cmd) == 0 {
		return fmt.Errorf("命令不能为空")
	}

	t.logger.Debug("准备执行终端命令",
		zap.String("命名空间", namespace),
		zap.String("Pod名称", podName),
		zap.String("容器名称", containerName),
		zap.Strings("命令", cmd))

	// 构建exec请求
	req := t.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	// 设置exec选项
	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	// 创建SPDY执行器
	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		t.logger.Error("创建SPDY执行器失败", zap.Error(err))
		return fmt.Errorf("创建SPDY执行器失败: %w", err)
	}

	// 开始流式传输
	t.logger.Debug("开始流式传输")
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	})

	if err != nil {
		t.logger.Error("流式传输失败", zap.Error(err))
		return fmt.Errorf("流式传输失败: %w", err)
	}

	t.logger.Debug("流式传输完成")
	return nil
}

// isValidShell 检查Shell类型是否受支持
// 支持的Shell类型包括: bash, sh, zsh, fish, ash, dash, ksh
func isValidShell(shell string) bool {
	// 支持的shell列表，包括常见的Unix shell
	validShells := []string{"bash", "sh", "zsh", "fish", "ash", "dash", "ksh", "csh", "tcsh"}

	for _, validShell := range validShells {
		if validShell == shell {
			return true
		}
	}
	return false
}

// buildShellFallbackList 构建shell fallback列表
// 按优先级返回可尝试的shell命令列表，覆盖更多容器类型
func buildShellFallbackList(preferredShell string) []string {
	var fallbackList []string

	// 1. 用户指定的shell（如果有效）
	if preferredShell != "" {
		fallbackList = append(fallbackList, preferredShell)
	}

	// 2. 常用shell（相对路径）- 按实用性排序
	commonShells := []string{"sh", "bash", "ash", "dash", "busybox"}
	for _, shell := range commonShells {
		if shell != preferredShell { // 避免重复
			fallbackList = append(fallbackList, shell)
		}
	}

	// 3. 标准路径shell（/bin目录）
	binShells := []string{"/bin/sh", "/bin/bash", "/bin/ash", "/bin/dash", "/bin/busybox"}
	for _, shell := range binShells {
		shellName := strings.TrimPrefix(shell, "/bin/")
		if shellName != preferredShell { // 避免重复
			fallbackList = append(fallbackList, shell)
		}
	}

	// 4. 系统路径shell（/usr/bin目录）
	usrBinShells := []string{"/usr/bin/sh", "/usr/bin/bash", "/usr/bin/ash", "/usr/bin/dash"}
	for _, shell := range usrBinShells {
		shellName := strings.TrimPrefix(shell, "/usr/bin/")
		if shellName != preferredShell { // 避免重复
			fallbackList = append(fallbackList, shell)
		}
	}

	// 5. Alpine Linux 和精简容器特殊路径
	alpineShells := []string{"/sbin/sh", "/system/bin/sh", "/usr/local/bin/sh"}
	for _, shell := range alpineShells {
		fallbackList = append(fallbackList, shell)
	}

	// 6. BusyBox特殊命令（针对极简容器）
	busyboxCommands := []string{"busybox sh", "/bin/busybox sh", "/usr/bin/busybox sh"}
	fallbackList = append(fallbackList, busyboxCommands...)

	// 7. 基本命令fallback（作为最后手段）
	basicCommands := []string{"cat", "/bin/cat", "/usr/bin/cat", "echo", "/bin/echo"}
	fallbackList = append(fallbackList, basicCommands...)

	// 8. 最后的fallback - 确保至少有一个基本选项
	if len(fallbackList) == 0 {
		fallbackList = []string{"sh", "/bin/sh", "cat"}
	}

	return fallbackList
}

// checkShell 兼容性函数，保持向后兼容
// 已弃用: 请使用 isValidShell
func checkShell(shell string) bool {
	return isValidShell(shell)
}

// writeErrorMessage 向WebSocket客户端发送错误消息
// 将错误信息格式化为标准的WebSocket消息并发送给客户端
func (t *terminaler) writeErrorMessage(session *Session, message string) error {
	if session == nil {
		return fmt.Errorf("会话为空")
	}

	// 检查连接是否已关闭
	if atomic.LoadInt32(&session.closed) == 1 {
		t.logger.Debug("连接已关闭，跳过错误消息发送")
		return fmt.Errorf("连接已关闭")
	}

	// 构造错误消息
	errorMsg := Message{
		Op:   "stdout",
		Data: fmt.Sprintf("\r\n错误: %s\r\n", message),
	}

	// 序列化消息
	msgBytes, err := json.Marshal(errorMsg)
	if err != nil {
		t.logger.Error("序列化错误消息失败", zap.Error(err))
		return fmt.Errorf("序列化错误消息失败: %w", err)
	}

	// 使用读锁保护连接操作
	session.mu.RLock()
	defer session.mu.RUnlock()

	// 再次检查连接状态
	if atomic.LoadInt32(&session.closed) == 1 {
		t.logger.Debug("连接已关闭，跳过错误消息发送")
		return fmt.Errorf("连接已关闭")
	}

	// 设置较短的写入超时（错误消息优先级较低）
	shortWriteTimeout := time.Second * 3
	if err := session.conn.SetWriteDeadline(time.Now().Add(shortWriteTimeout)); err != nil {
		t.logger.Debug("设置WebSocket写入超时失败", zap.Error(err)) // 降级为Debug
		return fmt.Errorf("设置写入超时失败: %w", err)
	}

	// 发送错误消息到WebSocket（可能失败，但不应该阻塞整个流程）
	if err := session.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		// 检查是否是预期的关闭错误
		if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			t.logger.Debug("连接已正常关闭，无法发送错误消息", zap.Error(err))
		} else {
			t.logger.Debug("发送错误消息到WebSocket失败", zap.Error(err)) // 降级为Debug，因为这在某些情况下是正常的
		}
		return fmt.Errorf("发送错误消息失败: %w", err)
	}

	t.logger.Debug("已发送错误消息到客户端", zap.String("消息", message))
	return nil
}

// detectAvailableCommands 检测容器中可用的基本命令
// 返回可用命令列表，用于优化shell fallback策略
func (t *terminaler) detectAvailableCommands(ctx context.Context, namespace, podName, containerName string) []string {
	t.logger.Debug("开始检测容器中的可用命令",
		zap.String("namespace", namespace),
		zap.String("podName", podName),
		zap.String("containerName", containerName))

	// 要检测的基本命令列表（按优先级排序）
	commandsToTest := []string{
		"sh", "bash", "ash", "dash", "busybox",
		"/bin/sh", "/bin/bash", "/bin/ash", "/bin/dash", "/bin/busybox",
		"/usr/bin/sh", "/usr/bin/bash", "/sbin/sh",
		"cat", "/bin/cat", "/usr/bin/cat",
		"echo", "/bin/echo", "/usr/bin/echo",
	}

	var availableCommands []string

	// 创建一个短超时的上下文，避免检测过程过长
	detectCtx, cancel := context.WithTimeout(ctx, 10*time.Second) // 增加超时时间
	defer cancel()

	t.logger.Debug("开始检测命令", zap.Int("总命令数", len(commandsToTest)))

	for i, cmd := range commandsToTest {
		t.logger.Debug("检测命令", zap.String("命令", cmd), zap.Int("进度", i+1), zap.Int("总数", len(commandsToTest)))

		if t.testCommandExists(detectCtx, namespace, podName, containerName, cmd) {
			availableCommands = append(availableCommands, cmd)
			t.logger.Debug("找到可用命令", zap.String("命令", cmd), zap.Int("已找到", len(availableCommands)))
			// 为了提高效率，找到一定数量的命令后就停止检测
			if len(availableCommands) >= 10 {
				t.logger.Debug("已找到足够的命令，停止检测", zap.Int("找到数量", len(availableCommands)))
				break
			}
		}
	}

	t.logger.Info("命令检测完成",
		zap.Int("检测到的命令数", len(availableCommands)),
		zap.Strings("可用命令", availableCommands))

	return availableCommands
}

// testCommandExists 测试指定命令是否在容器中存在
func (t *terminaler) testCommandExists(ctx context.Context, namespace, podName, containerName, cmd string) bool {
	// 使用更简单和直接的方式测试命令存在性
	testCommands := [][]string{
		// 优先使用最基本的测试方法
		{"ls", "-la", cmd},           // 直接检查文件是否存在
		{"test", "-f", cmd},          // 检查文件是否存在且为普通文件
		{"test", "-x", cmd},          // 检查文件是否存在且可执行
		{"which", cmd},               // 查找命令路径
		{"command", "-v", cmd},       // 检查命令是否可用
		{"/bin/ls", "-la", cmd},      // 使用绝对路径的ls
		{"/usr/bin/test", "-f", cmd}, // 使用绝对路径的test
	}

	for _, testCmd := range testCommands {
		if t.executeQuickTest(ctx, namespace, podName, containerName, testCmd) {
			t.logger.Debug("检测到可用命令", zap.String("命令", cmd), zap.Strings("测试命令", testCmd))
			return true
		}
	}

	// 作为最后的尝试，直接执行命令看是否存在
	// 对于shell命令，尝试执行一个简单的操作
	if strings.Contains(cmd, "sh") || strings.Contains(cmd, "bash") {
		if t.executeQuickTest(ctx, namespace, podName, containerName, []string{cmd, "-c", "echo test"}) {
			t.logger.Debug("通过直接执行检测到shell命令", zap.String("命令", cmd))
			return true
		}
	}

	return false
}

// executeQuickTest 执行快速测试命令
func (t *terminaler) executeQuickTest(ctx context.Context, namespace, podName, containerName string, cmd []string) bool {
	// 创建更短的超时上下文
	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// 构建exec请求
	req := t.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	// 设置exec选项 - 不使用TTY，仅获取退出状态
	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     false,
		Stdout:    false,
		Stderr:    false,
		TTY:       false,
	}, scheme.ParameterCodec)

	// 创建SPDY执行器
	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		t.logger.Debug("创建测试SPDY执行器失败", zap.Error(err), zap.Strings("命令", cmd))
		return false
	}

	// 执行测试命令
	err = exec.StreamWithContext(testCtx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: nil,
		Stderr: nil,
		Tty:    false,
	})

	// 如果命令成功执行（退出码0），则认为命令存在
	if err == nil {
		t.logger.Debug("命令测试成功", zap.Strings("测试命令", cmd))
		return true
	}

	// 检查错误类型，某些错误代码可能表示命令存在但参数不正确
	errorStr := err.Error()
	t.logger.Debug("命令测试结果", zap.Strings("测试命令", cmd), zap.String("错误", errorStr))

	if strings.Contains(errorStr, "exit code 1") ||
		strings.Contains(errorStr, "exit code 2") ||
		strings.Contains(errorStr, "invalid option") ||
		strings.Contains(errorStr, "usage:") {
		// 这些错误通常表示命令存在但使用不当
		t.logger.Debug("命令存在但使用不当，认为命令可用", zap.Strings("测试命令", cmd))
		return true
	}

	return false
}

// buildOptimizedShellList 构建优化的shell列表
func (t *terminaler) buildOptimizedShellList(preferredShell string, availableCommands []string) []string {
	var optimizedList []string
	commandSet := make(map[string]bool)

	// 转换为map便于快速查找
	for _, cmd := range availableCommands {
		commandSet[cmd] = true
	}

	// 1. 用户首选shell（如果可用）
	if preferredShell != "" && commandSet[preferredShell] {
		optimizedList = append(optimizedList, preferredShell)
	}

	// 2. 按优先级选择可用shell
	preferredOrder := []string{"bash", "sh", "ash", "dash", "/bin/bash", "/bin/sh", "/bin/ash", "/bin/dash", "/usr/bin/bash", "/usr/bin/sh", "busybox", "/bin/busybox"}

	for _, shell := range preferredOrder {
		if commandSet[shell] && !contains(optimizedList, shell) {
			optimizedList = append(optimizedList, shell)
		}
	}

	// 3. 添加busybox变体
	if commandSet["busybox"] || commandSet["/bin/busybox"] || commandSet["/usr/bin/busybox"] {
		busyboxVariants := []string{"busybox sh", "/bin/busybox sh", "/usr/bin/busybox sh"}
		for _, variant := range busyboxVariants {
			if !contains(optimizedList, variant) {
				optimizedList = append(optimizedList, variant)
			}
		}
	}

	// 4. 没有shell时尝试基本命令
	if len(optimizedList) == 0 {
		basicCommands := []string{"cat", "/bin/cat", "/usr/bin/cat", "echo", "/bin/echo", "/usr/bin/echo"}
		for _, cmd := range basicCommands {
			if commandSet[cmd] {
				optimizedList = append(optimizedList, cmd)
			}
		}
	}

	// 5. 最后使用默认fallback
	if len(optimizedList) == 0 {
		t.logger.Warn("没有检测到可用命令，使用默认fallback列表")
		return buildShellFallbackList(preferredShell)
	}

	t.logger.Debug("构建优化的shell列表完成", zap.Strings("shells", optimizedList))
	return optimizedList
}

// formatNoCommandsAvailableError 格式化无可用命令错误信息
func (t *terminaler) formatNoCommandsAvailableError() string {
	return `容器中没有检测到任何可用的基本命令。

可能的原因：
1. 容器使用了极简的基础镜像（如scratch、distroless）
2. 容器的PATH环境变量配置不正确  
3. 容器的文件系统权限配置过于严格
4. 容器正在启动过程中，基本工具尚未就绪

建议解决方案：
1. 使用包含基本Shell的镜像（如alpine、ubuntu、busybox）
2. 在Dockerfile中安装基本工具：RUN apk add --no-cache busybox 或 RUN apt-get update && apt-get install -y bash
3. 检查容器的启动状态和健康检查
4. 验证容器的运行用户权限

如需技术支持，请联系系统管理员。`
}

// contains 检查字符串切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// detectContainerInfo 检测容器信息和特征
// 通过执行基本的系统检测命令来识别容器类型
func (t *terminaler) detectContainerInfo(ctx context.Context, namespace, podName, containerName string) ContainerInfo {
	info := ContainerInfo{
		OS:             "unknown",
		Architecture:   "unknown",
		IsAlpine:       false,
		IsBusyBox:      false,
		IsDistroless:   false,
		PackageManager: "unknown",
		ShellFeatures:  []string{},
	}

	// 创建短超时上下文用于检测
	detectCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// 检测操作系统类型
	t.detectOS(detectCtx, namespace, podName, containerName, &info)

	// 检测架构
	t.detectArchitecture(detectCtx, namespace, podName, containerName, &info)

	// 检测包管理器
	t.detectPackageManager(detectCtx, namespace, podName, containerName, &info)

	// 检测特殊特征
	t.detectSpecialFeatures(detectCtx, namespace, podName, containerName, &info)

	t.logger.Debug("容器信息检测完成",
		zap.String("OS", info.OS),
		zap.String("架构", info.Architecture),
		zap.Bool("Alpine", info.IsAlpine),
		zap.Bool("BusyBox", info.IsBusyBox),
		zap.Bool("Distroless", info.IsDistroless),
		zap.String("包管理器", info.PackageManager))

	return info
}

// detectOS 检测操作系统类型
func (t *terminaler) detectOS(ctx context.Context, namespace, podName, containerName string, info *ContainerInfo) {
	// 检测Alpine Linux
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/alpine-release"}) {
		info.OS = "alpine"
		info.IsAlpine = true
		return
	}

	// 检测Ubuntu/Debian
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/debian_version"}) {
		if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/lsb-release"}) {
			info.OS = "ubuntu"
		} else {
			info.OS = "debian"
		}
		return
	}

	// 检测CentOS/RHEL
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/centos-release"}) {
		info.OS = "centos"
		return
	}
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/redhat-release"}) {
		info.OS = "rhel"
		return
	}

	// 检测其他发行版
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"cat", "/etc/os-release"}) {
		// 可以进一步解析os-release文件内容
		info.OS = "linux"
	}
}

// detectArchitecture 检测系统架构
func (t *terminaler) detectArchitecture(ctx context.Context, namespace, podName, containerName string, info *ContainerInfo) {
	// 尝试检测架构
	archCommands := [][]string{
		{"uname", "-m"},
		{"arch"},
		{"dpkg", "--print-architecture"},
	}

	for _, cmd := range archCommands {
		if t.executeSimpleTest(ctx, namespace, podName, containerName, cmd) {
			// 这里可以进一步解析输出来确定确切的架构
			info.Architecture = "detected"
			return
		}
	}
}

// detectPackageManager 检测包管理器
func (t *terminaler) detectPackageManager(ctx context.Context, namespace, podName, containerName string, info *ContainerInfo) {
	packageManagers := map[string][]string{
		"apk":    {"apk", "--version"},
		"apt":    {"apt", "--version"},
		"yum":    {"yum", "--version"},
		"dnf":    {"dnf", "--version"},
		"pacman": {"pacman", "--version"},
	}

	for pm, cmd := range packageManagers {
		if t.executeSimpleTest(ctx, namespace, podName, containerName, cmd) {
			info.PackageManager = pm
			return
		}
	}
}

// detectSpecialFeatures 检测特殊特征
func (t *terminaler) detectSpecialFeatures(ctx context.Context, namespace, podName, containerName string, info *ContainerInfo) {
	// 检测BusyBox
	if t.executeSimpleTest(ctx, namespace, podName, containerName, []string{"busybox", "--help"}) {
		info.IsBusyBox = true
		info.ShellFeatures = append(info.ShellFeatures, "busybox")
	}

	// 检测是否为Distroless（通常没有shell）
	hasBasicCommands := false
	basicCommands := [][]string{
		{"ls", "/"},
		{"cat", "/etc/passwd"},
		{"echo", "test"},
	}

	for _, cmd := range basicCommands {
		if t.executeSimpleTest(ctx, namespace, podName, containerName, cmd) {
			hasBasicCommands = true
			break
		}
	}

	if !hasBasicCommands {
		info.IsDistroless = true
	}

	// 检测可用的shell特性
	shellTests := map[string][]string{
		"bash_completion": {"bash", "-c", "type complete"},
		"zsh":             {"zsh", "--version"},
		"fish":            {"fish", "--version"},
		"ash":             {"ash", "-c", "echo test"},
	}

	for feature, cmd := range shellTests {
		if t.executeSimpleTest(ctx, namespace, podName, containerName, cmd) {
			info.ShellFeatures = append(info.ShellFeatures, feature)
		}
	}
}

// executeSimpleTest 执行简单的测试命令
func (t *terminaler) executeSimpleTest(ctx context.Context, namespace, podName, containerName string, cmd []string) bool {
	// 创建更短的超时
	testCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// 构建exec请求
	req := t.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	// 设置exec选项
	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     false,
		Stdout:    false,
		Stderr:    false,
		TTY:       false,
	}, scheme.ParameterCodec)

	// 创建SPDY执行器
	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		return false
	}

	// 执行命令
	err = exec.StreamWithContext(testCtx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: nil,
		Stderr: nil,
		Tty:    false,
	})

	return err == nil
}

// buildOptimizedShellListWithContainerInfo 根据容器信息构建优化的shell列表
func (t *terminaler) buildOptimizedShellListWithContainerInfo(preferredShell string, availableCommands []string, containerInfo ContainerInfo) []string {
	var optimizedList []string
	commandSet := make(map[string]bool)

	// 将可用命令转换为map
	for _, cmd := range availableCommands {
		commandSet[cmd] = true
	}

	// 1. 用户首选shell（如果可用）
	if preferredShell != "" && commandSet[preferredShell] {
		optimizedList = append(optimizedList, preferredShell)
	}

	// 2. 根据容器类型优化shell顺序
	switch {
	case containerInfo.IsAlpine:
		optimizedList = append(optimizedList, t.getAlpineOptimizedShells(commandSet, preferredShell)...)

	case containerInfo.IsBusyBox:
		optimizedList = append(optimizedList, t.getBusyBoxOptimizedShells(commandSet, preferredShell)...)

	case containerInfo.IsDistroless:
		optimizedList = append(optimizedList, t.getDistrolessOptimizedShells(commandSet, preferredShell)...)

	case containerInfo.OS == "ubuntu" || containerInfo.OS == "debian":
		optimizedList = append(optimizedList, t.getDebianOptimizedShells(commandSet, preferredShell)...)

	case containerInfo.OS == "centos" || containerInfo.OS == "rhel":
		optimizedList = append(optimizedList, t.getRHELOptimizedShells(commandSet, preferredShell)...)

	default:
		// 通用fallback
		optimizedList = append(optimizedList, t.getGenericOptimizedShells(commandSet, preferredShell)...)
	}

	// 3. 如果还是没有找到可用shell，使用原始方法
	if len(optimizedList) == 0 {
		t.logger.Warn("基于容器信息未找到优化shell，使用原始fallback")
		return t.buildOptimizedShellList(preferredShell, availableCommands)
	}

	// 去重
	uniqueList := t.removeDuplicates(optimizedList)

	t.logger.Debug("基于容器信息构建的优化shell列表",
		zap.Strings("shells", uniqueList),
		zap.String("容器类型", containerInfo.OS))

	return uniqueList
}

// getAlpineOptimizedShells 获取Alpine Linux优化的shell列表
func (t *terminaler) getAlpineOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	alpineShells := []string{
		"ash", // Alpine默认shell
		"/bin/ash",
		"sh", // 通常指向ash
		"/bin/sh",
		"busybox sh", // BusyBox的sh
		"/bin/busybox sh",
		"busybox ash", // BusyBox的ash
		"/bin/busybox ash",
		"bash", // 如果安装了bash
		"/bin/bash",
	}

	var available []string
	for _, shell := range alpineShells {
		if shell != preferredShell && commandSet[shell] {
			available = append(available, shell)
		}
	}
	return available
}

// getBusyBoxOptimizedShells 获取BusyBox优化的shell列表
func (t *terminaler) getBusyBoxOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	busyboxShells := []string{
		"busybox sh",
		"/bin/busybox sh",
		"busybox ash",
		"/bin/busybox ash",
		"ash",
		"/bin/ash",
		"sh",
		"/bin/sh",
	}

	var available []string
	for _, shell := range busyboxShells {
		if shell != preferredShell && commandSet[shell] {
			available = append(available, shell)
		}
	}
	return available
}

// getDistrolessOptimizedShells 获取Distroless优化的shell列表
func (t *terminaler) getDistrolessOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	// Distroless镜像通常没有shell，尝试基本命令
	basicCommands := []string{
		"cat",
		"/bin/cat",
		"/usr/bin/cat",
		"echo",
		"/bin/echo",
		"/usr/bin/echo",
	}

	var available []string
	for _, cmd := range basicCommands {
		if commandSet[cmd] {
			available = append(available, cmd)
		}
	}
	return available
}

// getDebianOptimizedShells 获取Debian/Ubuntu优化的shell列表
func (t *terminaler) getDebianOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	debianShells := []string{
		"bash", // Debian/Ubuntu默认
		"/bin/bash",
		"sh",
		"/bin/sh",
		"dash", // Ubuntu中sh通常指向dash
		"/bin/dash",
		"/usr/bin/bash",
	}

	var available []string
	for _, shell := range debianShells {
		if shell != preferredShell && commandSet[shell] {
			available = append(available, shell)
		}
	}
	return available
}

// getRHELOptimizedShells 获取RHEL/CentOS优化的shell列表
func (t *terminaler) getRHELOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	rhelShells := []string{
		"bash",
		"/bin/bash",
		"/usr/bin/bash",
		"sh",
		"/bin/sh",
		"zsh", // 有时会安装zsh
		"/bin/zsh",
	}

	var available []string
	for _, shell := range rhelShells {
		if shell != preferredShell && commandSet[shell] {
			available = append(available, shell)
		}
	}
	return available
}

// getGenericOptimizedShells 获取通用优化的shell列表
func (t *terminaler) getGenericOptimizedShells(commandSet map[string]bool, preferredShell string) []string {
	genericShells := []string{
		"bash", "sh", "ash", "dash",
		"/bin/bash", "/bin/sh", "/bin/ash", "/bin/dash",
		"/usr/bin/bash", "/usr/bin/sh",
		"busybox sh", "/bin/busybox sh",
	}

	var available []string
	for _, shell := range genericShells {
		if shell != preferredShell && commandSet[shell] {
			available = append(available, shell)
		}
	}
	return available
}

// removeDuplicates 移除重复项
func (t *terminaler) removeDuplicates(input []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range input {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
}
