package cache

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestCacheCapturesTypedRenewInterval(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("webhook.common_map_renew_interval_seconds", 99)
	for _, seconds := range []int{0, 1, 300, 86400} {
		cfg := &config.WebhookConfig{CommonMapRenewIntervalSeconds: seconds}
		cache := NewWebhookCache(cfg, zap.NewNop(), nil, nil).(*webhookCache)
		cfg.CommonMapRenewIntervalSeconds = 99
		if cache.renewInterval != time.Duration(seconds)*time.Second {
			t.Fatal("typed refresh interval drifted")
		}
	}
	if NewWebhookCache(nil, zap.NewNop(), nil, nil).(*webhookCache).renewInterval != 0 {
		t.Fatal("nil configuration read global interval")
	}
}
