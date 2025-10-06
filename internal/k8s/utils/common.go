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
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// K8s资源状态常量定义
const (
	// 通用状态
	StatusPending     = "Pending"
	StatusUnknown     = "Unknown"
	StatusReady       = "Ready"
	StatusTerminating = "Terminating"
	StatusRunning     = "Running"
	StatusUpdating    = "Updating"
	StatusSucceeded   = "Succeeded"
	StatusFailed      = "Failed"
	StatusEvicted     = "Evicted"
)

func ConvertUnstructuredToYAML(obj *unstructured.Unstructured) (string, error) {
	if obj == nil {
		return "", fmt.Errorf("unstructured对象不能为空")
	}

	jsonBytes, err := obj.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("序列化unstructured对象失败: %w", err)
	}

	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return "", fmt.Errorf("JSON转换为YAML失败: %w", err)
	}

	return string(yamlBytes), nil
}

// CalculateAge 计算资源的年龄，返回可读的时间格式
func CalculateAge(creationTime time.Time) string {
	duration := time.Since(creationTime)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		if days == 1 {
			return "1d"
		}
		return fmt.Sprintf("%dd", days)
	} else if hours > 0 {
		if hours == 1 {
			return "1h"
		}
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", minutes)
	} else {
		seconds := int(duration.Seconds())
		if seconds <= 1 {
			return "1s"
		}
		return fmt.Sprintf("%ds", seconds)
	}
}

// FilterByName 根据搜索关键字过滤资源名称
func FilterByName(name string, searchKeyword string) bool {
	if searchKeyword == "" {
		return true
	}
	return Contains(name, searchKeyword)
}

// Contains 不区分大小写的字符串包含检查
func Contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// Paginate 通用分页函数
func Paginate[T any](items []T, page, size int) ([]T, int64) {
	total := int64(len(items))
	if total == 0 {
		return []T{}, 0
	}

	// 如果没有设置分页参数，返回所有数据
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	start := int64((page - 1) * size)
	end := start + int64(size)

	if start >= total {
		return []T{}, total
	}
	if end > total {
		end = total
	}

	return items[start:end], total
}

// SortByCreationTime 按创建时间排序资源列表（最新的在前）
// 使用泛型函数，接受一个提取时间的函数
func SortByCreationTime[T any](items []T, getTime func(T) time.Time) {
	sort.Slice(items, func(i, j int) bool {
		return getTime(items[i]).After(getTime(items[j]))
	})
}
