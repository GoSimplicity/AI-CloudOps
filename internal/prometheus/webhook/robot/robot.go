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

package robot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/request"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// WebhookRobot 定义了Webhook机器人相关的接口
type WebhookRobot interface {
	// RefreshPrivateRobotToken 刷新私有机器人令牌
	RefreshPrivateRobotToken(ctx context.Context)
	// GetPrivateRobotToken 获取当前的私有机器人令牌
	GetPrivateRobotToken() string
	// GetTenantAccessToken 获取租户访问令牌
	GetTenantAccessToken(ctx context.Context) (string, error)
}

// webhookRobot 是 WebhookRobot 接口的实现
type webhookRobot struct {
	privateRobotToken string
	tenantToken       string

	logger          *zap.Logger
	privateTokenMux sync.RWMutex
	tenantExpireAt  time.Time
	tenantTokenMux  sync.RWMutex
	httpClient      *http.Client
}

// NewWebhookRobot 创建一个新的 webhookRobot 实例
func NewWebhookRobot(logger *zap.Logger) WebhookRobot {
	return &webhookRobot{
		logger: logger,
		httpClient: &http.Client{
			Timeout: time.Duration(viper.GetInt("webhook.im_feishu.request_timeout_seconds")) * time.Second,
		},
	}
}

// RefreshPrivateRobotToken 刷新私有机器人令牌
func (w *webhookRobot) RefreshPrivateRobotToken(ctx context.Context) {
	token, _, err := w.getTokenFromAPI(ctx)
	if err != nil {
		w.logger.Error("刷新私有机器人令牌失败", zap.Error(err))
		return
	}

	// 更新私有机器人令牌
	w.privateTokenMux.Lock()
	w.privateRobotToken = token
	w.privateTokenMux.Unlock()

	w.logger.Info("成功刷新私有机器人令牌")
}

// GetTenantAccessToken 获取租户访问令牌
func (w *webhookRobot) GetTenantAccessToken(ctx context.Context) (string, error) {
	// 读锁检查是否有有效的令牌
	w.tenantTokenMux.RLock()
	if w.tenantToken != "" && time.Now().Before(w.tenantExpireAt.Add(-60*time.Second)) {
		token := w.tenantToken
		w.tenantTokenMux.RUnlock()
		return token, nil
	}
	w.tenantTokenMux.RUnlock()

	// 写锁获取新的令牌
	w.tenantTokenMux.Lock()
	defer w.tenantTokenMux.Unlock()

	// 再次检查，防止其他 Goroutine 已经更新
	if w.tenantToken != "" && time.Now().Before(w.tenantExpireAt.Add(-60*time.Second)) {
		return w.tenantToken, nil
	}

	// 获取新的令牌
	token, expire, err := w.getTokenFromAPI(ctx)
	if err != nil {
		w.logger.Error("获取租户访问令牌失败", zap.Error(err))
		return "", err
	}

	// 更新租户访问令牌和过期时间
	w.tenantToken = token
	w.tenantExpireAt = time.Now().Add(time.Duration(expire) * time.Second)

	w.logger.Info("成功获取租户访问令牌")
	return w.tenantToken, nil
}

// GetPrivateRobotToken 获取当前的私有机器人令牌
func (w *webhookRobot) GetPrivateRobotToken() string {
	w.privateTokenMux.RLock()
	defer w.privateTokenMux.RUnlock()
	return w.privateRobotToken
}

// postWithJson 发送带有JSON字节的POST请求
func (w *webhookRobot) postWithJson(ctx context.Context, url string, jsonBytes []byte, headers map[string]string) ([]byte, error) {
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		w.logger.Error("创建HTTP请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 设置默认Content-Type
	if _, exists := headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.logger.Error("发送HTTP请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("读取响应体失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.logger.Error("服务器返回非2xx状态码",
			zap.String("url", url),
			zap.Int("statusCode", resp.StatusCode),
			zap.String("responseBody", string(bodyBytes)),
		)
		return bodyBytes, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return bodyBytes, nil
}

// getTokenFromAPI 是一个通用函数，用于获取 token
func (w *webhookRobot) getTokenFromAPI(ctx context.Context) (string, int, error) {
	requestData := request.RobotTenantAccessTokenReq{
		AppID:     viper.GetString("webhook.im_feishu.private_robot_app_id"),
		AppSecret: viper.GetString("webhook.im_feishu.private_robot_app_secret"),
	}

	// 序列化请求数据
	jsonBytes, err := json.Marshal(requestData)
	if err != nil {
		w.logger.Error("获取令牌时序列化请求数据失败",
			zap.Error(err),
			zap.String("AppID", requestData.AppID),
		)
		return "", 0, fmt.Errorf("failed to serialize request: %w", err)
	}

	// 发送 HTTP POST 请求
	url := viper.GetString("webhook.im_feishu.tenant_access_token_api")
	bodyBytes, err := w.postWithJson(ctx, url, jsonBytes, nil)
	if err != nil {
		w.logger.Error("获取令牌时发送HTTP请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return "", 0, fmt.Errorf("failed to send HTTP request: %w", err)
	}

	// 解析响应数据
	var responseData request.RobotTenantAccessTokenRes
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		w.logger.Error("解析响应JSON失败",
			zap.Error(err),
			zap.String("AppID", requestData.AppID),
			zap.String("responseBody", string(bodyBytes)),
		)
		return "", 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// 如果 API 返回错误码，记录错误并返回
	if responseData.Code != 0 {
		w.logger.Error("API返回错误",
			zap.Int("Code", responseData.Code),
			zap.String("Message", responseData.Message),
			zap.String("responseBody", string(bodyBytes)),
		)
		return "", 0, fmt.Errorf("API error: code=%d, message=%s", responseData.Code, responseData.Message)
	}

	return responseData.TenantAccessToken, responseData.Expire, nil
}
