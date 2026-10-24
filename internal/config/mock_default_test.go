package config

import (
	"testing"

	"github.com/spf13/viper"
)

func loadMockEnabledFromDefaults(t *testing.T) bool {
	t.Helper()
	viper.Reset()
	t.Cleanup(viper.Reset)
	setDefaults()
	viper.AutomaticEnv()
	bindEnvVars()
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		t.Fatal(err)
	}
	return cfg.Mock.Enabled
}

func TestMockDefaultFollowsEnvironment(t *testing.T) {
	for _, tt := range []struct {
		name, env, mockEnv string
		want               bool
	}{
		{"production defaults off", "production", "", false},
		{"development defaults on", "development", "", true},
		{"unset ENV fails closed", "", "", false},
		{"unknown ENV fails closed", "staging", "", false},
		{"test is not a supported mode", "test", "", false},
		{"testing is not a supported mode", "testing", "", false},
		{"production alias defaults off", " PROD ", "", false},
		{"dev alias defaults on", "Dev", "", true},
		{"explicit off in development", "development", "false", false},
		{"explicit on in production still loads (rejected by Validate)", "production", "true", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ENV", tt.env)
			t.Setenv("MOCK_ENABLED", tt.mockEnv)
			if got := loadMockEnabledFromDefaults(t); got != tt.want {
				t.Fatalf("mock.enabled=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestShippedConfigFilesSetMockExplicitly(t *testing.T) {
	for file, want := range map[string]bool{
		"../../config/config.production.yaml":  false,
		"../../config/config.development.yaml": true,
	} {
		v := viper.New()
		v.SetConfigFile(file)
		if err := v.ReadInConfig(); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if !v.IsSet("mock.enabled") || v.GetBool("mock.enabled") != want {
			t.Fatalf("%s: mock.enabled 应显式为 %v", file, want)
		}
	}
}
