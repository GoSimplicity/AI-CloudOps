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
package notification

import (
	"time"

	"github.com/spf13/viper"
)

// LoadNotificationConfig 从配置文件加载通知配置
func LoadNotificationConfig() (*NotificationConfig, error) {
	config := &NotificationConfig{}

	// 加载邮箱配置
	if viper.IsSet("notification.email") {
		emailConfig := &EmailConfig{
			BaseChannelConfig: BaseChannelConfig{
				Enabled:       viper.GetBool("notification.email.enabled"),
				MaxRetries:    viper.GetInt("notification.email.max_retries"),
				RetryInterval: parseDuration(viper.GetString("notification.email.retry_interval"), 5*time.Minute),
				Timeout:       parseDuration(viper.GetString("notification.email.timeout"), 30*time.Second),
			},
			SMTPHost: viper.GetString("notification.email.smtp_host"),
			SMTPPort: viper.GetInt("notification.email.smtp_port"),
			Username: viper.GetString("notification.email.username"),
			Password: viper.GetString("notification.email.password"),
			FromName: viper.GetString("notification.email.from_name"),
			UseTLS:   viper.GetBool("notification.email.use_tls"),
		}

		// 设置默认值
		if emailConfig.MaxRetries == 0 {
			emailConfig.MaxRetries = 3
		}
		if emailConfig.FromName == "" {
			emailConfig.FromName = "AI-CloudOps"
		}

		config.Email = emailConfig
	}

	// 加载飞书配置
	if viper.IsSet("notification.feishu") {
		feishuConfig := &FeishuConfig{
			BaseChannelConfig: BaseChannelConfig{
				Enabled:       viper.GetBool("notification.feishu.enabled"),
				MaxRetries:    viper.GetInt("notification.feishu.max_retries"),
				RetryInterval: parseDuration(viper.GetString("notification.feishu.retry_interval"), 5*time.Minute),
				Timeout:       parseDuration(viper.GetString("notification.feishu.timeout"), 10*time.Second),
			},
			AppID:                viper.GetString("notification.feishu.app_id"),
			AppSecret:            viper.GetString("notification.feishu.app_secret"),
			WebhookURL:           viper.GetString("notification.feishu.webhook_url"),
			PrivateMessageAPI:    viper.GetString("notification.feishu.private_message_api"),
			TenantAccessTokenAPI: viper.GetString("notification.feishu.tenant_access_token_api"),
		}

		// 设置默认值
		if feishuConfig.MaxRetries == 0 {
			feishuConfig.MaxRetries = 3
		}

		config.Feishu = feishuConfig
	}

	return config, nil
}

// parseDuration 解析时间间隔，失败时返回默认值
func parseDuration(s string, defaultDuration time.Duration) time.Duration {
	if s == "" {
		return defaultDuration
	}
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return defaultDuration
}

// GetDefaultEmailConfig 获取默认邮箱配置
func GetDefaultEmailConfig() *EmailConfig {
	return &EmailConfig{
		BaseChannelConfig: BaseChannelConfig{
			Enabled:       false,
			MaxRetries:    3,
			RetryInterval: 5 * time.Minute,
			Timeout:       30 * time.Second,
		},
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		FromName: "AI-CloudOps",
		UseTLS:   true,
	}
}

// GetDefaultFeishuConfig 获取默认飞书配置
func GetDefaultFeishuConfig() *FeishuConfig {
	return &FeishuConfig{
		BaseChannelConfig: BaseChannelConfig{
			Enabled:       false,
			MaxRetries:    3,
			RetryInterval: 5 * time.Minute,
			Timeout:       10 * time.Second,
		},
		WebhookURL:           "https://open.feishu.cn/open-apis/bot/v2/hook/",
		PrivateMessageAPI:    "https://open.feishu.cn/open-apis/im/v1/messages",
		TenantAccessTokenAPI: "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
	}
}

// MergeWithDefaults 将配置与默认值合并
func MergeWithDefaults(config *NotificationConfig) *NotificationConfig {
	if config == nil {
		config = &NotificationConfig{}
	}

	result := &NotificationConfig{}

	// 合并邮箱配置
	if config.Email != nil {
		result.Email = config.Email
	} else {
		result.Email = GetDefaultEmailConfig()
	}

	// 合并飞书配置
	if config.Feishu != nil {
		result.Feishu = config.Feishu
	} else {
		result.Feishu = GetDefaultFeishuConfig()
	}

	return result
}

// ConfigSummary 配置摘要
type ConfigSummary struct {
	Email  *ChannelSummary `json:"email,omitempty"`
	Feishu *ChannelSummary `json:"feishu,omitempty"`
}

// ChannelSummary 渠道配置摘要
type ChannelSummary struct {
	Enabled       bool   `json:"enabled"`
	MaxRetries    int    `json:"max_retries"`
	RetryInterval string `json:"retry_interval"`
	Timeout       string `json:"timeout"`
	Status        string `json:"status"` // "ok", "error", "disabled"
	ErrorMessage  string `json:"error_message,omitempty"`
}

// GetConfigSummary 获取配置摘要
func GetConfigSummary(config *NotificationConfig) *ConfigSummary {
	if config == nil {
		return &ConfigSummary{}
	}

	summary := &ConfigSummary{}

	// 邮箱配置摘要
	if config.Email != nil {
		emailSummary := &ChannelSummary{
			Enabled:       config.Email.IsEnabled(),
			MaxRetries:    config.Email.GetMaxRetries(),
			RetryInterval: config.Email.GetRetryInterval().String(),
			Timeout:       config.Email.GetTimeout().String(),
		}

		if config.Email.IsEnabled() {
			if err := config.Email.Validate(); err != nil {
				emailSummary.Status = "error"
				emailSummary.ErrorMessage = err.Error()
			} else {
				emailSummary.Status = "ok"
			}
		} else {
			emailSummary.Status = "disabled"
		}

		summary.Email = emailSummary
	}

	// 飞书配置摘要
	if config.Feishu != nil {
		feishuSummary := &ChannelSummary{
			Enabled:       config.Feishu.IsEnabled(),
			MaxRetries:    config.Feishu.GetMaxRetries(),
			RetryInterval: config.Feishu.GetRetryInterval().String(),
			Timeout:       config.Feishu.GetTimeout().String(),
		}

		if config.Feishu.IsEnabled() {
			if err := config.Feishu.Validate(); err != nil {
				feishuSummary.Status = "error"
				feishuSummary.ErrorMessage = err.Error()
			} else {
				feishuSummary.Status = "ok"
			}
		} else {
			feishuSummary.Status = "disabled"
		}

		summary.Feishu = feishuSummary
	}

	return summary
}
