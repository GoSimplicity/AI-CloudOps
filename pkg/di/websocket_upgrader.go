package di

import (
	ws "github.com/GoSimplicity/AI-CloudOps/pkg/websocket"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// InitWebSocketUpgrader 初始化WebSocket升级器
// 说明：
// 1. 升级逻辑属于传输层能力，必须通过依赖注入提供，避免全局单例
// 2. Error回调返回通用错误，避免暴露内部细节
func InitWebSocketUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  10 * time.Second,
		Subprotocols:      []string{ws.Protocol},
		EnableCompression: true,
		CheckOrigin:       ws.CheckOrigin,
		Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
			http.Error(w, "WebSocket 升级失败", status)
		},
	}
}
