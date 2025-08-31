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
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type TaintService interface {
	CheckTaintYaml(ctx context.Context, taint *model.CheckTaintYamlReq) error
	EnableSwitchNode(ctx context.Context, req *model.NodeCordonReq) error
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

func NewTaintService(manager manager.TaintManager, logger *zap.Logger) TaintService {
	return &taintService{
		manager: manager,
		logger:  logger,
	}
}

// AddNodeTaint implements TaintService.
func (t *taintService) AddNodeTaint(ctx context.Context, req *model.AddNodeTaintsReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(req.Taints) == 0 {
		return fmt.Errorf("污点列表不能为空")
	}

	// 构建污点YAML
	yamlData, err := utils.BuildTaintYaml(req.Taints)
	if err != nil {
		t.logger.Error("构建污点YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return fmt.Errorf("构建污点YAML失败: %w", err)
	}

	if err := t.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, yamlData, manager.ModTypeAdd); err != nil {
		t.logger.Error("添加节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// DeleteNodeTaint implements TaintService.
func (t *taintService) DeleteNodeTaint(ctx context.Context, req *model.DeleteNodeTaintsReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(req.TaintKeys) == 0 {
		return fmt.Errorf("污点键列表不能为空")
	}

	// 构建要删除的污点键字符串
	taintKeysStr := strings.Join(req.TaintKeys, ",")

	// 构建删除污点的YAML（使用空值表示删除）
	var taintsToDelete []model.NodeTaintEntity
	for _, key := range req.TaintKeys {
		taintsToDelete = append(taintsToDelete, model.NodeTaintEntity{
			Key: key,
		})
	}

	yamlData, err := utils.BuildTaintYaml(taintsToDelete)
	if err != nil {
		t.logger.Error("构建删除污点YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return fmt.Errorf("构建删除污点YAML失败: %w", err)
	}

	if err := t.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, yamlData, manager.ModTypeDelete); err != nil {
		t.logger.Error("删除节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.String("taintKeys", taintKeysStr),
			zap.Error(err))
		return err
	}

	return nil
}

// SwitchNodeSchedule implements TaintService.
func (t *taintService) SwitchNodeSchedule(ctx context.Context, req *model.SwitchNodeScheduleReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	if err := t.manager.EnableSwitchNode(ctx, req.ClusterID, req.NodeName, req.Enable); err != nil {
		t.logger.Error("切换节点调度状态失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Bool("enable", req.Enable),
			zap.Error(err))
		return err
	}

	return nil
}

// AddOrUpdateNodeTaint implements TaintService.
func (t *taintService) AddOrUpdateNodeTaint(ctx context.Context, req *model.AddNodeTaintsReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(req.Taints) == 0 {
		return fmt.Errorf("污点列表不能为空")
	}

	// 构建污点YAML
	yamlData, err := utils.BuildTaintYaml(req.Taints)
	if err != nil {
		t.logger.Error("构建污点YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return fmt.Errorf("构建污点YAML失败: %w", err)
	}

	if err := t.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, yamlData, manager.ModTypeUpdate); err != nil {
		t.logger.Error("添加或更新节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// BatchEnableSwitchNodes implements TaintService.
func (t *taintService) EnableSwitchNode(ctx context.Context, req *model.NodeCordonReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	// NodeCordonReq 用于禁用节点调度，所以这里 scheduleEnable 为 false
	if err := t.manager.EnableSwitchNode(ctx, req.ClusterID, req.NodeName, false); err != nil {
		t.logger.Error("禁用节点调度失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// CheckTaintYaml implements TaintService.
func (t *taintService) CheckTaintYaml(ctx context.Context, req *model.CheckTaintYamlReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if req.YamlData == "" {
		return fmt.Errorf("YAML数据不能为空")
	}

	if err := t.manager.CheckTaintYaml(ctx, req.ClusterID, req.NodeName, req.YamlData); err != nil {
		t.logger.Error("检查污点YAML配置失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// DrainPods implements TaintService.
func (t *taintService) DrainPods(ctx context.Context, req *model.DrainNodeReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	if err := t.manager.DrainPods(ctx, req.ClusterID, req.NodeName); err != nil {
		t.logger.Error("驱逐节点Pod失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}
