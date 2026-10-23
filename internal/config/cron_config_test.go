package config

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestCronAllowLocalExecDefaultsOffAndFollowsEnv(t *testing.T) {
	for _, file := range configFiles {
		for _, tt := range []struct {
			name, env string
			want      bool
		}{
			{"unset", "", false}, {"env true", "true", true}, {"env false", "false", false},
		} {
			t.Run(tt.name+"/"+file, func(t *testing.T) {
				viper.Reset()
				t.Cleanup(viper.Reset)
				t.Setenv("ENV", "development")
				for _, name := range requiredEnvNames {
					t.Setenv(name, "fake-"+strings.ToLower(name)+"-"+strings.Repeat("x", 32))
				}
				t.Setenv("CRON_ALLOW_LOCAL_EXEC", tt.env)
				cfg, _, err := loadFromFile(file)
				if err != nil {
					t.Fatalf("load: %v", err)
				}
				if cfg.Cron.AllowLocalExec != tt.want || CronLocalExecAllowed() != tt.want {
					t.Fatalf("allow_local_exec=%v/%v want %v", cfg.Cron.AllowLocalExec, CronLocalExecAllowed(), tt.want)
				}
			})
		}
	}
	t.Run("no config at all", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		if CronLocalExecAllowed() {
			t.Fatal("must default to false")
		}
	})
}
