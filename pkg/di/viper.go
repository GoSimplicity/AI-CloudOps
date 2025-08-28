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

package di

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// InitViper 初始化viper配置，支持环境变量优先级：环境变量 > 配置文件 > 默认值
func InitViper() error {
	// 支持通过命令行参数 --config 指定任意配置文件
	configFile := pflag.String("config", "", "配置文件路径")
	pflag.Parse()

	// 如果未通过命令行指定，则根据环境变量ENV选择默认配置文件
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

	// 设置配置文件类型和路径
	viper.SetConfigFile(*configFile)

	// 设置默认值（最低优先级）
	setDefaults()

	// 启用环境变量支持
	viper.AutomaticEnv()

	// 将点号替换为下划线以支持嵌套配置的环境变量
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件（中等优先级）
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，只是打印警告，继续使用环境变量和默认值
		fmt.Printf("Warning: Failed to read config file %s: %v\n", *configFile, err)
		fmt.Println("Using environment variables and default values only.")
	}

	// 绑定环境变量（最高优先级）- 必须在读取配置文件之后
	bindEnvVars()

	// 加载配置到全局变量
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// 加载外部配置（仅环境变量）
	loadExternalConfig()

	return nil
}

func InitWebHookViper() {
	configFile := pflag.String("config", "config/webhook.yaml", "配置文件路径")
	pflag.Parse()
	viper.SetConfigFile(*configFile)

	// 设置webhook默认值（最低优先级）
	setWebhookDefaults()

	// 启用环境变量支持
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 读取配置文件（中等优先级）
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Warning: Failed to read webhook config file: %v\n", err)
		fmt.Println("Using environment variables and default values only.")
	}

	// 绑定webhook环境变量（最高优先级）
	bindWebhookEnvVars()
}

// setDefaults 设置所有配置的默认值
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8889")

	// Log defaults
	viper.SetDefault("log.dir", "./logs")
	viper.SetDefault("log.level", "debug")

	// JWT defaults
	viper.SetDefault("jwt.key1", "ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0l")
	viper.SetDefault("jwt.key2", "ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0z")
	viper.SetDefault("jwt.issuer", "K5mBPBYNQeNWEBvCTE5msog3KSGTdhmx")
	viper.SetDefault("jwt.expiration", 3600)

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")

	// MySQL defaults
	viper.SetDefault("mysql.addr", "root:root@tcp(localhost:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=Local")

	// Tree defaults
	viper.SetDefault("tree.check_status_cron", "@every 300s")
	viper.SetDefault("tree.password_encryption_key", "ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0l")

	// K8s defaults
	viper.SetDefault("k8s.refresh_cron", "@every 300s")

	// Prometheus defaults
	viper.SetDefault("prometheus.refresh_cron", "@every 15s")
	viper.SetDefault("prometheus.enable_alert", 0)
	viper.SetDefault("prometheus.enable_record", 0)
	viper.SetDefault("prometheus.alert_webhook_addr", "http://localhost:8889/api/v1/alerts/receive")
	viper.SetDefault("prometheus.alert_webhook_file_dir", "/tmp/webhook_files")
	viper.SetDefault("prometheus.httpSdAPI", "http://localhost:8888/api/not_auth/getTreeNodeBindIps")

	// Mock defaults
	viper.SetDefault("mock.enabled", true)

	// Notification Email defaults
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

	// Notification Feishu defaults
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

// setWebhookDefaults 设置Webhook默认值
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

// bindEnvVars 绑定环境变量
func bindEnvVars() {
	// 使用反射自动绑定所有配置项到环境变量
	bindStructEnvVars(reflect.TypeOf(Config{}), "")
}

// bindWebhookEnvVars 绑定Webhook环境变量
func bindWebhookEnvVars() {
	bindStructEnvVars(reflect.TypeOf(WebhookConfig{}), "webhook")
}

// bindStructEnvVars 递归绑定结构体中的环境变量
func bindStructEnvVars(t reflect.Type, prefix string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 获取mapstructure标签作为配置键
		mapstructureTag := field.Tag.Get("mapstructure")
		if mapstructureTag == "" {
			// 如果没有 mapstructure 标签，跳过这个字段
			continue
		}

		// 构建完整的配置键
		var configKey string
		if prefix == "" {
			configKey = mapstructureTag
		} else {
			configKey = prefix + "." + mapstructureTag
		}

		// 获取实际类型（处理指针类型）
		actualType := field.Type
		if actualType.Kind() == reflect.Ptr {
			actualType = actualType.Elem()
		}

		// 如果是嵌套结构体，递归处理
		if actualType.Kind() == reflect.Struct {
			bindStructEnvVars(actualType, configKey)
		} else {
			// 对于非结构体字段，绑定环境变量
			// 获取env标签作为环境变量名
			envTag := field.Tag.Get("env")
			if envTag != "" {
				viper.BindEnv(configKey, envTag)
			} else {
				// 如果没有env标签，使用配置键自动生成环境变量名
				envName := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
				viper.BindEnv(configKey, envName)
			}
		}
	}
}

// loadExternalConfig 加载外部配置（仅环境变量）
func loadExternalConfig() {
	GlobalExternalConfig.LLM.APIKey = os.Getenv("LLM_API_KEY")
	GlobalExternalConfig.LLM.BaseURL = os.Getenv("LLM_BASE_URL")
	GlobalExternalConfig.Aliyun.AccessKeyID = os.Getenv("ALIYUN_ACCESS_KEY_ID")
	GlobalExternalConfig.Aliyun.AccessKeySecret = os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	GlobalExternalConfig.Tavily.APIKey = os.Getenv("TAVILY_API_KEY")
}
