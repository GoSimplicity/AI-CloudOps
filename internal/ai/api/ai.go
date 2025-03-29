package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/ai/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	service service.AIService
}

func NewAIHandler(service service.AIService) *AIHandler {
	return &AIHandler{
		service: service,
	}
}

func (h *AIHandler) RegisterRouters(server *gin.Engine) {
	aiGroup := server.Group("/api/ai")
	{
		aiGroup.POST("/chat", h.SendChatMessage)
		aiGroup.POST("/chat/stream", h.SendStreamingChatMessage)
		aiGroup.POST("/upload", h.UploadFile)
		aiGroup.POST("/stop", h.StopResponse)
		aiGroup.POST("/feedback", h.SendFeedback)
		aiGroup.POST("/suggested_questions", h.GetSuggestedQuestions)
		aiGroup.POST("/messages", h.GetMessages)
		aiGroup.POST("/conversations", h.GetConversations)
		aiGroup.POST("/conversation/delete", h.DeleteConversation)
		aiGroup.POST("/conversation/rename", h.RenameConversation)
	}
}

// SendChatMessage 发送聊天消息
func (h *AIHandler) SendChatMessage(ctx *gin.Context) {
	var req model.ChatMessage

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.SendChatMessage(ctx, req)
	})
}

// SendStreamingChatMessage 发送流式聊天消息
func (h *AIHandler) SendStreamingChatMessage(ctx *gin.Context) {
	var req model.ChatMessage

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	// 设置流式响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("X-Accel-Buffering", "no")

	// 确保立即发送头信息
	ctx.Writer.Flush()

	// 处理客户端断开连接的情况
	clientGone := ctx.Writer.CloseNotify()

	err := h.service.SendStreamingChatMessage(ctx, req, func(chunk model.ChunkChatCompletionResponse) error {
		// 检查客户端是否已断开连接
		select {
		case <-clientGone:
			return nil
		default:
			// 发送事件流数据
			ctx.SSEvent("message", chunk)
			ctx.Writer.Flush()
			return nil
		}
	})

	if err != nil {
		// 尝试发送错误消息，但客户端可能已断开连接
		select {
		case <-clientGone:
			return
		default:
			utils.ErrorWithMessage(ctx, err.Error())
		}
	}
}

// UploadFile 上传文件
func (h *AIHandler) UploadFile(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	user := ctx.PostForm("user")
	filePath := "/tmp/" + file.Filename
	if err := ctx.SaveUploadedFile(file, filePath); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.service.UploadFile(ctx, filePath, user)
	})
}

// StopResponse 停止响应
func (h *AIHandler) StopResponse(ctx *gin.Context) {
	var req struct {
		TaskID string `json:"task_id"`
		User   string `json:"user"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.StopResponse(ctx, req.TaskID, req.User)
	})
}

// SendFeedback 发送反馈
func (h *AIHandler) SendFeedback(ctx *gin.Context) {
	var req struct {
		MessageID string `json:"message_id"`
		Rating    string `json:"rating"`
		User      string `json:"user"`
		Content   string `json:"content"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.SendFeedback(ctx, req.MessageID, req.Rating, req.User, req.Content)
	})
}

// GetSuggestedQuestions 获取建议问题
func (h *AIHandler) GetSuggestedQuestions(ctx *gin.Context) {
	var req struct {
		MessageID string `json:"message_id"`
		User      string `json:"user"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetSuggestedQuestions(ctx, req.MessageID, req.User)
	})
}

// GetMessages 获取消息
func (h *AIHandler) GetMessages(ctx *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id"`
		User           string `json:"user"`
		FirstID        string `json:"first_id"`
		Limit          int    `json:"limit"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetMessages(ctx, req.ConversationID, req.User, req.FirstID, req.Limit)
	})
}

// GetConversations 获取会话列表
func (h *AIHandler) GetConversations(ctx *gin.Context) {
	var req struct {
		User   string `json:"user"`
		LastID string `json:"last_id"`
		Limit  int    `json:"limit"`
		SortBy string `json:"sort_by"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.GetConversations(ctx, req.User, req.LastID, req.Limit, req.SortBy)
	})
}

// DeleteConversation 删除会话
func (h *AIHandler) DeleteConversation(ctx *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id"`
		User           string `json:"user"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.service.DeleteConversation(ctx, req.ConversationID, req.User)
	})
}

// RenameConversation 重命名会话
func (h *AIHandler) RenameConversation(ctx *gin.Context) {
	var req struct {
		ConversationID string `json:"conversation_id"`
		Name           string `json:"name"`
		AutoGenerate   bool   `json:"auto_generate"`
		User           string `json:"user"`
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.service.RenameConversation(ctx, req.ConversationID, req.Name, req.AutoGenerate, req.User)
	})
}
