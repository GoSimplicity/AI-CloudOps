package envmode

import (
	"errors"
	"os"
	"strings"
	"testing"
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
