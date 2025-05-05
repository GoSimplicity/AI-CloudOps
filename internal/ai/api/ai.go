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

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/internal/ai/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AIHandler struct {
	service  service.AIService
	upgrader websocket.Upgrader
}

func NewAIHandler(service service.AIService) *AIHandler {
	return &AIHandler{
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *AIHandler) RegisterRouters(server *gin.Engine) {
	aiGroup := server.Group("/api/ai")
	{
		aiGroup.POST("/chat", h.SendChatMessage)
		aiGroup.GET("/chat/ws", h.HandleWebSocketChat)
	}
}

// SendChatMessage 发送常规聊天消息 (HTTP)
func (h *AIHandler) SendChatMessage(ctx *gin.Context) {
	var req model.ChatMessage

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.SendChatMessage(ctx, req)
	})
}

// HandleWebSocketChat 处理WebSocket连接的聊天
func (h *AIHandler) HandleWebSocketChat(ctx *gin.Context) {
	conn, err := h.upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 持续监听消息
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}

		// 解析请求
		var chatRequest model.ChatMessage
		if err := json.Unmarshal(message, &chatRequest); err != nil {
			h.sendErrorResponse(conn, "Invalid request format")
			continue
		}

		// 创建响应通道
		responseChan := make(chan model.StreamResponse)

		// 启动goroutine处理聊天请求
		go func() {
			err := h.service.StreamChatMessage(ctx, chatRequest, responseChan)
			if err != nil {
			}
		}()

		// 从通道读取响应并通过WebSocket发送
		for response := range responseChan {
			if response.Error != "" {
				h.sendErrorResponse(conn, response.Error)
				break
			}

			resp := model.WSResponse{
				Type:    "message",
				Content: response.Content,
				Done:    response.Done,
			}

			if err := conn.WriteJSON(resp); err != nil {
				break
			}
		}
	}
}

// sendErrorResponse 发送错误响应
func (h *AIHandler) sendErrorResponse(conn *websocket.Conn, errMsg string) error {
	resp := model.WSResponse{
		Type:  "error",
		Error: errMsg,
		Done:  true,
	}

	if err := conn.WriteJSON(resp); err != nil {
		return fmt.Errorf("发送错误响应失败: %w", err)
	}
	return nil
}
