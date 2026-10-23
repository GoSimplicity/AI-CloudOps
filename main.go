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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := loadDotEnv(dotEnvFile); err != nil {
		return err
	}

	cfg, _, err := config.Load()
	if err != nil {
		if config.IsNonFatal(err) {
			_, _ = fmt.Fprintf(os.Stderr, "配置加载告警，将继续使用环境变量/默认值: %v\n", err)
		} else {
			return err
		}
	}

	application, err := di.ProvideApp(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	return application.Run(ctx)
}

// dotEnvFile 启动时读取的本地环境变量文件
const dotEnvFile = ".env"

// errDotEnvInvalid .env 存在但无法读取或解析。固定文案：godotenv 的原始错误可能包含
// 出错行的原文（即密钥明文），因此不得输出或包装原始错误
var errDotEnvInvalid = errors.New("加载 .env 失败：文件无法读取或格式错误（为避免泄露密钥，不显示具体内容），请检查该文件")

// loadDotEnv 加载 .env：文件不存在时继续使用环境变量/配置文件；存在但出错时启动失败
func loadDotEnv(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		_, _ = fmt.Fprintln(os.Stderr, "未找到 .env，将使用环境变量/配置文件")
		return nil
	}
	if err := godotenv.Load(path); err != nil {
		return errDotEnvInvalid
	}
	return nil
}
