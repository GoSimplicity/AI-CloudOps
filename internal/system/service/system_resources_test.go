package service

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type resourceCache struct {
	redis.Cmdable
	value      string
	gets, sets int
	key        string
}

func (r *resourceCache) Get(_ context.Context, key string) *redis.StringCmd {
	r.gets++
	r.key = key
	if r.value == "" {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(r.value, nil)
}
func (r *resourceCache) Set(context.Context, string, interface{}, time.Duration) *redis.StatusCmd {
	r.sets++
	return redis.NewStatusResult("OK", nil)
}

func TestCriticalProbeFailuresNeverPublishHealthyOrCachePartialData(t *testing.T) {
	for _, stage := range []string{"cpu", "memory", "disk"} {
		for _, operation := range []string{"info", "metrics", "refresh"} {
			t.Run(stage+"/"+operation, func(t *testing.T) {
				cache := &resourceCache{}
				readers := systemResourceReaders{
					cpu: func(context.Context) (float64, error) {
						if stage == "cpu" {
							return 0, errors.New("private-probe-output")
						}
						return 10, nil
					},
					memory: func(context.Context) (map[string]uint64, error) {
						if stage == "memory" {
							return nil, errors.New("private-probe-output")
						}
						return map[string]uint64{"total": 100, "used": 20}, nil
					},
					disk: func(context.Context) (map[string]uint64, error) {
						if stage == "disk" {
							return nil, errors.New("private-probe-output")
						}
						return map[string]uint64{"total": 100, "used": 20}, nil
					},
				}
				s := &systemService{l: zap.NewNop(), redis: cache, resources: readers}
				var result *model.System
				var err error
				switch operation {
				case "info":
					result, err = s.GetCurrentSystemInfo(context.Background())
				case "metrics":
					result, err = s.GetSystemMetrics(context.Background())
				case "refresh":
					result, err = s.RefreshSystemInfo(context.Background())
				}
				if err == nil || result != nil || cache.sets != 0 || strings.Contains(err.Error(), "private-probe-output") {
					t.Fatal("probe failure returned or cached a successful/unsanitized snapshot")
				}
			})
		}
	}
}

func TestCriticalResourcesCommitTogetherAndRespectCancellation(t *testing.T) {
	for _, mode := range []string{"success", "nan", "capacity", "cancel", "failure", "memory missing used", "disk missing used"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := &systemService{resources: systemResourceReaders{
				cpu: func(context.Context) (float64, error) {
					if mode == "nan" {
						return math.NaN(), nil
					}
					return 0, nil
				},
				memory: func(context.Context) (map[string]uint64, error) {
					if mode == "memory missing used" {
						return map[string]uint64{"total": 100}, nil
					}
					return map[string]uint64{"total": 100, "used": 25}, nil
				},
				disk: func(context.Context) (map[string]uint64, error) {
					if mode == "disk missing used" {
						return map[string]uint64{"total": 100}, nil
					}
					if mode == "cancel" {
						cancel()
						return nil, ctx.Err()
					}
					if mode == "failure" {
						return nil, errors.New("probe")
					}
					if mode == "capacity" {
						return map[string]uint64{"total": 1, "used": 2}, nil
					}
					return map[string]uint64{"total": 100, "used": 50}, nil
				},
			}}
			info := model.System{CPUCores: 2, CPUUsage: 90, MemoryTotal: 1, DiskTotal: 2}
			before := info
			err := s.collectCriticalResources(ctx, &info)
			if mode == "success" {
				if err != nil || info.CPUUsage != 0 || info.MemoryUsage != 25 || info.DiskUsage != 50 {
					t.Fatal("valid measured idle/capacity data was not committed")
				}
				return
			}
			if err == nil || !reflect.DeepEqual(before, info) {
				t.Fatal("failed critical collection changed partial measurements")
			}
			if mode == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation classification lost")
			}
		})
	}
}

func TestSystemCacheOnlyAcceptsMeasuredFreshV3Snapshots(t *testing.T) {
	for _, mode := range []string{"fresh", "empty", "future", "stale", "zero capacity", "invalid cpu", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			info := model.System{CPUCores: 2, CPUUsage: 0, MemoryTotal: 100, DiskTotal: 100, LastUpdateTime: time.Now().Unix() - 5}
			switch mode {
			case "future":
				info.LastUpdateTime = time.Now().Unix() + 3600
			case "stale":
				info.LastUpdateTime = time.Now().Unix() - 301
			case "zero capacity":
				info.MemoryTotal = 0
			case "invalid cpu":
				info.CPUUsage = -1
			}
			data, _ := json.Marshal(info)
			cache := &resourceCache{value: string(data)}
			if mode == "empty" {
				cache.value = ""
			}
			if mode == "malformed" {
				cache.value = "{"
			}
			calls := 0
			s := &systemService{l: zap.NewNop(), redis: cache, resources: systemResourceReaders{cpu: func(context.Context) (float64, error) { calls++; return 0, errors.New("unavailable") }}}
			result, err := s.GetCurrentSystemInfo(context.Background())
			if cache.key != "system:info:v3" || cache.sets != 0 {
				t.Fatal("unverified legacy cache was used or failed measurements cached")
			}
			if mode == "fresh" {
				if err != nil || result == nil || calls != 0 {
					t.Fatal("valid measured idle cache was discarded")
				}
			} else if err == nil || result != nil || calls != 1 {
				t.Fatal("invalid cache was represented as fresh measured data")
			}
		})
	}
}

func TestUnsupportedWindowsAndCancelledCollectionReturnErrors(t *testing.T) {
	s := &systemService{l: zap.NewNop(), redis: &resourceCache{}}
	info := &model.System{}
	if s.collectWindowsSystemInfo(context.Background(), info) == nil {
		t.Fatal("unimplemented platform reported successful zero measurements")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.collectWindowsSystemInfo(ctx, info); !errors.Is(err, context.Canceled) {
		t.Fatal("unsupported platform lost cancellation")
	}
	if _, err := s.GetCurrentSystemInfo(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled request consulted cache or returned success")
	}
}

func TestPublicSystemMethodsPreserveInFlightCancellation(t *testing.T) {
	for _, operation := range []string{"info", "metrics", "refresh"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			cache := &resourceCache{}
			s := &systemService{l: zap.NewNop(), redis: cache, resources: systemResourceReaders{
				cpu: func(context.Context) (float64, error) {
					cancel()
					return 0, ctx.Err()
				},
			}}
			var result *model.System
			var err error
			switch operation {
			case "info":
				result, err = s.GetCurrentSystemInfo(ctx)
			case "metrics":
				result, err = s.GetSystemMetrics(ctx)
			case "refresh":
				result, err = s.RefreshSystemInfo(ctx)
			}
			if !errors.Is(err, context.Canceled) || result != nil || cache.sets != 0 {
				t.Fatal("public cancellation became a generic failure or successful cached data")
			}
		})
	}
}
