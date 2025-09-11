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
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"k8s.io/client-go/kubernetes"
)

// CleanClusterSensitiveInfo 清理集群敏感信息
func CleanClusterSensitiveInfo(cluster *model.K8sCluster) {
	if cluster != nil {
		cluster.KubeConfigContent = ""
	}
}

// CleanClusterSensitiveInfoList 清理集群列表中的敏感信息
func CleanClusterSensitiveInfoList(clusters []*model.K8sCluster) {
	for _, cluster := range clusters {
		CleanClusterSensitiveInfo(cluster)
	}
}

// ValidateClusterExists 验证集群是否存在
func ValidateClusterExists(cluster *model.K8sCluster, clusterID int) error {
	if cluster == nil {
		return fmt.Errorf("集群不存在，ID: %d", clusterID)
	}
	return nil
}

// ValidateResourceQuantities 验证资源配额格式
func ValidateResourceQuantities(cluster *model.K8sCluster) error {
	// TODO: 实现资源配额验证逻辑
	// 这里可以验证CPU、内存等资源配额的格式是否正确
	return nil
}

// AddClusterResourceLimit 添加集群资源限制
func AddClusterResourceLimit(ctx context.Context, client kubernetes.Interface, cluster *model.K8sCluster) error {
	// TODO: 实现添加集群资源限制的逻辑
	// 这里可以创建ResourceQuota或LimitRange等资源
	return nil
}
