package task

import (
	"errors"
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ErrForbiddenTaskEnv 任务自定义环境变量中包含会改变动态链接或 shell 行为的变量
var ErrForbiddenTaskEnv = errors.New("任务环境变量不允许设置")

// forbiddenTaskEnvNames 精确匹配（不区分大小写）的禁止变量名：
// 可让 shell 在启动时执行任意文件、改变分词/选项/调试输出，或让 glibc 加载任意模块、调整分配器行为
var forbiddenTaskEnvNames = map[string]struct{}{
	"BASH_ENV":       {},
	"ENV":            {},
	"IFS":            {},
	"SHELLOPTS":      {},
	"BASHOPTS":       {},
	"PS4":            {},
	"GCONV_PATH":     {},
	"GLIBC_TUNABLES": {},
	"LOCPATH":        {},
}

// forbiddenTaskEnvPrefixes 前缀匹配（不区分大小写）：动态链接器变量（LD_PRELOAD、DYLD_INSERT_LIBRARIES 等）、
// bash 导出函数（BASH_FUNC_name%%）、glibc malloc 调参变量（MALLOC_*）
var forbiddenTaskEnvPrefixes = []string{"LD_", "DYLD_", "BASH_FUNC_", "MALLOC_"}

// isBlankTaskEnvRow key（去首尾空白后）与 value 都为空的行：前端新增环境变量时会插入 {key:”,value:”} 空行
func isBlankTaskEnvRow(kv model.KeyValue) bool {
	return strings.TrimSpace(kv.Key) == "" && kv.Value == ""
}

// NormalizeTaskEnv 丢弃 key 与 value 都为空的行，其余行原样保留（顺序不变）。
// 保存前调用；不做其他改写，非法行仍交给 ValidateTaskEnv 拒绝
func NormalizeTaskEnv(env model.KeyValueList) model.KeyValueList {
	if env == nil {
		return nil
	}
	out := make(model.KeyValueList, 0, len(env))
	for _, kv := range env {
		if isBlankTaskEnvRow(kv) {
			continue
		}
		out = append(out, kv)
	}
	return out
}

// ValidateTaskEnv 校验命令行任务（JobType==Command）的自定义环境变量。
// key 与 value 都为空的行跳过；key 为空但 value 非空、key 含 '=' 或 NUL 时拒绝（避免拼接成 "LD_PRELOAD=..." 绕过检查）；
// 变量名统一按大写与拒绝名单比较。错误信息只包含变量名，不包含变量值
func ValidateTaskEnv(env model.KeyValueList) error {
	for _, kv := range env {
		if isBlankTaskEnvRow(kv) {
			continue
		}
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
