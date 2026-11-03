/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package utils

import (
	"fmt"
	"math"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func GetMemoryUsagePercentage(s *model.System) float64 {
	if s.MemoryTotal == 0 {
		return 0
	}
	return float64(s.MemoryUsed) / float64(s.MemoryTotal) * 100
}

func GetDiskUsagePercentage(s *model.System) float64 {
	if s.DiskTotal == 0 {
		return 0
	}
	return float64(s.DiskUsed) / float64(s.DiskTotal) * 100
}

func GetUptimeFormatted(s *model.System) string {
	duration := time.Duration(s.Uptime) * time.Second
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时%d分钟", hours, minutes)
	} else {
		return fmt.Sprintf("%d分钟", minutes)
	}
}

// FormatBytes 格式化字节大小
func FormatBytes(bytes uint64, unit string) string {
	if unit == "GB" {
		if bytes < 1024 {
			return fmt.Sprintf("%.2f GB", float64(bytes))
		}
		tb := float64(bytes) / 1024
		return fmt.Sprintf("%.2f TB", tb)
	}

	if bytes < 1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes))
	}
	gb := float64(bytes) / 1024
	return fmt.Sprintf("%.2f GB", gb)
}

// HasValidResourceMetrics rejects incomplete snapshots before health classification.
func HasValidResourceMetrics(s *model.System) bool {
	return s != nil && s.CPUCores > 0 && !math.IsNaN(s.CPUUsage) && !math.IsInf(s.CPUUsage, 0) &&
		s.CPUUsage >= 0 && s.CPUUsage <= 100 && s.MemoryTotal > 0 && s.MemoryUsed <= s.MemoryTotal &&
		s.DiskTotal > 0 && s.DiskUsed <= s.DiskTotal
}

func ToResponse(s *model.System) *model.SystemInfoResponse {
	if !HasValidResourceMetrics(s) {
		return &model.SystemInfoResponse{MemoryUsageFormatted: "不可用", DiskUsageFormatted: "不可用", UptimeFormatted: "不可用", SystemStatus: "未知"}
	}
	status := "健康"
	if s.CPUUsage > 80 || GetMemoryUsagePercentage(s) > 85 || GetDiskUsagePercentage(s) > 90 {
		status = "告警"
	} else if s.CPUUsage > 60 || GetMemoryUsagePercentage(s) > 70 || GetDiskUsagePercentage(s) > 80 {
		status = "注意"
	}

	return &model.SystemInfoResponse{
		System:               s,
		MemoryUsageFormatted: fmt.Sprintf("%s / %s", FormatBytes(s.MemoryUsed, "MB"), FormatBytes(s.MemoryTotal, "MB")),
		DiskUsageFormatted:   fmt.Sprintf("%s / %s", FormatBytes(s.DiskUsed, "GB"), FormatBytes(s.DiskTotal, "GB")),
		UptimeFormatted:      GetUptimeFormatted(s),
		SystemStatus:         status,
	}
}
