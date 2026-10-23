package config

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

const webhookConfigFile = "../../config/webhook.yaml"

// webhook.yaml 不得写入 MySQL DSN（含密码）
func TestWebhookConfigFileContainsNoDSN(t *testing.T) {
	v := viper.New()
	v.SetConfigFile(webhookConfigFile)
	if err := v.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if !v.IsSet(webhookMySQLAddrKey) {
		t.Fatalf("缺少占位字段 %s", webhookMySQLAddrKey)
	}
	if v.GetString(webhookMySQLAddrKey) != "" {
		t.Fatalf("%s 不应在配置文件中写入值", webhookMySQLAddrKey) // 不输出值本身
	}
	raw, err := os.ReadFile(webhookConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "@tcp(") {
		t.Fatal("webhook.yaml 中不得出现 DSN")
	}
}

func TestLoadWebhookFailsFastWithoutDSN(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "  ")
	// 主程序的 MYSQL_ADDR 不能替代 WEBHOOK_MYSQL_ADDR
	t.Setenv("MYSQL_ADDR", "fake_main_user:fake_main_pass@tcp(127.0.0.1:3306)/fake")
	_, _, err := loadWebhookFromFile(webhookConfigFile)
	if err == nil {
		t.Fatal("缺少 WEBHOOK_MYSQL_ADDR 时 webhook 必须启动失败")
	}
	if !strings.Contains(err.Error(), WebhookMySQLAddrEnv) || !strings.Contains(err.Error(), "缺少必需的密钥环境变量") {
		t.Fatalf("错误信息应列出缺失的变量名: %v", err)
	}
	if IsNonFatal(err) {
		t.Fatal("缺少 DSN 不能是可继续运行的告警")
	}
	assertNoValuesLeaked(t, err.Error())
}

func TestLoadWebhookInjectsDSNFromEnv(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	want := "fake_hook_user:fake_hook_pass@tcp(127.0.0.1:3306)/fake"
	t.Setenv(WebhookMySQLAddrEnv, want)
	t.Setenv("MYSQL_ADDR", "fake_main_user:fake_main_pass@tcp(127.0.0.1:3306)/fake")
	cfg, secrets, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg == nil || cfg.Port == "" {
		t.Fatal("webhook 其他配置应正常加载")
	}
	// DSN 只经返回值传给 di.InitDB
	if secrets == nil || secrets.MySQLAddr != want {
		t.Fatalf("DSN 应取自 %s（且优先于 MYSQL_ADDR）", WebhookMySQLAddrEnv)
	}
}

// P3-2：DSN 不得写入全局 viper；webhook.* 仍发布给子系统现有的全局读取方
func TestLoadWebhookDoesNotSetGlobalMySQLAddr(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake_hook_user:fake_hook_pass@tcp(127.0.0.1:3306)/fake")
	t.Setenv("MYSQL_ADDR", "fake_main_user:fake_main_pass@tcp(127.0.0.1:3306)/fake")
	t.Setenv("WEBHOOK_FRONT_DOMAIN", "env-front.example")
	if _, _, err := loadWebhookFromFile(webhookConfigFile); err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if viper.IsSet(webhookMySQLAddrKey) || viper.GetString(webhookMySQLAddrKey) != "" {
		t.Fatalf("全局 viper 不应包含 %s", webhookMySQLAddrKey)
	}
	for _, key := range viper.AllKeys() {
		if !strings.HasPrefix(key, webhookGlobalPrefix) {
			t.Fatalf("全局 viper 只应发布 webhook.* 键, got %s", key)
		}
	}
	if got := viper.GetString("webhook.port"); got != "8889" {
		t.Fatalf("webhook.port 应发布到全局 viper（取配置文件值）, got %q", got)
	}
	if got := viper.GetString("webhook.front_domain"); got != "env-front.example" {
		t.Fatalf("webhook.front_domain 应保持环境变量优先, got %q", got)
	}
	if got := viper.GetInt("webhook.fixed_workers"); got != 10 {
		t.Fatalf("默认值 webhook.fixed_workers 应发布到全局 viper, got %d", got)
	}
}

func TestLoadWebhookMissingFileStillRequiresDSN(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "")
	if _, _, err := loadWebhookFromFile(t.TempDir() + "/missing.yaml"); err == nil || IsNonFatal(err) {
		t.Fatalf("配置文件缺失时也必须因缺少 DSN 失败, got %v", err)
	}
}
