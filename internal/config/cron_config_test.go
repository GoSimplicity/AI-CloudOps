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
				if cfg.Cron.AllowLocalExec != tt.want || NewCronLocalExecPolicy().IsAllowed() != tt.want {
					t.Fatalf("allow_local_exec=%v/%v want %v", cfg.Cron.AllowLocalExec, NewCronLocalExecPolicy().IsAllowed(), tt.want)
				}
			})
		}
	}
	t.Run("no config at all", func(t *testing.T) {
		viper.Reset()
		t.Cleanup(viper.Reset)
		if NewCronLocalExecPolicy().IsAllowed() {
			t.Fatal("must default to false")
		}
	})
}

func TestCronLocalExecPolicyReadsCurrentConfiguration(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	loaded := viper.GetViper()
	policy := NewCronLocalExecPolicy()
	for _, tt := range []struct {
		name  string
		value bool
	}{
		{"enable", true}, {"revoke", false}, {"enable again", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			loaded.Set(CronAllowLocalExecKey, tt.value)
			if policy.IsAllowed() != tt.value {
				t.Fatal("policy cached an earlier permission value")
			}
		})
	}
	// 已构造的应用绑定原配置实例，不能被另一应用/测试替换全局实例。
	viper.Reset()
	viper.Set(CronAllowLocalExecKey, false)
	if !policy.IsAllowed() {
		t.Fatal("policy followed a replacement global instance")
	}
}

func TestCronLocalExecPolicyReadsEnvironmentAtEveryCheck(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	policy := NewCronLocalExecPolicy()
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{"true", true}, {"false", false}, {"", false}, {"invalid", false}, {"true", true},
	} {
		t.Setenv("CRON_ALLOW_LOCAL_EXEC", tt.value)
		if policy.IsAllowed() != tt.want {
			t.Fatalf("dynamic environment=%q: want %v", tt.value, tt.want)
		}
	}
}

func TestNilCronLocalExecPolicyDeniesExecution(t *testing.T) {
	var policy CronLocalExecPolicy
	if policy.IsAllowed() {
		t.Fatal("missing policy must fail closed")
	}
}
