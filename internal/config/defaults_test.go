package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/spf13/viper"
)

func structDefaultTag(t *testing.T, typ reflect.Type, field string) string {
	t.Helper()
	f, ok := typ.FieldByName(field)
	if !ok {
		t.Fatalf("%s 缺少字段 %s", typ.Name(), field)
	}
	return f.Tag.Get("default")
}

// webhook 端口默认值与 config/webhook.yaml 一致为 8889（viper 默认值、结构体默认值、配置文件三处对齐）
func TestWebhookPortDefaultIs8889(t *testing.T) {
	v := viper.New()
	setWebhookDefaults(v)
	if got := v.GetString("webhook.port"); got != "8889" {
		t.Fatalf("loader 中 webhook.port 默认值应为 8889, got %q", got)
	}
	if got := structDefaultTag(t, reflect.TypeOf(WebhookConfig{}), "Port"); got != "8889" {
		t.Fatalf("WebhookConfig.Port 的 default 标签应为 8889, got %q", got)
	}
	file := viper.New()
	file.SetConfigFile(webhookConfigFile)
	if err := file.ReadInConfig(); err != nil {
		t.Fatal(err)
	}
	if got := file.GetString("webhook.port"); got != "8889" {
		t.Fatalf("webhook.yaml 中 webhook.port 应为 8889, got %q", got)
	}
}

// JWT_EXPIRATION 单位为分钟，默认 30（30 分钟）：loader 默认值、结构体默认值、所有 config.*.yaml 对齐
func TestJWTExpirationDefaultIs30Minutes(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	if got := viper.GetInt64("jwt.expiration"); got != 30 {
		t.Fatalf("loader 中 jwt.expiration 默认值应为 30（分钟）, got %d", got)
	}
	if got := structDefaultTag(t, reflect.TypeOf(JWTConfig{}), "Expiration"); got != "30" {
		t.Fatalf("JWTConfig.Expiration 的 default 标签应为 30, got %q", got)
	}
	for _, f := range []string{"../../config/config.development.yaml", "../../config/config.production.yaml"} {
		v := viper.New()
		v.SetConfigFile(f)
		if err := v.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		if got := v.GetInt64("jwt.expiration"); got != 30 {
			t.Fatalf("%s 中 jwt.expiration 应为 30（分钟）, got %d", f, got)
		}
	}
}

func TestLoginLockoutDefaults(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	if got := viper.GetInt("auth.login_lockout.username_max_failures"); got != 5 {
		t.Fatalf("username_max_failures=%d", got)
	}
	if got := viper.GetDuration("auth.login_lockout.username_window"); got != 15*time.Minute {
		t.Fatalf("username_window=%s", got)
	}
	if got := viper.GetDuration("auth.login_lockout.username_lock"); got != 15*time.Minute {
		t.Fatalf("username_lock=%s", got)
	}
	if got := viper.GetInt("auth.login_lockout.ip_max_failures"); got != 20 {
		t.Fatalf("ip_max_failures=%d", got)
	}
	if got := viper.GetDuration("auth.login_lockout.ip_window"); got != 15*time.Minute {
		t.Fatalf("ip_window=%s", got)
	}
	if got := viper.GetDuration("auth.login_lockout.ip_lock"); got != 15*time.Minute {
		t.Fatalf("ip_lock=%s", got)
	}
	got := LoginLockoutConfig{}.Normalized()
	if got.UsernameMaxFailures != 5 || got.IPMaxFailures != 20 || got.UsernameLock != 15*time.Minute {
		t.Fatalf("normalized zero config = %+v", got)
	}
	custom := LoginLockoutConfig{UsernameMaxFailures: 3, IPMaxFailures: 9, UsernameWindow: time.Minute}.Normalized()
	if custom.UsernameMaxFailures != 3 || custom.IPMaxFailures != 9 || custom.UsernameWindow != time.Minute || custom.IPLock != 15*time.Minute {
		t.Fatalf("explicit values changed: %+v", custom)
	}
	for _, f := range []string{"../../config/config.development.yaml", "../../config/config.production.yaml"} {
		v := viper.New()
		v.SetConfigFile(f)
		if err := v.ReadInConfig(); err != nil {
			t.Fatal(err)
		}
		if v.GetInt("auth.login_lockout.username_max_failures") != 5 || v.GetInt("auth.login_lockout.ip_max_failures") != 20 {
			t.Fatalf("%s 登录锁定默认值不正确", f)
		}
		if v.GetDuration("auth.login_lockout.username_window") != 15*time.Minute || v.GetDuration("auth.login_lockout.ip_lock") != 15*time.Minute {
			t.Fatalf("%s 登录锁定时长不正确", f)
		}
	}
}
