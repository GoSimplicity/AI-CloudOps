// Package envmode 统一解析后端进程的运行环境（环境变量 ENV）。
//
// 只依赖标准库，供 internal/config、pkg/ssh 等包复用同一套判定，
// 避免各处直接比较字符串而因大小写、空白或别名把生产环境误判为非生产环境。
//
// 规则（窝窝头 2026-10-24 授权口径）：先 TrimSpace 再转小写；
// production、prod、prd 归为 production；development、dev 归为 development；
// 未设置、空串（含只有空白）和其它任何取值（包括 test、testing）一律拒绝。
package envmode

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

// EnvKey 是承载运行环境的环境变量名。
const EnvKey = "ENV"

// Mode 表示规范化后的运行环境。
type Mode string

const (
	// Development 开发环境，对应 config/config.development.yaml。
	Development Mode = "development"
	// Production 生产环境，对应 config/config.production.yaml，并启用全部生产校验。
	Production Mode = "production"
)

// Allowed 是错误信息里列出的允许取值（大小写不敏感，首尾空白忽略）。
const Allowed = "production/prod/prd（生产）、development/dev（开发）"

var (
	// ErrUnset 表示 ENV 未设置、为空串或只有空白。
	ErrUnset = errors.New("环境变量 ENV 未设置或为空，允许的取值：" + Allowed)
	// ErrUnknown 表示 ENV 取值不在允许列表里。
	ErrUnknown = errors.New("环境变量 ENV 取值无法识别")
)

// maxShownLen 是错误信息里回显 ENV 原值的最大字符数；更长的值只报长度，
// 避免把误填进 ENV 的长串（例如口令）原样打进日志。
const maxShownLen = 16

var aliases = map[string]Mode{
	"production":  Production,
	"prod":        Production,
	"prd":         Production,
	"development": Development,
	"dev":         Development,
}

// Parse 按上面的规则解析 raw。失败时返回 ErrUnset 或包装了 ErrUnknown 的错误。
func Parse(raw string) (Mode, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", ErrUnset
	}
	if m, ok := aliases[strings.ToLower(trimmed)]; ok {
		return m, nil
	}
	shown := fmt.Sprintf("%q", trimmed)
	if n := utf8.RuneCountInString(trimmed); n > maxShownLen {
		shown = fmt.Sprintf("（长度 %d，不回显）", n)
	}
	return "", fmt.Errorf("%w: %s，允许的取值：%s", ErrUnknown, shown, Allowed)
}

// Current 解析当前进程的 ENV。
func Current() (Mode, error) {
	return Parse(os.Getenv(EnvKey))
}

// IsProduction 判断当前进程是否按生产环境处理。
//
// 失败即关闭：只有 ENV 明确解析为 development 时返回 false；
// 生产别名、未设置、空串和无法识别的取值都返回 true。
// 启动入口 config.Load 已对后三种直接拒绝启动，这里兜住绕过 Load 的调用方。
func IsProduction() bool {
	m, err := Current()
	return err != nil || m == Production
}
