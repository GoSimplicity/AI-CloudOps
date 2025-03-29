package model

// ChatMessage 聊天消息请求
type ChatMessage struct {
	Query            string                 `json:"query"`
	Inputs           map[string]interface{} `json:"inputs,omitempty"`
	ResponseMode     string                 `json:"response_mode"`
	User             string                 `json:"user"`
	ConversationID   string                 `json:"conversation_id,omitempty"`
	Files            []FileInfo             `json:"files,omitempty"`
	AutoGenerateName bool                   `json:"auto_generate_name,omitempty"`
}

// FileInfo 文件信息
type FileInfo struct {
	Type           string `json:"type"`
	TransferMethod string `json:"transfer_method"`
	URL            string `json:"url,omitempty"`
	UploadFileID   string `json:"upload_file_id,omitempty"`
}

// ChatCompletionResponse 阻塞模式的响应
type ChatCompletionResponse struct {
	MessageID      string                 `json:"message_id"`
	ConversationID string                 `json:"conversation_id"`
	Mode           string                 `json:"mode"`
	Answer         string                 `json:"answer"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      int64                  `json:"created_at"`
}

// ChunkChatCompletionResponse 流式模式的响应块
type ChunkChatCompletionResponse struct {
	Event          string                 `json:"event"`
	MessageID      string                 `json:"message_id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	Answer         string                 `json:"answer,omitempty"`
	TaskID         string                 `json:"task_id,omitempty"`
	Audio          string                 `json:"audio,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      int64                  `json:"created_at,omitempty"`
}

// FileUploadResponse 文件上传响应
type FileUploadResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Size      int    `json:"size"`
	Extension string `json:"extension"`
	MimeType  string `json:"mime_type"`
	CreatedBy string `json:"created_by"`
	CreatedAt int64  `json:"created_at"`
}
