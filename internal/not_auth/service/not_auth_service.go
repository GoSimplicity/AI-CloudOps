package service

import (
	"context"
	"fmt"
	treeNode "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/tree_node"
	pkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/general"
	promPkg "github.com/GoSimplicity/AI-CloudOps/pkg/utils/prometheus"
	promModel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l           *zap.Logger
	treeNodeDao treeNode.TreeNodeDAO
}

func NewNotAuthService(l *zap.Logger, treeNodeDao treeNode.TreeNodeDAO) NotAuthService {
	return &notAuthService{
		l:           l,
		treeNodeDao: treeNodeDao,
	}
}

// BuildPrometheusServiceDiscovery 构建 Prometheus 服务发现的目标组，支持多个标签
func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error) {
	leafNodeIntList, err := pkg.ConvertToIntList(leafNodeIdList)
	if err != nil {
		n.l.Warn("无效的 leafNodeIdList", zap.Strings("leafNodeIdList", leafNodeIdList), zap.Error(err))
		return nil, err
	}

	leafNodeList, err := n.treeNodeDao.GetByIDs(ctx, leafNodeIntList)
	if err != nil {
		n.l.Error("根据 leafNodeIdList 获取树节点失败", zap.Ints("leafNodeIdList", leafNodeIntList), zap.Error(err))
		return nil, fmt.Errorf("获取树节点失败: %w", err)
	}

	// 初始化目标组列表
	tgList := make([]*targetgroup.Group, 0, len(leafNodeList))

	for _, node := range leafNodeList {
		// 检查节点是否绑定了 ECS 实例
		if node.BindEcs == nil || len(node.BindEcs) == 0 {
			n.l.Warn("leaf node without bind ecs", zap.Int("node_id", node.ID))
			continue
		}

		for _, ecs := range node.BindEcs {
			// 确保 ecs.Tags 有偶数个元素，以便成对解析
			if len(ecs.Tags) == 0 || len(ecs.Tags)%2 != 0 {
				n.l.Warn("ECS 实例的 Tags 格式不正确，必须为偶数个元素", zap.Int("node_id", node.ID), zap.Any("ecs", ecs))
				continue
			}

			// 构建 Prometheus 的标签映射
			labels, err := promPkg.ParseTags(ecs.Tags)
			if err != nil {
				n.l.Warn("解析 ECS 实例的 Tags 失败", zap.Int("node_id", node.ID), zap.Any("ecs", ecs), zap.Error(err))
				continue
			}

			target := fmt.Sprintf("%s:%d", ecs.IpAddr, port)

			// 创建目标组
			tg := &targetgroup.Group{
				Targets: []promModel.LabelSet{
					{
						promModel.AddressLabel: promModel.LabelValue(target),
					},
				},
				Labels: labels,
			}

			tgList = append(tgList, tg)
		}
	}

	return tgList, nil
}
