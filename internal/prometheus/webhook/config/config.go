package config

import (
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"os"
)

// AlertWebhookConfig Private告警Webhook的配置
type AlertWebhookConfig struct {
	HTTPAddr                      string        `yaml:"http_addr"`                           // HTTP 地址
	LogLevel                      string        `yaml:"log_level"`                           // 日志级别
	LogFilePath                   string        `yaml:"log_file_path"`                       // 日志文件路径
	AlertReceiveQueueSize         int           `yaml:"alert_receive_queue_size"`            // 告警接收队列大小
	CommonMapRenewIntervalSeconds int           `yaml:"common_map_renew_interval_seconds"`   // 通用映射刷新间隔（秒）
	MySQLConfig                   *mysql.Config `yaml:"mysql"`                               // MySQL 配置（使用 GORM 默认配置）
	HTTPRequestTimeoutSeconds     int           `yaml:"http_request_global_timeout_seconds"` // HTTP 请求超时（秒）
	AlertManagerAPI               string        `yaml:"alert_manager_api"`                   // 告警管理 API
	FrontDomain                   string        `yaml:"front_domain"`                        // 前端域名
	BackendDomain                 string        `yaml:"backend_domain"`                      // 后端域名
	HostName                      string        `yaml:"-"`                                   // 主机名（不在配置中）
	LocalIP                       string        `yaml:"-"`                                   // 本地 IP（不在配置中）
	IMFeishuConfig                *IMFeishu     `yaml:"im_feishu"`                           // 飞书 IM 配置
	Logger                        *zap.Logger   `yaml:"-"`                                   // Logger（不在配置中，运行时设置）
}

// IMFeishu Private飞书 IM 配置
type IMFeishu struct {
	GroupChatMessageAPI       string `yaml:"group_message_api"`        // 群聊消息 API
	PrivateChatMessageAPI     string `yaml:"private_message_api"`      // 私聊消息 API
	TenantAccessTokenAPI      string `yaml:"tenant_access_token_api"`  // 租户访问令牌 API
	PrivateChatRobotAppID     string `yaml:"private_robot_app_id"`     // 私聊机器人 App ID
	PrivateChatRobotAppSecret string `yaml:"private_robot_app_secret"` // 私聊机器人 App Secret
	RequestTimeoutSeconds     int    `yaml:"request_timeout_seconds"`  // 请求超时时间（秒）
}

// LoadAlertWebhook 加载并解析配置文件
func LoadAlertWebhook(filename string) (*AlertWebhookConfig, error) {
	cfg := &AlertWebhookConfig{}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(content, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
