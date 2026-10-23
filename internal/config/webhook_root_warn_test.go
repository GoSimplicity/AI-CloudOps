package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// 配置文件存在但没有 webhook 根节点：记一条告警，只列根节点键名，不输出任何值；仍按环境变量/默认值继续加载
func TestLoadWebhookWarnsWhenFileHasNoWebhookRoot(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	warns := captureConfigWarn(t)
	path := writeWebhookYAML(t, "mysql:\n  addr: \"\"\nwebhok:\n  port: \"7001\"\nnote: test-secret-value\n")

	cfg, _, err := loadWebhookFromFile(path)
	if err != nil {
		t.Fatalf("没有 webhook 根节点时应告警并继续加载: %v", err)
	}
	if cfg.Port != "8889" {
		t.Fatalf("应回落到默认端口, got %q", cfg.Port)
	}
	if len(*warns) != 1 {
		t.Fatalf("应只有一条告警, got %d", len(*warns))
	}
	w := (*warns)[0]
	for _, want := range []string{"webhook 根节点", path, "mysql, note, webhok"} {
		if !strings.Contains(w, want) {
			t.Fatalf("告警应包含 %q: %s", want, w)
		}
	}
	for _, leak := range []string{"test-secret-value", "7001"} {
		if strings.Contains(w, leak) {
			t.Fatalf("告警不应输出配置值: %s", w)
		}
	}
}

// 有 webhook 根节点、配置文件缺失时都不产生该告警
func TestLoadWebhookNoRootWarnWhenRootPresentOrFileMissing(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	for name, path := range map[string]string{
		"root-present": writeWebhookYAML(t, "webhook:\n  port: \"8889\"\n"),
		"repo-file":    webhookConfigFile,
		"missing":      t.TempDir() + "/missing.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			warns := captureConfigWarn(t)
			if _, _, err := loadWebhookFromFile(path); err != nil && !IsNonFatal(err) {
				t.Fatalf("应正常加载: %v", err)
			}
			for _, w := range *warns {
				if strings.Contains(w, "webhook 根节点") {
					t.Fatalf("不应告警: %s", w)
				}
			}
		})
	}
}

// 空配置文件：没有任何根节点键，同样告警，键名列表为“无”
func TestLoadWebhookWarnsOnEmptyFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	warns := captureConfigWarn(t)
	if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, "# only comments\n")); err != nil {
		t.Fatalf("空文件应告警并继续加载: %v", err)
	}
	if len(*warns) != 1 || !strings.HasSuffix((*warns)[0], "文件根节点下的键：无") {
		t.Fatalf("空文件应告警且键名为“无”, got %v", *warns)
	}
}

// webhook: 写了但值为 null（子键全部注释掉）：InConfig 为 false，同样告警
func TestLoadWebhookWarnsWhenWebhookRootIsNull(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	warns := captureConfigWarn(t)
	if _, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n#  port: \"7001\"\nmysql:\n  addr: \"\"\n")); err != nil {
		t.Fatalf("应告警并继续加载: %v", err)
	}
	if len(*warns) != 1 || !strings.HasSuffix((*warns)[0], "文件根节点下的键：mysql") {
		t.Fatalf("webhook 为 null 时应告警且只列 mysql, got %v", *warns)
	}
}
