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

package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// NonFatalError 用于标记“可继续运行”的告警类错误（例如配置文件不存在）。
type NonFatalError struct {
	err error
}

func (e NonFatalError) Error() string {
	return e.err.Error()
}

func (e NonFatalError) Unwrap() error {
	return e.err
}

func IsNonFatal(err error) bool {
	var nf NonFatalError
	return errors.As(err, &nf)
}

// Load 加载主程序配置，支持优先级：环境变量 > 配置文件 > 默认值。
func Load() (*Config, *ExternalConfig, error) {
	configFile := pflag.String("config", "", "配置文件路径")
	pflag.Parse()

	if *configFile == "" {
		env := os.Getenv("ENV")
		if env == "" {
			env = "development"
		}
		switch env {
		case "production":
			*configFile = "config/config.production.yaml"
		default:
			*configFile = "config/config.development.yaml"
		}
	}

	return loadFromFile(*configFile)
}

// loadFromFile 按“环境变量 > 配置文件 > 默认值”加载指定配置文件（不解析命令行参数，便于测试）
func loadFromFile(configFile string) (*Config, *ExternalConfig, error) {
	viper.SetConfigFile(configFile)
	setDefaults()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	readErr := viper.ReadInConfig()
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("读取配置文件失败: %w", readErr)
	}

	bindEnvVars()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, nil, fmt.Errorf("解析配置失败: %w", err)
	}

	ext := &ExternalConfig{}
	loadExternalConfig(ext)

	if err := cfg.Validate(); err != nil {
		return cfg, ext, fmt.Errorf("配置校验失败: %w", err)
	}

	if readErr != nil {
		return cfg, ext, NonFatalError{err: fmt.Errorf("读取配置文件失败: %w", readErr)}
	}

	return cfg, ext, nil
}

// LoadWebhook 加载 Webhook 子系统配置，支持优先级：环境变量 > 配置文件 > 默认值。
func LoadWebhook() (*WebhookConfig, error) {
	configFile := pflag.String("config", "config/webhook.yaml", "配置文件路径")
	pflag.Parse()

	viper.SetConfigFile(*configFile)
	setWebhookDefaults()
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	readErr := viper.ReadInConfig()
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, fmt.Errorf("读取Webhook配置文件失败: %w", readErr)
	}

	bindWebhookEnvVars()

	cfg := &WebhookConfig{}
	if err := viper.UnmarshalKey("webhook", cfg); err != nil {
		// 兼容旧配置：如果没有 webhook 根节点，则尝试直接 unmarshal
		if err2 := viper.Unmarshal(cfg); err2 != nil {
			return nil, fmt.Errorf("解析Webhook配置失败: %w", err)
		}
	}

	if readErr != nil {
		return cfg, NonFatalError{err: fmt.Errorf("读取Webhook配置文件失败: %w", readErr)}
	}
	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("server.port", "8889")

	viper.SetDefault("log.dir", "./logs")
	viper.SetDefault("log.level", "debug")

	// jwt.key1/key2/issuer、mysql.addr、tree.password_encryption_key 无内置默认值，
	// 必须通过环境变量注入，缺失时 Config.Validate 直接失败（见 requiredSecretEnvVars）
	viper.SetDefault("jwt.expiration", 3600)

	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")

	viper.SetDefault("mysql.max_open_conns", 20)
	viper.SetDefault("mysql.max_idle_conns", 5)
	viper.SetDefault("mysql.conn_max_lifetime", "30m")
	viper.SetDefault("mysql.conn_max_idle_time", "5m")
	viper.SetDefault("mysql.connect_timeout", "5s")
	viper.SetDefault("mysql.startup_timeout", "10s")

	viper.SetDefault("tree.check_status_cron", "@every 300s")

	viper.SetDefault("k8s.refresh_cron", "@every 300s")

	viper.SetDefault("prometheus.refresh_cron", "@every 15s")
	viper.SetDefault("prometheus.enable_alert", 0)
	viper.SetDefault("prometheus.enable_record", 0)
	viper.SetDefault("prometheus.alert_webhook_addr", "http://localhost:8889/api/v1/alerts/receive")
	viper.SetDefault("prometheus.alert_webhook_file_dir", "/tmp/webhook_files")
	viper.SetDefault("prometheus.httpSdAPI", "http://localhost:8889/api/not_auth/getTreeNodeBindIps")

	viper.SetDefault("mock.enabled", true)

	viper.SetDefault("notification.email.enabled", false)
	viper.SetDefault("notification.email.smtp_host", "smtp.gmail.com")
	viper.SetDefault("notification.email.smtp_port", 587)
	viper.SetDefault("notification.email.username", "")
	viper.SetDefault("notification.email.password", "")
	viper.SetDefault("notification.email.from_name", "AI-CloudOps")
	viper.SetDefault("notification.email.max_retries", 3)
	viper.SetDefault("notification.email.retry_interval", "5m")
	viper.SetDefault("notification.email.timeout", "30s")
	viper.SetDefault("notification.email.use_tls", true)

	viper.SetDefault("notification.feishu.enabled", false)
	viper.SetDefault("notification.feishu.app_id", "")
	viper.SetDefault("notification.feishu.app_secret", "")
	viper.SetDefault("notification.feishu.webhook_url", "https://open.feishu.cn/open-apis/bot/v2/hook/")
	viper.SetDefault("notification.feishu.private_message_api", "https://open.feishu.cn/open-apis/im/v1/messages")
	viper.SetDefault("notification.feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
	viper.SetDefault("notification.feishu.max_retries", 3)
	viper.SetDefault("notification.feishu.retry_interval", "5m")
	viper.SetDefault("notification.feishu.timeout", "10s")
}

func setWebhookDefaults() {
	viper.SetDefault("webhook.port", "8888")
	viper.SetDefault("webhook.fixed_workers", 10)
	viper.SetDefault("webhook.front_domain", "http://localhost:3000")
	viper.SetDefault("webhook.backend_domain", "http://localhost:8889")
	viper.SetDefault("webhook.default_upgrade_minutes", 60)
	viper.SetDefault("webhook.alert_manager_api", "http://localhost:9093")
	viper.SetDefault("webhook.common_map_renew_interval_seconds", 300)
	viper.SetDefault("webhook.im_feishu.group_message_api", "https://open.feishu.cn/open-apis/im/v1/messages")
	viper.SetDefault("webhook.im_feishu.request_timeout_seconds", 10)
	viper.SetDefault("webhook.im_feishu.private_robot_app_id", "")
	viper.SetDefault("webhook.im_feishu.private_robot_app_secret", "")
	viper.SetDefault("webhook.im_feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
}

func bindEnvVars() {
	bindStructEnvVars(reflect.TypeOf(Config{}), "")
}

func bindWebhookEnvVars() {
	bindStructEnvVars(reflect.TypeOf(WebhookConfig{}), "webhook")
}

func bindStructEnvVars(t reflect.Type, prefix string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag == "" {
			continue
		}

		var configKey string
		if prefix == "" {
			configKey = mapstructureTag
		} else {
			configKey = prefix + "." + mapstructureTag
		}

		actualType := field.Type
		if actualType.Kind() == reflect.Ptr {
			actualType = actualType.Elem()
		}

		if actualType.Kind() == reflect.Struct {
			bindStructEnvVars(actualType, configKey)
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag != "" {
			_ = viper.BindEnv(configKey, envTag)
			continue
		}

		envName := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
		_ = viper.BindEnv(configKey, envName)
	}
}

func loadExternalConfig(ext *ExternalConfig) {
	if ext == nil {
		return
	}
	ext.LLM.APIKey = os.Getenv("LLM_API_KEY")
	ext.LLM.BaseURL = os.Getenv("LLM_BASE_URL")
	ext.Aliyun.AccessKeyID = os.Getenv("ALIYUN_ACCESS_KEY_ID")
	ext.Aliyun.AccessKeySecret = os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	ext.Tavily.APIKey = os.Getenv("TAVILY_API_KEY")
}
