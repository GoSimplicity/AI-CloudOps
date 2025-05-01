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

	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l           *zap.Logger
}

func NewNotAuthService(l *zap.Logger) NotAuthService {
	return &notAuthService{
		l: l,
	}
}

// BuildPrometheusServiceDiscovery 构建 Prometheus 服务发现的目标组，支持多个标签
func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error) {
	// leafNodeIntList, err := promPkg.ConvertToIntList(leafNodeIdList)
	// if err != nil {
	// 	n.l.Warn("无效的 leafNodeIdList", zap.Strings("leafNodeIdList", leafNodeIdList), zap.Error(err))
	// 	return nil, err
	// }

	// leafNodeList, err := n.treeNodeDao.GetByIDs(ctx, leafNodeIntList)
	// if err != nil {
	// 	n.l.Error("根据 leafNodeIdList 获取树节点失败", zap.Ints("leafNodeIdList", leafNodeIntList), zap.Error(err))
	// 	return nil, fmt.Errorf("获取树节点失败: %w", err)
	// }

	// 初始化目标组列表
	// tgList := make([]*targetgroup.Group, 0, len(leafNodeList))

	// for _, node := range leafNodeList {
	// 	// 检查节点是否绑定了 ECS 实例
	// 	if node.BindEcs == nil || len(node.BindEcs) == 0 {
	// 		n.l.Warn("leaf node without bind ecs", zap.Int("node_id", node.ID))
	// 		continue
	// 	}

	// 	for _, ecs := range node.BindEcs {
	// 		// 检查 ecs.Tags 是否为空或有偶数个元素
	// 		if len(ecs.Tags) == 0 {
	// 			// 如果标签为空,直接跳过
	// 			continue
	// 		}
	// 		if len(ecs.Tags)%2 != 0 {
	// 			n.l.Warn("ECS 实例的 Tags 格式不正确，必须为偶数个元素", zap.Int("node_id", node.ID), zap.Any("ecs", ecs))
	// 			continue
	// 		}

	// 		// 构建 Prometheus 的标签映射
	// 		labels, err := promPkg.ParseTags(ecs.Tags)
	// 		if err != nil {
	// 			n.l.Warn("解析 ECS 实例的 Tags 失败", zap.Int("node_id", node.ID), zap.Any("ecs", ecs), zap.Error(err))
	// 			continue
	// 		}

	// 		target := fmt.Sprintf("%s:%d", ecs.IpAddr, port)

	// 		// 创建目标组
	// 		tg := &targetgroup.Group{
	// 			Targets: []promModel.LabelSet{
	// 				{
	// 					promModel.AddressLabel: promModel.LabelValue(target),
	// 				},
	// 			},
	// 			Labels: labels,
	// 		}

	// 		tgList = append(tgList, tg)
	// 	}
	// }

	return nil, nil
}
