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

// 第二层审查：变量名先按白名单格式校验，再过拒绝名单（归一化：去首尾空白、统一大写）
func TestValidateTaskEnvNameAllowlistFormat(t *testing.T) {
	const value = "fake-env-value-canary"
	rejected := []string{
		// bash 导出函数（实测可注入）
		"BASH_FUNC_echo%%", "BASH_FUNC_x%%", "bash_func_echo%%", " BASH_FUNC_echo%% ",
		// 夹带零宽字符 / 格式字符的变体
		"L\u200bD_PRELOAD", "LD_\u200bPRELOAD", "\ufeffLD_PRELOAD", "BASH\u202e_ENV", "PS\u200b4",
		// SHELLOPTS=xtrace 配合 PS4=$(...) 可注入
		"SHELLOPTS", "shellopts", " SHELLOPTS ", "PS4", "ps4", "\tPS4\n",
		"PROMPT_COMMAND", "prompt_command", " Prompt_Command ",
		"BASH_ENV", "bash_env", " BASH_ENV ", "ENV", "env", " Env ",
		// 格式非法
		"1ABC", "A-B", "A.B", "A B", "A=B", "A\x00B", "Ä_VAR", "$PATH",
	}
	for _, name := range rejected {
		err := ValidateTaskEnv(model.KeyValueList{{Key: "SAFE_NAME", Value: "1"}, {Key: name, Value: value}})
		if !errors.Is(err, ErrForbiddenTaskEnv) {
			t.Fatalf("变量名 %q 应被拒绝, got %v", name, err)
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("错误信息不应包含变量值: %v", err)
		}
	}
	allowed := model.KeyValueList{
		{Key: "PATH", Value: "/usr/bin"}, {Key: "_PRIVATE", Value: "1"}, {Key: "app_mode", Value: "x"},
		{Key: "Var2", Value: "1"}, {Key: " TRIMMED_OK ", Value: "1"}, {Key: "A", Value: "1"},
		{Key: "PYTHONPATH", Value: "/opt/lib"}, {Key: "NODE_OPTIONS", Value: "--max-old-space-size=512"}, {Key: "PERL5OPT", Value: "-w"},
		{Key: "PS1", Value: "1"}, {Key: "PROMPT", Value: "1"},
	}
	if err := ValidateTaskEnv(allowed); err != nil {
		t.Fatalf("合法变量名不应被拒绝: %v", err)
	}
}

// NormalizeTaskEnv 存库前去掉 key 首尾空白，不改大小写和 value
func TestNormalizeTaskEnvTrimsKeys(t *testing.T) {
	got := NormalizeTaskEnv(model.KeyValueList{{Key: " FOO ", Value: " v "}, {Key: "\tBar\n", Value: "x"}, {Key: "  ", Value: ""}})
	want := model.KeyValueList{{Key: "FOO", Value: " v "}, {Key: "Bar", Value: "x"}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// 去首尾空白后的变量名重复即拒绝；大小写不同不算重复（Linux 环境变量区分大小写，os/exec 也按区分大小写去重）
func TestValidateTaskEnvRejectsDuplicateNames(t *testing.T) {
	const value = "fake-env-value-canary"
	for _, env := range []model.KeyValueList{
		{{Key: "FOO", Value: "1"}, {Key: "FOO", Value: value}},
		{{Key: " FOO", Value: "1"}, {Key: "FOO", Value: value}},
		{{Key: "FOO", Value: "1"}, {Key: "BAR", Value: "2"}, {Key: "FOO\t", Value: value}},
		{{Key: "foo", Value: "1"}, {Key: " foo ", Value: value}},
	} {
		err := ValidateTaskEnv(env)
		if !errors.Is(err, ErrForbiddenTaskEnv) || !strings.Contains(err.Error(), "变量名重复") {
			t.Fatalf("%v 应因变量名重复被拒绝, got %v", env, err)
		}
		if strings.Contains(err.Error(), value) {
			t.Fatalf("错误信息不应包含变量值: %v", err)
		}
	}
	// 空行（key/value 都为空）可以出现多次，不算重复
	if err := ValidateTaskEnv(model.KeyValueList{{Key: "", Value: ""}, {Key: " ", Value: ""}, {Key: "A", Value: "1"}}); err != nil {
		t.Fatalf("多个空行不应算重复: %v", err)
	}
	// 大小写不同是两个变量
	if err := ValidateTaskEnv(model.KeyValueList{{Key: "FOO", Value: "1"}, {Key: "foo", Value: "2"}, {Key: "Foo", Value: "3"}}); err != nil {
		t.Fatalf("大小写不同不应算重复: %v", err)
	}
}
