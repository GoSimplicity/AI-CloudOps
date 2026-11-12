package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type rootHeldWait int

const (
	rootHeldWaitAccepted rootHeldWait = iota
	rootHeldWaitMissed
	rootHeldWaitCompleted
	rootHeldWaitNotReached
	rootHeldWaitNotReleased
)

// rootObserveHeldCommand waits until the proxy has accepted the targeted Redis
// command, then until publication returns. The proxy stores held and closes the
// client before it signals seen, so publication can return while seen is still
// open. That error is the injected unknown result. A return while held is still
// false means the command was not issued.
func rootObserveHeldCommand(ctx context.Context, seen <-chan struct{}, finished <-chan error, held *atomic.Bool) (rootHeldWait, error) {
	select {
	case <-seen:
	case err := <-finished:
		if !held.Load() {
			return rootHeldWaitMissed, err
		}
		if err == nil {
			return rootHeldWaitCompleted, nil
		}
		return rootHeldWaitAccepted, nil
	case <-ctx.Done():
		return rootHeldWaitNotReached, ctx.Err()
	}
	select {
	case err := <-finished:
		if err == nil {
			return rootHeldWaitCompleted, nil
		}
		return rootHeldWaitAccepted, nil
	case <-ctx.Done():
		return rootHeldWaitNotReleased, ctx.Err()
	}
}

func TestRootObserveHeldCommandAcceptsReturnBeforeSeen(t *testing.T) {
	seen := make(chan struct{})
	finished := make(chan error, 1)
	var held atomic.Bool
	held.Store(true)
	finished <- errors.New("AlertManager配置发布未完成，池9881，阶段持久化投影")
	got, err := rootObserveHeldCommand(context.Background(), seen, finished, &held)
	if got != rootHeldWaitAccepted || err != nil {
		t.Fatalf("held intent removal was reported missing: %v %v", got, err)
	}
}

func TestRootObserveHeldCommandRejectsReturnBeforeHold(t *testing.T) {
	seen := make(chan struct{})
	finished := make(chan error, 1)
	var held atomic.Bool
	cause := errors.New("AlertManager配置发布未完成，池9881，阶段持久化投影")
	finished <- cause
	got, err := rootObserveHeldCommand(context.Background(), seen, finished, &held)
	if got != rootHeldWaitMissed || !errors.Is(err, cause) {
		t.Fatalf("early publication return was accepted: %v %v", got, err)
	}
}

func TestRootObserveHeldCommandRequiresFailureAfterSeen(t *testing.T) {
	seen := make(chan struct{})
	close(seen)
	finished := make(chan error, 1)
	var held atomic.Bool
	held.Store(true)
	finished <- errors.New("redis EOF")
	got, err := rootObserveHeldCommand(context.Background(), seen, finished, &held)
	if got != rootHeldWaitAccepted || err != nil {
		t.Fatalf("observed command did not fail closed: %v %v", got, err)
	}
	finished = make(chan error, 1)
	finished <- nil
	got, err = rootObserveHeldCommand(context.Background(), seen, finished, &held)
	if got != rootHeldWaitCompleted || err != nil {
		t.Fatalf("successful publication was accepted: %v %v", got, err)
	}
}

func TestRootObserveHeldCommandTimesOutWithoutCommand(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	seen := make(chan struct{})
	finished := make(chan error)
	var held atomic.Bool
	got, err := rootObserveHeldCommand(ctx, seen, finished, &held)
	if got != rootHeldWaitNotReached || !errors.Is(err, context.Canceled) {
		t.Fatalf("missing command did not time out: %v %v", got, err)
	}
}

func TestRootObserveHeldCommandConcurrentCloseBeforeSeen(t *testing.T) {
	for i := 0; i < 100; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		seen := make(chan struct{})
		finished := make(chan error, 1)
		var held atomic.Bool
		done := make(chan struct{})
		var got rootHeldWait
		var err error
		go func() {
			got, err = rootObserveHeldCommand(ctx, seen, finished, &held)
			close(done)
		}()
		held.Store(true)
		finished <- errors.New("persist")
		close(seen)
		select {
		case <-done:
			cancel()
		case <-ctx.Done():
			cancel()
			t.Fatal("held command wait hung")
		}
		if got != rootHeldWaitAccepted || err != nil {
			t.Fatalf("iteration %d reported %v %v", i, got, err)
		}
	}
}
