package di

import (
	"sync"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/cache"
	"github.com/redis/go-redis/v9"
)

// Wire owns this dedicated, no-retry Store pool. A later construction failure
// runs cleanup automatically; main defers it until application writers stop.
func ProvideProjectionStore(client *redis.Client) (*cache.ProjectionStore, func(), error) {
	store, err := cache.NewProjectionStore(client)
	if err != nil {
		return nil, nil, err
	}
	var once sync.Once
	cleanup := func() { once.Do(func() { _ = store.Close() }) }
	return store, cleanup, nil
}

func ProvideProjectionRecoveryOptions(cfg *config.Config) cache.ProjectionRecoveryOptions {
	options := cache.DefaultProjectionRecoveryOptions()
	if cfg == nil {
		return options
	}
	values := cfg.Prometheus
	if values.ProjectionRecoveryInterval != 0 {
		options.Interval = values.ProjectionRecoveryInterval
	}
	if values.ProjectionRecoveryItemTimeout != 0 {
		options.ItemTimeout = values.ProjectionRecoveryItemTimeout
	}
	if values.ProjectionRecoveryCycleTimeout != 0 {
		options.CycleTimeout = values.ProjectionRecoveryCycleTimeout
	}
	if values.ProjectionRecoveryPageSize != 0 {
		options.PageSize = values.ProjectionRecoveryPageSize
	}
	if values.ProjectionRecoveryMaxPerCycle != 0 {
		options.MaxPerCycle = values.ProjectionRecoveryMaxPerCycle
	}
	return options
}
