package config

import (
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// webhookIntFieldCase 每个整数字段的测试数据；yaml 生成只设置该字段的配置文件（im_feishu 字段需要嵌套）
type webhookIntFieldCase struct {
	key       string
	env       string
	allowZero bool
	max       int
	get       func(*WebhookConfig) int
	yaml      func(v string) string
	wantErr   string // 取自环境变量时的报错：只含键名、变量名和上下限，不含取值
}

// wantErrFor 按来源给出期望报错：配置文件来源写“配置文件中的 KEY”，不提环境变量
func (c webhookIntFieldCase) wantErrFor(source string) string {
	if source == "yaml" {
		return strings.Replace(c.wantErr, c.key+"（环境变量 "+c.env+"）", "配置文件中的 "+c.key+" ", 1)
	}
	return c.wantErr
}

func webhookIntFieldCases() []webhookIntFieldCase {
	top := func(k string) func(string) string {
		return func(v string) string { return "webhook:\n  port: \"8889\"\n  " + k + ": " + v + "\n" }
	}
	return []webhookIntFieldCase{
		{"webhook.fixed_workers", "WEBHOOK_FIXED_WORKERS", false, 1024,
			func(c *WebhookConfig) int { return c.FixedWorkers }, top("fixed_workers"),
			"Webhook 整数配置超出允许范围: webhook.fixed_workers（环境变量 WEBHOOK_FIXED_WORKERS）必须在 1 到 1024 之间"},
		{"webhook.default_upgrade_minutes", "WEBHOOK_DEFAULT_UPGRADE_MINUTES", false, 10080,
			func(c *WebhookConfig) int { return c.DefaultUpgradeMinutes }, top("default_upgrade_minutes"),
			"Webhook 整数配置超出允许范围: webhook.default_upgrade_minutes（环境变量 WEBHOOK_DEFAULT_UPGRADE_MINUTES）必须在 1 到 10080 之间"},
		{"webhook.common_map_renew_interval_seconds", "WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", false, 86400,
			func(c *WebhookConfig) int { return c.CommonMapRenewIntervalSeconds }, top("common_map_renew_interval_seconds"),
			"Webhook 整数配置超出允许范围: webhook.common_map_renew_interval_seconds（环境变量 WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS）必须在 1 到 86400 之间"},
		{"webhook.im_feishu.request_timeout_seconds", "WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", false, 600,
			func(c *WebhookConfig) int { return c.ImFeishu.RequestTimeoutSeconds },
			func(v string) string {
				return "webhook:\n  port: \"8889\"\n  im_feishu:\n    request_timeout_seconds: " + v + "\n"
			},
			"Webhook 整数配置超出允许范围: webhook.im_feishu.request_timeout_seconds（环境变量 WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS）必须在 1 到 600 之间"},
	}
}

// clearWebhookIntEnv 把全部整数字段的环境变量置空（视为未设置），避免互相干扰
func clearWebhookIntEnv(t *testing.T) {
	t.Helper()
	for _, c := range webhookIntFieldCases() {
		t.Setenv(c.env, "")
	}
}

// loadWebhookIntCase 按来源（env / yaml）给字段 c 设置取值 v 并加载
func loadWebhookIntCase(t *testing.T, c webhookIntFieldCase, source, v string) (*WebhookConfig, error) {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearWebhookIntEnv(t)
	body := "webhook:\n  port: \"8889\"\n"
	switch source {
	case "env":
		t.Setenv(c.env, v)
	case "yaml":
		body = c.yaml(v)
	default:
		t.Fatalf("未知来源 %s", source)
	}
	cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, body))
	return cfg, err
}

// 每个整数字段：负数一律拒绝；不允许 0 的字段写 0 拒绝（全部字段都不允许 0，im_feishu.request_timeout_seconds 自第二轮起最小 1）；
// 报错按来源写“KEY（环境变量 ENV）”或“配置文件中的 KEY”，只含键名、变量名和上下限、不含取值；失败不发布全局。环境变量与 YAML 两个来源都覆盖
func TestLoadWebhookIntFieldsRejectOutOfRange(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	for _, c := range webhookIntFieldCases() {
		bads := []string{"-7391", "-1"}
		if !c.allowZero {
			bads = append(bads, "0")
		}
		for _, source := range []string{"env", "yaml"} {
			for _, bad := range bads {
				t.Run(c.key+"/"+source+"/"+strings.ReplaceAll(bad, "7391", "large"), func(t *testing.T) {
					_, err := loadWebhookIntCase(t, c, source, bad)
					if err == nil || IsNonFatal(err) {
						t.Fatalf("%s=%s 应导致启动失败, got %v", c.key, bad, err)
					}
					msg := err.Error()
					if !strings.Contains(msg, c.key) || strings.Contains(msg, c.env) != (source == "env") {
						t.Fatalf("报错应包含键名，且只有环境变量来源才提变量名: %s", msg)
					}
					// 单字符 "0" 会和上限（如 1024）中的数字撞上，靠下面的精确比对保证不含取值
					if len(bad) > 1 && strings.Contains(msg, bad) {
						t.Fatalf("报错不应包含取值: %s", msg)
					}
					if want := c.wantErrFor(source); msg != want {
						t.Fatalf("报错应只描述键名、来源和上下限\n got: %s\nwant: %s", msg, want)
					}
					if len(viper.AllKeys()) != 0 {
						t.Fatal("失败时不应发布任何键到全局 viper")
					}
				})
			}
		}
	}
}

// 允许 0 的字段写 0 接受；所有字段在上下限内（含上限）的正值接受。cfg 与全局取值一致
func TestLoadWebhookIntFieldsAcceptInRange(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	for _, c := range webhookIntFieldCases() {
		goods := []int{1, 7, c.max}
		if c.allowZero {
			goods = append(goods, 0)
		}
		for _, source := range []string{"env", "yaml"} {
			for _, good := range goods {
				s := strconv.Itoa(good)
				t.Run(c.key+"/"+source+"/"+s, func(t *testing.T) {
					cfg, err := loadWebhookIntCase(t, c, source, s)
					if err != nil {
						t.Fatalf("%s=%s 应通过: %v", c.key, s, err)
					}
					if c.get(cfg) != good || viper.GetInt(c.key) != good {
						t.Fatalf("got cfg=%d 全局=%d, want %d", c.get(cfg), viper.GetInt(c.key), good)
					}
				})
			}
		}
	}
}

// 多个字段同时越界：一次列出全部键名（按字段顺序），仍不含取值
func TestLoadWebhookIntFieldsListAllViolations(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	clearWebhookIntEnv(t)
	t.Setenv("WEBHOOK_FIXED_WORKERS", "0")
	t.Setenv("WEBHOOK_DEFAULT_UPGRADE_MINUTES", "-4821")
	t.Setenv("WEBHOOK_COMMON_MAP_RENEW_INTERVAL_SECONDS", "-4822")
	t.Setenv("WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", "-4823")
	_, _, err := loadWebhookFromFile(writeWebhookYAML(t, "webhook:\n  port: \"8889\"\n"))
	if err == nil {
		t.Fatal("应启动失败")
	}
	msg := err.Error()
	last := -1
	for _, c := range webhookIntFieldCases() {
		i := strings.Index(msg, c.key+"（")
		if i < 0 || i < last {
			t.Fatalf("应按字段顺序列出 %s: %s", c.key, msg)
		}
		last = i
	}
	if strings.Contains(msg, "482") {
		t.Fatalf("报错不应包含取值: %s", msg)
	}
	if len(viper.AllKeys()) != 0 {
		t.Fatal("失败时不应发布任何键到全局 viper")
	}
}

// 仓库自带 webhook.yaml 与内置默认值都满足整数范围规则
func TestRepoWebhookConfigAndDefaultsSatisfyIntRules(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	clearWebhookIntEnv(t)
	for name, path := range map[string]string{
		"repo-file": webhookConfigFile,
		"defaults":  t.TempDir() + "/missing.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			cfg, _, err := loadWebhookFromFile(path)
			if err != nil && !IsNonFatal(err) {
				t.Fatalf("应正常加载: %v", err)
			}
			if err := validateWebhookIntFields(cfg, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}
