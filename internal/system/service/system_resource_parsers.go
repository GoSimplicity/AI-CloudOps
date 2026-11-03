package service

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var errCPUProbe = errors.New("CPU计数无效或不完整")
var errMemoryProbe = errors.New("内存计数无效或不完整")

type cpuStat struct {
	counters [10]uint64
	count    int
	total    uint64
	idle     uint64
}

// /proc/stat begins with user,nice,system,idle; subsequent accounting fields
// are optional on older kernels. Every supplied counter must parse, including
// guest fields. guest/guest_nice are already included in user/nice, so only the
// first eight counters contribute to elapsed CPU time.
func parseLinuxCPUStat(output string) (*cpuStat, error) {
	first, _, _ := strings.Cut(output, "\n")
	fields := strings.Fields(first)
	if len(fields) < 5 || len(fields) > 11 || fields[0] != "cpu" {
		return nil, errCPUProbe
	}
	stat := &cpuStat{count: len(fields) - 1}
	for i, field := range fields[1:] {
		value, err := parseUnsignedCounter(field)
		if err != nil {
			return nil, errCPUProbe
		}
		stat.counters[i] = value
		if i < 8 {
			if value > math.MaxUint64-stat.total {
				return nil, errCPUProbe
			}
			stat.total += value
		}
	}
	stat.idle = stat.counters[3]
	return stat, nil
}
func linuxCPUUsageBetween(before, after *cpuStat) (float64, error) {
	if before == nil || after == nil || before.count != after.count {
		return 0, errCPUProbe
	}
	for i := 0; i < before.count; i++ {
		if after.counters[i] < before.counters[i] {
			return 0, errCPUProbe
		}
	}
	if after.total <= before.total || after.idle < before.idle {
		return 0, errCPUProbe
	}
	elapsed, idle := after.total-before.total, after.idle-before.idle
	if idle > elapsed {
		return 0, errCPUProbe
	}
	return float64(elapsed-idle) / float64(elapsed) * 100, nil
}
func parseUnsignedCounter(text string) (uint64, error) {
	if text == "" {
		return 0, errMemoryProbe
	}
	for _, ch := range text {
		if ch < '0' || ch > '9' {
			return 0, errMemoryProbe
		}
	}
	value, err := strconv.ParseUint(text, 10, 64)
	if err != nil {
		return 0, errMemoryProbe
	}
	return value, nil
}

var macCPUUsageLine = regexp.MustCompile(`^CPU usage:\s+([0-9]+(?:\.[0-9]{1,2})?)% user,\s+([0-9]+(?:\.[0-9]{1,2})?)% sys,\s+([0-9]+(?:\.[0-9]{1,2})?)% idle$`)

// top reports user/system/idle. Apple's globalstats.c also has versions that
// print the hundredths integer without zero padding (11.6 means 11.06). Accept
// that format only when all three components establish a coherent 100% sample;
// conventional decimal output remains accepted by the same complete contract.
func parseMacCPUUsage(output string) (float64, error) {
	var sample []string
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "CPU usage:") {
			continue
		}
		if sample != nil {
			return 0, errCPUProbe
		}
		sample = macCPUUsageLine.FindStringSubmatch(strings.TrimSpace(line))
		if sample == nil {
			return 0, errCPUProbe
		}
	}
	if sample == nil {
		return 0, errCPUProbe
	}
	for _, appleHundredths := range []bool{false, true} {
		var values [3]float64
		valid := true
		for i, token := range sample[1:] {
			value, err := strconv.ParseFloat(token, 64)
			if appleHundredths {
				whole, fraction, hasDot := strings.Cut(token, ".")
				integer, wholeErr := parseUnsignedCounter(whole)
				if wholeErr != nil {
					valid = false
					break
				}
				value = float64(integer)
				if hasDot {
					part, partErr := parseUnsignedCounter(fraction)
					if partErr != nil {
						valid = false
						break
					}
					value += float64(part) / 100
				}
			}
			if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 100 {
				valid = false
				break
			}
			values[i] = value
		}
		// Each displayed component can be truncated/rounded at hundredth precision.
		if valid && math.Abs(values[0]+values[1]+values[2]-100) <= 0.031 {
			return math.Min(100, values[0]+values[1]), nil
		}
	}
	return 0, errCPUProbe
}

var macVMHeader = regexp.MustCompile(`^Mach Virtual Memory Statistics: \(page size of ([0-9]+) bytes\)$`)

func parseMacMemoryInfo(totalOutput, vmOutput string) (map[string]uint64, error) {
	total, err := parseUnsignedCounter(strings.TrimSpace(totalOutput))
	if err != nil || total < 1024*1024 {
		return nil, errMemoryProbe
	}
	var pageSize, free, inactive uint64
	var headerSeen, freeSeen, inactiveSeen bool
	for _, raw := range strings.Split(vmOutput, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "Mach Virtual Memory Statistics:") {
			match := macVMHeader.FindStringSubmatch(line)
			if headerSeen || match == nil {
				return nil, errMemoryProbe
			}
			pageSize, err = parseUnsignedCounter(match[1])
			if err != nil || pageSize == 0 || pageSize&(pageSize-1) != 0 {
				return nil, errMemoryProbe
			}
			headerSeen = true
		}
		key, value, hasColon := strings.Cut(line, ":")
		if !hasColon || (key != "Pages free" && key != "Pages inactive") {
			continue
		}
		text := strings.TrimSpace(value)
		if !strings.HasSuffix(text, ".") {
			return nil, errMemoryProbe
		}
		count, err := parseUnsignedCounter(strings.TrimSuffix(text, "."))
		if err != nil {
			return nil, errMemoryProbe
		}
		if key == "Pages free" {
			if freeSeen {
				return nil, errMemoryProbe
			}
			free, freeSeen = count, true
		} else {
			if inactiveSeen {
				return nil, errMemoryProbe
			}
			inactive, inactiveSeen = count, true
		}
	}
	if !headerSeen || !freeSeen || !inactiveSeen || pageSize > total || total%pageSize != 0 || free > math.MaxUint64-inactive {
		return nil, errMemoryProbe
	}
	availablePages := free + inactive
	if availablePages > math.MaxUint64/pageSize {
		return nil, errMemoryProbe
	}
	available := availablePages * pageSize
	if available > total {
		return nil, errMemoryProbe
	}
	return map[string]uint64{"total": total / (1024 * 1024), "used": (total - available) / (1024 * 1024)}, nil
}
