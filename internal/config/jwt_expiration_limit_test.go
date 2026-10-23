package config

import (
	"fmt"
	"strings"
	"testing"
)

// captureConfigWarn 替换 configWarnf，返回收集到的告警
func captureConfigWarn(t *testing.T) *[]string {
	t.Helper()
	var got []string
	orig := configWarnf
	configWarnf = func(format string, args ...any) { got = append(got, fmt.Sprintf(format, args...)) }
	t.Cleanup(func() { configWarnf = orig })
	return &got
}

func expirationTestConfig(exp int64) *Config {
	c := validConfigForTest()
	c.Tree.PasswordEncryptionKey = randomTestKey()[:32] // 生产环境要求恰好32字节
	c.JWT.Expiration = exp
	return c
}

func TestJWTExpirationUpperBoundProductionRejects1441(t *testing.T) {
	t.Setenv("ENV", "production")
	warns := captureConfigWarn(t)
	err := expirationTestConfig(MaxJWTExpirationMinutes + 1).Validate()
	if err == nil || !strings.Contains(err.Error(), "JWT_EXPIRATION") {
		t.Fatalf("production 下 1441 分钟应拒绝启动, got %v", err)
	}
	if len(*warns) != 0 {
		t.Fatalf("production 下应直接返回错误而非告警: %v", *warns)
	}
}

func TestJWTExpirationUpperBound1440Passes(t *testing.T) {
	for _, env := range allEnvs {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			warns := captureConfigWarn(t)
			if err := expirationTestConfig(MaxJWTExpirationMinutes).Validate(); err != nil {
				t.Fatalf("1440 分钟应通过: %v", err)
			}
			if len(*warns) != 0 {
				t.Fatalf("1440 分钟不应告警: %v", *warns)
			}
		})
	}
}

// 只有 envmode 识别为开发环境（development/dev，去首尾空白、忽略大小写）时 1441 分钟才告警继续；
// 空值、test、production-x 失败即关闭，按生产拒绝（不告警）
func TestJWTExpirationUpperBoundNonProductionWarns(t *testing.T) {
	for _, env := range []string{"", "test", "production-x"} {
		t.Run("fail-closed/"+env, func(t *testing.T) {
			t.Setenv("ENV", env)
			warns := captureConfigWarn(t)
			err := expirationTestConfig(MaxJWTExpirationMinutes + 1).Validate()
			if err == nil || !strings.Contains(err.Error(), "JWT_EXPIRATION") {
				t.Fatalf("ENV=%q 应按生产拒绝 1441 分钟, got %v", env, err)
			}
			if len(*warns) != 0 {
				t.Fatalf("ENV=%q 应直接返回错误而非告警: %v", env, *warns)
			}
		})
	}
	for _, env := range []string{"development", "dev", " DEV "} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			warns := captureConfigWarn(t)
			if err := expirationTestConfig(MaxJWTExpirationMinutes + 1).Validate(); err != nil {
				t.Fatalf("非 production 下 1441 分钟应继续启动, got %v", err)
			}
			if len(*warns) != 1 || !strings.Contains((*warns)[0], "JWT_EXPIRATION") || !strings.Contains((*warns)[0], "1441") {
				t.Fatalf("非 production 下 1441 分钟应记一条告警, got %v", *warns)
			}
		})
	}
}

// 历史 env.example 的 3600（按分钟即 60 小时）在 production 被拒
func TestJWTExpirationLegacy3600RejectedInProduction(t *testing.T) {
	t.Setenv("ENV", "production")
	if err := expirationTestConfig(3600).Validate(); err == nil {
		t.Fatal("production 下 3600 分钟应拒绝启动")
	}
}

// P3：unicode.Cf 格式字符（零宽空格、BOM、RLO）嵌在密钥中间也应拒绝
func TestJWTKeyRejectsUnicodeFormatChars(t *testing.T) {
	for _, r := range []rune{'\u200B', '\uFEFF', '\u202E'} {
		for _, slot := range []string{"JWT_KEY1", "JWT_KEY2"} {
			t.Run(fmt.Sprintf("U+%04X/%s", r, slot), func(t *testing.T) {
				if !hasControlChar("a" + string(r) + "b") {
					t.Fatalf("hasControlChar 应识别 U+%04X", r)
				}
				t.Setenv("ENV", "development")
				c := validConfigForTest()
				bad := randomTestKey()[:20] + string(r) + randomTestKey()[:20]
				if slot == "JWT_KEY1" {
					c.JWT.Key1 = bad
				} else {
					c.JWT.Key2 = bad
				}
				assertRejectedWithoutLeak(t, c.Validate(), slot+" 首尾不能有空白且不能包含控制字符", bad)
			})
		}
	}
	if hasControlChar(randomTestKey()) {
		t.Fatal("普通 base64 密钥不应被识别为含控制/格式字符")
	}
}
