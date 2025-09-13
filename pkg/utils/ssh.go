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

package utils

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// WebSocket 升级器配置常量
	DefaultWSReadBufferSize   = 4096             // 读缓冲区大小（增加以提高性能）
	DefaultWSWriteBufferSize  = 4096             // 写缓冲区大小（增加以提高性能）
	DefaultWSHandshakeTimeout = 10 * time.Second // 握手超时时间
)

// UpGrader 升级HTTP连接为WebSocket连接
// 包含更好的错误处理和性能配置
var UpGrader = websocket.Upgrader{
	ReadBufferSize:    DefaultWSReadBufferSize,
	WriteBufferSize:   DefaultWSWriteBufferSize,
	HandshakeTimeout:  DefaultWSHandshakeTimeout,
	EnableCompression: true, // 启用压缩以提高性能
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中，应该实现适当的来源检查
		// 目前为了兼容性允许所有来源
		return true
	},
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		// 自定义错误处理，避免暴露敏感信息
		http.Error(w, "WebSocket upgrade failed", status)
	},
}
