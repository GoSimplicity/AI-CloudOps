package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// viper 对配置键不区分大小写（统一转小写）。记录当前行为：
// - 根节点下的大写 PORT: 等同 port:，按旧平铺格式启动失败，报错列出小写键名 port，不含取值；
// - webhook / WEBHOOK 根节点下的 PORT: 等同 webhook.port，正常生效，环境变量 WEBHOOK_PORT 仍可覆盖
func TestLoadWebhookUppercasePortKey(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)

	for name, body := range map[string]string{
		"root-only":         "PORT: \"7001\"\n",
		"root-with-webhook": "webhook:\n  port: \"8889\"\nPORT: \"7001\"\n",
		"root-mixed-case":   "Port: \"7001\"\nmysql:\n  addr: \"\"\n",
	} {
		t.Run("reject/"+name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("WEBHOOK_PORT", "")
			_, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
			if err == nil || IsNonFatal(err) {
				t.Fatalf("根节点下的大写 PORT 应按旧平铺格式启动失败, got %v", err)
			}
			if !strings.Contains(err.Error(), "webhook 字段 port") || !strings.Contains(err.Error(), "webhook:") {
				t.Fatalf("报错应列出小写键名 port 并提示移到 webhook: 下: %v", err)
			}
			if strings.Contains(err.Error(), "7001") {
				t.Fatalf("报错不应包含取值: %v", err)
			}
			if len(viper.AllKeys()) != 0 {
				t.Fatal("失败时不应发布任何键到全局 viper")
			}
		})
	}

	for _, tc := range []struct{ name, body, env, want string }{
		{"under-webhook", "webhook:\n  PORT: \"7001\"\n", "", "7001"},
		{"under-uppercase-root", "WEBHOOK:\n  PORT: \"7001\"\n", "", "7001"},
		{"env-overrides-uppercase", "webhook:\n  PORT: \"7001\"\n", "7003", "7003"},
	} {
		t.Run("accept/"+tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("WEBHOOK_PORT", tc.env)
			warns := captureConfigWarn(t)
			cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, tc.body))
			if err != nil {
				t.Fatalf("webhook 根节点下的大写 PORT 应正常加载: %v", err)
			}
			if cfg.Port != tc.want || viper.GetString("webhook.port") != tc.want {
				t.Fatalf("got cfg=%q 全局=%q, want %q", cfg.Port, viper.GetString("webhook.port"), tc.want)
			}
			if len(*warns) != 0 {
				t.Fatalf("大写 WEBHOOK 根节点应被识别为 webhook，不应告警: %v", *warns)
			}
		})
	}
}
