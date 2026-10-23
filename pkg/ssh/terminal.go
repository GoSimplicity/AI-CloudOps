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

package ssh

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// TerminalReader 实现从WebSocket读取数据的io.Reader接口
type TerminalReader struct {
	conn         *websocket.Conn
	logger       *zap.Logger
	onDisconnect context.CancelFunc
}

// TerminalWriter 实现向WebSocket写入数据的io.Writer接口
type TerminalWriter struct {
	mu     sync.Mutex
	conn   *websocket.Conn
	logger *zap.Logger
}

// Read 从WebSocket读取用户输入的命令数据
func (r *TerminalReader) Read(p []byte) (n int, err error) {
	// 从WebSocket客户端接收消息
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		r.logger.Error("从WebSocket读取消息失败", zap.Error(err))
		if r.onDisconnect != nil {
			r.onDisconnect()
		}
		return 0, fmt.Errorf("从WebSocket读取消息失败: %w", err)
	}

	// 将接收到的消息转换为命令字符串
	cmdStr := string(message)

	// 确保命令以换行符结尾（终端需要）
	if !strings.HasSuffix(cmdStr, "\n") {
		cmdStr = cmdStr + "\n"
	}

	// 将命令复制到读取缓冲区
	cmdBytes := []byte(cmdStr)
	n = copy(p, cmdBytes)

	return n, nil
}

// Write 向WebSocket发送终端输出数据
func (w *TerminalWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(p) == 0 {
		return 0, nil
	}

	// 向WebSocket客户端发送文本消息
	err = w.conn.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		w.logger.Error("向WebSocket写入消息失败", zap.Error(err))
		return 0, fmt.Errorf("向WebSocket写入消息失败: %w", err)
	}

	return len(p), nil
}

// WebTerminal 提供Web终端功能
func (c *client) WebTerminal(userID int, conn *websocket.Conn) error {
	return c.WebTerminalContext(context.Background(), userID, conn)
}

// WebTerminalContext 只中断当前连接捕获的传输，最终Close由会话所有者执行
func (c *client) WebTerminalContext(parent context.Context, userID int, conn *websocket.Conn) error {
	if parent == nil || conn == nil {
		return fmt.Errorf("终端上下文与WebSocket不能为空")
	}
	if err := parent.Err(); err != nil {
		return err
	}
	transport := c.sshClient
	if transport == nil {
		return fmt.Errorf("SSH客户端未连接")
	}
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var setupExpired atomic.Bool
	setupTimer := time.AfterFunc(10*time.Second, func() { setupExpired.Store(true); cancel() })
	defer setupTimer.Stop()
	interrupted := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(interrupted)
		_ = transport.Close()
		_ = conn.Close()
	})
	defer func() {
		if !stop() {
			<-interrupted
		}
	}()
	err := c.webTerminal(userID, conn, cancel, func() { setupTimer.Stop() })
	if parent.Err() != nil {
		return parent.Err()
	}
	if setupExpired.Load() {
		return fmt.Errorf("SSH终端初始化超时: %w", context.DeadlineExceeded)
	}
	return err
}

func (c *client) webTerminal(userID int, conn *websocket.Conn, onDisconnect context.CancelFunc, onStarted func()) error {
	if conn == nil {
		return fmt.Errorf("WebSocket连接不能为空")
	}

	// 确保在函数结束时清理所有资源
	defer func() {
		// 关闭WebSocket连接
		if err := conn.Close(); err != nil {
			c.logger.Error("关闭WebSocket连接失败", zap.Error(err))
		}

		// 关闭SSH会话
		if err := c.CloseSession(userID); err != nil {
			c.logger.Error("关闭SSH会话失败", zap.Int("用户ID", userID), zap.Error(err))
		}

		c.logger.Info("Web终端SSH会话已清理", zap.Int("用户ID", userID))
	}()

	// 获取或创建SSH会话
	session := c.GetSession(userID)
	if session == nil {
		if err := c.CreateSession(userID); err != nil {
			errMsg := "创建SSH会话失败"
			c.logger.Error(errMsg, zap.Int("用户ID", userID), zap.Error(err))
			if writeErr := conn.WriteMessage(websocket.TextMessage, []byte(errMsg+": "+err.Error()+"\r\n")); writeErr != nil {
				c.logger.Error("向WebSocket发送错误消息失败", zap.Error(writeErr))
			}
			return fmt.Errorf("%s: %w", errMsg, err)
		}
		session = c.GetSession(userID)
	}

	if session == nil {
		errMsg := "SSH会话未建立"
		c.logger.Error(errMsg, zap.Int("用户ID", userID))
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte("错误: "+errMsg+"\r\n")); writeErr != nil {
			c.logger.Error("向WebSocket发送错误消息失败", zap.Error(writeErr))
		}
		return fmt.Errorf("%s", errMsg)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显（避免重复显示用户输入）
		ssh.TTY_OP_ISPEED: 14400, // 输入波特率
		ssh.TTY_OP_OSPEED: 14400, // 输出波特率
	}

	// 请求伪终端（PTY）
	if err := session.RequestPty("xterm", 25, 80, modes); err != nil {
		errMsg := "请求伪终端失败"
		c.logger.Error(errMsg, zap.Error(err))
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte("错误: 无法创建终端\r\n")); writeErr != nil {
			c.logger.Error("向WebSocket发送错误消息失败", zap.Error(writeErr))
		}
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	reader := &TerminalReader{conn: conn, logger: c.logger, onDisconnect: onDisconnect}
	writer := &TerminalWriter{conn: conn, logger: c.logger}

	// 将WebSocket连接设置为SSH会话的输入输出流
	session.Stdin = reader  // 标准输入：从WebSocket读取用户命令
	session.Stdout = writer // 标准输出：向WebSocket发送命令结果
	session.Stderr = writer // 错误输出：向WebSocket发送错误信息

	// 启动交互式Shell
	if err := session.Shell(); err != nil {
		errMsg := "启动Shell失败"
		c.logger.Error(errMsg, zap.Error(err))
		if writeErr := conn.WriteMessage(websocket.TextMessage, []byte("错误: 无法启动Shell\r\n")); writeErr != nil {
			c.logger.Error("向WebSocket发送错误消息失败", zap.Error(writeErr))
		}
		return fmt.Errorf("%s: %w", errMsg, err)
	}

	onStarted()
	c.logger.Info("Web终端SSH会话已启动", zap.Int("用户ID", userID))

	// 等待SSH会话结束（阻塞直到用户退出或连接断开）
	if err := session.Wait(); err != nil {
		c.logger.Info("SSH会话结束", zap.Int("用户ID", userID), zap.Error(err))
	} else {
		c.logger.Info("SSH会话正常结束", zap.Int("用户ID", userID))
	}

	return nil
}
