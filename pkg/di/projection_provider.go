package di

import (
	"sync"

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
