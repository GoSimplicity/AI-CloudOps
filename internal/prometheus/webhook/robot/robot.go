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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/request"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var ErrPrivateRobotDisabled = errors.New("private robot credentials are not configured")
var ErrTokenRequest = errors.New("token request construction failed")
var ErrTokenTransport = errors.New("token HTTP transport failed")
var ErrTokenRead = errors.New("token response read failed")
var ErrTokenEncode = errors.New("token request encoding failed")
var ErrTokenDecode = errors.New("token response decoding failed")
var ErrTokenInvalid = errors.New("token provider returned an unusable token or expiry")

type TokenHTTPStatusError struct{ StatusCode int }

func (err *TokenHTTPStatusError) Error() string {
	return fmt.Sprintf("token HTTP status %d", err.StatusCode)
}

type TokenProviderError struct{ Code int }

func (err *TokenProviderError) Error() string { return fmt.Sprintf("token provider code %d", err.Code) }

// WebhookRobot 定义了Webhook机器人相关的接口
type WebhookRobot interface {
	// RefreshPrivateRobotToken 刷新私有机器人令牌
	RefreshPrivateRobotToken(ctx context.Context)
	GetPrivateRobotToken() string
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
	if errors.Is(err, ErrPrivateRobotDisabled) {
		return
	}
	if err != nil {
		w.logger.Error("刷新私有机器人令牌失败", zap.String("operation", "private_token_refresh"))
		return
	}

	w.privateTokenMux.Lock()
	w.privateRobotToken = token
	w.privateTokenMux.Unlock()

	w.logger.Info("成功刷新私有机器人令牌")
}

func (w *webhookRobot) GetTenantAccessToken(ctx context.Context) (string, error) {
	w.tenantTokenMux.RLock()
	if w.tenantToken != "" && time.Now().Before(w.tenantExpireAt.Add(-60*time.Second)) {
		token := w.tenantToken
		w.tenantTokenMux.RUnlock()
		return token, nil
	}
	w.tenantTokenMux.RUnlock()

	w.tenantTokenMux.Lock()
	defer w.tenantTokenMux.Unlock()

	// 再次检查，防止其他 Goroutine 已经更新
	if w.tenantToken != "" && time.Now().Before(w.tenantExpireAt.Add(-60*time.Second)) {
		return w.tenantToken, nil
	}

	token, expire, err := w.getTokenFromAPI(ctx)
	if errors.Is(err, ErrPrivateRobotDisabled) {
		return "", err
	}
	if err != nil {
		w.logger.Error("获取租户访问令牌失败", zap.String("operation", "tenant_token_get"))
		return "", err
	}

	// 更新租户访问令牌和过期时间
	w.tenantToken = token
	w.tenantExpireAt = time.Now().Add(time.Duration(expire) * time.Second)

	w.logger.Info("成功获取租户访问令牌")
	return w.tenantToken, nil
}

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
		w.logger.Error("创建HTTP请求失败", zap.String("operation", "token_request_build"))
		return nil, ErrTokenRequest
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 设置默认Content-Type
	if _, exists := headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.logger.Error("发送HTTP请求失败", zap.String("operation", "token_transport"))
		return nil, ErrTokenTransport
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.Error("读取响应体失败", zap.String("operation", "token_response_read"))
		return nil, ErrTokenRead
	}

	// 检查HTTP状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		w.logger.Error("服务器返回非2xx状态码", zap.String("operation", "token_http_status"), zap.Int("status_code", resp.StatusCode))
		return nil, &TokenHTTPStatusError{StatusCode: resp.StatusCode}
	}

	return bodyBytes, nil
}

// getTokenFromAPI 是一个通用函数，用于获取 token
func (w *webhookRobot) getTokenFromAPI(ctx context.Context) (string, int, error) {
	requestData := request.RobotTenantAccessTokenReq{
		AppID:     viper.GetString("webhook.im_feishu.private_robot_app_id"),
		AppSecret: viper.GetString("webhook.im_feishu.private_robot_app_secret"),
	}

	if strings.TrimSpace(requestData.AppID) == "" || strings.TrimSpace(requestData.AppSecret) == "" {
		return "", 0, ErrPrivateRobotDisabled
	}

	jsonBytes, err := json.Marshal(requestData)
	if err != nil {
		w.logger.Error("获取令牌时序列化请求数据失败", zap.String("operation", "token_request_encode"))
		return "", 0, ErrTokenEncode
	}

	// 发送 HTTP POST 请求
	url := viper.GetString("webhook.im_feishu.tenant_access_token_api")
	bodyBytes, err := w.postWithJson(ctx, url, jsonBytes, nil)
	if err != nil {
		w.logger.Error("获取令牌时发送HTTP请求失败", zap.String("operation", "token_exchange"))
		return "", 0, err
	}

	var responseData request.RobotTenantAccessTokenRes
	err = json.Unmarshal(bodyBytes, &responseData)
	if err != nil {
		w.logger.Error("解析响应JSON失败", zap.String("operation", "token_response_decode"))
		return "", 0, ErrTokenDecode
	}

	// 如果 API 返回错误码，记录错误并返回
	if responseData.Code != 0 {
		w.logger.Error("API返回错误", zap.String("operation", "token_provider"), zap.Int("provider_code", responseData.Code))
		return "", 0, &TokenProviderError{Code: responseData.Code}
	}

	if strings.TrimSpace(responseData.TenantAccessToken) == "" || responseData.Expire <= 0 || int64(responseData.Expire) > int64(time.Duration(1<<63-1)/time.Second) {
		w.logger.Error("令牌响应不可用", zap.String("operation", "token_response_validate"))
		return "", 0, ErrTokenInvalid
	}
	return responseData.TenantAccessToken, responseData.Expire, nil
}
