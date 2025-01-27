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
	// K8s 客户端刷新任务 - 每5分钟
	if err := s.registerTask(
		RefreshK8sClientsTask,
		"@every 5m",
	); err != nil {
		return err
	}

	// Prometheus 缓存刷新任务 - 每10秒
	if err := s.registerTask(
		RefreshPrometheusCacheTask,
		"@every 10s",
	); err != nil {
		return err
	}

	// 主机状态检查任务 - 每10秒
	if err := s.registerTask(
		CheckHostStatusTask,
		"@every 10s",
	); err != nil {
		return err
	}

	// K8s 状态检查任务 - 每10秒
	if err := s.registerTask(
		CheckK8sStatusTask,
		"@every 10s",
	); err != nil {
		return err
	}

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
