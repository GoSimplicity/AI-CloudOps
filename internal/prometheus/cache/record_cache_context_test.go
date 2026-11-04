package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestRecordingCanceledRefreshDoesNotReadOrPublish(t *testing.T) {
	cache, source, storage, _ := newRecordingContractCache(t, []*model.MonitorRecordRule{{Name: "valid:up", Expr: "up", Enable: 1}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cache.GenerateMainConfig(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled refresh did not preserve cancellation")
	}
	if source.calls != 0 || storage.writes != 0 {
		t.Fatal("canceled refresh read or published configuration")
	}
}
