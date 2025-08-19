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

package startup

import (
	"context"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"go.uber.org/zap"
)

type ApplicationBootstrap interface {
	InitializeK8sClients(ctx context.Context) error
}

type applicationBootstrap struct {
	clusterMgr manager.ClusterManager
	logger     *zap.Logger
}

func NewApplicationBootstrap(clusterMgr manager.ClusterManager, logger *zap.Logger) ApplicationBootstrap {
	return &applicationBootstrap{
		clusterMgr: clusterMgr,
		logger:     logger,
	}
}

func (ab *applicationBootstrap) InitializeK8sClients(ctx context.Context) error {
	ab.logger.Info("开始初始化K8s集群客户端")

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := ab.clusterMgr.InitializeAllClusters(ctx); err != nil {
		ab.logger.Error("初始化K8s集群客户端失败", zap.Error(err))
		return err
	}

	ab.logger.Info("成功初始化K8s集群客户端")
	return nil
}
