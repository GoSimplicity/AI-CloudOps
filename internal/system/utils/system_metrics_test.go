package utils

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestResourceHealthRequiresCompleteFiniteMeasurements(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*model.System)
	}{
		{"missing cores", func(s *model.System) { s.CPUCores = 0 }},
		{"cpu negative", func(s *model.System) { s.CPUUsage = -1 }},
		{"cpu overflow", func(s *model.System) { s.CPUUsage = 101 }},
		{"cpu nan", func(s *model.System) { s.CPUUsage = math.NaN() }},
		{"cpu infinity", func(s *model.System) { s.CPUUsage = math.Inf(1) }},
		{"memory absent", func(s *model.System) { s.MemoryTotal = 0 }},
		{"memory overflow", func(s *model.System) { s.MemoryUsed = 101 }},
		{"disk absent", func(s *model.System) { s.DiskTotal = 0 }},
		{"disk overflow", func(s *model.System) { s.DiskUsed = 101 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &model.System{CPUCores: 2, MemoryTotal: 100, MemoryUsed: 20, DiskTotal: 100, DiskUsed: 20}
			tc.edit(s)
			r := ToResponse(s)
			if HasValidResourceMetrics(s) || r.SystemStatus != "未知" || r.System != nil {
				t.Fatal("invalid measurements were represented as a measured health state")
			}
			if _, err := json.Marshal(r); err != nil {
				t.Fatal("unknown response retained non-serializable metrics")
			}
		})
	}
	if ToResponse(nil).SystemStatus != "未知" {
		t.Fatal("nil snapshot must remain unknown")
	}
}

func TestMeasuredIdleAndExistingHealthThresholdsRemainValid(t *testing.T) {
	for _, tc := range []struct {
		cpu          float64
		memory, disk uint64
		status       string
	}{
		{0, 0, 0, "健康"},
		{60, 70, 80, "健康"},
		{61, 20, 20, "注意"},
		{20, 71, 20, "注意"},
		{20, 20, 81, "注意"},
		{81, 20, 20, "告警"},
		{20, 86, 20, "告警"},
		{20, 20, 91, "告警"},
	} {
		s := &model.System{CPUCores: 1, CPUUsage: tc.cpu, MemoryTotal: 100, MemoryUsed: tc.memory, DiskTotal: 100, DiskUsed: tc.disk}
		if !HasValidResourceMetrics(s) || ToResponse(s).SystemStatus != tc.status {
			t.Fatalf("measured values changed health classification: cpu=%v memory=%d disk=%d", tc.cpu, tc.memory, tc.disk)
		}
	}
}
