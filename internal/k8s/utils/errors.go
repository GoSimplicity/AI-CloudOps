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

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
)

// K8sError 定义Kubernetes相关错误类型
type K8sError struct {
	Operation string
	Resource  string
	Name      string
	Namespace string
	ClusterID int
	Err       error
}

func (e *K8sError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s %s %s/%s in cluster %d failed: %v",
			e.Operation, e.Resource, e.Namespace, e.Name, e.ClusterID, e.Err)
	}
	return fmt.Sprintf("%s %s %s in cluster %d failed: %v",
		e.Operation, e.Resource, e.Name, e.ClusterID, e.Err)
}

func NewK8sError(operation, resource, name, namespace string, clusterID int, err error) *K8sError {
	return &K8sError{
		Operation: operation,
		Resource:  resource,
		Name:      name,
		Namespace: namespace,
		ClusterID: clusterID,
		Err:       err,
	}
}

func ValidateRequest(req interface{}) error {
	if req == nil {
		return fmt.Errorf("request cannot be empty")
	}
	return nil
}

func ValidateClusterID(clusterID int) error {
	if clusterID <= 0 {
		return fmt.Errorf("cluster ID must be positive, got %d", clusterID)
	}
	return nil
}

func ValidateName(name string, resource string) error {
	if name == "" {
		return fmt.Errorf("%s name cannot be empty", resource)
	}
	return nil
}

func ValidateNamespace(namespace string) error {
	if namespace == "" {
		return fmt.Errorf("namespace cannot be empty")
	}
	return nil
}

// LogAndWrapError 记录错误并包装
func LogAndWrapError(logger *zap.Logger, err error, operation string, fields ...zap.Field) error {
	logger.Error(operation+" failed", append(fields, zap.Error(err))...)
	return fmt.Errorf("%s failed: %w", operation, err)
}

// IsNotFoundError 检查是否为资源不存在错误
func IsNotFoundError(err error) bool {
	return errors.IsNotFound(err)
}

// IsAlreadyExistsError 检查是否为资源已存在错误
func IsAlreadyExistsError(err error) bool {
	return errors.IsAlreadyExists(err)
}

// IsConflictError 检查是否为冲突错误
func IsConflictError(err error) bool {
	return errors.IsConflict(err)
}

// IsForbiddenError 检查是否为权限不足错误
func IsForbiddenError(err error) bool {
	return errors.IsForbidden(err)
}

func HandleK8sError(err error, operation, resource, name, namespace string, clusterID int) error {
	if err == nil {
		return nil
	}

	k8sErr := NewK8sError(operation, resource, name, namespace, clusterID, err)

	// 针对不同类型的错误返回更友好的信息
	switch {
	case IsNotFoundError(err):
		return fmt.Errorf("%s not found", resource)
	case IsAlreadyExistsError(err):
		return fmt.Errorf("%s already exists", resource)
	case IsForbiddenError(err):
		return fmt.Errorf("insufficient permissions to %s %s", operation, resource)
	case IsConflictError(err):
		return fmt.Errorf("conflict occurred while trying to %s %s", operation, resource)
	default:
		return k8sErr
	}
}
