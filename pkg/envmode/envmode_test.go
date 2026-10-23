package envmode

import (
	"errors"
	"strings"
	"testing"
)

func TestParseNormalizesCaseWhitespaceAndAliases(t *testing.T) {
	for raw, want := range map[string]Mode{
		"production":     Production,
		"Production":     Production,
		" production":    Production,
		"production\n":   Production,
		"\tPRODUCTION  ": Production,
		"prod":           Production,
		"PROD":           Production,
		"prd":            Production,
		" Prd ":          Production,
		"development":    Development,
		"Development":    Development,
		"dev":            Development,
		" DEV ":          Development,
		"test":           Test,
		"TEST":           Test,
		"testing":        Test,
	} {
		got, err := Parse(raw)
		if err != nil || got != want {
			t.Fatalf("Parse(%q)=%q,%v want %q", raw, got, err, want)
		}
	}
}

func TestParseRejectsUnsetAndUnknown(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t\n"} {
		if _, err := Parse(raw); !errors.Is(err, ErrUnset) {
			t.Fatalf("Parse(%q) err=%v want ErrUnset", raw, err)
		}
	}
	if !strings.Contains(ErrUnset.Error(), "ENV=development") || !strings.Contains(ErrUnset.Error(), "ENV=production") {
		t.Fatalf("未设置提示应给出可选取值: %v", ErrUnset)
	}
	for _, raw := range []string{"staging", "produciton", "pro", "production1", "local", "prod uction"} {
		_, err := Parse(raw)
		if !errors.Is(err, ErrUnknown) || errors.Is(err, ErrUnset) {
			t.Fatalf("Parse(%q) err=%v want ErrUnknown", raw, err)
		}
		if !strings.Contains(err.Error(), "production") {
			t.Fatalf("未知取值提示应列出允许值: %v", err)
		}
	}
}

func TestParseDoesNotEchoLongValues(t *testing.T) {
	long := "x-" + strings.Repeat("q", maxShownLen+8)
	_, err := Parse(long)
	if !errors.Is(err, ErrUnknown) || strings.Contains(err.Error(), long) {
		t.Fatalf("长取值不应回显: %v", err)
	}
	_, err = Parse("Staging")
	if err == nil || !strings.Contains(err.Error(), `"staging"`) {
		t.Fatalf("短取值应回显便于排障: %v", err)
	}
}

func TestIsProductionFailsClosed(t *testing.T) {
	for _, tt := range []struct {
		env  string
		want bool
	}{
		{"production", true},
		{"Production", true},
		{" production", true},
		{"prod", true},
		{"prd", true},
		{"PROD", true},
		{"", true},        // 未设置：按生产处理
		{"staging", true}, // 未知：按生产处理
		{"development", false},
		{"dev", false},
		{"test", false},
	} {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv(EnvKey, tt.env)
			if got := IsProduction(); got != tt.want {
				t.Fatalf("ENV=%q IsProduction=%v want %v", tt.env, got, tt.want)
			}
		})
	}
}
