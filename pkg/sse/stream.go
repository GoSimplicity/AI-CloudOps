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

package sse

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Producer 生产者函数类型，用于产生SSE数据
type Producer func(ctx context.Context, msgChan chan<- interface{})

// Config SSE配置
type Config struct {
	BufferSize int    `json:"buffer_size"` // 消息通道缓冲区大小，默认为1
	EventName  string `json:"event_name"`  // SSE事件名称，默认为"message"
}

// Handler SSE处理器接口
type Handler interface {
	// Stream 启动SSE流式推送
	Stream(ctx *gin.Context, producer Producer, config ...*Config) error
	// StreamWithContext 使用自定义上下文启动SSE流式推送
	StreamWithContext(ctx context.Context, writer io.Writer, producer Producer, config ...*Config) error
}

// handler SSE处理器实现
type handler struct {
	logger *zap.Logger
}

// NewHandler 创建新的SSE处理器
func NewHandler(logger *zap.Logger) Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &handler{
		logger: logger,
	}
}

// Stream 启动SSE流式推送（基于Gin Context）
func (h *handler) Stream(ctx *gin.Context, producer Producer, config ...*Config) error {
	if ctx == nil {
		return fmt.Errorf("gin Context不能为空")
	}
	if producer == nil {
		return fmt.Errorf("生产者函数不能为空")
	}

	// 获取配置
	cfg := h.getConfig(config...)

	// 设置SSE响应头
	h.setSSEHeaders(ctx)

	// 创建消息通道
	msgChan := make(chan interface{}, cfg.BufferSize)

	// 创建可取消的上下文
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 启动连接监听协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.monitorConnection(ctx, cancelCtx, cancel)
	}()

	// 启动生产者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(msgChan)
		h.runProducer(cancelCtx, producer, msgChan)
	}()

	// 主流式推送循环
	ctx.Stream(func(w io.Writer) bool {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				h.logger.Debug("消息通道已关闭，停止SSE推送")
				return false
			}
			ctx.SSEvent(cfg.EventName, msg)
			return true
		case <-cancelCtx.Done():
			h.logger.Debug("上下文已取消，停止SSE推送")
			return false
		}
	})

	// 等待所有协程结束
	wg.Wait()
	h.logger.Debug("SSE流式推送已完成")

	return nil
}

// StreamWithContext 使用自定义上下文启动SSE流式推送
func (h *handler) StreamWithContext(ctx context.Context, writer io.Writer, producer Producer, config ...*Config) error {
	if ctx == nil {
		return fmt.Errorf("上下文不能为空")
	}
	if writer == nil {
		return fmt.Errorf("writer不能为空")
	}
	if producer == nil {
		return fmt.Errorf("生产者函数不能为空")
	}

	// 获取配置
	cfg := h.getConfig(config...)

	// 创建消息通道
	msgChan := make(chan interface{}, cfg.BufferSize)

	// 创建可取消的上下文
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	// 启动生产者协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(msgChan)
		h.runProducer(cancelCtx, producer, msgChan)
	}()

	// 主流式推送循环
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				h.logger.Debug("消息通道已关闭，停止SSE推送")
				wg.Wait()
				return nil
			}

			// 写入SSE格式的数据
			if err := h.writeSSEMessage(writer, cfg.EventName, msg); err != nil {
				h.logger.Error("写入SSE消息失败", zap.Error(err))
				cancel()
				wg.Wait()
				return fmt.Errorf("写入SSE消息失败: %w", err)
			}

		case <-cancelCtx.Done():
			h.logger.Debug("上下文已取消，停止SSE推送")
			wg.Wait()
			return ctx.Err()
		}
	}
}

// getConfig 获取有效配置
func (h *handler) getConfig(config ...*Config) *Config {
	// 默认配置
	cfg := &Config{
		BufferSize: 1,
		EventName:  "message",
	}

	// 使用提供的配置覆盖默认值
	if len(config) > 0 && config[0] != nil {
		if config[0].BufferSize > 0 {
			cfg.BufferSize = config[0].BufferSize
		}
		if config[0].EventName != "" {
			cfg.EventName = config[0].EventName
		}
	}

	return cfg
}

// setSSEHeaders 设置SSE响应头
func (h *handler) setSSEHeaders(ctx *gin.Context) {
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("Access-Control-Allow-Headers", "Cache-Control")
}

// monitorConnection 监听客户端连接状态
func (h *handler) monitorConnection(ctx *gin.Context, cancelCtx context.Context, cancel context.CancelFunc) {
	select {
	case <-ctx.Request.Context().Done():
		h.logger.Info("客户端连接已断开，停止SSE推送")
		cancel()
	case <-cancelCtx.Done():
		h.logger.Debug("SSE推送已完成")
	}
}

// runProducer 运行生产者函数
func (h *handler) runProducer(ctx context.Context, producer Producer, msgChan chan<- interface{}) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("生产者函数发生panic", zap.Any("panic", r))
		}
	}()

	h.logger.Debug("开始运行生产者函数")
	producer(ctx, msgChan)
	h.logger.Debug("生产者函数运行完成")
}

// writeSSEMessage 写入SSE格式的消息
func (h *handler) writeSSEMessage(writer io.Writer, eventName string, data interface{}) error {
	// 格式化SSE消息
	message := fmt.Sprintf("event: %s\ndata: %v\n\n", eventName, data)

	// 写入数据
	_, err := writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("写入SSE消息失败: %w", err)
	}

	// 如果writer支持Flush，则立即刷新
	if flusher, ok := writer.(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}

// StreamSimple 简化的SSE流式推送函数（向后兼容）
func StreamSimple(ctx *gin.Context, producer Producer, logger *zap.Logger) error {
	handler := NewHandler(logger)
	return handler.Stream(ctx, producer)
}
