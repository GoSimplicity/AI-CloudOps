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
	"fmt"
	"net/http"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/di"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, secrets, err := config.LoadWebhook()
	if err != nil {
		if config.IsNonFatal(err) {
			_, _ = fmt.Fprintf(os.Stderr, "配置加载告警，将继续使用环境变量/默认值: %v\n", err)
		} else {
			return err
		}
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("初始化日志失败: %w", err)
	}
	defer func() { _ = logger.Sync() }()

	cmd, err := di.InitWebServer(cfg, di.MySQLDSN(secrets.MySQLAddr))
	if err != nil {
		return err
	}
	cmd.Server.GET("/headers", printHeaders)
	cmd.Start()

	logger.Info("Webhook服务开始监听", zap.String("port", cfg.Port))
	if err := cmd.Server.Run(":" + cfg.Port); err != nil {
		return fmt.Errorf("启动Webhook服务失败: %w", err)
	}
	return nil
}

func printHeaders(c *gin.Context) {
	headers := c.Request.Header
	for key, values := range headers {
		for _, value := range values {
			c.String(http.StatusOK, "%s: %s\n", key, value)
		}
	}
}
