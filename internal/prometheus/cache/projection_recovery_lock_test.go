package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	configdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/config"
	"go.uber.org/zap"
)

type recoveryLockResult struct {
	err      error
	panicked bool
}

func callRecoveryWithoutPanic(ctx context.Context, publisher PoolProjectionRecoverer, ref configdao.PoolReservation) (result recoveryLockResult) {
	defer func() {
		if recover() != nil {
			result.panicked = true
		}
	}()
	result.err = publisher.RecoverCommittedPool(ctx, ref)
	return result
}
func lockedRecoverySubject(family int8) (sync.Locker, PoolProjectionRecoverer) {
	switch family {
	case 1:
		p := &prometheusConfigCache{}
		return &p.mu, p
	case 3:
		p := &alertRuleConfigCache{}
		return &p.mu, p
	case 4:
		p := &recordRuleConfigCache{}
		return &p.mu, p
	default:
		p := &alertManagerConfigCache{}
		return &p.mu, p
	}
}
func TestProjectionRecoveryCanceledEntryDoesNotWaitForNormalPublisherLock(t *testing.T) {
	for _, family := range []int8{1, 3, 4, 2, 5} {
		t.Run(fmt.Sprint(family), func(t *testing.T) {
			lock, publisher := lockedRecoverySubject(family)
			lock.Lock()
			var once sync.Once
			unlock := func() { once.Do(lock.Unlock) }
			defer unlock()
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			ref := configdao.PoolReservation{OperationID: 1, Family: family, PoolID: 7, Generation: 1, Nonce: [16]byte{1}}
			done := make(chan recoveryLockResult, 1)
			go func() { done <- callRecoveryWithoutPanic(ctx, publisher, ref) }()
			timer := time.NewTimer(250 * time.Millisecond)
			defer timer.Stop()
			select {
			case result := <-done:
				if result.panicked || !errors.Is(result.err, context.Canceled) {
					t.Fatal("canceled entry touched unavailable dependencies or lost context.Canceled")
				}
			case <-timer.C:
				t.Error("already-canceled recovery blocked behind the normal publisher lock")
				// Always release the real mutex and collect the worker, including on the
				// old failing implementation; no blocked goroutine escapes this subtest.
				unlock()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Fatal("recovery goroutine remained after releasing its lock")
				}
			}
		})
	}
}
func TestProjectionRecoveryWorkerCancellationFinishesWhileNormalPublisherKeepsLock(t *testing.T) {
	f := newRecordProjectionFixture(t, nil, nil, nil, nil)
	ref := commitRecoveryEmpty(t, f, 4, 7).Reservation
	lock, publisher := lockedRecoverySubject(4)
	lock.Lock()
	var once sync.Once
	unlock := func() { once.Do(lock.Unlock) }
	defer unlock()
	entered := make(chan context.Context, 1)
	dispatcher := recoveryDispatchFunc(func(ctx context.Context, got configdao.PoolReservation) error {
		if got != ref {
			return errors.New("unexpected real candidate")
		}
		entered <- ctx
		result := callRecoveryWithoutPanic(ctx, publisher, got)
		if result.panicked {
			return errors.New("unavailable dependency reached after lock release")
		}
		return result.err
	})
	options := recoveryOptions()
	options.Interval = time.Hour
	options.ItemTimeout = 50 * time.Millisecond
	worker, err := NewProjectionRecoveryWorker(configdao.NewProjectionRecoveryJournalDAO(zap.NewNop(), f.db), dispatcher, zap.NewNop(), options)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	watchdog := time.NewTimer(2 * time.Second)
	defer watchdog.Stop()
	var itemCtx context.Context
	select {
	case itemCtx = <-entered:
	case <-watchdog.C:
		unlock()
		cancel()
		<-done
		t.Fatal("real pending candidate did not reach recovery")
	}
	select {
	case <-itemCtx.Done():
	case <-watchdog.C:
		unlock()
		cancel()
		<-done
		t.Fatal("configured item timeout did not expire")
	}
	cancel()
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-timer.C:
		t.Error("worker shutdown waited for a normal generator lock after item and parent cancellation")
		unlock()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker remained after releasing the normal publisher lock")
		}
	}
}
