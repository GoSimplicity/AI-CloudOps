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

package constants

import "errors"

var (
	ErrorK8sClientNotReady     = errors.New("k8s client not ready")
	ErrorMetricsClientNotReady = errors.New("metrics client not ready")

	ErrorNodeNotFound       = errors.New("node not found")
	ErrorTaintsKeyDuplicate = errors.New("taints key exist")
)

// K8s 资源默认配置常量
const (
	// 默认资源限制
	DefaultCPULimit      = "1000m"
	DefaultMemoryLimit   = "1Gi"
	DefaultCPURequest    = "500m"
	DefaultMemoryRequest = "512Mi"

	// 默认名称
	DefaultLimitRangeName = "default-limits"
	DefaultQuotaName      = "default-quota"

	// 模拟使用量（用于测试和演示）
	MockCPUUsage    = "300m"
	MockMemoryUsage = "256Mi"
)
