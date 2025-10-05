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
