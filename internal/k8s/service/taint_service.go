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

package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type TaintService interface {
	// CheckTaintYaml 检查 Taint YAML 配置是否合法
	CheckTaintYaml(ctx context.Context, taint *model.CheckTaintYamlReq) error
	BatchEnableSwitchNodes(ctx context.Context, req *model.NodeCordonReq) error
	AddOrUpdateNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error
	DrainPods(ctx context.Context, req *model.DrainNodeReq) error
	DeleteNodeTaint(ctx context.Context, taint *model.DeleteNodeTaintsReq) error
	SwitchNodeSchedule(ctx context.Context, req *model.SwitchNodeScheduleReq) error
	AddNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error
}

type taintService struct {
	manager manager.TaintManager
	logger  *zap.Logger
}

// AddNodeTaint implements TaintService.
func (t *taintService) AddNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error {
	panic("unimplemented")
}

// DeleteNodeTaint implements TaintService.
func (t *taintService) DeleteNodeTaint(ctx context.Context, taint *model.DeleteNodeTaintsReq) error {
	panic("unimplemented")
}

// SwitchNodeSchedule implements TaintService.
func (t *taintService) SwitchNodeSchedule(ctx context.Context, req *model.SwitchNodeScheduleReq) error {
	panic("unimplemented")
}

// AddOrUpdateNodeTaint implements TaintService.
func (t *taintService) AddOrUpdateNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error {
	panic("unimplemented")
}

// BatchEnableSwitchNodes implements TaintService.
func (t *taintService) BatchEnableSwitchNodes(ctx context.Context, req *model.NodeCordonReq) error {
	panic("unimplemented")
}

// CheckTaintYaml implements TaintService.
func (t *taintService) CheckTaintYaml(ctx context.Context, taint *model.CheckTaintYamlReq) error {
	panic("unimplemented")
}

// DrainPods implements TaintService.
func (t *taintService) DrainPods(ctx context.Context, req *model.DrainNodeReq) error {
	panic("unimplemented")
}

func NewTaintService(manager manager.TaintManager, logger *zap.Logger) TaintService {
	return &taintService{
		manager: manager,
		logger:  logger,
	}
}

// // CheckTaintYaml 检查 Taint YAML 配置是否合法
// func (t *taintService) CheckTaintYaml(ctx context.Context, req *model.AddNodeTaintsReq) error {
// 	return t.manager.CheckTaintYaml(ctx, req.ClusterID, req.NodeName, req.TaintYaml)
// }

// // BatchEnableSwitchNodes 批量启用或禁用节点
// func (t *taintService) BatchEnableSwitchNodes(ctx context.Context, req *model.BatchEnableSwitchNodesReq) error {
// 	return t.manager.BatchEnableSwitchNodes(ctx, req.ClusterID, req.NodeName, req.ScheduleEnable)
// }

// // AddOrUpdateNodeTaint 更新节点的 Taint
// func (t *taintService) AddOrUpdateNodeTaint(ctx context.Context, req *model.TaintK8sNodesReq) error {
// 	return t.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, req.TaintYaml, req.ModType)
// }

// // DrainPods 并发驱逐 Pods
// func (t *taintService) DrainPods(ctx context.Context, req *model.K8sClusterNodesReq) error {
// 	return t.manager.DrainPods(ctx, req.ClusterID, req.NodeName)
// }
