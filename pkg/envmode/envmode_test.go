package envmode

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// unsetForTest 在 t.Setenv 登记恢复之后真正删掉 ENV，覆盖“未设置”而不只是“空串”。
func unsetForTest(t *testing.T) {
	t.Helper()
	if err := os.Unsetenv(EnvKey); err != nil {
		t.Fatal(err)
	}
}

func TestParseAccepts(t *testing.T) {
	for raw, want := range map[string]Mode{
		"production": Production, "prod": Production, "prd": Production,
		"PRODUCTION": Production, "Prod": Production, "pRd": Production,
		" production": Production, "prod \t": Production, "\n PRD \r\n": Production,
		"development": Development, "dev": Development,
		"DEVELOPMENT": Development, "Dev": Development, "\tdev ": Development,
	} {
		got, err := Parse(raw)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %q, %v；want %q, nil", raw, got, err, want)
		}
	}
}

func TestParseRejectsUnsetAndBlank(t *testing.T) {
	for _, raw := range []string{"", " ", "\t", "\r\n", " \t \n "} {
		got, err := Parse(raw)
		if !errors.Is(err, ErrUnset) || got != "" {
			t.Errorf("Parse(%q) = %q, %v；want ErrUnset", raw, got, err)
		}
	}
}

func TestParseRejectsOthers(t *testing.T) {
	for _, raw := range []string{
		"test", "testing", "TEST", "staging", "local", "pro", "produc", "production1",
		"prod-cn", "dev2", "devel", "develop", "p rod", "prod\x00", "ｐｒｏｄ", "\"prod\"", "'dev'",
	} {
		got, err := Parse(raw)
		if !errors.Is(err, ErrUnknown) || got != "" {
			t.Errorf("Parse(%q) = %q, %v；want ErrUnknown", raw, got, err)
			continue
		}
		if !strings.Contains(err.Error(), Allowed) {
			t.Errorf("Parse(%q) 的错误没有列出允许取值：%v", raw, err)
		}
	}
	if !strings.Contains(ErrUnset.Error(), Allowed) {
		t.Errorf("ErrUnset 没有列出允许取值：%v", ErrUnset)
	}
}

func TestParseErrorDoesNotEchoLongValues(t *testing.T) {
	secret := "sk-" + strings.Repeat("Z9", 20)
	_, err := Parse(secret)
	if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "Z9Z9") {
		t.Fatalf("长值被回显：%v", err)
	}
	if !strings.Contains(err.Error(), "长度 43") {
		t.Fatalf("长值应只报长度：%v", err)
	}
	// 短值用 %q 回显，控制字符被转义而不是原样写进日志
	_, err = Parse("ab\x1b[0m")
	if err == nil || strings.Contains(err.Error(), "\x1b") || !strings.Contains(err.Error(), `"ab\x1b[0m"`) {
		t.Fatalf("短值应经 %%q 转义回显：%v", err)
	}
}

func TestCurrentAndIsProduction(t *testing.T) {
	for _, tt := range []struct {
		raw     string
		set     bool
		mode    Mode
		wantErr bool
		prod    bool
	}{
		{"", false, "", true, true},
		{"", true, "", true, true},
		{"  ", true, "", true, true},
		{"Prod", true, Production, false, true},
		{" prd ", true, Production, false, true},
		{"DEV", true, Development, false, false},
		{"development", true, Development, false, false},
		{"testing", true, "", true, true},
		{"staging", true, "", true, true},
	} {
		if tt.set {
			t.Setenv(EnvKey, tt.raw)
		} else {
			t.Setenv(EnvKey, "")
			unsetForTest(t)
		}
		m, err := Current()
		if m != tt.mode || (err != nil) != tt.wantErr {
			t.Errorf("ENV=%q(set=%v): Current() = %q, %v", tt.raw, tt.set, m, err)
		}
		if got := IsProduction(); got != tt.prod {
			t.Errorf("ENV=%q(set=%v): IsProduction() = %v, want %v", tt.raw, tt.set, got, tt.prod)
		}
	}
}

// P2-2：回显上限按字符（rune）计数，16 个字符以内经 %q 回显，17 个起只报长度。
// 多字节用例防止按字节计数（M10），16/17 两侧用例防止边界错位（M12）。
func TestParseEchoBoundaryCountsRunes(t *testing.T) {
	for _, tt := range []struct {
		name  string
		raw   string
		runes int
		shown bool
	}{
		{"ascii-16", strings.Repeat("q", 16), 16, true},
		{"ascii-17", strings.Repeat("q", 17), 17, false},
		{"cjk-16", strings.Repeat("环", 16), 16, true}, // 48 字节
		{"cjk-17", strings.Repeat("环", 17), 17, false},
		{"latin1-16", strings.Repeat("é", 16), 16, true}, // 32 字节
		{"emoji-5", strings.Repeat("😀", 5), 5, true},     // 20 字节
		{"mixed-16", "dev环境" + strings.Repeat("x", 11), 16, true},
		{"mixed-17", "dev环境" + strings.Repeat("x", 12), 17, false},
		{"padded-cjk-16", "  " + strings.Repeat("产", 16) + "\t", 16, true}, // 先去首尾空白再计数
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.raw)
			if !errors.Is(err, ErrUnknown) {
				t.Fatalf("Parse(%q) 应返回 ErrUnknown，got %v", tt.raw, err)
			}
			trimmed := strings.TrimSpace(tt.raw)
			if n := utf8.RuneCountInString(trimmed); n != tt.runes {
				t.Fatalf("用例自检：字符数 %d，期望 %d", n, tt.runes)
			}
			msg := err.Error()
			quoted := strconv.Quote(trimmed)
			if tt.shown {
				if !strings.Contains(msg, quoted) || strings.Contains(msg, "不回显") {
					t.Fatalf("%d 个字符应原样回显，got %q", tt.runes, msg)
				}
				return
			}
			if strings.Contains(msg, trimmed) || strings.Contains(msg, quoted) {
				t.Fatalf("%d 个字符不应回显原值，got %q", tt.runes, msg)
			}
			if want := fmt.Sprintf("（长度 %d，不回显）", tt.runes); !strings.Contains(msg, want) {
				t.Fatalf("应按字符数报长度 %q，got %q", want, msg)
			}
		})
	}
}
