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

import "github.com/hibiken/asynq"

type Routes struct {
	CreateK8sCluster *CreateK8sClusterTask
	UpdateK8sCluster *UpdateK8sClusterTask
	TimedTask        *TimedTask
}

func NewRoutes(createK8sCluster *CreateK8sClusterTask, updateK8sCluster *UpdateK8sClusterTask, timedTask *TimedTask) *Routes {
	return &Routes{
		CreateK8sCluster: createK8sCluster,
		UpdateK8sCluster: updateK8sCluster,
		TimedTask:        timedTask,
	}
}

func (r *Routes) RegisterHandlers() *asynq.ServeMux {
	mux := asynq.NewServeMux()

	// 注册任务
	// mux.HandleFunc(DeferCreateK8sCluster, r.CreateK8sCluster.ProcessTask)
	// mux.HandleFunc(DeferUpdateK8sCluster, r.UpdateK8sCluster.ProcessTask)
	mux.HandleFunc(DeferTimedTask, r.TimedTask.ProcessTask)

	return mux
}
