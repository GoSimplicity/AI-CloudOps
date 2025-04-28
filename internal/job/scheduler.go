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
package job

import (
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const (
	RefreshK8sClientsTask      = "refresh_k8s_clients"
	RefreshPrometheusCacheTask = "refresh_prometheus_cache"
	CheckHostStatusTask        = "check_host_status"
	CheckK8sStatusTask         = "check_k8s_status"
)

type TimedScheduler struct {
	scheduler *asynq.Scheduler
}

func NewTimedScheduler(scheduler *asynq.Scheduler) *TimedScheduler {
	return &TimedScheduler{
		scheduler: scheduler,
	}
}

func (s *TimedScheduler) RegisterTimedTasks() error {
	// // K8s 客户端刷新任务 - 每5分钟
	// if err := s.registerTask(
	// 	RefreshK8sClientsTask,
	// 	"@every 5m",
	// ); err != nil {
	// 	return err
	// }

	// Prometheus 缓存刷新任务 - 每10秒
	if err := s.registerTask(
		RefreshPrometheusCacheTask,
		"@every 30s",
	); err != nil {
		return err
	}

	// // 主机状态检查任务 - 每10秒
	// if err := s.registerTask(
	// 	CheckHostStatusTask,
	// 	"@every 10s",
	// ); err != nil {
	// 	return err
	// }

	// // K8s 状态检查任务 - 每10秒
	// if err := s.registerTask(
	// 	CheckK8sStatusTask,
	// 	"@every 10s",
	// ); err != nil {
	// 	return err
	// }

	return nil
}

func (s *TimedScheduler) registerTask(taskName, cronSpec string) error {
	payload := TimedPayload{
		TaskName:    taskName,
		LastRunTime: time.Now(),
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	task := asynq.NewTask(DeferTimedTask, payloadBytes)
	_, err = s.scheduler.Register(cronSpec, task)
	return err
}

func (s *TimedScheduler) Run() error {
	return s.scheduler.Run()
}

func (s *TimedScheduler) Stop() {
	s.scheduler.Shutdown()
}
