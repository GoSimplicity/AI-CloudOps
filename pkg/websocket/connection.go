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

package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// 默认配置常量
	DefaultWriteWait       = 10 * time.Second
	DefaultPongWait        = 60 * time.Second
	DefaultPingPeriod      = (DefaultPongWait * 9) / 10
	DefaultMaxMessageSize  = 512
	DefaultReadBufferSize  = 1024
	DefaultWriteBufferSize = 1024
)

// Config WebSocket配置
type Config struct {
	ReadBufferSize  int           `json:"read_buffer_size"`  // 读缓冲区大小
	WriteBufferSize int           `json:"write_buffer_size"` // 写缓冲区大小
	WriteWait       time.Duration `json:"write_wait"`        // 写超时时间
	PongWait        time.Duration `json:"pong_wait"`         // Pong等待时间
	PingPeriod      time.Duration `json:"ping_period"`       // Ping发送间隔
	MaxMessageSize  int64         `json:"max_message_size"`  // 最大消息大小
	CheckOrigin     bool          `json:"check_origin"`      // 是否检查来源
}

// Connection WebSocket连接接口
type Connection interface {
	// WriteMessage 写入消息
	WriteMessage(messageType int, data []byte) error
	// ReadMessage 读取消息
	ReadMessage() (messageType int, p []byte, err error)
	// WriteJSON 写入JSON消息
	WriteJSON(v interface{}) error
	// ReadJSON 读取JSON消息
	ReadJSON(v interface{}) error
	// Close 关闭连接
	Close() error
	// SetWriteDeadline 设置写超时
	SetWriteDeadline(t time.Time) error
	// SetReadDeadline 设置读超时
	SetReadDeadline(t time.Time) error
	// SetPongHandler 设置Pong处理器
	SetPongHandler(h func(appData string) error)
	// SetCloseHandler 设置关闭处理器
	SetCloseHandler(h func(code int, text string) error)
	// LocalAddr 获取本地地址
	LocalAddr() string
	// RemoteAddr 获取远程地址
	RemoteAddr() string
}

// Manager WebSocket管理器接口
type Manager interface {
	// Upgrade 升级HTTP连接为WebSocket
	Upgrade(ctx *gin.Context, responseHeader http.Header) (Connection, error)
	// UpgradeHTTP 升级原生HTTP连接为WebSocket
	UpgradeHTTP(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Connection, error)
	// StartHeartbeat 启动心跳检测
	StartHeartbeat(ctx context.Context, conn Connection) error
	// HandleConnection 处理WebSocket连接
	HandleConnection(ctx context.Context, conn Connection, handler ConnectionHandler) error
}

// ConnectionHandler WebSocket连接处理器
type ConnectionHandler interface {
	// OnConnect 连接建立时调用
	OnConnect(conn Connection) error
	// OnMessage 收到消息时调用
	OnMessage(conn Connection, messageType int, data []byte) error
	// OnClose 连接关闭时调用
	OnClose(conn Connection, code int, text string) error
	// OnError 发生错误时调用
	OnError(conn Connection, err error) error
}

// connection WebSocket连接实现
type connection struct {
	*websocket.Conn
	logger *zap.Logger
}

// manager WebSocket管理器实现
type manager struct {
	upgrader *websocket.Upgrader
	config   *Config
	logger   *zap.Logger
}

// NewManager 创建WebSocket管理器
func NewManager(config *Config, logger *zap.Logger) Manager {
	if logger == nil {
		logger = zap.NewNop()
	}

	// 应用默认配置
	cfg := getDefaultConfig()
	if config != nil {
		if config.ReadBufferSize > 0 {
			cfg.ReadBufferSize = config.ReadBufferSize
		}
		if config.WriteBufferSize > 0 {
			cfg.WriteBufferSize = config.WriteBufferSize
		}
		if config.WriteWait > 0 {
			cfg.WriteWait = config.WriteWait
		}
		if config.PongWait > 0 {
			cfg.PongWait = config.PongWait
		}
		if config.PingPeriod > 0 {
			cfg.PingPeriod = config.PingPeriod
		}
		if config.MaxMessageSize > 0 {
			cfg.MaxMessageSize = config.MaxMessageSize
		}
		cfg.CheckOrigin = config.CheckOrigin
	}

	// 创建升级器
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  cfg.ReadBufferSize,
		WriteBufferSize: cfg.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return !cfg.CheckOrigin || true // 不检查来源或通过检查
		},
	}

	return &manager{
		upgrader: upgrader,
		config:   cfg,
		logger:   logger,
	}
}

// Upgrade 升级HTTP连接为WebSocket
func (m *manager) Upgrade(ctx *gin.Context, responseHeader http.Header) (Connection, error) {
	conn, err := m.upgrader.Upgrade(ctx.Writer, ctx.Request, responseHeader)
	if err != nil {
		m.logger.Error("WebSocket升级失败", zap.Error(err))
		return nil, fmt.Errorf("WebSocket升级失败: %w", err)
	}

	// 配置连接参数
	conn.SetReadLimit(m.config.MaxMessageSize)

	wsConn := &connection{
		Conn:   conn,
		logger: m.logger,
	}

	m.logger.Info("WebSocket连接已建立",
		zap.String("远程地址", conn.RemoteAddr().String()),
		zap.String("本地地址", conn.LocalAddr().String()))

	return wsConn, nil
}

// UpgradeHTTP 升级原生HTTP连接为WebSocket
func (m *manager) UpgradeHTTP(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Connection, error) {
	conn, err := m.upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		m.logger.Error("WebSocket升级失败", zap.Error(err))
		return nil, fmt.Errorf("WebSocket升级失败: %w", err)
	}

	// 配置连接参数
	conn.SetReadLimit(m.config.MaxMessageSize)

	wsConn := &connection{
		Conn:   conn,
		logger: m.logger,
	}

	m.logger.Info("WebSocket连接已建立",
		zap.String("远程地址", conn.RemoteAddr().String()),
		zap.String("本地地址", conn.LocalAddr().String()))

	return wsConn, nil
}

// StartHeartbeat 启动心跳检测
func (m *manager) StartHeartbeat(ctx context.Context, conn Connection) error {
	if conn == nil {
		return fmt.Errorf("连接不能为空")
	}

	// 配置Pong处理器
	conn.SetPongHandler(func(string) error {
		m.logger.Debug("收到Pong消息")
		conn.SetReadDeadline(time.Now().Add(m.config.PongWait))
		return nil
	})

	// 设置初始读超时
	conn.SetReadDeadline(time.Now().Add(m.config.PongWait))

	// 启动Ping发送协程
	go func() {
		ticker := time.NewTicker(m.config.PingPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				m.logger.Debug("上下文已取消，停止心跳检测")
				return
			case <-ticker.C:
				if err := conn.SetWriteDeadline(time.Now().Add(m.config.WriteWait)); err != nil {
					m.logger.Error("设置写超时失败", zap.Error(err))
					return
				}
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					m.logger.Error("发送Ping消息失败", zap.Error(err))
					return
				}
				m.logger.Debug("已发送Ping消息")
			}
		}
	}()

	return nil
}

// HandleConnection 处理WebSocket连接
func (m *manager) HandleConnection(ctx context.Context, conn Connection, handler ConnectionHandler) error {
	if conn == nil {
		return fmt.Errorf("连接不能为空")
	}
	if handler == nil {
		return fmt.Errorf("处理器不能为空")
	}

	// 创建可取消上下文
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// 配置关闭处理器
	conn.SetCloseHandler(func(code int, text string) error {
		m.logger.Info("WebSocket连接已关闭", zap.Int("代码", code), zap.String("原因", text))
		handler.OnClose(conn, code, text)
		cancel()
		return nil
	})

	// 调用连接建立处理器
	if err := handler.OnConnect(conn); err != nil {
		m.logger.Error("连接建立处理失败", zap.Error(err))
		return fmt.Errorf("连接建立处理失败: %w", err)
	}

	// 启动心跳检测
	if err := m.StartHeartbeat(ctx, conn); err != nil {
		m.logger.Error("启动心跳检测失败", zap.Error(err))
		return fmt.Errorf("启动心跳检测失败: %w", err)
	}

	// 启动消息读取协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.handleMessages(ctx, conn, handler, cancel)
	}()

	// 等待上下文取消
	<-ctx.Done()
	m.logger.Debug("WebSocket连接处理结束")

	// 等待所有协程结束
	wg.Wait()

	return nil
}

// handleMessages 处理消息读取
func (m *manager) handleMessages(ctx context.Context, conn Connection, handler ConnectionHandler, cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			m.logger.Debug("上下文已取消，停止消息处理")
			return
		default:
			messageType, data, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					m.logger.Error("WebSocket连接异常关闭", zap.Error(err))
					handler.OnError(conn, err)
				} else {
					m.logger.Debug("WebSocket连接正常关闭", zap.Error(err))
				}
				return
			}

			// 处理消息
			if err := handler.OnMessage(conn, messageType, data); err != nil {
				m.logger.Error("消息处理失败", zap.Error(err))
				handler.OnError(conn, err)
				return
			}
		}
	}
}

// LocalAddr 获取本地地址
func (c *connection) LocalAddr() string {
	return c.Conn.LocalAddr().String()
}

// RemoteAddr 获取远程地址
func (c *connection) RemoteAddr() string {
	return c.Conn.RemoteAddr().String()
}

// getDefaultConfig 获取默认配置
func getDefaultConfig() *Config {
	return &Config{
		ReadBufferSize:  DefaultReadBufferSize,
		WriteBufferSize: DefaultWriteBufferSize,
		WriteWait:       DefaultWriteWait,
		PongWait:        DefaultPongWait,
		PingPeriod:      DefaultPingPeriod,
		MaxMessageSize:  DefaultMaxMessageSize,
		CheckOrigin:     false,
	}
}

// NewSimpleUpgrader 创建简单的WebSocket升级器
func NewSimpleUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  DefaultReadBufferSize,
		WriteBufferSize: DefaultWriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}
