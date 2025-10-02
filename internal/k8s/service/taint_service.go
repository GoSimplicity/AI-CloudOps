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

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/utils"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

type TaintService interface {
	CheckTaintYaml(ctx context.Context, taint *model.CheckTaintYamlReq) error
	AddNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error
	DeleteNodeTaint(ctx context.Context, taint *model.DeleteNodeTaintsReq) error
	AddOrUpdateNodeTaint(ctx context.Context, taint *model.AddNodeTaintsReq) error
	DrainPods(ctx context.Context, req *model.DrainNodeReq) error
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

// AddNodeTaint 添加节点污点
func (s *taintService) AddNodeTaint(ctx context.Context, req *model.AddNodeTaintsReq) error {
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
	yamlData, err := utils.BuildTaintYamlFromK8sTaints(req.Taints)
	if err != nil {
		s.logger.Error("构建污点YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return fmt.Errorf("构建污点YAML失败: %w", err)
	}

	if err := s.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, yamlData, manager.ModTypeAdd); err != nil {
		s.logger.Error("添加节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// DeleteNodeTaint 删除节点污点
func (s *taintService) DeleteNodeTaint(ctx context.Context, req *model.DeleteNodeTaintsReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if len(req.TaintKeys) == 0 {
		return fmt.Errorf("污点键列表不能为空")
	}

	if err := s.manager.DeleteNodeTaintsByKeys(ctx, req.ClusterID, req.NodeName, req.TaintKeys); err != nil {
		s.logger.Error("删除节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Strings("taintKeys", req.TaintKeys),
			zap.Error(err))
		return err
	}

	return nil
}

// AddOrUpdateNodeTaint 添加或更新节点污点
func (s *taintService) AddOrUpdateNodeTaint(ctx context.Context, req *model.AddNodeTaintsReq) error {
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
	yamlData, err := utils.BuildTaintYamlFromK8sTaints(req.Taints)
	if err != nil {
		s.logger.Error("构建污点YAML失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return fmt.Errorf("构建污点YAML失败: %w", err)
	}

	if err := s.manager.AddOrUpdateNodeTaint(ctx, req.ClusterID, req.NodeName, yamlData, manager.ModTypeUpdate); err != nil {
		s.logger.Error("添加或更新节点污点失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// CheckTaintYaml 检查污点YAML配置
func (s *taintService) CheckTaintYaml(ctx context.Context, req *model.CheckTaintYamlReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}
	if req.YamlData == "" {
		return fmt.Errorf("YAML数据不能为空")
	}

	if err := s.manager.CheckTaintYaml(ctx, req.ClusterID, req.NodeName, req.YamlData); err != nil {
		s.logger.Error("检查污点YAML配置失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}

// DrainPods 驱逐节点Pod
func (s *taintService) DrainPods(ctx context.Context, req *model.DrainNodeReq) error {
	if req.ClusterID <= 0 {
		return fmt.Errorf("集群ID不能为空")
	}
	if req.NodeName == "" {
		return fmt.Errorf("节点名称不能为空")
	}

	if err := s.manager.DrainPods(ctx, req.ClusterID, req.NodeName); err != nil {
		s.logger.Error("驱逐节点Pod失败",
			zap.Int("clusterID", req.ClusterID),
			zap.String("nodeName", req.NodeName),
			zap.Error(err))
		return err
	}

	return nil
}
