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
