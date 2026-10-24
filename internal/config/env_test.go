package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
)

func TestIsProductionNormalizesEnv(t *testing.T) {
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
		{"development", false},
		{"dev", false},
		{"test", true},
		{"testing", true},
		{"", true},
		{"staging", true},
	} {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			if got := IsProduction(); got != tt.want {
				t.Fatalf("ENV=%q IsProduction=%v want %v", tt.env, got, tt.want)
			}
		})
	}
}

func TestResolveConfigFileByEnv(t *testing.T) {
	for env, want := range map[string]string{
		"production":  "config/config.production.yaml",
		"Production":  "config/config.production.yaml",
		" production": "config/config.production.yaml",
		"prod":        "config/config.production.yaml",
		"prd":         "config/config.production.yaml",
		"PROD":        "config/config.production.yaml",
		"development": "config/config.development.yaml",
		"dev":         "config/config.development.yaml",
	} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			got, err := resolveConfigFile("")
			if err != nil || got != want {
				t.Fatalf("ENV=%q got=%q err=%v want %q", env, got, err, want)
			}
		})
	}
}

func TestResolveConfigFileRejectsUnsetAndUnknownEnv(t *testing.T) {
	for _, tt := range []struct {
		env  string
		want error
	}{
		{"", envmode.ErrUnset},
		{"  ", envmode.ErrUnset},
		{"staging", envmode.ErrUnknown},
		{"test", envmode.ErrUnknown},
		{"testing", envmode.ErrUnknown},
		{"produciton", envmode.ErrUnknown},
	} {
		t.Run(tt.env, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			for _, explicit := range []string{"", "custom.yaml"} {
				got, err := resolveConfigFile(explicit)
				if !errors.Is(err, tt.want) || got != "" {
					t.Fatalf("ENV=%q explicit=%q got=%q err=%v", tt.env, explicit, got, err)
				}
				if !strings.Contains(err.Error(), "ENV") {
					t.Fatalf("错误提示应指明 ENV: %v", err)
				}
			}
		})
	}
	t.Setenv("ENV", "")
	_, err := resolveConfigFile("")
	if !strings.Contains(err.Error(), envmode.Allowed) {
		t.Fatalf("未设置时应提示允许的 ENV 取值: %v", err)
	}
}

func TestResolveConfigFileKeepsExplicitFlag(t *testing.T) {
	t.Setenv("ENV", "Prod")
	if got, err := resolveConfigFile("custom.yaml"); err != nil || got != "custom.yaml" {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestValidateAppliesProductionChecksForAliases(t *testing.T) {
	for _, env := range []string{"Production", " production", "prod", "prd", "PROD", ""} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("ENV", env)
			c := prodTestConfig()
			c.Mock.Enabled = true
			if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "MOCK_ENABLED") {
				t.Fatalf("ENV=%q 应按生产校验拒绝 mock: %v", env, err)
			}
		})
	}
	t.Setenv("ENV", "development")
	c := validConfigForTest()
	c.Mock.Enabled = true
	if err := c.Validate(); err != nil {
		t.Fatalf("开发环境允许 mock: %v", err)
	}
}
