package consumer

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"testing"
)

func TestConsumerCapturesTypedWorkers(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("webhook.fixed_workers", 99)
	for _, workers := range []int{0, 1, 10, 1024} {
		cfg := &config.WebhookConfig{FixedWorkers: workers}
		consumer := NewWebhookConsumer(cfg, zap.NewNop(), nil, nil, nil, nil).(*webhookConsumer)
		cfg.FixedWorkers = 99
		if consumer.workerCount != workers {
			t.Fatal("typed worker configuration drifted")
		}
	}
	if NewWebhookConsumer(nil, zap.NewNop(), nil, nil, nil, nil).(*webhookConsumer).workerCount != 0 {
		t.Fatal("nil configuration read global worker count")
	}
}
