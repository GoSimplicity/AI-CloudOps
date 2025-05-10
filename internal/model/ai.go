package model

// ChatMessage 聊天消息请求
type ChatMessage struct {
	Role        string           `json:"role"`
	Style       string           `json:"style"`
	Question    string           `json:"question"`
	ChatHistory []HistoryMessage `json:"chatHistory"`
}

// HistoryMessage 聊天历史中的消息
type HistoryMessage struct {
	Role    string `json:"role"`    // user 或 assistant
	Content string `json:"content"` // 消息内容
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	Answer string `json:"answer"`
}

// StreamResponse 流式响应
type StreamResponse struct {
	Content  string `json:"content"`
	Error    string `json:"error,omitempty"`
	Done     bool   `json:"done"`
	ToolName string `json:"toolName"`
	ToolDesc string `json:"toolDesc"`
}

// WSResponse WebSocket响应格式
type WSResponse struct {
	Type    string `json:"type"` // message 或 error
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
	Done    bool   `json:"done"`
}
