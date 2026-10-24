package task

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ErrForbiddenTaskEnv 任务自定义环境变量中包含会改变动态链接或 shell 行为的变量
var ErrForbiddenTaskEnv = errors.New("任务环境变量不允许设置")

// forbiddenTaskEnvNames 精确匹配（不区分大小写）的禁止变量名：
// 可让 shell 在启动时执行任意文件或命令（BASH_ENV、ENV、PROMPT_COMMAND）、改变分词/选项/调试输出
// （IFS、SHELLOPTS=xtrace 配合 PS4=$(...) 可执行命令），或让 glibc 加载任意模块、调整分配器行为。
// PYTHONPATH、NODE_OPTIONS、PERL5OPT 等解释器变量有正当用途，暂不禁止（风险见 README 与 docs/release/w1-cron-env-audit.md）
var forbiddenTaskEnvNames = map[string]struct{}{
	"BASH_ENV":       {},
	"ENV":            {},
	"IFS":            {},
	"SHELLOPTS":      {},
	"BASHOPTS":       {},
	"PS4":            {},
	"PROMPT_COMMAND": {},
	"GCONV_PATH":     {},
	"GLIBC_TUNABLES": {},
	"LOCPATH":        {},
}

// forbiddenTaskEnvPrefixes 前缀匹配（不区分大小写）：动态链接器变量（LD_PRELOAD、DYLD_INSERT_LIBRARIES 等）、
// bash 导出函数（BASH_FUNC_name%%）、glibc malloc 调参变量（MALLOC_*）
var forbiddenTaskEnvPrefixes = []string{"LD_", "DYLD_", "BASH_FUNC_", "MALLOC_"}

// taskEnvNamePattern 变量名白名单格式（POSIX 可移植变量名）：字母或下划线开头，其后只能是字母、数字、下划线。
// 先按格式拒绝，再过拒绝名单；可拦掉 BASH_FUNC_x%%、夹带零宽字符（如 U+200B）或其他 Unicode 字符的变体
var taskEnvNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// isBlankTaskEnvRow key（去首尾空白后）与 value 都为空的行：前端新增环境变量时会插入 {key:”,value:”} 空行
func isBlankTaskEnvRow(kv model.KeyValue) bool {
	return strings.TrimSpace(kv.Key) == "" && kv.Value == ""
}

// NormalizeTaskEnv 丢弃 key 与 value 都为空的行，其余行的 key 去首尾空白、value 原样保留（顺序不变）。
// 保存前与注入子进程前调用；不做其他改写（不改大小写），非法行仍交给 ValidateTaskEnv 拒绝
func NormalizeTaskEnv(env model.KeyValueList) model.KeyValueList {
	if env == nil {
		return nil
	}
	out := make(model.KeyValueList, 0, len(env))
	for _, kv := range env {
		if isBlankTaskEnvRow(kv) {
			continue
		}
		kv.Key = strings.TrimSpace(kv.Key)
		out = append(out, kv)
	}
	return out
}

// ValidateTaskEnv 校验命令行任务（JobType==Command）的自定义环境变量。
// 这是纵深防御：类型 2 任务本身就能执行任意命令，这里只防止通过环境变量改变动态链接器或 shell 的行为。
// 规则：key 与 value 都为空的行跳过；变量名去首尾空白后必须匹配 ^[A-Za-z_][A-Za-z0-9_]*$（同时挡住空名、'='、NUL、
// '%'、零宽字符等），再统一转大写与拒绝名单比较；去首尾空白后的变量名重复（区分大小写，与 Linux 及 os/exec
// 注入行为一致：FOO 与 foo 是两个变量，同名时只有最后一个生效）时拒绝。错误信息只包含变量名（格式非法时用 %q 显示不可见字符），不包含变量值
func ValidateTaskEnv(env model.KeyValueList) error {
	seen := make(map[string]struct{}, len(env))
	for _, kv := range env {
		if isBlankTaskEnvRow(kv) {
			continue
		}
		name := strings.TrimSpace(kv.Key)
		if name == "" {
			return fmt.Errorf("%w: 变量名为空", ErrForbiddenTaskEnv)
		}
		if !taskEnvNamePattern.MatchString(name) {
			return fmt.Errorf("%w: 变量名 %q 包含非法字符（只允许字母、数字、下划线，且不能以数字开头）", ErrForbiddenTaskEnv, name)
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
		if _, dup := seen[name]; dup {
			return fmt.Errorf("%w: 变量名重复: %s", ErrForbiddenTaskEnv, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
