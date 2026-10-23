package di

import (
	"errors"
	"reflect"
	"testing"
	"time"

	cronHandler "github.com/GoSimplicity/AI-CloudOps/internal/cron/handler"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func TestWorkerConfigKeepsSharedQueueAndCronSemantics(t *testing.T) {
	manager, err := notification.NewManager(emptyNotificationConfig{}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	cfg := workerConfig(manager)
	if cfg.Concurrency != 10 || len(cfg.Queues) != 1 || cfg.Queues["default"] != 1 {
		t.Fatal("shared queue or concurrency changed")
	}
	if reflect.ValueOf(cfg.IsFailure).Pointer() != reflect.ValueOf(cronHandler.IsTaskFailure).Pointer() {
		t.Fatal("Cron lease failure classifier replaced")
	}
	if cfg.RetryDelayFunc == nil {
		t.Fatal("retry routing not wired")
	}
	// Asynq v0.22.0 默认 n=0 为 15~44 秒，通知不可覆盖 Cron 的原生退避
	for _, kind := range []string{"cron:task", "unknown"} {
		got := cfg.RetryDelayFunc(0, errors.New("fake failure"), asynq.NewTask(kind, nil))
		if got < 15*time.Second || got > 44*time.Second {
			t.Fatalf("%s native retry delay changed: %s", kind, got)
		}
	}
}

type emptyNotificationConfig struct{}

func (emptyNotificationConfig) GetEmail() notification.EmailConfig   { return nil }
func (emptyNotificationConfig) GetFeishu() notification.FeishuConfig { return nil }

type asyncInboxConfig struct{ base string }

func (asyncInboxConfig) GetEmail() notification.EmailConfig     { return nil }
func (c asyncInboxConfig) GetFeishu() notification.FeishuConfig { return asyncInboxFeishu{c.base} }

type asyncInboxFeishu struct{ base string }

func (asyncInboxFeishu) IsEnabled() bool                   { return true }
func (asyncInboxFeishu) GetMaxRetries() int                { return 4 }
func (asyncInboxFeishu) GetRetryInterval() time.Duration   { return time.Minute }
func (asyncInboxFeishu) GetTimeout() time.Duration         { return 2 * time.Second }
func (asyncInboxFeishu) GetChannelName() string            { return "feishu" }
func (asyncInboxFeishu) Validate() error                   { return nil }
func (asyncInboxFeishu) GetAppID() string                  { return "local-inbox-only" }
func (asyncInboxFeishu) GetAppSecret() string              { return "local-inbox-only" }
func (c asyncInboxFeishu) GetWebhookURL() string           { return c.base + "/forbidden-webhook" }
func (c asyncInboxFeishu) GetPrivateMessageAPI() string    { return c.base + "/messages" }
func (c asyncInboxFeishu) GetTenantAccessTokenAPI() string { return c.base + "/token" }

func TestWorkerConfigNotificationDelayUsesRequestOverrides(t *testing.T) {
	manager, err := notification.NewManager(asyncInboxConfig{"http://127.0.0.1"}, nil, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	cfg := workerConfig(manager)
	zero, one := 0, 1
	for _, tt := range []struct {
		name    string
		minutes *int
		want    time.Duration
	}{
		{"channel default", nil, time.Minute}, {"zero", &zero, 0}, {"one minute", &one, time.Minute},
	} {
		t.Run(tt.name, func(t *testing.T) {
			task, err := notification.CreateSendNotificationTask(&notification.SendRequest{Channel: "feishu", RetryIntervalMinutes: tt.minutes}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got := cfg.RetryDelayFunc(0, errors.New("fake"), task); got != tt.want {
				t.Fatalf("got=%s want=%s", got, tt.want)
			}
		})
	}
}
