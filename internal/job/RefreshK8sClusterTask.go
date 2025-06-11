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

package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/client"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao/admin"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type RefreshK8sClusterPayload struct {
	ClusterID int `json:"cluster_id"`
}

type RefreshK8sClusterTask struct {
	l      *zap.Logger
	client client.K8sClient
	dao    admin.ClusterDAO
}

func NewRefreshK8sClusterTask(l *zap.Logger, client client.K8sClient, dao admin.ClusterDAO) *RefreshK8sClusterTask {
	return &RefreshK8sClusterTask{
		l:      l,
		client: client,
		dao:    dao,
	}
}

// ProcessTask 处理刷新K8s集群状态的任务
func (t *RefreshK8sClusterTask) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload RefreshK8sClusterPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		t.l.Error("RefreshK8sClusterTask: 解析任务载荷失败", zap.Error(err))
		return fmt.Errorf("RefreshK8sClusterTask: 解析任务载荷失败: %w", err)
	}

	t.l.Info("RefreshK8sClusterTask: 开始处理刷新K8s集群状态任务", zap.Int("clusterID", payload.ClusterID))

	const (
		maxRetries     = 3               // 最大重试次数
		baseRetryDelay = 3 * time.Second // 基础重试延迟
	)

	var (
		retryCount int
		lastError  error
	)

	for retryCount < maxRetries {
		select {
		case <-ctx.Done():
			t.l.Error("RefreshK8sClusterTask: 任务被取消", zap.Int("clusterID", payload.ClusterID))
			return ctx.Err()
		default:
			// 获取集群信息
			cluster, err := t.dao.GetClusterByID(ctx, payload.ClusterID)
			if err != nil {
				lastError = fmt.Errorf("获取集群信息失败: %w", err)
				t.l.Error("RefreshK8sClusterTask: 获取集群信息失败",
					zap.Int("clusterID", payload.ClusterID),
					zap.Error(err))
				retryCount++
				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					t.l.Info("RefreshK8sClusterTask: 任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}
				return lastError
			}

			if cluster == nil {
				t.l.Error("RefreshK8sClusterTask: 集群不存在", zap.Int("clusterID", payload.ClusterID))
				return fmt.Errorf("集群不存在，ID: %d", payload.ClusterID)
			}

			// 检查集群连接状态
			if err := t.client.CheckClusterConnection(payload.ClusterID); err != nil {
				t.l.Error("RefreshK8sClusterTask: 集群连接检查失败",
					zap.Int("clusterID", payload.ClusterID),
					zap.Error(err))

				// 更新集群状态为错误
				if updateErr := t.dao.UpdateClusterStatus(ctx, payload.ClusterID, "ERROR"); updateErr != nil {
					t.l.Error("RefreshK8sClusterTask: 更新集群状态失败",
						zap.Int("clusterID", payload.ClusterID),
						zap.Error(updateErr))
				}

				lastError = fmt.Errorf("集群连接检查失败: %w", err)
				retryCount++
				if retryCount < maxRetries {
					delay := time.Duration(retryCount) * baseRetryDelay
					t.l.Info("RefreshK8sClusterTask: 任务重试",
						zap.Int("重试次数", retryCount),
						zap.Duration("延迟时间", delay),
						zap.Error(lastError))
					time.Sleep(delay)
					continue
				}
				return lastError
			}

			// 更新集群状态为成功
			if err := t.dao.UpdateClusterStatus(ctx, payload.ClusterID, "SUCCESS"); err != nil {
				t.l.Error("RefreshK8sClusterTask: 更新集群状态失败",
					zap.Int("clusterID", payload.ClusterID),
					zap.Error(err))
				return fmt.Errorf("更新集群状态失败: %w", err)
			}

			t.l.Info("RefreshK8sClusterTask: 成功刷新K8s集群状态", zap.Int("clusterID", payload.ClusterID))
			return nil
		}
	}

	return fmt.Errorf("达到最大重试次数，任务失败: %w", lastError)
}
