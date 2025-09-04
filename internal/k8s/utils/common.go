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
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	statusPending     = "Pending"
	statusUnknown     = "Unknown"
	statusReady       = "Ready"
	statusTerminating = "Terminating"
	statusRunning     = "Running"
	statusUpdating    = "Updating"
	statusSucceeded   = "Succeeded"
	statusFailed      = "Failed"
	statusEvicted     = "Evicted"
)

// ConvertToMetaV1ListOptions 将K8sGetResourceListReq转换为metav1.ListOptions
func ConvertToMetaV1ListOptions(req *model.K8sGetResourceListReq) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
		Continue:      req.Continue,
	}
}

// ConvertK8sListReqToMetaV1ListOptions 将K8sListReq转换为metav1.ListOptions
func ConvertK8sListReqToMetaV1ListOptions(req *model.K8sListReq) metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: req.LabelSelector,
		FieldSelector: req.FieldSelector,
		Limit:         req.Limit,
		Continue:      req.Continue,
	}
}

func ConvertUnstructuredToYAML(obj *unstructured.Unstructured) (string, error) {
	jsonBytes, err := obj.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("无法序列化unstructured对象为json: %v", err)
	}
	yamlBytes, err := yaml.JSONToYAML(jsonBytes)
	if err != nil {
		return "", fmt.Errorf("无法将json转换为yaml: %v", err)
	}
	return string(yamlBytes), nil
}
