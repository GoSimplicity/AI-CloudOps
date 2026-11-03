package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestParseLinuxCPUStatRequiresCompleteValidCounters(t *testing.T) {
	for _, tc := range []struct {
		name, raw   string
		total, idle uint64
		valid       bool
	}{
		{"minimal", "cpu 10 2 3 85\ncpu0 1 0 0 3", 100, 85, true},
		{"modern", "cpu 10 2 3 85 1 2 3 4 5 1", 110, 85, true},
		{"boot zero", "cpu 0 0 0 0", 0, 0, true},
		{"empty", "", 0, 0, false}, {"missing aggregate", "cpu0 1 2 3 4", 0, 0, false},
		{"missing idle", "cpu 1 2 3", 0, 0, false}, {"bad user", "cpu invalid 2 3 4", 0, 0, false},
		{"bad optional", "cpu 1 2 3 4 invalid", 0, 0, false}, {"bad guest", "cpu 1 2 3 4 0 0 0 0 invalid 0", 0, 0, false},
		{"negative", "cpu -1 2 3 4", 0, 0, false}, {"plus sign", "cpu +1 2 3 4", 0, 0, false},
		{"counter overflow", "cpu 18446744073709551616 0 0 0", 0, 0, false},
		{"total overflow", "cpu 18446744073709551615 1 0 0", 0, 0, false},
		{"extra fields", "cpu 1 2 3 4 5 6 7 8 9 10 11", 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLinuxCPUStat(tc.raw)
			if !tc.valid {
				if err == nil {
					t.Fatal("invalid counters accepted")
				}
				if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "184467") {
					t.Fatal("raw counter echoed")
				}
				return
			}
			if err != nil || got.total != tc.total || got.idle != tc.idle {
				t.Fatalf("unexpected stat: %+v error=%v", got, err)
			}
		})
	}
}
func TestLinuxCPUUsageRejectsRollbackAndUnobservedIntervals(t *testing.T) {
	for _, tc := range []struct {
		name, before, after string
		want                float64
		valid               bool
	}{
		{"busy", "cpu 10 2 3 85", "cpu 20 2 3 95", 50, true},
		{"real idle", "cpu 10 2 3 85", "cpu 10 2 3 105", 0, true},
		{"all busy", "cpu 10 2 3 85", "cpu 30 2 3 85", 100, true},
		{"guest not double counted", "cpu 10 0 0 90 0 0 0 0 5 0", "cpu 20 0 0 100 0 0 0 0 10 0", 50, true},
		{"no elapsed", "cpu 10 2 3 85", "cpu 10 2 3 85", 0, false},
		{"zero no elapsed", "cpu 0 0 0 0", "cpu 0 0 0 0", 0, false},
		{"total rollback", "cpu 10 2 3 85", "cpu 9 2 3 84", 0, false},
		{"hidden counter rollback", "cpu 10 2 3 85", "cpu 9 2 3 95", 0, false},
		{"idle rollback", "cpu 10 2 3 85", "cpu 30 2 3 84", 0, false},
		{"format changed", "cpu 10 2 3 85", "cpu 10 2 3 95 0", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, e := parseLinuxCPUStat(tc.before)
			if e != nil {
				t.Fatal(e)
			}
			b, e := parseLinuxCPUStat(tc.after)
			if e != nil {
				t.Fatal(e)
			}
			got, err := linuxCPUUsageBetween(a, b)
			if !tc.valid {
				if err == nil {
					t.Fatal("unobserved interval accepted")
				}
				return
			}
			if err != nil || math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("usage=%v error=%v", got, err)
			}
		})
	}
}
func TestParseMacCPUUsageRequiresUserSystemIdle(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		want      float64
		valid     bool
	}{
		{"native Apple unpadded hundredths", "CPU usage: 4.10% user, 11.6% sys, 84.82% idle ", 15.16, true},
		{"native Apple two digits", "CPU usage: 6.77% user, 12.50% sys, 80.72% idle", 19.27, true},
		{"conventional decimal", "CPU usage: 5.5% user, 4.5% sys, 90.0% idle", 10, true},
		{"real idle", "CPU usage: 0.0% user, 0.0% sys, 100.0% idle", 0, true},
		{"system-only busy", "CPU usage: 0.00% user, 80.00% sys, 20.00% idle", 80, true},
		{"rounding", "CPU usage: 33.34% user, 66.67% sys, 0.00% idle", 100, true},
		{"empty", "", 0, false}, {"missing system idle", "CPU usage: 0% user,", 0, false},
		{"missing idle", "CPU usage: 10% user, 20% sys,", 0, false},
		{"wrong label", "CPU usage: 10% user, 20% system, 70% idle", 0, false},
		{"all zero", "CPU usage: 0% user, 0% sys, 0% idle", 0, false},
		{"inconsistent total", "CPU usage: 0% user, 10% sys, 20% idle", 0, false},
		{"bad field", "CPU usage: bad% user, 0% sys, 100% idle", 0, false},
		{"NaN", "CPU usage: NaN% user, 0% sys, 100% idle", 0, false},
		{"overflow", "CPU usage: 101% user, 0% sys, 0% idle", 0, false},
		{"negative", "CPU usage: -1% user, 1% sys, 100% idle", 0, false},
		{"duplicate", "CPU usage: 0% user, 0% sys, 100% idle\nCPU usage: 0% user, 0% sys, 100% idle", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMacCPUUsage(tc.raw)
			if !tc.valid {
				if err == nil {
					t.Fatal("partial CPU sample accepted")
				}
				return
			}
			if err != nil || math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("usage=%v error=%v", got, err)
			}
		})
	}
}
func TestParseMacMemoryUsesReportedPageSizeAndRequiredCounters(t *testing.T) {
	const header4 = "Mach Virtual Memory Statistics: (page size of 4096 bytes)\n"
	const header16 = "Mach Virtual Memory Statistics: (page size of 16384 bytes)\n"
	for _, tc := range []struct {
		name, total, raw string
		wantUsed         uint64
		valid            bool
	}{
		{"4K", "1073741824", header4 + "Pages free: 32768.\nPages inactive: 32768.\n", 768, true},
		{"16K", "1073741824", header16 + "Pages free: 32768.\nPages inactive: 32768.\n", 0, true},
		{"zero free", "1073741824", header16 + "Pages free: 0.\nPages inactive: 0.\n", 1024, true},
		{"missing header", "1073741824", "Pages free: 0.\nPages inactive: 0.\n", 0, false},
		{"missing free", "1073741824", header4 + "Pages inactive: 0.\n", 0, false},
		{"missing inactive", "1073741824", header4 + "Pages free: 0.\n", 0, false},
		{"bad free", "1073741824", header4 + "Pages free: invalid.\nPages inactive: 0.\n", 0, false},
		{"negative inactive", "1073741824", header4 + "Pages free: 0.\nPages inactive: -1.\n", 0, false},
		{"duplicate", "1073741824", header4 + "Pages free: 0.\nPages free: 0.\nPages inactive: 0.\n", 0, false},
		{"zero total", "0", header4 + "Pages free: 0.\nPages inactive: 0.\n", 0, false},
		{"bad total", "invalid", header4 + "Pages free: 0.\nPages inactive: 0.\n", 0, false},
		{"total overflow", "18446744073709551616", header4 + "Pages free: 0.\nPages inactive: 0.\n", 0, false},
		{"sum overflow", "1073741824", header4 + "Pages free: 18446744073709551615.\nPages inactive: 1.\n", 0, false},
		{"multiply overflow", "1073741824", header4 + "Pages free: 18446744073709551615.\nPages inactive: 0.\n", 0, false},
		{"above capacity", "1073741824", header4 + "Pages free: 262145.\nPages inactive: 0.\n", 0, false},
		{"zero page", "1073741824", "Mach Virtual Memory Statistics: (page size of 0 bytes)\nPages free: 0.\nPages inactive: 0.\n", 0, false},
		{"bad page", "1073741824", "Mach Virtual Memory Statistics: (page size of invalid bytes)\nPages free: 0.\nPages inactive: 0.\n", 0, false},
		{"unaligned total", "1073741825", header4 + "Pages free: 0.\nPages inactive: 0.\n", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseMacMemoryInfo(tc.total, tc.raw)
			if !tc.valid {
				if err == nil {
					t.Fatal("incomplete/invalid memory sample accepted")
				}
				return
			}
			if err != nil || got["total"] != 1024 || got["used"] != tc.wantUsed {
				t.Fatalf("memory=%v error=%v", got, err)
			}
		})
	}
}

func TestCriticalProbeCommandCancellation(t *testing.T) {
	s := &systemService{}
	for _, tc := range []struct {
		name string
		call func(context.Context) error
	}{
		{"linux cpu", func(ctx context.Context) error { _, err := s.getLinuxCPUUsage(ctx); return err }},
		{"darwin cpu", func(ctx context.Context) error { _, err := s.getMacOSCPUUsage(ctx); return err }},
		{"darwin memory", func(ctx context.Context) error { _, err := s.getMacOSMemoryInfo(ctx); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := tc.call(ctx); !errors.Is(err, context.Canceled) {
				t.Fatal("cancellation identity lost")
			}
		})
	}
}
