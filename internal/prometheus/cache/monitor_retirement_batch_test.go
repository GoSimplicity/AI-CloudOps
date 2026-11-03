package cache

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"go.uber.org/zap"
)

type recoveryBatchProm struct {
	PrometheusConfigCache
	retire func(context.Context) error
	normal func(context.Context) error
}

func (t recoveryBatchProm) ReconcileDeletedScrapePools(ctx context.Context) error {
	return t.retire(ctx)
}
func (t recoveryBatchProm) GenerateMainConfig(ctx context.Context) error { return t.normal(ctx) }

type recoveryBatchAM struct {
	AlertManagerConfigCache
	retire func(context.Context) error
	normal func(context.Context) error
}

func (t recoveryBatchAM) ReconcileDeletedAlertManagerPools(ctx context.Context) error {
	return t.retire(ctx)
}
func (t recoveryBatchAM) GenerateMainConfig(ctx context.Context) error { return t.normal(ctx) }

type recoveryBatchRule struct {
	AlertRuleConfigCache
	retire func(context.Context) error
}

func (t recoveryBatchRule) ReconcileDeletedScrapePools(ctx context.Context) error {
	return t.retire(ctx)
}
func (recoveryBatchRule) GenerateMainConfig(context.Context) error { return nil }

type recoveryBatchRecord struct {
	RecordRuleConfigCache
	retire func(context.Context) error
}

func (t recoveryBatchRecord) ReconcileDeletedScrapePools(ctx context.Context) error {
	return t.retire(ctx)
}
func (recoveryBatchRecord) GenerateMainConfig(context.Context) error { return nil }

func TestRetirementBatchSurvivesNormalFailureAndIndependentOwnerConflict(t *testing.T) {
	for _, mode := range []string{"normal fail-fast", "AM foreign owner", "AM foreign owner plus normal failure"} {
		t.Run(mode, func(t *testing.T) {
			var retired atomic.Int32
			var normals atomic.Int32
			foreignErr, normalErr := errors.New("AM foreign owner"), errors.New("normal generator failure")
			// Every controller reaches the batch concurrently, without sleeps. An AM
			// failure must not cancel the other controllers while they are waiting.
			allStarted := make(chan struct{})
			var started atomic.Int32
			retire := func(ctx context.Context) error {
				if started.Add(1) == 4 {
					close(allStarted)
				}
				<-allStarted
				if err := ctx.Err(); err != nil {
					return err
				}
				retired.Add(1)
				return nil
			}
			amRetire := func(ctx context.Context) error {
				if err := retire(ctx); err != nil {
					return err
				}
				if mode != "normal fail-fast" {
					return foreignErr
				}
				return nil
			}
			normal := func(context.Context) error {
				normals.Add(1)
				if retired.Load() != 4 {
					t.Error("normal generation began before independent retirement attempts completed")
				}
				if mode == "AM foreign owner" {
					return nil
				}
				return normalErr
			}
			unrelatedNormal := func(context.Context) error { normals.Add(1); return nil }
			manager, err := NewMonitorCache(&config.Config{}, recoveryBatchProm{retire: retire, normal: normal}, recoveryBatchAM{retire: amRetire, normal: unrelatedNormal}, recoveryBatchRule{retire: retire}, recoveryBatchRecord{retire: retire}, zap.NewNop())
			if err != nil {
				t.Fatal(err)
			}
			err = manager.MonitorCacheManager(context.Background())
			if retired.Load() != 4 || normals.Load() != 2 || errors.Is(err, normalErr) != (mode != "AM foreign owner") {
				t.Fatal("retirement or normal task was starved")
			}
			if errors.Is(err, foreignErr) != (mode != "normal fail-fast") {
				t.Fatal("independent retirement failure was lost")
			}
		})
	}
}
func TestRetirementBatchParentCancellationStopsAllAndSkipsNormalGeneration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var started, stopped, normal atomic.Int32
	allStarted := make(chan struct{})
	retire := func(ctx context.Context) error {
		if started.Add(1) == 4 {
			close(allStarted)
		}
		<-ctx.Done()
		stopped.Add(1)
		return ctx.Err()
	}
	gen := func(context.Context) error { normal.Add(1); return nil }
	manager, err := NewMonitorCache(&config.Config{}, recoveryBatchProm{retire: retire, normal: gen}, recoveryBatchAM{retire: retire, normal: gen}, recoveryBatchRule{retire: retire}, recoveryBatchRecord{retire: retire}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- manager.MonitorCacheManager(ctx) }()
	<-allStarted
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) || stopped.Load() != 4 || normal.Load() != 0 {
		t.Fatal("parent cancellation did not stop all retirement attempts")
	}
}
