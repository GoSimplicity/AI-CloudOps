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
	"sort"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/pkg/runenv"
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
		// 与 Config.Validate 使用同一判断（去首尾空白、忽略大小写），避免选了开发配置却按生产规则校验
		if runenv.IsProduction() {
			*configFile = "config/config.production.yaml"
		} else {
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

// WebhookSecrets Webhook 进程的敏感配置，仅保存在返回值中，不写入全局 viper
type WebhookSecrets struct {
	// MySQLAddr 来自 WEBHOOK_MYSQL_ADDR 的 MySQL DSN（含账号密码）
	MySQLAddr string
}

// LoadWebhook 加载 Webhook 子系统配置，支持优先级：环境变量 > 配置文件 > 默认值。
func LoadWebhook() (*WebhookConfig, *WebhookSecrets, error) {
	configFile := pflag.String("config", "config/webhook.yaml", "配置文件路径")
	pflag.Parse()

	return loadWebhookFromFile(*configFile)
}

// WebhookMySQLAddrEnv Webhook 进程的 MySQL DSN（含账号密码）环境变量，必需，无内置默认值
const WebhookMySQLAddrEnv = "WEBHOOK_MYSQL_ADDR"

// webhookMySQLAddrKey webhook.yaml 中仅保留为空占位的 DSN 键，其值从不被读取
const webhookMySQLAddrKey = "mysql.addr"

// webhookGlobalPrefix 发布到全局 viper 的键前缀：webhook 子系统其余代码仍通过全局 viper 读取 webhook.*
const webhookGlobalPrefix = "webhook."

// loadWebhookFromFile 加载指定的 Webhook 配置文件（不解析命令行参数，便于测试）。
// 使用独立的 viper 实例读取文件与环境变量：DSN 只经由返回的 WebhookSecrets 传给数据库初始化，
// 不写入全局 viper（mysql.addr 不会出现在全局配置中），也不受主程序 MYSQL_ADDR 影响
func loadWebhookFromFile(configFile string) (*WebhookConfig, *WebhookSecrets, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	setWebhookDefaults(v)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	readErr := v.ReadInConfig()
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return nil, nil, fmt.Errorf("读取Webhook配置文件失败: %w", readErr)
	}

	// 不再支持没有 webhook 根节点的旧平铺格式：这些键只会被忽略、静默回落到默认值（端口可能变化），
	// 所以直接失败并提示移到 webhook: 下
	if misplaced := misplacedWebhookRootKeys(v); len(misplaced) > 0 {
		// 只列出键名，不输出值
		return nil, nil, fmt.Errorf("Webhook 配置文件根节点下出现了 webhook 字段 %s：不再支持没有 webhook 根节点的旧格式，请把这些键移到 webhook: 下",
			strings.Join(misplaced, ", "))
	}

	bindStructEnvVars(v, reflect.TypeOf(WebhookConfig{}), "webhook")

	// SEC-02A/SEC-22：DSN 整体由 WEBHOOK_MYSQL_ADDR 注入，配置文件不写、无默认值，缺失即失败
	addr := strings.TrimSpace(os.Getenv(WebhookMySQLAddrEnv))
	if addr == "" {
		// 只列出变量名，不输出任何值
		return nil, nil, fmt.Errorf("缺少必需的密钥环境变量: %s（不提供内置默认值，请通过环境变量注入）", WebhookMySQLAddrEnv)
	}

	// 逐键取合并后的值（环境变量 > 配置文件 > 默认值）。v.UnmarshalKey("webhook") 只拿到配置文件里的
	// webhook 子树，环境变量和默认值进不了 cfg（WEBHOOK_PORT 因此从未生效），所以不能直接用。
	// 只构造这一次：解码 cfg 与发布到全局 viper 共用同一份、同一顺序
	merged := collectWebhookLeafValues(v)
	rawPort, hasPort := merged.vals[webhookPortKey]
	port := strings.TrimSpace(fmt.Sprint(rawPort))
	if !hasPort || rawPort == nil || !isValidPort(port) {
		// 只列出键名和变量名，不输出值
		return nil, nil, fmt.Errorf("Webhook 端口 %s（环境变量 WEBHOOK_PORT）必须是 1-65535 的纯数字", webhookPortKey)
	}
	// 去掉前导零后的规范值（如 000080 → 80）同时用于监听和全局 webhook.port
	merged.vals[webhookPortKey] = normalizePort(port)

	cfg, err := decodeWebhookConfig(merged)
	if err != nil {
		return nil, nil, err
	}

	// 只把 webhook.* 发布到全局 viper，供 webhook 子系统现有读取方使用；mysql.* 等其他键不发布。
	// 与 cfg 使用同一份合并后的值
	merged.apply(viper.Set)

	secrets := &WebhookSecrets{MySQLAddr: addr}
	if readErr != nil {
		return cfg, secrets, NonFatalError{err: fmt.Errorf("读取Webhook配置文件失败: %w", readErr)}
	}
	return cfg, secrets, nil
}

// webhookPortKey Webhook 监听端口的配置键
const webhookPortKey = "webhook.port"

// misplacedWebhookRootKeys 返回配置文件根节点下出现的 WebhookConfig 字段键名（如 port、fixed_workers），
// 按结构体字段顺序。只看配置文件本身（InConfig），不受环境变量（如 PORT）和默认值影响
func misplacedWebhookRootKeys(v *viper.Viper) []string {
	var keys []string
	t := reflect.TypeOf(WebhookConfig{})
	for i := 0; i < t.NumField(); i++ {
		key := t.Field(i).Tag.Get("mapstructure")
		if key != "" && v.InConfig(key) {
			keys = append(keys, key)
		}
	}
	return keys
}

// webhookValues 逐键合并后的 webhook.* 叶子键取值；keys 已排序，写入顺序确定
type webhookValues struct {
	keys []string
	vals map[string]any
}

// apply 按 keys 的固定顺序逐个写入（tmp.Set / viper.Set 共用）
func (m webhookValues) apply(set func(key string, val any)) {
	for _, key := range m.keys {
		set(key, m.vals[key])
	}
}

// collectWebhookLeafValues 取 webhook.* 下每个叶子键合并后的值（环境变量 > 配置文件 > 默认值）。
// 跳过父键：配置文件写成 `im_feishu:`（null）时 AllKeys 会同时包含 webhook.im_feishu 和它的子键，
// 父键 Get 得到的是默认值组成的 map，若写入顺序在子键之后会把子键的环境变量值覆盖掉。
// 判定父键用两条：还有以 "key." 开头的其他键，或取值本身是 map
func collectWebhookLeafValues(v *viper.Viper) webhookValues {
	var all []string
	for _, key := range v.AllKeys() {
		if strings.HasPrefix(key, webhookGlobalPrefix) {
			all = append(all, key)
		}
	}
	sort.Strings(all)

	m := webhookValues{vals: make(map[string]any, len(all))}
	for _, key := range all {
		if hasDescendantKey(all, key) {
			continue
		}
		val := v.Get(key)
		if isMapValue(val) {
			continue
		}
		m.keys = append(m.keys, key)
		m.vals[key] = val
	}
	return m
}

// hasDescendantKey 判断 keys 中是否存在 key 的子键
func hasDescendantKey(keys []string, key string) bool {
	prefix := key + "."
	for _, k := range keys {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func isMapValue(val any) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Map
}

// decodeWebhookConfig 把逐键合并后的 webhook.* 值解码为 WebhookConfig。
// 借一个空白 viper 实例承载这些键再 UnmarshalKey，沿用 viper 默认的解码设置
// （WeaklyTypedInput 与 decode hook），字符串 "3" 可转为 int；不新增直接依赖
func decodeWebhookConfig(merged webhookValues) (*WebhookConfig, error) {
	tmp := viper.New()
	merged.apply(tmp.Set)
	cfg := &WebhookConfig{}
	if err := tmp.UnmarshalKey("webhook", cfg); err != nil {
		return nil, fmt.Errorf("解析Webhook配置失败: %w", err)
	}
	return cfg, nil
}

// isValidPort 端口必须是纯 ASCII 数字且在 1-65535 之间（调用方先去首尾空白）。
// 先去掉前导零再查长度和范围：000080 视为 80；全是零（0、0000）仍拒绝
func isValidPort(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	t := strings.TrimLeft(s, "0")
	if t == "" || len(t) > 5 {
		return false
	}
	n := 0
	for _, r := range t {
		n = n*10 + int(r-'0')
	}
	return n >= 1 && n <= 65535
}

// normalizePort 去掉合法端口的前导零（000080 → 80），只在 isValidPort 通过后调用
func normalizePort(s string) string {
	return strings.TrimLeft(s, "0")
}

func setDefaults() {
	viper.SetDefault("server.port", "8889")

	viper.SetDefault("log.dir", "./logs")
	viper.SetDefault("log.level", "debug")

	// jwt.key1/key2、mysql.addr、tree.password_encryption_key 无内置默认值，
	// 必须通过环境变量注入，缺失时 Config.Validate 直接失败（见 requiredSecretEnvVars）。
	// jwt.issuer 不是密钥，取自 config/*.yaml，可用环境变量 JWT_ISSUER 覆盖
	viper.SetDefault("jwt.expiration", 30) // 单位：分钟

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

	viper.SetDefault(CronAllowLocalExecKey, false)

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

func setWebhookDefaults(v *viper.Viper) {
	v.SetDefault("webhook.port", "8889")
	v.SetDefault("webhook.fixed_workers", 10)
	v.SetDefault("webhook.front_domain", "http://localhost:3000")
	v.SetDefault("webhook.backend_domain", "http://localhost:8889")
	v.SetDefault("webhook.default_upgrade_minutes", 60)
	v.SetDefault("webhook.alert_manager_api", "http://localhost:9093")
	v.SetDefault("webhook.common_map_renew_interval_seconds", 300)
	v.SetDefault("webhook.im_feishu.group_message_api", "https://open.feishu.cn/open-apis/im/v1/messages")
	v.SetDefault("webhook.im_feishu.request_timeout_seconds", 10)
	v.SetDefault("webhook.im_feishu.private_robot_app_id", "")
	v.SetDefault("webhook.im_feishu.private_robot_app_secret", "")
	v.SetDefault("webhook.im_feishu.tenant_access_token_api", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal")
}

func bindEnvVars() {
	bindStructEnvVars(viper.GetViper(), reflect.TypeOf(Config{}), "")
}

func bindStructEnvVars(v *viper.Viper, t reflect.Type, prefix string) {
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
			bindStructEnvVars(v, actualType, configKey)
			continue
		}

		envTag := field.Tag.Get("env")
		if envTag != "" {
			_ = v.BindEnv(configKey, envTag)
			continue
		}

		envName := strings.ToUpper(strings.ReplaceAll(configKey, ".", "_"))
		_ = v.BindEnv(configKey, envName)
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

// CronAllowLocalExecKey 是否允许本地命令/脚本任务的配置键（环境变量 CRON_ALLOW_LOCAL_EXEC）
const CronAllowLocalExecKey = "cron.allow_local_exec"

// CronLocalExecAllowed 返回当前是否允许执行命令行(2)/脚本(4)任务；未配置时为 false
func CronLocalExecAllowed() bool {
	return viper.GetBool(CronAllowLocalExecKey)
}
