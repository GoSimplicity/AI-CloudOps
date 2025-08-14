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

	treeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, port int, treeNodeIDs []int) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l       *zap.Logger
	treeDao treeDao.TreeNodeDAO
}

func NewNotAuthService(l *zap.Logger) NotAuthService {
	return &notAuthService{
		l: l,
	}
}

// BuildPrometheusServiceDiscovery 构建 Prometheus HTTP SD 目标组
func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, port int, treeNodeIDs []int) ([]*targetgroup.Group, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("端口无效")
	}
	if len(treeNodeIDs) == 0 {
		return nil, fmt.Errorf("tree_node_ids 不能为空")
	}
	if n.treeDao == nil {
		return nil, fmt.Errorf("内部配置缺失: TreeNodeDAO 未初始化")
	}

	// 收集所有绑定资源
	targetsSet := make(map[string]struct{})
	var orderedAddrs []string

	for _, nodeID := range treeNodeIDs {
		node, err := n.treeDao.GetNode(ctx, nodeID)
		if err != nil {
			return nil, err
		}
		for _, res := range node.TreeLocalResources {
			if res.IpAddr == "" {
				continue
			}
			addr := fmt.Sprintf("%s:%d", res.IpAddr, port)
			if _, ok := targetsSet[addr]; ok {
				continue
			}
			targetsSet[addr] = struct{}{}
			orderedAddrs = append(orderedAddrs, addr)
		}
	}

	// 生成目标组（保持顺序稳定）
	var targets []model.LabelSet
	for _, addr := range orderedAddrs {
		targets = append(targets, model.LabelSet{
			model.AddressLabel: model.LabelValue(addr),
		})
	}

	if len(targets) == 0 {
		// 返回空切片，Prometheus 会忽略
		return []*targetgroup.Group{}, nil
	}

	tg := &targetgroup.Group{
		Targets: targets,
		Labels: model.LabelSet{
			"instance":      model.LabelValue("ai-cloudops-tree"),
			"tree_node_ids": model.LabelValue(strings.Join(intSliceToStrings(treeNodeIDs), ",")),
		},
	}

	return []*targetgroup.Group{tg}, nil
}

// intSliceToStrings 将 int 切片安全转换为字符串切片
func intSliceToStrings(nums []int) []string {
	if len(nums) == 0 {
		return []string{}
	}
	ss := make([]string, 0, len(nums))
	for _, n := range nums {
		ss = append(ss, fmt.Sprintf("%d", n))
	}
	return ss
}
