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
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
)

type Config struct {
	AIOps        AIOpsConfig        `mapstructure:"aiops"`
	Server       ServerConfig       `mapstructure:"server"`
	Log          LogConfig          `mapstructure:"log"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Redis        RedisConfig        `mapstructure:"redis"`
	MySQL        MySQLConfig        `mapstructure:"mysql"`
	Tree         TreeConfig         `mapstructure:"tree"`
	K8s          K8sConfig          `mapstructure:"k8s"`
	Prometheus   PrometheusConfig   `mapstructure:"prometheus"`
	Mock         MockConfig         `mapstructure:"mock"`
	Notification NotificationConfig `mapstructure:"notification"`
	Webhook      WebhookConfig      `mapstructure:"webhook"`
}

func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("配置不能为空")
	}
	if c.Server.Port == "" {
		return fmt.Errorf("server.port 不能为空")
	}
	if c.Log.Dir == "" {
		return fmt.Errorf("log.dir 不能为空")
	}
	if c.MySQL.Addr == "" {
		return fmt.Errorf("mysql.addr 不能为空")
	}
	if err := c.MySQL.Validate(); err != nil {
		return err
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr 不能为空")
	}
	if c.AIOps.BaseURL != "" {
		u, err := url.Parse(c.AIOps.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return fmt.Errorf("aiops.base_url 必须是有效的 HTTP(S) 地址")
		}
	}
	// 失败即关闭：ENV 未设置/无法识别时同样执行生产校验（启动入口 Load 已先行拒绝这两种情况）
	if envmode.IsProduction() {
		// 按固定顺序检查（不用 map：map 遍历顺序随机，同样的输入每次会报不同的变量名）；错误信息只含变量名
		for _, k := range []struct{ name, value string }{
			{"JWT_KEY1", c.JWT.Key1},
			{"JWT_KEY2", c.JWT.Key2},
			{"TREE_PASSWORD_ENCRYPTION_KEY", c.Tree.PasswordEncryptionKey},
		} {
			if len(k.value) < 32 || strings.HasPrefix(k.value, "ebe3vx") {
				return fmt.Errorf("生产环境 %s 必须配置独立的至少32字符密钥", k.name)
			}
		}
		if len(c.Tree.PasswordEncryptionKey) != 32 {
			return fmt.Errorf("TREE_PASSWORD_ENCRYPTION_KEY 必须恰好32字节")
		}
		if c.JWT.Key1 == c.JWT.Key2 {
			return fmt.Errorf("JWT 两个密钥不得相同")
		}
		if c.Mock.Enabled {
			return fmt.Errorf("生产环境必须关闭 MOCK_ENABLED")
		}
		if c.AIOps.BaseURL != "" && len(c.AIOps.APIKey) < 32 {
			return fmt.Errorf("生产环境 AIOPS_API_KEY 至少32字符")
		}
	}
	return nil
}

type AIOpsConfig struct {
	BaseURL string `mapstructure:"base_url" env:"AIOPS_BASE_URL"`
	APIKey  string `mapstructure:"api_key" env:"AIOPS_API_KEY"`
}

type ServerConfig struct {
	AllowedOrigins string `mapstructure:"allowed_origins" env:"SERVER_ALLOWED_ORIGINS"`
	Port           string `mapstructure:"port" env:"SERVER_PORT" default:"8889"`
}

type LogConfig struct {
	Dir   string `mapstructure:"dir" env:"LOG_DIR" default:"./logs"`
	Level string `mapstructure:"level" env:"LOG_LEVEL" default:"debug"`
}

// JWTConfig JWT配置
type JWTConfig struct {
	Key1       string `mapstructure:"key1" env:"JWT_KEY1" default:"ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0l"`
	Key2       string `mapstructure:"key2" env:"JWT_KEY2" default:"ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0z"`
	Issuer     string `mapstructure:"issuer" env:"JWT_ISSUER" default:"K5mBPBYNQeNWEBvCTE5msog3KSGTdhmx"`
	Expiration int64  `mapstructure:"expiration" env:"JWT_EXPIRATION" default:"3600"`
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr" env:"REDIS_ADDR" default:"localhost:6379"`
	Password string `mapstructure:"password" env:"REDIS_PASSWORD" default:""`
}

// MySQLConfig MySQL配置
type MySQLConfig struct {
	MaxOpenConns    int           `mapstructure:"max_open_conns" env:"MYSQL_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" env:"MYSQL_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" env:"MYSQL_CONN_MAX_LIFETIME"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" env:"MYSQL_CONN_MAX_IDLE_TIME"`
	ConnectTimeout  time.Duration `mapstructure:"connect_timeout" env:"MYSQL_CONNECT_TIMEOUT"`
	StartupTimeout  time.Duration `mapstructure:"startup_timeout" env:"MYSQL_STARTUP_TIMEOUT"`
	Addr            string        `mapstructure:"addr" env:"MYSQL_ADDR" default:"root:root@tcp(localhost:3306)/cloudops?charset=utf8mb4&parseTime=True&loc=Local"`
}

type TreeConfig struct {
	CheckStatusCron       string `mapstructure:"check_status_cron" env:"TREE_CHECK_STATUS_CRON" default:"@every 300s"`
	PasswordEncryptionKey string `mapstructure:"password_encryption_key" env:"TREE_PASSWORD_ENCRYPTION_KEY" default:"ebe3vxIP7sblVvUHXb7ZaiMPuz4oXo0l"`
}

// K8sConfig Kubernetes配置
type K8sConfig struct {
	RefreshCron string `mapstructure:"refresh_cron" env:"K8S_REFRESH_CRON" default:"@every 300s"`
}

// PrometheusConfig Prometheus配置
type PrometheusConfig struct {
	RefreshCron         string `mapstructure:"refresh_cron" env:"PROMETHEUS_REFRESH_CRON" default:"@every 15s"`
	EnableAlert         int    `mapstructure:"enable_alert" env:"PROMETHEUS_ENABLE_ALERT" default:"0"`
	EnableRecord        int    `mapstructure:"enable_record" env:"PROMETHEUS_ENABLE_RECORD" default:"0"`
	AlertWebhookAddr    string `mapstructure:"alert_webhook_addr" env:"PROMETHEUS_ALERT_WEBHOOK_ADDR" default:"http://localhost:8889/api/v1/alerts/receive"`
	AlertWebhookFileDir string `mapstructure:"alert_webhook_file_dir" env:"PROMETHEUS_ALERT_WEBHOOK_FILE_DIR" default:"/tmp/webhook_files"`
	HttpSdAPI           string `mapstructure:"httpSdAPI" env:"PROMETHEUS_HTTP_SD_API" default:"http://localhost:8889/api/not_auth/getTreeNodeBindIps"`
}

// MockConfig Mock配置
type MockConfig struct {
	Enabled bool `mapstructure:"enabled" env:"MOCK_ENABLED" default:"true"`
}

type NotificationConfig struct {
	Email  *EmailConfig  `mapstructure:"email"`
	Feishu *FeishuConfig `mapstructure:"feishu"`
}

func (c *NotificationConfig) GetEmail() *EmailConfig {
	if c == nil {
		return nil
	}
	return c.Email
}

func (c *NotificationConfig) GetFeishu() *FeishuConfig {
	if c == nil {
		return nil
	}
	return c.Feishu
}

type EmailConfig struct {
	Enabled       bool   `mapstructure:"enabled" env:"NOTIFICATION_EMAIL_ENABLED" default:"false"`
	SMTPHost      string `mapstructure:"smtp_host" env:"NOTIFICATION_EMAIL_SMTP_HOST" default:"smtp.gmail.com"`
	SMTPPort      int    `mapstructure:"smtp_port" env:"NOTIFICATION_EMAIL_SMTP_PORT" default:"587"`
	Username      string `mapstructure:"username" env:"NOTIFICATION_EMAIL_USERNAME" default:""`
	Password      string `mapstructure:"password" env:"NOTIFICATION_EMAIL_PASSWORD" default:""`
	FromName      string `mapstructure:"from_name" env:"NOTIFICATION_EMAIL_FROM_NAME" default:"AI-CloudOps"`
	MaxRetries    int    `mapstructure:"max_retries" env:"NOTIFICATION_EMAIL_MAX_RETRIES" default:"3"`
	RetryInterval string `mapstructure:"retry_interval" env:"NOTIFICATION_EMAIL_RETRY_INTERVAL" default:"5m"`
	Timeout       string `mapstructure:"timeout" env:"NOTIFICATION_EMAIL_TIMEOUT" default:"30s"`
	UseTLS        bool   `mapstructure:"use_tls" env:"NOTIFICATION_EMAIL_USE_TLS" default:"true"`
}

func (c *EmailConfig) IsEnabled() bool {
	if c == nil {
		return false
	}
	return c.Enabled
}

func (c *EmailConfig) GetMaxRetries() int {
	if c == nil {
		return 3
	}
	if c.MaxRetries <= 0 {
		return 3
	}
	return c.MaxRetries
}

func (c *EmailConfig) GetRetryInterval() time.Duration {
	if c == nil || c.RetryInterval == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(c.RetryInterval)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}

func (c *EmailConfig) GetTimeout() time.Duration {
	if c == nil || c.Timeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
}

func (c *EmailConfig) GetChannelName() string {
	return "email"
}

func (c *EmailConfig) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	if c.SMTPHost == "" {
		return fmt.Errorf("邮件 SMTPHost 不能为空")
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return fmt.Errorf("邮件 SMTPPort 无效: %d", c.SMTPPort)
	}
	if c.Username == "" {
		return fmt.Errorf("邮件 Username 不能为空")
	}
	if c.Password == "" {
		return fmt.Errorf("邮件 Password 不能为空")
	}
	if strings.Contains(c.Username, "xxx@") || c.Password == "xxx" {
		return fmt.Errorf("邮件通知配置仍为占位符，请替换为真实授权信息")
	}
	return nil
}

func (c *EmailConfig) GetSMTPHost() string {
	if c == nil {
		return ""
	}
	return c.SMTPHost
}

func (c *EmailConfig) GetSMTPPort() int {
	if c == nil {
		return 0
	}
	return c.SMTPPort
}

func (c *EmailConfig) GetUsername() string {
	if c == nil {
		return ""
	}
	return c.Username
}

func (c *EmailConfig) GetPassword() string {
	if c == nil {
		return ""
	}
	return c.Password
}

func (c *EmailConfig) GetFromName() string {
	if c == nil || c.FromName == "" {
		return "AI-CloudOps"
	}
	return c.FromName
}

func (c *EmailConfig) GetUseTLS() bool {
	if c == nil {
		return false
	}
	return c.UseTLS
}

type FeishuConfig struct {
	Enabled              bool   `mapstructure:"enabled" env:"NOTIFICATION_FEISHU_ENABLED" default:"false"`
	AppID                string `mapstructure:"app_id" env:"NOTIFICATION_FEISHU_APP_ID" default:""`
	AppSecret            string `mapstructure:"app_secret" env:"NOTIFICATION_FEISHU_APP_SECRET" default:""`
	WebhookURL           string `mapstructure:"webhook_url" env:"NOTIFICATION_FEISHU_WEBHOOK_URL" default:"https://open.feishu.cn/open-apis/bot/v2/hook/"`
	PrivateMessageAPI    string `mapstructure:"private_message_api" env:"NOTIFICATION_FEISHU_PRIVATE_MESSAGE_API" default:"https://open.feishu.cn/open-apis/im/v1/messages"`
	TenantAccessTokenAPI string `mapstructure:"tenant_access_token_api" env:"NOTIFICATION_FEISHU_TENANT_ACCESS_TOKEN_API" default:"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"`
	MaxRetries           int    `mapstructure:"max_retries" env:"NOTIFICATION_FEISHU_MAX_RETRIES" default:"3"`
	RetryInterval        string `mapstructure:"retry_interval" env:"NOTIFICATION_FEISHU_RETRY_INTERVAL" default:"5m"`
	Timeout              string `mapstructure:"timeout" env:"NOTIFICATION_FEISHU_TIMEOUT" default:"10s"`
}

func (c *FeishuConfig) IsEnabled() bool {
	if c == nil {
		return false
	}
	return c.Enabled
}

func (c *FeishuConfig) GetMaxRetries() int {
	if c == nil {
		return 3
	}
	if c.MaxRetries <= 0 {
		return 3
	}
	return c.MaxRetries
}

func (c *FeishuConfig) GetRetryInterval() time.Duration {
	if c == nil || c.RetryInterval == "" {
		return 5 * time.Minute
	}
	d, err := time.ParseDuration(c.RetryInterval)
	if err != nil {
		return 5 * time.Minute
	}
	return d
}

func (c *FeishuConfig) GetTimeout() time.Duration {
	if c == nil || c.Timeout == "" {
		return 10 * time.Second
	}
	d, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return 10 * time.Second
	}
	return d
}

func (c *FeishuConfig) GetChannelName() string {
	return "feishu"
}

func (c *FeishuConfig) Validate() error {
	if c == nil || !c.Enabled {
		return nil
	}
	if c.AppID == "" {
		return fmt.Errorf("飞书 AppID 不能为空")
	}
	if c.AppSecret == "" {
		return fmt.Errorf("飞书 AppSecret 不能为空")
	}
	if c.WebhookURL == "" {
		return fmt.Errorf("飞书 WebhookURL 不能为空")
	}
	if c.PrivateMessageAPI == "" {
		return fmt.Errorf("飞书 PrivateMessageAPI 不能为空")
	}
	if c.TenantAccessTokenAPI == "" {
		return fmt.Errorf("飞书 TenantAccessTokenAPI 不能为空")
	}
	if c.AppID == "xxx" || c.AppSecret == "xxx" {
		return fmt.Errorf("飞书通知配置仍为占位符，请替换为真实凭据")
	}
	return nil
}

func (c *FeishuConfig) GetAppID() string {
	if c == nil {
		return ""
	}
	return c.AppID
}

func (c *FeishuConfig) GetAppSecret() string {
	if c == nil {
		return ""
	}
	return c.AppSecret
}

func (c *FeishuConfig) GetWebhookURL() string {
	if c == nil {
		return ""
	}
	return c.WebhookURL
}

func (c *FeishuConfig) GetPrivateMessageAPI() string {
	if c == nil {
		return ""
	}
	return c.PrivateMessageAPI
}

func (c *FeishuConfig) GetTenantAccessTokenAPI() string {
	if c == nil {
		return ""
	}
	return c.TenantAccessTokenAPI
}

// WebhookConfig Webhook配置（用于webhook子系统）
type WebhookConfig struct {
	Port                          string         `mapstructure:"port" env:"WEBHOOK_PORT" default:"8888"`
	FixedWorkers                  int            `mapstructure:"fixed_workers" env:"WEBHOOK_FIXED_WORKERS" default:"10"`
	FrontDomain                   string         `mapstructure:"front_domain" env:"WEBHOOK_FRONT_DOMAIN" default:"http://localhost:3000"`
	BackendDomain                 string         `mapstructure:"backend_domain" env:"WEBHOOK_BACKEND_DOMAIN" default:"http://localhost:8889"`
	DefaultUpgradeMinutes         int            `mapstructure:"default_upgrade_minutes" env:"WEBHOOK_DEFAULT_UPGRADE_MINUTES" default:"60"`
	AlertManagerAPI               string         `mapstructure:"alert_manager_api" env:"WEBHOOK_ALERT_MANAGER_API" default:"http://localhost:9093"`
	CommonMapRenewIntervalSeconds int            `mapstructure:"common_map_renew_interval_seconds" env:"WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS" default:"300"`
	ImFeishu                      ImFeishuConfig `mapstructure:"im_feishu"`
}

type ImFeishuConfig struct {
	GroupMessageAPI       string `mapstructure:"group_message_api" env:"WEBHOOK_IM_FEISHU_GROUP_MESSAGE_API" default:"https://open.feishu.cn/open-apis/im/v1/messages"`
	RequestTimeoutSeconds int    `mapstructure:"request_timeout_seconds" env:"WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS" default:"10"`
	PrivateRobotAppID     string `mapstructure:"private_robot_app_id" env:"WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_ID" default:""`
	PrivateRobotAppSecret string `mapstructure:"private_robot_app_secret" env:"WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET" default:""`
	TenantAccessTokenAPI  string `mapstructure:"tenant_access_token_api" env:"WEBHOOK_IM_FEISHU_TENANT_ACCESS_TOKEN_API" default:"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal"`
}

// LLMConfig LLM配置（来自环境变量）
type LLMConfig struct {
	APIKey  string `env:"LLM_API_KEY" default:""`
	BaseURL string `env:"LLM_BASE_URL" default:""`
}

// AliyunConfig 阿里云配置（来自环境变量）
type AliyunConfig struct {
	AccessKeyID     string `env:"ALIYUN_ACCESS_KEY_ID" default:""`
	AccessKeySecret string `env:"ALIYUN_ACCESS_KEY_SECRET" default:""`
}

// TavilyConfig Tavily配置（来自环境变量）
type TavilyConfig struct {
	APIKey string `env:"TAVILY_API_KEY" default:""`
}

// ExternalConfig 外部服务配置（仅来自环境变量）
type ExternalConfig struct {
	LLM    LLMConfig    `mapstructure:"llm"`
	Aliyun AliyunConfig `mapstructure:"aliyun"`
	Tavily TavilyConfig `mapstructure:"tavily"`
}

// WithDefaults 为程序内直接构造及省略的配置提供有限资源边界
func (c MySQLConfig) WithDefaults() MySQLConfig {
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 20
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 5
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 5 * time.Minute
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	if c.StartupTimeout == 0 {
		c.StartupTimeout = 10 * time.Second
	}
	return c
}

func (c MySQLConfig) Validate() error {
	c = c.WithDefaults()
	if c.MaxOpenConns < 1 || c.MaxOpenConns > 10000 {
		return fmt.Errorf("mysql.max_open_conns 必须在1至10000之间")
	}
	if c.MaxIdleConns < 1 || c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("mysql.max_idle_conns 必须在1至max_open_conns之间")
	}
	if c.ConnMaxLifetime < 0 || c.ConnMaxIdleTime < 0 || c.ConnectTimeout < 0 || c.StartupTimeout < 0 {
		return fmt.Errorf("MySQL连接期限必须为正数，零值使用默认配置")
	}
	return nil
}
