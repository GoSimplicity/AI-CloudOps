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

package di

import (
	"context"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func InitRedis(cfg *config.Config, logger *zap.Logger) (*redis.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// 初始化 Redis 客户端
	client := redis.NewClient(&redis.Options{
		ContextTimeoutEnabled: true,
		Addr:                  cfg.Redis.Addr,
		Password:              cfg.Redis.Password,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Redis连接失败",
			zap.String("addr", cfg.Redis.Addr),
			zap.Error(err),
		)
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	logger.Info("Redis初始化成功", zap.String("addr", cfg.Redis.Addr))
	return client, nil
}
