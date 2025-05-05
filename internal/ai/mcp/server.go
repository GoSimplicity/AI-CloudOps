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

package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	getime "github.com/GoSimplicity/AI-CloudOps/internal/ai/mcp/tools/time"
	"github.com/mark3labs/mcp-go/server"
)

type MCP struct {
	Name          string
	Version       string
	Port          int
	ServerOptions []server.ServerOption
	SSEOption     []server.SSEOption
	Mode          MCPServerMode
	sseServer     *server.SSEServer
	mu            sync.Mutex
	stopCh        chan struct{} // 用于控制服务器停止的通道
}

type MCPServerMode string

const (
	MCPServerModeSSE MCPServerMode = "sse"
)

func NewMCPServer() *MCP {
	return &MCP{
		Mode:    MCPServerModeSSE,
		Name:    "AI-CloudOps",
		Version: "1.0.0",
		Port:    9000,
		stopCh:  make(chan struct{}), // 初始化停止通道
	}
}

// Start 启动MCP服务器
func (s *MCP) Start() error {
	if s.Name == "" {
		return fmt.Errorf("服务器名称不能为空")
	}
	if s.Version == "" {
		return fmt.Errorf("服务器版本不能为空")
	}
	if s.Port <= 0 || s.Port > 65535 {
		return fmt.Errorf("无效的端口号: %d", s.Port)
	}
	return RunMCPServerWithOption(s)
}

// Stop 停止MCP服务器
func (s *MCP) Stop() error {
	s.mu.Lock()
	sseServer := s.sseServer
	s.mu.Unlock()

	if sseServer != nil {
		// 先发送停止信号
		close(s.stopCh)

		// 设置更长的超时时间，确保有足够时间处理连接关闭
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := sseServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("服务器关闭失败: %w", err)
		}
		log.Println("MCP服务器已成功关闭")
		return nil
	}

	return nil
}

// RunMCPServerWithOption 启动MCP服务器
func RunMCPServerWithOption(cfg *MCP) error {
	if cfg == nil {
		return fmt.Errorf("服务器配置不能为空")
	}

	// 添加自定义选项处理SSE连接关闭
	sseOptions := append(cfg.SSEOption, server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
		return ctx
	}))

	// 创建基础服务
	baseServer := server.NewMCPServer(
		cfg.Name,
		cfg.Version,
		cfg.ServerOptions...,
	)

	// 注册工具
	RegisterTools(baseServer)

	// 初始化SSE服务器
	cfg.mu.Lock()
	cfg.sseServer = server.NewSSEServer(baseServer, sseOptions...)
	cfg.mu.Unlock()

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		serverErr := make(chan error, 1)

		go func() {
			if err := cfg.sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}()

		// 等待停止信号或错误
		select {
		case <-cfg.stopCh:
			log.Println("接收到停止信号，准备关闭SSE服务器")
		case err := <-serverErr:
			log.Fatalf("SSE服务器启动失败: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	log.Println("SSE服务器启动成功")

	return nil
}

func RegisterTools(server *server.MCPServer) {
	getime.RegisterTools(server)
	// TODO: 注册其他工具
}
