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
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	cronHandler "github.com/GoSimplicity/AI-CloudOps/internal/cron/handler"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
)

func InitAsynqClient(cfg *config.Config) *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	})
}

func InitAsynqServer(cfg *config.Config, manager *notification.Manager) *asynq.Server {
	return asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
		},
		workerConfig(manager),
	)
}

func InitScheduler(cfg *config.Config) *asynq.Scheduler {
	return asynq.NewScheduler(asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
	}, nil)
}

// 复用现有 default 队列和 Cron 预算规则，通知仅覆盖自己的重试间隔
func workerConfig(manager *notification.Manager) asynq.Config {
	return asynq.Config{
		IsFailure:      cronHandler.IsTaskFailure,
		Concurrency:    10,
		Queues:         map[string]int{"default": 1},
		RetryDelayFunc: manager.RetryDelay,
	}
}
