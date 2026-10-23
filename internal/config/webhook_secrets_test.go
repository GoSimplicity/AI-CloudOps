package config

import (
	"os"
	"path/filepath"
	"reflect"
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

// 环境变量 WEBHOOK_PORT 覆盖 webhook.yaml 中的 port: "8889"，返回值与全局 viper 一致
func TestLoadWebhookPortEnvOverridesConfigFile(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake_hook_user:fake_hook_pass@tcp(127.0.0.1:3306)/fake")
	t.Setenv("WEBHOOK_PORT", "9999")
	cfg, _, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg.Port != "9999" {
		t.Fatalf("cfg.Port 应取环境变量 WEBHOOK_PORT=9999（覆盖配置文件 8889）, got %q", cfg.Port)
	}
	if got := viper.GetString("webhook.port"); got != "9999" {
		t.Fatalf("全局 viper webhook.port 应为 9999, got %q", got)
	}
}

// 环境变量 WEBHOOK_FIXED_WORKERS 覆盖默认值 10（webhook.yaml 未设置该键）
func TestLoadWebhookFixedWorkersEnvOverridesDefault(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, "fake_hook_user:fake_hook_pass@tcp(127.0.0.1:3306)/fake")
	t.Setenv("WEBHOOK_FIXED_WORKERS", "3")
	cfg, _, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg.FixedWorkers != 3 {
		t.Fatalf("cfg.FixedWorkers 应取环境变量 WEBHOOK_FIXED_WORKERS=3（覆盖默认值 10）, got %d", cfg.FixedWorkers)
	}
	if got := viper.GetInt("webhook.fixed_workers"); got != 3 {
		t.Fatalf("全局 viper webhook.fixed_workers 应为 3, got %d", got)
	}
}

const fakeWebhookDSN = "fake_hook_user:fake_hook_pass@tcp(127.0.0.1:3306)/fake"

// writeWebhookYAML 写一个临时 webhook 配置文件，返回路径
func writeWebhookYAML(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "webhook.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// compareWebhookCfgWithGlobal 逐字段（含嵌套 im_feishu）比对 cfg 与全局 viper 的 webhook.*，返回比对的字段数
func compareWebhookCfgWithGlobal(t *testing.T, val reflect.Value, prefix string) int {
	t.Helper()
	n := 0
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		key := prefix + "." + f.Tag.Get("mapstructure")
		fv := val.Field(i)
		switch fv.Kind() {
		case reflect.Struct:
			n += compareWebhookCfgWithGlobal(t, fv, key)
		case reflect.String:
			if !viper.IsSet(key) || viper.GetString(key) != fv.String() {
				t.Errorf("%s: cfg 与全局 viper 不一致（或全局未设置）", key) // 不输出值，im_feishu 含密钥
			}
			n++
		case reflect.Int:
			if !viper.IsSet(key) || int64(viper.GetInt(key)) != fv.Int() {
				t.Errorf("%s: cfg=%d, 全局=%d", key, fv.Int(), viper.GetInt(key))
			}
			n++
		default:
			t.Fatalf("%s: 未覆盖的字段类型 %s", key, fv.Kind())
		}
	}
	return n
}

// cfg 与全局 webhook.* 来自同一份合并值：每个字段（含嵌套 im_feishu）都一致，且三层来源都进入 cfg
func TestLoadWebhookCfgMatchesGlobalForEveryField(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "9999")                           // 环境变量覆盖文件
	t.Setenv("WEBHOOK_IM_FEISHU_REQUEST_TIMEOUT_SECONDS", "7") // 嵌套字段的环境变量
	t.Setenv("WEBHOOK_IM_FEISHU_PRIVATE_ROBOT_APP_SECRET", "fake-app-secret-value")
	cfg, _, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if n := compareWebhookCfgWithGlobal(t, reflect.ValueOf(*cfg), "webhook"); n != 12 {
		t.Fatalf("应比对 WebhookConfig 全部 12 个字段, got %d", n)
	}
	if cfg.Port != "9999" || cfg.ImFeishu.RequestTimeoutSeconds != 7 || cfg.ImFeishu.PrivateRobotAppSecret != "fake-app-secret-value" {
		t.Fatal("环境变量（含嵌套 im_feishu）应进入 cfg")
	}
	if cfg.DefaultUpgradeMinutes != 30 || cfg.CommonMapRenewIntervalSeconds != 10 {
		t.Fatalf("配置文件值应进入 cfg（覆盖默认值 60/300）, got %d/%d", cfg.DefaultUpgradeMinutes, cfg.CommonMapRenewIntervalSeconds)
	}
	if cfg.FixedWorkers != 10 {
		t.Fatalf("默认值应进入 cfg, got fixed_workers=%d", cfg.FixedWorkers)
	}
}

// webhook.yaml 未设置 fixed_workers：默认值 10 进入 cfg
func TestLoadWebhookDefaultFixedWorkersInCfg(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_FIXED_WORKERS", "")
	cfg, _, err := loadWebhookFromFile(webhookConfigFile)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg.FixedWorkers != 10 || viper.GetInt("webhook.fixed_workers") != 10 {
		t.Fatalf("默认 fixed_workers=10 应进入 cfg 和全局, got cfg=%d", cfg.FixedWorkers)
	}
}

// 配置文件里没有 port：只靠环境变量提供；都没有时取默认值 8889
func TestLoadWebhookPortFromEnvOnlyOrDefault(t *testing.T) {
	path := writeWebhookYAML(t, "webhook:\n  front_domain: \"file-front.example\"\n")
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)

	viper.Reset()
	t.Cleanup(viper.Reset)
	t.Setenv("WEBHOOK_PORT", "7777")
	cfg, _, err := loadWebhookFromFile(path)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg.Port != "7777" || viper.GetString("webhook.port") != "7777" {
		t.Fatalf("文件无 port 时应取环境变量 7777, got cfg=%q 全局=%q", cfg.Port, viper.GetString("webhook.port"))
	}
	if cfg.FrontDomain != "file-front.example" {
		t.Fatalf("文件中的其他键应保留, got %q", cfg.FrontDomain)
	}

	viper.Reset()
	t.Setenv("WEBHOOK_PORT", "")
	cfg, _, err = loadWebhookFromFile(path)
	if err != nil {
		t.Fatalf("加载 webhook 配置失败: %v", err)
	}
	if cfg.Port != "8889" || viper.GetString("webhook.port") != "8889" {
		t.Fatalf("文件与环境变量都没有 port 时应取默认值 8889, got cfg=%q", cfg.Port)
	}
}

// 非法端口启动失败（错误只含键名/变量名，不含值，且不发布到全局）；首尾空白去掉后合法则通过
func TestLoadWebhookRejectsInvalidPort(t *testing.T) {
	t.Setenv(WebhookMySQLAddrEnv, fakeWebhookDSN)
	t.Setenv("WEBHOOK_PORT", "")
	for _, bad := range []string{"", "   ", "abc", "0", "65536", "-1", "+80", "80a", "8 0", "99999999999999999999"} {
		t.Run("bad/"+bad, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			path := writeWebhookYAML(t, "webhook:\n  port: \""+bad+"\"\n")
			_, _, err := loadWebhookFromFile(path)
			if err == nil || IsNonFatal(err) {
				t.Fatalf("端口 %q 应导致启动失败, got %v", bad, err)
			}
			if !strings.Contains(err.Error(), "webhook.port") || !strings.Contains(err.Error(), "WEBHOOK_PORT") {
				t.Fatalf("错误应列出键名和变量名: %v", err)
			}
			if strings.TrimSpace(bad) != "" && strings.Contains(err.Error(), strings.TrimSpace(bad)) {
				t.Fatalf("错误信息不应包含端口值: %v", err)
			}
			if len(viper.AllKeys()) != 0 {
				t.Fatal("失败时不应发布任何键到全局 viper")
			}
		})
	}
	// 环境变量提供的非法端口同样失败
	viper.Reset()
	t.Setenv("WEBHOOK_PORT", "abc")
	if _, _, err := loadWebhookFromFile(webhookConfigFile); err == nil {
		t.Fatal("WEBHOOK_PORT=abc 应导致启动失败")
	}
	// 带首尾空白的合法端口：去空白后通过，cfg 与全局都是去空白后的值
	for _, tc := range []struct{ name, file, env, want string }{
		{"file", "webhook:\n  port: \"  8080 \"\n", "", "8080"},
		{"env", "webhook:\n  port: \"8889\"\n", " 9998\t", "9998"},
		{"unquoted-int", "webhook:\n  port: 65535\n", "", "65535"},
		{"min", "webhook:\n  port: \"1\"\n", "", "1"},
	} {
		t.Run("ok/"+tc.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			t.Setenv("WEBHOOK_PORT", tc.env)
			cfg, _, err := loadWebhookFromFile(writeWebhookYAML(t, tc.file))
			if err != nil {
				t.Fatalf("应通过: %v", err)
			}
			if cfg.Port != tc.want || viper.GetString("webhook.port") != tc.want {
				t.Fatalf("got cfg=%q 全局=%q, want %q", cfg.Port, viper.GetString("webhook.port"), tc.want)
			}
		})
	}
}
