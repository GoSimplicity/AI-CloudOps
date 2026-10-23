package config

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// callLoadIsolated 在隔离的命令行环境里调用 Load：
// os.Args 只留程序名和给定参数（不带 go test 的 -test.* 参数），pflag.CommandLine 每次换新
// （Load 会重复定义 --config），结束后恢复全局状态并清空 viper。
func callLoadIsolated(t *testing.T, args ...string) (*Config, *ExternalConfig, error) {
	t.Helper()
	origArgs, origFlags := os.Args, pflag.CommandLine
	t.Cleanup(func() {
		os.Args, pflag.CommandLine = origArgs, origFlags
		viper.Reset()
	})
	os.Args = append([]string{"cloudops-load-test"}, args...)
	fs := pflag.NewFlagSet("cloudops-load-test", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	pflag.CommandLine = fs
	viper.Reset()
	return Load()
}

// P2-3：启动入口 Load 必须把 ENV 校验错误原样返回（不论是否传了 --config），不能吞掉后继续按默认文件加载。
func TestLoadRejectsInvalidEnv(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "custom.yaml")
	if err := os.WriteFile(cfgFile, []byte("server:\n  port: \"8889\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		raw  string
		set  bool
		args []string
		want error
	}{
		{"unset", "", false, nil, envmode.ErrUnset},
		{"empty", "", true, nil, envmode.ErrUnset},
		{"blank", " \t ", true, nil, envmode.ErrUnset},
		{"test", "test", true, nil, envmode.ErrUnknown},
		{"production-x", "production-x", true, nil, envmode.ErrUnknown},
		{"long-17", strings.Repeat("q", 17), true, nil, envmode.ErrUnknown},
		{"unset+config", "", false, []string{"--config", cfgFile}, envmode.ErrUnset},
		{"staging+config", "staging", true, []string{"--config=" + cfgFile}, envmode.ErrUnknown},
	} {
		t.Run(tt.name, func(t *testing.T) {
			setOrUnsetEnv(t, tt.raw, tt.set)
			cfg, ext, err := callLoadIsolated(t, tt.args...)
			if !errors.Is(err, tt.want) {
				t.Fatalf("ENV=%q(set=%v) args=%v: Load() err=%v，期望包装 %v", tt.raw, tt.set, tt.args, err, tt.want)
			}
			if IsNonFatal(err) {
				t.Fatalf("ENV 校验失败不能按“可继续运行”的告警返回：%v", err)
			}
			if cfg != nil || ext != nil {
				t.Fatalf("ENV 校验失败时不应返回配置：cfg=%v ext=%v", cfg != nil, ext != nil)
			}
			if !strings.Contains(err.Error(), "启动环境校验失败") {
				t.Fatalf("错误信息应说明是启动环境校验失败：%v", err)
			}
		})
	}
	// 对照：合法 ENV 能越过 ENV 校验（之后的结果与本用例无关，只要不是 ENV 错误）
	setOrUnsetEnv(t, " Dev ", true)
	if _, _, err := callLoadIsolated(t, "--config", cfgFile); errors.Is(err, envmode.ErrUnset) || errors.Is(err, envmode.ErrUnknown) {
		t.Fatalf("ENV=\" Dev \" 不应报 ENV 错误：%v", err)
	}
}
