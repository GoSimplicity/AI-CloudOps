package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
)

func setOrUnsetEnv(t *testing.T, raw string, set bool) {
	t.Helper()
	t.Setenv("ENV", raw)
	if !set {
		if err := os.Unsetenv("ENV"); err != nil {
			t.Fatal(err)
		}
	}
}

// 生产密钥校验必须跟随规范化后的 ENV：别名、大小写、空白都算生产；
// 未设置、空白、未知取值失败即关闭，同样执行生产校验。
func TestProductionChecksFollowNormalizedEnv(t *testing.T) {
	weak := func() *Config {
		return &Config{Server: ServerConfig{Port: "8889"}, Log: LogConfig{Dir: "logs"}, MySQL: MySQLConfig{Addr: "db"}, Redis: RedisConfig{Addr: "redis"},
			// JWT 密钥强度与 issuer 在所有环境都校验（w1 validateKeys），这里给合规值；只留生产专属的弱项（TREE 密钥过短、mock 开启）
			JWT: JWTConfig{Key1: randomTestKey(), Key2: randomTestKey(), Issuer: "fake-issuer-value"}, Tree: TreeConfig{PasswordEncryptionKey: "short3"}, Mock: MockConfig{Enabled: true}}
	}
	for _, tt := range []struct {
		raw     string
		set     bool
		enforce bool
	}{
		{"production", true, true}, {"prod", true, true}, {"prd", true, true},
		{"PROD", true, true}, {"Production", true, true}, {" prd\t", true, true}, {"\nPROD \r\n", true, true},
		{"", false, true}, {"", true, true}, {"   ", true, true},
		{"test", true, true}, {"testing", true, true}, {"staging", true, true},
		{"development", true, false}, {"dev", true, false}, {" DEV ", true, false}, {"Development", true, false},
	} {
		setOrUnsetEnv(t, tt.raw, tt.set)
		err := weak().Validate()
		if (err != nil) != tt.enforce {
			t.Errorf("ENV=%q(set=%v): Validate() = %v，期望执行生产校验=%v", tt.raw, tt.set, err, tt.enforce)
		}
	}
	// 合规的生产配置在别名下也能通过
	setOrUnsetEnv(t, " Prod ", true)
	ok := &Config{Server: ServerConfig{Port: "8889"}, Log: LogConfig{Dir: "logs"}, MySQL: MySQLConfig{Addr: "db"}, Redis: RedisConfig{Addr: "redis"},
		JWT: JWTConfig{Key1: randomTestKey(), Key2: randomTestKey(), Issuer: "fake-issuer-value"}, Tree: TreeConfig{PasswordEncryptionKey: strings.Repeat("c", 32)}}
	if err := ok.Validate(); err != nil {
		t.Errorf("ENV=\" Prod \" 下合规配置不应失败：%v", err)
	}
}

// 启动入口：ENV 无论是否传了 --config 都要校验；默认配置文件按规范化后的 ENV 选择。
func TestResolveConfigFile(t *testing.T) {
	for _, tt := range []struct {
		raw, flag, want string
		set             bool
	}{
		{"production", "", "config/config.production.yaml", true},
		{"PRD", "", "config/config.production.yaml", true},
		{" prod ", "", "config/config.production.yaml", true},
		{"development", "", "config/config.development.yaml", true},
		{"Dev", "", "config/config.development.yaml", true},
		{"prod", "custom.yaml", "custom.yaml", true},
		{"dev", "custom.yaml", "custom.yaml", true},
	} {
		setOrUnsetEnv(t, tt.raw, tt.set)
		got, err := resolveConfigFile(tt.flag)
		if err != nil || got != tt.want {
			t.Errorf("ENV=%q flag=%q: resolveConfigFile = %q, %v；want %q", tt.raw, tt.flag, got, err, tt.want)
		}
	}
	for _, tt := range []struct {
		raw, flag string
		set       bool
		want      error
	}{
		{"", "", false, envmode.ErrUnset},
		{"", "", true, envmode.ErrUnset},
		{" \t ", "", true, envmode.ErrUnset},
		{"", "custom.yaml", false, envmode.ErrUnset},
		{"test", "", true, envmode.ErrUnknown},
		{"testing", "", true, envmode.ErrUnknown},
		{"staging", "custom.yaml", true, envmode.ErrUnknown},
		{"productio", "", true, envmode.ErrUnknown},
	} {
		setOrUnsetEnv(t, tt.raw, tt.set)
		got, err := resolveConfigFile(tt.flag)
		if !errors.Is(err, tt.want) || got != "" {
			t.Errorf("ENV=%q(set=%v) flag=%q: resolveConfigFile = %q, %v；want 拒绝启动(%v)", tt.raw, tt.set, tt.flag, got, err, tt.want)
		}
	}
}
