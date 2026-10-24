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

// Package runenv 统一判断进程运行环境（环境变量 ENV）是否为生产环境。
// 只依赖标准库，供 internal/config、pkg/ssh 等不同层级的包复用同一判断，避免各处自行比较字符串。
package runenv

import (
	"os"
	"strings"
)

// EnvKey 承载运行环境的环境变量名
const EnvKey = "ENV"

// IsProductionValue 判断给定的 ENV 取值是否表示生产环境：去首尾空白后忽略大小写等于 "production"
func IsProductionValue(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "production")
}

// IsProduction 当前进程的 ENV 是否表示生产环境
func IsProduction() bool {
	return IsProductionValue(os.Getenv(EnvKey))
}
