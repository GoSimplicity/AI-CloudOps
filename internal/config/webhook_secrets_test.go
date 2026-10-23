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
	_, err := loadWebhookFromFile(webhookConfigFile)
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
	cfg, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg == nil || cfg.Port == "" {
		t.Fatal("webhook 其他配置应正常加载")
	}
	// internal/prometheus/webhook/di/gorm.go 通过 viper.GetString("mysql.addr") 读取 DSN
	if viper.GetString("mysql.addr") != want {
		t.Fatalf("mysql.addr 应取自 %s（且优先于 MYSQL_ADDR）", WebhookMySQLAddrEnv)
	}
}

func TestLoadWebhookMissingFileStillRequiresDSN(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "")
	if _, err := loadWebhookFromFile(t.TempDir() + "/missing.yaml"); err == nil || IsNonFatal(err) {
		t.Fatalf("配置文件缺失时也必须因缺少 DSN 失败, got %v", err)
	}
}
