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
	"net"
	"strconv"
	"strings"

	treeDao "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

const (
	// MaxDiscoveryNodeIDs 限制一次公共服务发现的数据库扇出
	MaxDiscoveryNodeIDs     = 100
	MaxDiscoveryIDValues    = 1000
	MaxDiscoveryIDsLength   = 4096
	MaxDiscoveryQueryLength = 8192
)

// NormalizeDiscoveryNodeIDs 保序去重，直接调用服务也必须遵守同一边界
func NormalizeDiscoveryNodeIDs(ids []int) ([]int, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("tree_node_ids 不能为空")
	}
	if len(ids) > MaxDiscoveryIDValues {
		return nil, fmt.Errorf("tree_node_ids 条目过多")
	}
	unique := make([]int, 0, min(len(ids), MaxDiscoveryNodeIDs))
	seen := make(map[int]struct{}, cap(unique))
	for _, id := range ids {
		if id <= 0 {
			return nil, fmt.Errorf("tree_node_ids 必须是正整数")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if len(unique) >= MaxDiscoveryNodeIDs {
			return nil, fmt.Errorf("tree_node_ids 最多包含 %d 个不同节点", MaxDiscoveryNodeIDs)
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique, nil
}

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, port int, treeNodeIDs []int) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l       *zap.Logger
	treeDao treeDao.TreeNodeDAO
}

func NewNotAuthService(l *zap.Logger, treeNodeDAO treeDao.TreeNodeDAO) NotAuthService {
	return &notAuthService{
		l:       l,
		treeDao: treeNodeDAO,
	}
}

// BuildPrometheusServiceDiscovery 构建 Prometheus HTTP SD 目标组
func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, port int, treeNodeIDs []int) ([]*targetgroup.Group, error) {
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("端口无效")
	}
	ids, err := NormalizeDiscoveryNodeIDs(treeNodeIDs)
	if err != nil {
		return nil, err
	}
	treeNodeIDs = ids
	if n.treeDao == nil {
		return nil, fmt.Errorf("内部配置缺失: TreeNodeDAO 未初始化")
	}

	targetsSet := make(map[string]struct{})
	var orderedAddrs []string

	for _, nodeID := range treeNodeIDs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		node, err := n.treeDao.GetNode(ctx, nodeID)
		if err != nil {
			return nil, err
		}
		for _, res := range node.TreeLocalResources {
			if res.IpAddr == "" {
				continue
			}
			addr := net.JoinHostPort(res.IpAddr, strconv.Itoa(port))
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
