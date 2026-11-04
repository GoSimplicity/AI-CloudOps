package app

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProjectionRecoveryShutdownCancelsAndWaitsForWriter(t *testing.T) {
	writerCtx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	application := &App{projectionRecoveryCancel: cancel, projectionRecoveryDone: done}
	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	settled := make(chan error, 1)
	go func() { settled <- application.waitProjectionRecovery(ctx) }()
	select {
	case <-writerCtx.Done():
	case <-ctx.Done():
		t.Fatal("shutdown never cancelled writer")
	}
	select {
	case <-settled:
		t.Fatal("shutdown returned before pending lease settled")
	default:
	}
	close(done)
	select {
	case err := <-settled:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("settled writer blocked shutdown")
	}
	if err := application.waitProjectionRecovery(ctx); err != nil {
		t.Fatal("settled shutdown was not idempotent", err)
	}
}
func TestProjectionRecoveryShutdownTimeoutDoesNotPretendWriterStopped(t *testing.T) {
	writerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	application := &App{projectionRecoveryCancel: cancel, projectionRecoveryDone: make(chan struct{})}
	ctx, stop := context.WithCancel(context.Background())
	stop()
	if err := application.waitProjectionRecovery(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("unfinished writer reported clean shutdown", err)
	}
	if writerCtx.Err() == nil {
		t.Fatal("timeout omitted writer cancellation")
	}
}
