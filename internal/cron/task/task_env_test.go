package task

import (
	"errors"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestValidateTaskEnvRejectsDangerousNames(t *testing.T) {
	const value = "fake-env-value-canary"
	rejected := []string{
		"LD_PRELOAD", "LD_LIBRARY_PATH", "ld_preload", "Ld_Audit", " LD_PRELOAD ",
		"DYLD_INSERT_LIBRARIES", "dyld_library_path",
		"BASH_ENV", "bash_env", "ENV", "env", "IFS", "ifs",
		"", "  ", "LD_PRELOAD=/tmp/x.so", "FOO=BAR", "A\x00B",
	}
	for _, name := range rejected {
		err := ValidateTaskEnv(model.KeyValueList{{Key: "SAFE", Value: "1"}, {Key: name, Value: value}})
		if !errors.Is(err, ErrForbiddenTaskEnv) {
			t.Fatalf("变量名 %q 应被拒绝, got %v", name, err)
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("错误信息不应包含变量值: %v", err)
		}
	}
}

func TestValidateTaskEnvAllowsOrdinaryNames(t *testing.T) {
	allowed := model.KeyValueList{
		{Key: "PATH", Value: "/usr/bin"}, {Key: "APP_MODE", Value: "x"},
		{Key: "ENVIRONMENT", Value: "dev"}, {Key: "MY_ENV", Value: "1"},
		{Key: "OLD_PATH", Value: "1"}, {Key: "IFS_MODE", Value: "1"}, {Key: "BUILD_DIR", Value: "1"},
	}
	if err := ValidateTaskEnv(allowed); err != nil {
		t.Fatalf("普通变量名不应被拒绝: %v", err)
	}
	if err := ValidateTaskEnv(nil); err != nil {
		t.Fatalf("空环境变量列表应通过: %v", err)
	}
}

// P3-4：补充的拒绝名单（精确名与前缀，均不区分大小写）
func TestValidateTaskEnvRejectsExtendedNames(t *testing.T) {
	const value = "fake-env-value-canary"
	for _, name := range []string{
		"BASH_FUNC_foo%%", "bash_func_x", "SHELLOPTS", "shellopts", "BASHOPTS", "PS4", "ps4",
		"GCONV_PATH", "GLIBC_TUNABLES", "glibc_tunables", "LOCPATH", "MALLOC_CHECK_", "malloc_perturb_", "MALLOC_ARENA_MAX",
	} {
		err := ValidateTaskEnv(model.KeyValueList{{Key: name, Value: value}})
		if !errors.Is(err, ErrForbiddenTaskEnv) {
			t.Fatalf("变量名 %q 应被拒绝, got %v", name, err)
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("错误信息不应包含变量值: %v", err)
		}
	}
	// 只是包含相似片段的普通变量名不受影响
	if err := ValidateTaskEnv(model.KeyValueList{{Key: "MY_MALLOC", Value: "1"}, {Key: "PS1", Value: "1"}, {Key: "XBASH_FUNC_", Value: "1"}}); err != nil {
		t.Fatalf("普通变量名不应被拒绝: %v", err)
	}
}

// P2-2：key/value 都为空的行跳过；key 为空但 value 非空、key 含 '=' 或 NUL 仍拒绝
func TestValidateTaskEnvBlankRows(t *testing.T) {
	if err := ValidateTaskEnv(model.KeyValueList{{Key: "", Value: ""}, {Key: "APP", Value: "1"}, {Key: "  ", Value: ""}}); err != nil {
		t.Fatalf("空行应跳过: %v", err)
	}
	for _, kv := range []model.KeyValue{{Key: "", Value: "v"}, {Key: " ", Value: " "}, {Key: "A=B", Value: ""}, {Key: "A\x00B", Value: ""}} {
		if err := ValidateTaskEnv(model.KeyValueList{kv}); !errors.Is(err, ErrForbiddenTaskEnv) {
			t.Fatalf("%q 应被拒绝, got %v", kv.Key, err)
		}
	}
}

func TestNormalizeTaskEnvDropsBlankRows(t *testing.T) {
	got := NormalizeTaskEnv(model.KeyValueList{{Key: "", Value: ""}, {Key: "A", Value: "1"}, {Key: " ", Value: ""}, {Key: "", Value: "v"}, {Key: "B", Value: ""}})
	want := model.KeyValueList{{Key: "A", Value: "1"}, {Key: "", Value: "v"}, {Key: "B", Value: ""}}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if NormalizeTaskEnv(nil) != nil {
		t.Fatal("nil 应保持 nil")
	}
}
