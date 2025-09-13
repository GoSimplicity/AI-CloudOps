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

// Write 实现io.Writer接口，向WebSocket客户端发送数据
func (t *Session) Write(p []byte) (int, error) {
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

	// 关闭WebSocket连接
	if err := t.conn.Close(); err != nil {
		t.logger.Error("关闭WebSocket连接失败", zap.Error(err))
		return fmt.Errorf("关闭WebSocket连接失败: %w", err)
	}

	t.logger.Debug("终端会话已关闭")
	return nil
}

// Read 实现io.Reader接口，从WebSocket客户端读取数据
func (t *Session) Read(p []byte) (int, error) {
	var msg Message

	// 从WebSocket读取JSON消息
	if err := t.conn.ReadJSON(&msg); err != nil {
		t.logger.Error("从WebSocket读取消息失败", zap.Error(err))
		return copy(p, endOfTransmission), fmt.Errorf("读取WebSocket消息失败: %w", err)
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

	default:
		// 未知消息类型
		t.logger.Error("接收到未知消息类型", zap.String("类型", msg.Op))
		return copy(p, endOfTransmission), fmt.Errorf("未知消息类型: %s", msg.Op)
	}
}

// Next 实现remotecommand.TerminalSizeQueue接口
// 返回下一个终端大小变化，如果通道关闭则返回nil
func (t *Session) Next() *remotecommand.TerminalSize {
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

	// 启动Ping/Pong心跳机制
	go t.startHeartbeat(ctx, conn, cancel)

	// 设置Pong处理器
	t.setupPongHandler(conn)

	// 处理终端会话
	t.handleTerminalSession(ctx, shell, namespace, podName, containerName, conn)
}

// startHeartbeat 启动WebSocket心跳机制
func (t *terminaler) startHeartbeat(ctx context.Context, conn *websocket.Conn, cancel context.CancelFunc) {
	wait.UntilWithContext(ctx, func(ctx context.Context) {
		// 发送Ping消息
		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
			t.logger.Error("发送Ping消息失败", zap.Error(err))
			cancel()         // 取消上下文
			_ = conn.Close() // 关闭WebSocket连接
		}
	}, pingPeriod)
}

// setupPongHandler 设置Pong消息处理器
func (t *terminaler) setupPongHandler(conn *websocket.Conn) {
	// 设置初始读取超时
	conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint

	// 设置Pong消息处理器
	conn.SetPongHandler(func(string) error {
		t.logger.Debug("接收到Pong消息")
		// 更新读取超时
		conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint
		return nil
	})
}

// handleTerminalSession 处理终端会话的核心逻辑
func (t *terminaler) handleTerminalSession(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn) {
	// 创建终端会话
	session := &Session{
		conn:     conn,
		sizeChan: make(chan remotecommand.TerminalSize, 1), // 带缓冲的通道防止阻塞
		logger:   t.logger.With(zap.String("组件", "TerminalSession")),
	}

	// 确保会话清理
	defer func() {
		if err := session.Close(); err != nil {
			t.logger.Error("关闭终端会话失败", zap.Error(err))
		}
	}()

	// 验证并设置Shell命令
	cmd := t.validateAndSetupShell(shell)

	// 执行终端命令
	err := t.executeTerminalCommand(ctx, namespace, podName, containerName, cmd, session)
	if err != nil && !errors.Is(err, context.Canceled) {
		// 发送错误信息到WebSocket客户端
		errorMsg := fmt.Sprintf("终端会话错误: %v", err)
		t.logger.Error("终端会话执行失败", zap.Error(err))

		if writeErr := t.writeErrorMessage(session, errorMsg); writeErr != nil {
			t.logger.Error("发送错误消息失败", zap.Error(writeErr))
		}
	}

	t.logger.Info("终端会话处理完成")
}

// validateAndSetupShell 验证并设置Shell命令
func (t *terminaler) validateAndSetupShell(shell string) []string {
	// 默认使用sh
	cmd := []string{"sh"}

	// 验证shell参数
	if shell != "" && len(shell) <= maxShellLength {
		if checkShell(shell) {
			cmd = []string{shell}
			t.logger.Debug("使用指定Shell", zap.String("shell", shell))
		} else {
			t.logger.Warn("不支持的Shell类型，使用默认sh", zap.String("shell", shell))
		}
	} else if len(shell) > maxShellLength {
		t.logger.Warn("Shell名称过长，使用默认sh",
			zap.String("shell", shell),
			zap.Int("长度", len(shell)),
			zap.Int("最大长度", maxShellLength))
	}

	return cmd
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

// checkShell 检查Shell类型是否受支持
// 支持的Shell类型包括: bash, sh, zsh, fish, ash
func checkShell(shell string) bool {
	// 支持的shell列表
	validShells := []string{"bash", "sh", "zsh", "fish", "ash"}

	for _, validShell := range validShells {
		if validShell == shell {
			return true
		}
	}
	return false
}

// writeErrorMessage 向WebSocket客户端发送错误消息
// 将错误信息格式化为标准的WebSocket消息并发送给客户端
func (t *terminaler) writeErrorMessage(session *Session, message string) error {
	if session == nil {
		return fmt.Errorf("会话为空")
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

	// 设置写入超时
	if err := session.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		t.logger.Error("设置WebSocket写入超时失败", zap.Error(err))
		return fmt.Errorf("设置写入超时失败: %w", err)
	}

	// 发送错误消息到WebSocket
	if err := session.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		t.logger.Error("发送错误消息到WebSocket失败", zap.Error(err))
		return fmt.Errorf("发送错误消息失败: %w", err)
	}

	t.logger.Debug("已发送错误消息到客户端", zap.String("消息", message))
	return nil
}
