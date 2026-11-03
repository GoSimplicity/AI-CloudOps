package service

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
)

type systemResourceReaders struct {
	cpu    func(context.Context) (float64, error)
	memory func(context.Context) (map[string]uint64, error)
	disk   func(context.Context) (map[string]uint64, error)
}

// Commit critical measurements together, only after all three probes succeed.
func (s *systemService) collectCriticalResources(ctx context.Context, info *model.System) error {
	if info == nil {
		return errors.New("系统指标目标不能为空")
	}
	readers := s.resources
	if readers.cpu == nil {
		readers.cpu = s.getCPUUsage
	}
	if readers.memory == nil {
		readers.memory = s.getMemoryInfo
	}
	if readers.disk == nil {
		readers.disk = s.getDiskInfo
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	cpu, err := readers.cpu(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("无法采集CPU使用率")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	memory, err := readers.memory(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("无法采集内存使用情况")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	disk, err := readers.disk(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("无法采集磁盘使用情况")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_, memoryTotalPresent := memory["total"]
	_, memoryUsedPresent := memory["used"]
	_, diskTotalPresent := disk["total"]
	_, diskUsedPresent := disk["used"]
	if !memoryTotalPresent || !memoryUsedPresent || !diskTotalPresent || !diskUsedPresent {
		return errors.New("关键系统指标无效或不完整")
	}
	candidate := model.System{
		CPUCores: info.CPUCores, CPUUsage: cpu,
		MemoryTotal: memory["total"], MemoryUsed: memory["used"],
		DiskTotal: disk["total"], DiskUsed: disk["used"],
	}
	if !utils.HasValidResourceMetrics(&candidate) {
		return errors.New("关键系统指标无效或不完整")
	}
	info.CPUUsage = candidate.CPUUsage
	info.MemoryTotal, info.MemoryUsed = candidate.MemoryTotal, candidate.MemoryUsed
	info.DiskTotal, info.DiskUsed = candidate.DiskTotal, candidate.DiskUsed
	info.MemoryUsage = utils.GetMemoryUsagePercentage(&candidate)
	info.DiskUsage = utils.GetDiskUsagePercentage(&candidate)
	return nil
}
