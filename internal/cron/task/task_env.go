package task

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ErrForbiddenTaskEnv 任务自定义环境变量中包含会改变动态链接或 shell 行为的变量
var ErrForbiddenTaskEnv = errors.New("任务环境变量不允许设置")

// forbiddenTaskEnvNames 精确匹配（不区分大小写）的禁止变量名：可让 shell 在启动时执行任意文件或改变分词
var forbiddenTaskEnvNames = map[string]struct{}{
	"BASH_ENV": {},
	"ENV":      {},
	"IFS":      {},
}

// forbiddenTaskEnvPrefixes 前缀匹配（不区分大小写）：动态链接器变量（LD_PRELOAD、DYLD_INSERT_LIBRARIES 等）
var forbiddenTaskEnvPrefixes = []string{"LD_", "DYLD_"}

// ValidateTaskEnv 校验命令行任务的自定义环境变量。
// 变量名统一按大写比较；变量名为空或含 '='、NUL 时也拒绝，避免拼接成 "LD_PRELOAD=..." 绕过检查。
// 错误信息只包含变量名，不包含变量值
func ValidateTaskEnv(env model.KeyValueList) error {
	for _, kv := range env {
		name := strings.TrimSpace(kv.Key)
		if name == "" || strings.ContainsAny(name, "=\x00") {
			return fmt.Errorf("%w: 变量名为空或包含非法字符", ErrForbiddenTaskEnv)
		}
		upper := strings.ToUpper(name)
		if _, ok := forbiddenTaskEnvNames[upper]; ok {
			return fmt.Errorf("%w: %s", ErrForbiddenTaskEnv, name)
		}
		for _, prefix := range forbiddenTaskEnvPrefixes {
			if strings.HasPrefix(upper, prefix) {
				return fmt.Errorf("%w: %s（禁止 %s* 系列）", ErrForbiddenTaskEnv, name, prefix)
			}
		}
	}
	return nil
}
