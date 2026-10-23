/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

// Package envmode 统一解析进程运行环境（环境变量 ENV）。
//
// 该包只依赖标准库，供 internal/config、pkg/ssh 等不同层级的包复用同一套判定，
// 避免各处自行比较字符串导致大小写、空白或别名不一致而误判为非生产环境。
package envmode

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// EnvKey 是承载运行环境的环境变量名。
const EnvKey = "ENV"

// Mode 表示规范化后的运行环境。
type Mode string

const (
	// Development 本地开发环境，对应 config/config.development.yaml。
	Development Mode = "development"
	// Test 测试环境；仓库没有独立的测试配置文件，与开发环境共用 config/config.development.yaml。
	Test Mode = "test"
	// Production 生产环境，对应 config/config.production.yaml，并启用全部生产校验。
	Production Mode = "production"
)

var (
	// ErrUnset 表示 ENV 未设置（或只有空白）。
	ErrUnset = errors.New("环境变量 ENV 未设置，请设置 ENV=development 或 ENV=production")
	// ErrUnknown 表示 ENV 取值无法识别。
	ErrUnknown = errors.New("环境变量 ENV 取值无法识别")
)

// maxShownLen 是错误信息中回显 ENV 原值的最大长度。
const maxShownLen = 16

// aliases 将规范化（TrimSpace + ToLower）后的取值映射到 Mode。
var aliases = map[string]Mode{
	"production":  Production,
	"prod":        Production,
	"prd":         Production,
	"development": Development,
	"dev":         Development,
	"test":        Test,
	"testing":     Test,
}

// Parse 解析 ENV 取值：先 TrimSpace、ToLower，再按别名表匹配。
// 未设置返回 ErrUnset，未知取值返回包装了 ErrUnknown 的错误，二者都不会被当作开发环境。
func Parse(raw string) (Mode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return "", ErrUnset
	}
	if m, ok := aliases[v]; ok {
		return m, nil
	}
	// 只回显短取值，避免把误填进 ENV 的长串（例如口令）原样打进日志。
	shown := fmt.Sprintf("%q", v)
	if len(v) > maxShownLen {
		shown = fmt.Sprintf("(长度%d，不回显)", len(v))
	}
	return "", fmt.Errorf("%w: %s，允许的取值为 development/dev、test/testing、production/prod/prd", ErrUnknown, shown)
}

// Current 解析当前进程的 ENV。
func Current() (Mode, error) {
	return Parse(os.Getenv(EnvKey))
}

// IsProduction 判断当前进程是否应按生产环境处理。
//
// 失败即关闭（fail closed）：只有 ENV 明确解析为开发/测试环境时才返回 false；
// ENV 为生产别名、未设置或无法识别时都返回 true，调用方据此执行最严格的策略。
// 启动入口（config.Load）会对未设置/未知取值直接拒绝启动，这里的兜底用于绕过 Load 的调用方。
func IsProduction() bool {
	m, err := Current()
	return err != nil || m == Production
}
