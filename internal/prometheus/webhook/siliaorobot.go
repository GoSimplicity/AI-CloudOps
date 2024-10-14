package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/request"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"go.uber.org/zap"
	"sync"
)

// WebhookRobot 定义了用于管理Webhook机器人的接口
type WebhookRobot interface {
	// RefreshPrivateRobotToken 刷新私有机器人令牌
	RefreshPrivateRobotToken(ctx context.Context) error
	// GetPrivateRobotToken 获取当前的私有机器人令牌
	GetPrivateRobotToken() string
}

type webhookRobot struct {
	logger            *zap.Logger
	config            *config.AlertWebhookConfig
	privateRobotToken string
	mux               sync.RWMutex
}

func NewWebhookRobot(logger *zap.Logger, config *config.AlertWebhookConfig) WebhookRobot {
	return &webhookRobot{
		logger: logger,
		config: config,
	}
}

// RefreshPrivateRobotToken 刷新私有机器人令牌
func (w *webhookRobot) RefreshPrivateRobotToken(ctx context.Context) error {
	// 构造请求数据
	requestData := request.RobotTenantAccessTokenReq{
		AppID:     w.config.IMFeishuConfig.PrivateChatRobotAppID,
		AppSecret: w.config.IMFeishuConfig.PrivateChatRobotAppSecret,
	}

	// 将请求数据序列化为JSON
	jsonBytes, err := json.Marshal(requestData)
	if err != nil {
		w.logger.Error("刷新私有机器人令牌时序列化请求数据失败",
			zap.Error(err),
			zap.String("AppID", w.config.IMFeishuConfig.PrivateChatRobotAppID),
		)
		return fmt.Errorf("failed to marshal RobotTenantAccessTokenReq: %w", err)
	}

	// 发送HTTP POST请求
	bodyBytes, err := apiresponse.PostWithJsonString(
		w.logger,
		"RefreshPrivateRobotToken",
		w.config.IMFeishuConfig.RequestTimeoutSeconds,
		w.config.IMFeishuConfig.TenantAccessTokenAPI,
		string(jsonBytes),
		nil, // paramsMap
		nil, // headerMap
	)
	if err != nil {
		w.logger.Error("刷新私有机器人令牌时发送HTTP请求失败",
			zap.Error(err),
			zap.String("funcName", "RefreshPrivateRobotToken"),
			zap.String("url", w.config.IMFeishuConfig.TenantAccessTokenAPI),
		)
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}

	// 解析响应数据
	var responseData request.RobotTenantAccessTokenRes
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		w.logger.Error("刷新私有机器人令牌时解析响应JSON失败",
			zap.Error(err),
			zap.String("AppID", w.config.IMFeishuConfig.PrivateChatRobotAppID),
			zap.String("responseBody", string(bodyBytes)),
		)
		return fmt.Errorf("failed to unmarshal RobotTenantAccessTokenRes: %w", err)
	}

	if responseData.Code != 0 {
		w.logger.Error("刷新私有机器人令牌时API返回错误",
			zap.Int("Code", responseData.Code),
			zap.String("Message", responseData.Message),
			zap.String("responseBody", string(bodyBytes)),
		)
		return fmt.Errorf("API returned error code %d: %s", responseData.Code, responseData.Message)
	}

	// 更新私有机器人令牌
	w.mux.Lock()
	w.privateRobotToken = responseData.TenantAccessToken
	w.mux.Unlock()

	w.logger.Info("成功刷新私有机器人令牌",
		zap.String("AppID", w.config.IMFeishuConfig.PrivateChatRobotAppID),
	)

	return nil
}

// GetPrivateRobotToken 获取当前的私有机器人令牌
func (w *webhookRobot) GetPrivateRobotToken() string {
	w.mux.RLock()
	defer w.mux.RUnlock()
	return w.privateRobotToken
}
