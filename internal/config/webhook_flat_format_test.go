package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// 旧的平铺格式（没有 webhook 根节点）不再支持：加载即失败，提示把键移到 webhook: 下，只列键名不输出值
func TestLoadWebhookRejectsFlatFormatWithoutValues(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake-hook-dsn")
	path := writeWebhookYAML(t, `port: "fake-port-9911"
fixed_workers: 77
front_domain: "fake-front.example"
im_feishu:
  private_robot_app_secret: "fake-feishu-secret"
mysql:
  addr: ""
`)
	cfg, secrets, err := loadWebhookFromFile(path)
	if err == nil {
		t.Fatalf("平铺格式必须加载失败，实际 cfg=%v secrets!=nil:%v", cfg != nil, secrets != nil)
	}
	if IsNonFatal(err) {
		t.Fatal("平铺格式不能是可继续运行的告警")
	}
	msg := err.Error()
	for _, want := range []string{"port", "fixed_workers", "front_domain", "im_feishu", "webhook:", "不再支持没有 webhook 根节点的旧格式"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("错误信息应包含 %q: %s", want, msg)
		}
	}
	if strings.Contains(msg, "mysql") {
		t.Fatalf("mysql 不是 webhook 字段，不应被列出: %s", msg)
	}
	for _, leaked := range []string{"9911", "77", "fake-"} {
		if strings.Contains(msg, leaked) {
			t.Fatalf("错误信息泄露了配置值: %s", msg)
		}
	}
	if viper.IsSet(webhookPortKey) {
		t.Fatal("加载失败时不应向全局 viper 发布 webhook.*")
	}
}

// 有 webhook 根节点、但根节点下还残留平铺字段时同样失败（避免两处取值不一致）
func TestLoadWebhookRejectsLeftoverRootKeys(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake-hook-dsn")
	path := writeWebhookYAML(t, `webhook:
  port: "9001"
port: "9002"
`)
	_, _, err := loadWebhookFromFile(path)
	if err == nil || !strings.Contains(err.Error(), "port") || !strings.Contains(err.Error(), "webhook:") {
		t.Fatalf("根节点残留 port 时应失败并提示移到 webhook: 下: %v", err)
	}
	if strings.Contains(err.Error(), "900") {
		t.Fatalf("错误信息泄露了配置值: %v", err)
	}
}

// 只检查配置文件本身：环境变量 PORT / FIXED_WORKERS 等（AutomaticEnv 可见）不算平铺字段
func TestLoadWebhookFlatCheckIgnoresEnvAndDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake-hook-dsn")
	t.Setenv("PORT", "1234")
	t.Setenv("FIXED_WORKERS", "5")
	path := writeWebhookYAML(t, `webhook:
  port: "9003"
mysql:
  addr: ""
`)
	cfg, _, err := loadWebhookFromFile(path)
	if err != nil {
		t.Fatalf("有 webhook 根节点时应正常加载: %v", err)
	}
	if cfg.Port != "9003" || cfg.FixedWorkers != 10 {
		t.Fatalf("取值不符合 环境变量 > 配置文件 > 默认值: port=%q fixed_workers=%d", cfg.Port, cfg.FixedWorkers)
	}
}

// 仓库内的 webhook 配置文件必须全部写在 webhook 根节点下
func TestRepoWebhookConfigHasNoRootLevelWebhookKeys(t *testing.T) {
	v := viper.New()
	v.SetConfigFile(webhookConfigFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if !v.InConfig("webhook") {
		t.Fatal("config/webhook.yaml 缺少 webhook 根节点")
	}
	if keys := misplacedWebhookRootKeys(v); len(keys) > 0 {
		t.Fatalf("config/webhook.yaml 根节点下有 webhook 字段: %v", keys)
	}
}

// 配置文件缺失时不触发平铺检查，仍按默认值加载（可继续运行的告警）
func TestLoadWebhookMissingFileSkipsFlatCheck(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake-hook-dsn")
	cfg, _, err := loadWebhookFromFile(t.TempDir() + "/not-exist.yaml")
	if err != nil && !IsNonFatal(err) {
		t.Fatalf("配置文件缺失应是可继续运行的告警: %v", err)
	}
	if cfg == nil || cfg.Port != "8889" {
		t.Fatal("配置文件缺失时应取默认端口")
	}
}
